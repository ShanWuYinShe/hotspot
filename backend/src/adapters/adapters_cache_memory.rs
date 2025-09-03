/// 内存缓存适配器
/// 
/// 基于标准库HashMap的简单内存缓存实现

use async_trait::async_trait;
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{Duration, Instant};
use crate::domain::{DomainResult, DomainError};
use crate::ports::{CachePort, CacheStats};

/// 缓存条目
#[derive(Debug, Clone)]
struct CacheEntry {
    data: String,
    #[allow(dead_code)]
    created_at: Instant,
    expires_at: Option<Instant>,
}

impl CacheEntry {
    /// 检查是否过期
    fn is_expired(&self) -> bool {
        if let Some(expires_at) = self.expires_at {
            Instant::now() > expires_at
        } else {
            false
        }
    }
}

/// 内存缓存实现
#[derive(Clone)]
pub struct MemoryCache {
    store: Arc<RwLock<HashMap<String, CacheEntry>>>,
    stats: Arc<RwLock<CacheStats>>,
}

impl MemoryCache {
    /// 创建新的内存缓存
    pub fn new() -> Self {
        Self {
            store: Arc::new(RwLock::new(HashMap::new())),
            stats: Arc::new(RwLock::new(CacheStats {
                hits: 0,
                misses: 0,
                entries: 0,
                memory_usage: 0,
            })),
        }
    }

    /// 清理过期的缓存条目
    fn cleanup_expired(&self) {
        let mut store = self.store.write().unwrap();
        let mut stats = self.stats.write().unwrap();
        
        let before_count = store.len();
        store.retain(|_, entry| !entry.is_expired());
        let after_count = store.len();
        
        stats.entries = after_count as u64;
        
        // 简单的内存使用量估算
        let memory_usage = store
            .iter()
            .map(|(k, v)| k.len() + v.data.len() + 64) // 估算每个条目的内存开销
            .sum::<usize>();
        
        stats.memory_usage = memory_usage as u64;
        
        if before_count != after_count {
            log::debug!("Cleaned up {} expired cache entries", before_count - after_count);
        }
    }
}

impl Default for MemoryCache {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait]
impl CachePort for MemoryCache {
    async fn get<T>(&self, key: &str) -> DomainResult<Option<T>>
    where
        T: serde::de::DeserializeOwned + Send,
    {
        // 先清理过期条目
        self.cleanup_expired();
        
        let store = self.store.read().unwrap();
        let mut stats = self.stats.write().unwrap();
        
        if let Some(entry) = store.get(key) {
            if entry.is_expired() {
                stats.misses += 1;
                return Ok(None);
            }
            
            stats.hits += 1;
            let value: T = serde_json::from_str(&entry.data)
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("缓存数据反序列化失败: {}", e),
                })?;
            Ok(Some(value))
        } else {
            stats.misses += 1;
            Ok(None)
        }
    }

    async fn set<T>(&self, key: &str, value: &T, ttl: Option<Duration>) -> DomainResult<()>
    where
        T: serde::Serialize + Send + Sync,
    {
        let data = serde_json::to_string(value)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("缓存数据序列化失败: {}", e),
            })?;

        let expires_at = ttl.map(|duration| Instant::now() + duration);
        
        let entry = CacheEntry {
            data,
            created_at: Instant::now(),
            expires_at,
        };

        let mut store = self.store.write().unwrap();
        let mut stats = self.stats.write().unwrap();
        
        let is_new_key = !store.contains_key(key);
        store.insert(key.to_string(), entry);
        
        if is_new_key {
            stats.entries += 1;
        }
        
        // 更新内存使用量估算
        let memory_usage = store
            .iter()
            .map(|(k, v)| k.len() + v.data.len() + 64)
            .sum::<usize>();
        
        stats.memory_usage = memory_usage as u64;
        
        Ok(())
    }

    async fn delete(&self, key: &str) -> DomainResult<()> {
        let mut store = self.store.write().unwrap();
        let mut stats = self.stats.write().unwrap();
        
        if store.remove(key).is_some() {
            stats.entries -= 1;
            
            // 更新内存使用量估算
            let memory_usage = store
                .iter()
                .map(|(k, v)| k.len() + v.data.len() + 64)
                .sum::<usize>();
            
            stats.memory_usage = memory_usage as u64;
        }
        
        Ok(())
    }

    async fn delete_batch(&self, keys: &[&str]) -> DomainResult<()> {
        let mut store = self.store.write().unwrap();
        let mut stats = self.stats.write().unwrap();
        
        let mut deleted_count = 0;
        for &key in keys {
            if store.remove(key).is_some() {
                deleted_count += 1;
            }
        }
        
        stats.entries -= deleted_count;
        
        // 更新内存使用量估算
        let memory_usage = store
            .iter()
            .map(|(k, v)| k.len() + v.data.len() + 64)
            .sum::<usize>();
        
        stats.memory_usage = memory_usage as u64;
        
        Ok(())
    }

    async fn exists(&self, key: &str) -> DomainResult<bool> {
        self.cleanup_expired();
        
        let store = self.store.read().unwrap();
        Ok(store.get(key).map_or(false, |entry| !entry.is_expired()))
    }

    async fn expire(&self, key: &str, ttl: Duration) -> DomainResult<bool> {
        let mut store = self.store.write().unwrap();
        
        if let Some(entry) = store.get_mut(key) {
            entry.expires_at = Some(Instant::now() + ttl);
            Ok(true)
        } else {
            Ok(false)
        }
    }

    async fn clear_all(&self) -> DomainResult<()> {
        let mut store = self.store.write().unwrap();
        let mut stats = self.stats.write().unwrap();
        
        store.clear();
        stats.entries = 0;
        stats.memory_usage = 0;
        
        Ok(())
    }

    async fn get_stats(&self) -> DomainResult<CacheStats> {
        self.cleanup_expired();
        let stats = self.stats.read().unwrap();
        Ok(stats.clone())
    }
}

