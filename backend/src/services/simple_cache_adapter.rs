/// 简单缓存适配器 - 为nonce服务提供简化的缓存接口
/// 
/// 将复杂的缓存端口适配为简单的字符串键值存储
use async_trait::async_trait;
use std::sync::Arc;

use crate::services::nonce_service::CacheService as SimpleCacheService;
use crate::ports::cache::Cache;

/// 简单缓存适配器
pub struct SimpleCacheAdapter<C: Cache> {
    cache: Arc<C>,
}

impl<C: Cache> SimpleCacheAdapter<C> {
    pub fn new(cache: Arc<C>) -> Self {
        Self { cache }
    }
}

#[async_trait]
impl<C: Cache> SimpleCacheService for SimpleCacheAdapter<C> {
    async fn get(&self, key: &str) -> Result<Option<String>, String> {
        match self.cache.get::<String>(key).await {
            Ok(value) => Ok(value),
            Err(e) => Err(format!("缓存获取失败: {:?}", e)),
        }
    }
    
    async fn set(&self, key: &str, value: &str, ttl: Option<u64>) -> Result<(), String> {
        let duration = ttl.map(|seconds| std::time::Duration::from_secs(seconds));
        match self.cache.set(key, &value.to_string(), duration).await {
            Ok(_) => Ok(()),
            Err(e) => Err(format!("缓存设置失败: {:?}", e)),
        }
    }
    
    async fn delete(&self, key: &str) -> Result<(), String> {
        match self.cache.delete(key).await {
            Ok(_) => Ok(()),
            Err(e) => Err(format!("缓存删除失败: {:?}", e)),
        }
    }
    
    async fn exists(&self, key: &str) -> Result<bool, String> {
        match self.cache.exists(key).await {
            Ok(exists) => Ok(exists),
            Err(e) => Err(format!("缓存检查失败: {:?}", e)),
        }
    }
    
    async fn expire(&self, key: &str, ttl: u64) -> Result<(), String> {
        let duration = std::time::Duration::from_secs(ttl);
        match self.cache.expire(key, duration).await {
            Ok(_) => Ok(()),
            Err(e) => Err(format!("缓存过期设置失败: {:?}", e)),
        }
    }
}

/// 内存缓存适配器 - 用于开发和测试
use std::collections::HashMap;
use std::sync::Mutex;

pub struct MemoryCacheAdapter {
    data: Arc<Mutex<HashMap<String, String>>>,
}

impl MemoryCacheAdapter {
    pub fn new() -> Self {
        Self {
            data: Arc::new(Mutex::new(HashMap::new())),
        }
    }
}

impl Default for MemoryCacheAdapter {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait]
impl SimpleCacheService for MemoryCacheAdapter {
    async fn get(&self, key: &str) -> Result<Option<String>, String> {
        let data = self.data.lock().map_err(|e| format!("锁获取失败: {}", e))?;
        Ok(data.get(key).cloned())
    }
    
    async fn set(&self, key: &str, value: &str, _ttl: Option<u64>) -> Result<(), String> {
        let mut data = self.data.lock().map_err(|e| format!("锁获取失败: {}", e))?;
        data.insert(key.to_string(), value.to_string());
        Ok(())
    }
    
    async fn delete(&self, key: &str) -> Result<(), String> {
        let mut data = self.data.lock().map_err(|e| format!("锁获取失败: {}", e))?;
        data.remove(key);
        Ok(())
    }
    
    async fn exists(&self, key: &str) -> Result<bool, String> {
        let data = self.data.lock().map_err(|e| format!("锁获取失败: {}", e))?;
        Ok(data.contains_key(key))
    }
    
    async fn expire(&self, _key: &str, _ttl: u64) -> Result<(), String> {
        // 内存缓存不支持TTL，直接返回成功
        Ok(())
    }
}
