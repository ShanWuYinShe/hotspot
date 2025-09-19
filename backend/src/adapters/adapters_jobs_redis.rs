/// Redis任务队列适配器
/// 
/// 基于Redis的任务队列实现
use crate::ports::{JobDispatcherPort, JobId, JobStatus, QueueStats, JobPayload};
use crate::ports::job_dispatcher_ports::Job;
use crate::domain::DomainResult;
use async_trait::async_trait;
use redis::{Client, AsyncCommands, RedisResult};
use redis::aio::ConnectionManager;
use serde::{Serialize, Deserialize};
use tracing::{debug, info, warn, error};
use uuid::Uuid;
use std::time::{SystemTime, UNIX_EPOCH};

/// Redis任务队列适配器
#[derive(Clone)]
pub struct RedisJobDispatcher {
    connection_manager: ConnectionManager,
    queue_prefix: String,
}

/// Redis中存储的任务数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RedisJobData {
    pub job_id: String,
    pub job_type: String,
    pub payload: serde_json::Value,
    pub created_at: u64,
    pub retry_count: u32,
    pub max_retries: u32,
    pub status: String,
}

impl RedisJobDispatcher {
    /// 从配置创建Redis任务队列适配器
    pub async fn from_config(redis_url: &str, queue_prefix: Option<String>) -> DomainResult<Self> {
        debug!("正在连接Redis任务队列: {}", redis_url);
        
        let client = Client::open(redis_url)
            .map_err(|e| {
                error!("Redis客户端创建失败: {}", e);
                crate::domain::DomainError::BusinessRuleViolation { 
                    rule: format!("Redis连接失败: {}", e) 
                }
            })?;
            
        let connection_manager = ConnectionManager::new(client).await
            .map_err(|e| {
                error!("Redis任务队列连接管理器创建失败: {}", e);
                crate::domain::DomainError::BusinessRuleViolation { 
                    rule: format!("Redis连接管理器失败: {}", e) 
                }
            })?;
            
        let queue_prefix = queue_prefix.unwrap_or_else(|| "hotspot:jobs".to_string());
        debug!("Redis任务队列连接建立成功，队列前缀: {}", queue_prefix);
        
        Ok(Self { 
            connection_manager,
            queue_prefix,
        })
    }
    
    /// 获取队列键名
    fn queue_key(&self, suffix: &str) -> String {
        format!("{}:{}", self.queue_prefix, suffix)
    }
    
    /// 获取任务状态键名
    fn job_status_key(&self, job_id: &str) -> String {
        format!("{}:status:{}", self.queue_prefix, job_id)
    }
    
    /// 获取统计信息键名
    fn stats_key(&self, stat_type: &str) -> String {
        format!("{}:stats:{}", self.queue_prefix, stat_type)
    }
    
    /// 序列化任务数据
    fn serialize_job_data(&self, job_data: &RedisJobData) -> DomainResult<String> {
        serde_json::to_string(job_data)
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation { 
                rule: format!("任务数据序列化失败: {}", e) 
            })
    }
    
    /// 反序列化任务数据
    fn deserialize_job_data(&self, data: &str) -> DomainResult<RedisJobData> {
        serde_json::from_str(data)
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation { 
                rule: format!("任务数据反序列化失败: {}", e) 
            })
    }
    
    /// 获取当前时间戳
    fn current_timestamp() -> u64 {
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs()
    }
}

#[async_trait]
impl JobDispatcherPort for RedisJobDispatcher {
    async fn dispatch_job<T>(&self, job: Job<T>) -> DomainResult<JobId>
    where
        T: JobPayload + Send + Sync,
    {
        let job_id = Uuid::new_v4().to_string();
        info!("Redis任务分发请求: {} (类型: {})", job_id, job.job_type);
        
        // 序列化任务载荷
        let payload = serde_json::to_value(&job.payload)
            .map_err(|e| crate::domain::DomainError::BusinessRuleViolation { 
                rule: format!("任务载荷序列化失败: {}", e) 
            })?;
        
        let redis_job_data = RedisJobData {
            job_id: job_id.clone(),
            job_type: job.job_type.clone(),
            payload,
            created_at: Self::current_timestamp(),
            retry_count: 0,
            max_retries: job.max_retries.unwrap_or(3),
            status: "queued".to_string(),
        };
        
        let serialized_data = self.serialize_job_data(&redis_job_data)?;
        let mut conn = self.connection_manager.clone();
        
        // 将任务添加到队列
        let queue_key = self.queue_key("pending");
        let result: RedisResult<i32> = conn.lpush(&queue_key, &serialized_data).await;
        
        match result {
            Ok(_) => {
                // 设置任务状态
                let status_key = self.job_status_key(&job_id);
                let _: RedisResult<()> = conn.set(&status_key, "queued").await;
                
                // 更新统计信息
                let stats_key = self.stats_key("queued");
                let _: RedisResult<i32> = conn.incr(&stats_key, 1).await;
                
                info!("Redis任务分发成功: {}", job_id);
                Ok(job_id)
            }
            Err(e) => {
                error!("Redis任务分发失败: {} - {}", job_id, e);
                Err(crate::domain::DomainError::BusinessRuleViolation { 
                    rule: format!("任务分发失败: {}", e) 
                })
            }
        }
    }

    async fn get_job_status(&self, job_id: &JobId) -> DomainResult<Option<JobStatus>> {
        debug!("Redis任务状态查询: {}", job_id);
        
        let mut conn = self.connection_manager.clone();
        let status_key = self.job_status_key(job_id);
        let result: RedisResult<Option<String>> = conn.get(&status_key).await;
        
        match result {
            Ok(Some(status_str)) => {
                let status = match status_str.as_str() {
                    "queued" => JobStatus::Queued,
                    "running" => JobStatus::Running,
                    "completed" => JobStatus::Completed,
                    "failed" => JobStatus::Failed,
                    "cancelled" => JobStatus::Cancelled,
                    _ => JobStatus::Queued, // 默认状态
                };
                debug!("Redis任务状态: {} = {:?}", job_id, status);
                Ok(Some(status))
            }
            Ok(None) => {
                debug!("Redis任务状态未找到: {}", job_id);
                Ok(None)
            }
            Err(e) => {
                error!("Redis任务状态查询失败: {} - {}", job_id, e);
                Err(crate::domain::DomainError::BusinessRuleViolation { 
                    rule: format!("任务状态查询失败: {}", e) 
                })
            }
        }
    }

    async fn cancel_job(&self, job_id: &JobId) -> DomainResult<bool> {
        info!("Redis任务取消请求: {}", job_id);
        
        let mut conn = self.connection_manager.clone();
        
        // 首先检查任务是否存在
        let status_key = self.job_status_key(job_id);
        let current_status: RedisResult<Option<String>> = conn.get(&status_key).await;
        
        match current_status {
            Ok(Some(status)) if status == "running" => {
                // 正在运行的任务无法取消
                warn!("无法取消正在运行的任务: {}", job_id);
                Ok(false)
            }
            Ok(Some(_)) => {
                // 设置任务状态为已取消
                let _: RedisResult<()> = conn.set(&status_key, "cancelled").await;
                
                // 从待处理队列中移除任务（如果存在）
                let queue_key = self.queue_key("pending");
                let _: RedisResult<i32> = conn.lrem(&queue_key, 0, job_id).await;
                
                // 更新统计信息
                let stats_key = self.stats_key("cancelled");
                let _: RedisResult<i32> = conn.incr(&stats_key, 1).await;
                
                info!("Redis任务取消成功: {}", job_id);
                Ok(true)
            }
            Ok(None) => {
                warn!("要取消的任务不存在: {}", job_id);
                Ok(false)
            }
            Err(e) => {
                error!("Redis任务取消失败: {} - {}", job_id, e);
                Err(crate::domain::DomainError::BusinessRuleViolation { 
                    rule: format!("任务取消失败: {}", e) 
                })
            }
        }
    }

    async fn retry_job(&self, job_id: &JobId) -> DomainResult<bool> {
        info!("Redis任务重试请求: {}", job_id);
        
        let mut conn = self.connection_manager.clone();
        
        // 查找失败的任务数据
        let failed_queue_key = self.queue_key("failed");
        let result: RedisResult<Option<String>> = conn.rpop(&failed_queue_key, None).await;
        
        match result {
            Ok(Some(job_data_str)) => {
                let mut job_data = self.deserialize_job_data(&job_data_str)?;
                
                if job_data.job_id == *job_id {
                    // 检查重试次数
                    if job_data.retry_count >= job_data.max_retries {
                        warn!("任务重试次数已达上限: {} ({}/{})", job_id, job_data.retry_count, job_data.max_retries);
                        // 重新放回失败队列
                        let _: RedisResult<i32> = conn.lpush(&failed_queue_key, &job_data_str).await;
                        return Ok(false);
                    }
                    
                    // 增加重试次数
                    job_data.retry_count += 1;
                    job_data.status = "queued".to_string();
                    
                    let updated_data = self.serialize_job_data(&job_data)?;
                    
                    // 重新加入待处理队列
                    let queue_key = self.queue_key("pending");
                    let _: RedisResult<i32> = conn.lpush(&queue_key, &updated_data).await;
                    
                    // 更新任务状态
                    let status_key = self.job_status_key(job_id);
                    let _: RedisResult<()> = conn.set(&status_key, "queued").await;
                    
                    // 更新统计信息
                    let stats_key = self.stats_key("retried");
                    let _: RedisResult<i32> = conn.incr(&stats_key, 1).await;
                    
                    info!("Redis任务重试成功: {} (第{}次重试)", job_id, job_data.retry_count);
                    Ok(true)
                } else {
                    // 不是目标任务，重新放回队列
                    let _: RedisResult<i32> = conn.lpush(&failed_queue_key, &job_data_str).await;
                    warn!("未找到要重试的任务: {}", job_id);
                    Ok(false)
                }
            }
            Ok(None) => {
                debug!("失败队列中没有任务可重试");
                Ok(false)
            }
            Err(e) => {
                error!("Redis任务重试失败: {} - {}", job_id, e);
                Err(crate::domain::DomainError::BusinessRuleViolation { 
                    rule: format!("任务重试失败: {}", e) 
                })
            }
        }
    }

    async fn get_queue_stats(&self) -> DomainResult<QueueStats> {
        debug!("Redis队列统计信息请求");
        
        let mut conn = self.connection_manager.clone();
        
        // 获取各队列长度
        let pending_key = self.queue_key("pending");
        let running_key = self.queue_key("running");
        let failed_key = self.queue_key("failed");
        
        let queued_jobs: i64 = conn.llen(&pending_key).await.unwrap_or(0);
        let running_jobs: i64 = conn.llen(&running_key).await.unwrap_or(0);
        let failed_jobs: i64 = conn.llen(&failed_key).await.unwrap_or(0);
        
        // 获取统计信息
        let completed_key = self.stats_key("completed");
        let completed_jobs: i64 = conn.get(&completed_key).await.unwrap_or(0);
        
        let stats = QueueStats {
            queued_jobs: queued_jobs as u64,
            running_jobs: running_jobs as u64,
            completed_jobs: completed_jobs as u64,
            failed_jobs: failed_jobs as u64,
            avg_processing_time_ms: 0, // 暂时不实现平均处理时间统计
        };
        
        debug!("Redis队列统计信息: {:?}", stats);
        Ok(stats)
    }
}
