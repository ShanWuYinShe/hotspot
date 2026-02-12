//! 布隆过滤器工具
//!
//! 使用 Redis 位图实现的布隆过滤器，用于快速过滤不存在的资源请求
//! 防止恶意请求穿透缓存直接查询数据库
//!
//! 特点：
//! - 如果返回 false，资源一定不存在
//! - 如果返回 true，资源可能存在（存在假阳性）
//! - 假阳性率约 1%（使用 7 个哈希函数和 10 bits/element）

use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use std::collections::hash_map::DefaultHasher;
use std::hash::{Hash, Hasher};

/// 布隆过滤器配置
#[derive(Debug, Clone)]
pub struct BloomFilterConfig {
    /// 每个元素使用的位数
    pub bits_per_element: usize,
    /// 哈希函数数量
    pub hash_count: usize,
    /// 预期元素数量
    pub expected_elements: usize,
}

impl Default for BloomFilterConfig {
    fn default() -> Self {
        Self {
            bits_per_element: 10,
            hash_count: 7,
            expected_elements: 1_000_000, // 默认支持 100 万元素
        }
    }
}

/// Redis 布隆过滤器
#[derive(Clone)]
pub struct RedisBloomFilter {
    redis: ConnectionManager,
    config: BloomFilterConfig,
}

impl RedisBloomFilter {
    pub fn new(redis: ConnectionManager) -> Self {
        Self::with_config(redis, BloomFilterConfig::default())
    }

    pub fn with_config(redis: ConnectionManager, config: BloomFilterConfig) -> Self {
        Self { redis, config }
    }

    /// 计算给定键的哈希值数组
    fn get_hash_indices(&self, key: &str, filter_name: &str) -> Vec<usize> {
        let total_bits = self.config.bits_per_element * self.config.expected_elements;
        let mut indices = Vec::with_capacity(self.config.hash_count);

        // 使用两个基础哈希值生成多个哈希值（Kirsch-Mitzenmacher 优化）
        let mut hasher1 = DefaultHasher::new();
        key.hash(&mut hasher1);
        filter_name.hash(&mut hasher1);
        let h1 = hasher1.finish() as usize;

        let mut hasher2 = DefaultHasher::new();
        filter_name.hash(&mut hasher2);
        key.hash(&mut hasher2);
        "salt".hash(&mut hasher2);
        let h2 = hasher2.finish() as usize;

        for i in 0..self.config.hash_count {
            let index = (h1.wrapping_add(i.wrapping_mul(h2))) % total_bits;
            indices.push(index);
        }

        indices
    }

    /// 获取 Redis 键名
    fn get_redis_key(&self, filter_name: &str) -> String {
        format!("bloom:{}", filter_name)
    }

    /// 添加元素到布隆过滤器
    pub async fn add(&self, filter_name: &str, key: &str) -> Result<(), redis::RedisError> {
        let redis_key = self.get_redis_key(filter_name);
        let indices = self.get_hash_indices(key, filter_name);

        let mut conn = self.redis.clone();
        let mut pipe = redis::pipe();

        for index in indices {
            pipe.cmd("SETBIT").arg(&redis_key).arg(index).arg(1);
        }

        pipe.query_async::<Vec<i32>>(&mut conn).await?;
        Ok(())
    }

    /// 批量添加元素
    pub async fn add_batch(
        &self,
        filter_name: &str,
        keys: &[String],
    ) -> Result<(), redis::RedisError> {
        if keys.is_empty() {
            return Ok(());
        }

        let redis_key = self.get_redis_key(filter_name);
        let mut conn = self.redis.clone();
        let mut pipe = redis::pipe();

        for key in keys {
            let indices = self.get_hash_indices(key, filter_name);
            for index in indices {
                pipe.cmd("SETBIT").arg(&redis_key).arg(index).arg(1);
            }
        }

        pipe.query_async::<Vec<i32>>(&mut conn).await?;
        Ok(())
    }

    /// 检查元素是否可能存在
    ///
    /// 返回值：
    /// - false: 元素一定不存在
    /// - true: 元素可能存在（有假阳性）
    pub async fn might_contain(
        &self,
        filter_name: &str,
        key: &str,
    ) -> Result<bool, redis::RedisError> {
        let redis_key = self.get_redis_key(filter_name);
        let indices = self.get_hash_indices(key, filter_name);

        let mut conn = self.redis.clone();
        let mut pipe = redis::pipe();

        for index in &indices {
            pipe.cmd("GETBIT").arg(&redis_key).arg(*index);
        }

        let results: Vec<i32> = pipe.query_async(&mut conn).await?;

        // 所有位都为 1 才可能存在
        Ok(results.iter().all(|&bit| bit == 1))
    }

    /// 批量检查元素是否可能存在
    pub async fn might_contain_batch(
        &self,
        filter_name: &str,
        keys: &[String],
    ) -> Result<Vec<bool>, redis::RedisError> {
        if keys.is_empty() {
            return Ok(vec![]);
        }

        let redis_key = self.get_redis_key(filter_name);
        let mut conn = self.redis.clone();

        // 收集所有索引和对应的键位置
        let mut all_indices: Vec<(usize, Vec<usize>)> = Vec::with_capacity(keys.len());
        for (i, key) in keys.iter().enumerate() {
            let indices = self.get_hash_indices(key, filter_name);
            all_indices.push((i, indices));
        }

        // 批量查询
        let mut pipe = redis::pipe();
        for (_, indices) in &all_indices {
            for index in indices {
                pipe.cmd("GETBIT").arg(&redis_key).arg(*index);
            }
        }

        let results: Vec<i32> = pipe.query_async(&mut conn).await?;

        // 解析结果
        let mut output = vec![false; keys.len()];
        let mut result_idx = 0;

        for (key_idx, indices) in all_indices {
            let mut all_set = true;
            for _ in 0..indices.len() {
                if results.get(result_idx).copied().unwrap_or(0) != 1 {
                    all_set = false;
                }
                result_idx += 1;
            }
            output[key_idx] = all_set;
        }

        Ok(output)
    }

    /// 清空布隆过滤器
    pub async fn clear(&self, filter_name: &str) -> Result<(), redis::RedisError> {
        let redis_key = self.get_redis_key(filter_name);
        let mut conn = self.redis.clone();
        conn.del::<_, ()>(&redis_key).await?;
        Ok(())
    }

    /// 获取布隆过滤器的大小（字节数）
    pub async fn size_bytes(&self, filter_name: &str) -> Result<usize, redis::RedisError> {
        let redis_key = self.get_redis_key(filter_name);
        let mut conn = self.redis.clone();
        let len: usize = conn.strlen(&redis_key).await?;
        Ok(len)
    }
}

/// 布隆过滤器名称常量
pub mod filter_names {
    /// 用户 ID 过滤器
    pub const USER_IDS: &str = "user_ids";
    /// 群组 ID 过滤器
    pub const GROUP_IDS: &str = "group_ids";
    /// 瞬间 ID 过滤器
    pub const MOMENT_IDS: &str = "moment_ids";
    /// 评论 ID 过滤器
    pub const COMMENT_IDS: &str = "comment_ids";
    /// 消息 ID 过滤器
    pub const MESSAGE_IDS: &str = "message_ids";
}

const BLOOM_READY_KEY: &str = "bloom:ready";

/// 资源存在性检查器
///
/// 封装布隆过滤器检查逻辑，提供更友好的 API
#[derive(Clone)]
pub struct ResourceExistenceChecker {
    bloom: RedisBloomFilter,
}

impl ResourceExistenceChecker {
    pub fn new(redis: ConnectionManager) -> Self {
        Self {
            bloom: RedisBloomFilter::new(redis),
        }
    }

    /// 检查布隆过滤器是否已完成初始化
    pub async fn is_ready(&self) -> Result<bool, redis::RedisError> {
        let mut conn = self.bloom.redis.clone();
        let exists: bool = redis::cmd("EXISTS")
            .arg(BLOOM_READY_KEY)
            .query_async(&mut conn)
            .await?;
        Ok(exists)
    }

    /// 标记布隆过滤器已完成初始化
    pub async fn set_ready(&self) -> Result<(), redis::RedisError> {
        let mut conn = self.bloom.redis.clone();
        let _: () = redis::cmd("SET")
            .arg(BLOOM_READY_KEY)
            .arg("1")
            .query_async(&mut conn)
            .await?;
        Ok(())
    }

    /// 检查用户是否可能存在
    pub async fn user_might_exist(&self, user_id: i64) -> Result<bool, redis::RedisError> {
        self.bloom
            .might_contain(filter_names::USER_IDS, &user_id.to_string())
            .await
    }

    /// 检查群组是否可能存在
    pub async fn group_might_exist(&self, group_id: &str) -> Result<bool, redis::RedisError> {
        self.bloom
            .might_contain(filter_names::GROUP_IDS, group_id)
            .await
    }

    /// 检查瞬间是否可能存在
    pub async fn moment_might_exist(&self, moment_id: &str) -> Result<bool, redis::RedisError> {
        self.bloom
            .might_contain(filter_names::MOMENT_IDS, moment_id)
            .await
    }

    /// 检查评论是否可能存在
    pub async fn comment_might_exist(&self, comment_id: &str) -> Result<bool, redis::RedisError> {
        self.bloom
            .might_contain(filter_names::COMMENT_IDS, comment_id)
            .await
    }

    /// 检查消息是否可能存在
    pub async fn message_might_exist(&self, message_id: &str) -> Result<bool, redis::RedisError> {
        self.bloom
            .might_contain(filter_names::MESSAGE_IDS, message_id)
            .await
    }

    /// 记录消息存在
    pub async fn record_message(&self, message_id: &str) -> Result<(), redis::RedisError> {
        self.bloom.add(filter_names::MESSAGE_IDS, message_id).await
    }

    /// 批量检查用户是否可能存在
    pub async fn user_might_exist_batch(
        &self,
        user_ids: &[i64],
    ) -> Result<Vec<bool>, redis::RedisError> {
        let keys: Vec<String> = user_ids.iter().map(|id| id.to_string()).collect();
        self.bloom
            .might_contain_batch(filter_names::USER_IDS, &keys)
            .await
    }

    /// 批量检查群组是否可能存在
    pub async fn group_might_exist_batch(
        &self,
        group_ids: &[String],
    ) -> Result<Vec<bool>, redis::RedisError> {
        self.bloom
            .might_contain_batch(filter_names::GROUP_IDS, group_ids)
            .await
    }

    /// 批量检查瞬间是否可能存在
    pub async fn moment_might_exist_batch(
        &self,
        moment_ids: &[String],
    ) -> Result<Vec<bool>, redis::RedisError> {
        self.bloom
            .might_contain_batch(filter_names::MOMENT_IDS, moment_ids)
            .await
    }

    /// 批量检查评论是否可能存在
    pub async fn comment_might_exist_batch(
        &self,
        comment_ids: &[String],
    ) -> Result<Vec<bool>, redis::RedisError> {
        self.bloom
            .might_contain_batch(filter_names::COMMENT_IDS, comment_ids)
            .await
    }

    /// 批量检查消息是否可能存在
    pub async fn message_might_exist_batch(
        &self,
        message_ids: &[String],
    ) -> Result<Vec<bool>, redis::RedisError> {
        self.bloom
            .might_contain_batch(filter_names::MESSAGE_IDS, message_ids)
            .await
    }

    /// 批量同步用户到布隆过滤器
    pub async fn sync_users(&self, user_ids: &[i64]) -> Result<(), redis::RedisError> {
        let keys: Vec<String> = user_ids.iter().map(|id| id.to_string()).collect();
        self.bloom.add_batch(filter_names::USER_IDS, &keys).await
    }

    /// 批量同步群组到布隆过滤器
    pub async fn sync_groups(&self, group_ids: &[String]) -> Result<(), redis::RedisError> {
        self.bloom
            .add_batch(filter_names::GROUP_IDS, group_ids)
            .await
    }

    /// 批量同步瞬间到布隆过滤器
    pub async fn sync_moments(&self, moment_ids: &[String]) -> Result<(), redis::RedisError> {
        self.bloom
            .add_batch(filter_names::MOMENT_IDS, moment_ids)
            .await
    }

    /// 批量同步评论到布隆过滤器
    pub async fn sync_comments(&self, comment_ids: &[String]) -> Result<(), redis::RedisError> {
        self.bloom
            .add_batch(filter_names::COMMENT_IDS, comment_ids)
            .await
    }

    /// 批量同步消息到布隆过滤器
    pub async fn sync_messages(&self, message_ids: &[String]) -> Result<(), redis::RedisError> {
        self.bloom
            .add_batch(filter_names::MESSAGE_IDS, message_ids)
            .await
    }

    /// 清空全部布隆过滤器（用于重新初始化）
    pub async fn clear_all(&self) -> Result<(), redis::RedisError> {
        for name in [
            filter_names::USER_IDS,
            filter_names::GROUP_IDS,
            filter_names::MOMENT_IDS,
            filter_names::COMMENT_IDS,
            filter_names::MESSAGE_IDS,
        ] {
            self.bloom.clear(name).await?;
        }
        Ok(())
    }

    /// 记录布隆过滤器大小（仅用于诊断/日志）
    pub async fn log_filter_sizes(&self) {
        for name in [
            filter_names::USER_IDS,
            filter_names::GROUP_IDS,
            filter_names::MOMENT_IDS,
            filter_names::COMMENT_IDS,
            filter_names::MESSAGE_IDS,
        ] {
            match self.bloom.size_bytes(name).await {
                Ok(size) => {
                    tracing::info!(filter = name, size_bytes = size, "bloom size");
                }
                Err(e) => {
                    tracing::warn!(filter = name, error = %e, "bloom size check failed");
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_hash_indices() {
        let config = BloomFilterConfig {
            bits_per_element: 10,
            hash_count: 7,
            expected_elements: 1000,
        };

        // 简单验证哈希索引生成
        let filter = RedisBloomFilter {
            redis: unsafe { std::mem::zeroed() }, // 测试时不使用 Redis
            config,
        };

        let indices = filter.get_hash_indices("test_key", "test_filter");
        assert_eq!(indices.len(), 7);

        // 相同的键应该产生相同的索引
        let indices2 = filter.get_hash_indices("test_key", "test_filter");
        assert_eq!(indices, indices2);

        // 不同的键应该产生不同的索引
        let indices3 = filter.get_hash_indices("different_key", "test_filter");
        assert_ne!(indices, indices3);
    }
}
