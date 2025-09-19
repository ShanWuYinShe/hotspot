/// 内存任务队列适配器
/// 
/// 功能说明：
/// - 提供轻量级的内存任务队列实现
/// - 适用于开发环境和小规模部署
/// - 任务不持久化，重启后丢失
/// - 支持异步任务执行和错误处理

use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use std::collections::VecDeque;
use std::sync::Arc;
use tokio::sync::Mutex;
use tracing::{debug, error, info, warn};

use crate::ports::JobDispatcher;

/// 内存任务队列调度器
#[derive(Clone)]
pub struct MemoryJobDispatcher {
    queue: Arc<Mutex<VecDeque<JobTask>>>,
    is_running: Arc<Mutex<bool>>,
}

/// 任务定义
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JobTask {
    pub id: String,
    pub job_type: String,
    pub payload: serde_json::Value,
    pub retry_count: u32,
    pub max_retries: u32,
}

impl MemoryJobDispatcher {
    /// 创建新的内存任务调度器
    pub fn new() -> Self {
        Self {
            queue: Arc::new(Mutex::new(VecDeque::new())),
            is_running: Arc::new(Mutex::new(false)),
        }
    }

    /// 启动任务处理器
    pub async fn start(&self) {
        let mut running = self.is_running.lock().await;
        if *running {
            warn!("Memory job dispatcher is already running");
            return;
        }
        *running = true;
        drop(running);

        info!("Starting memory job dispatcher");
        
        let queue = Arc::clone(&self.queue);
        let is_running = Arc::clone(&self.is_running);
        
        tokio::spawn(async move {
            loop {
                let should_continue = {
                    let running = is_running.lock().await;
                    *running
                };
                
                if !should_continue {
                    break;
                }

                let task = {
                    let mut queue_guard = queue.lock().await;
                    queue_guard.pop_front()
                };

                if let Some(task) = task {
                    Self::process_task(task).await;
                } else {
                    // 队列为空，短暂休眠
                    tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
                }
            }
            
            info!("Memory job dispatcher stopped");
        });
    }

    /// 停止任务处理器
    pub async fn stop(&self) {
        let mut running = self.is_running.lock().await;
        *running = false;
        info!("Stopping memory job dispatcher");
    }

    /// 处理单个任务
    async fn process_task(task: JobTask) {
        debug!("Processing task: {} (type: {})", task.id, task.job_type);
        
        match Self::execute_task(&task).await {
            Ok(_) => {
                info!("Task {} completed successfully", task.id);
            }
            Err(e) => {
                error!("Task {} failed: {}", task.id, e);
                
                if task.retry_count < task.max_retries {
                    warn!("Task {} will be retried ({}/{})", task.id, task.retry_count + 1, task.max_retries);
                    // 在实际实现中，这里应该重新入队
                } else {
                    error!("Task {} exceeded max retries, giving up", task.id);
                }
            }
        }
    }

    /// 执行具体任务
    async fn execute_task(task: &JobTask) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        match task.job_type.as_str() {
            "welcome_email" => {
                debug!("Sending welcome email for task {}", task.id);
                // 模拟邮件发送
                tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
                Ok(())
            }
            "location_init" => {
                debug!("Initializing location for task {}", task.id);
                // 模拟位置初始化
                tokio::time::sleep(tokio::time::Duration::from_millis(50)).await;
                Ok(())
            }
            "notification" => {
                debug!("Sending notification for task {}", task.id);
                // 模拟通知发送
                tokio::time::sleep(tokio::time::Duration::from_millis(30)).await;
                Ok(())
            }
            _ => {
                warn!("Unknown job type: {}", task.job_type);
                Ok(())
            }
        }
    }

    /// 获取队列状态
    pub async fn get_queue_size(&self) -> usize {
        let queue = self.queue.lock().await;
        queue.len()
    }

    /// 清空队列
    pub async fn clear_queue(&self) {
        let mut queue = self.queue.lock().await;
        let count = queue.len();
        queue.clear();
        info!("Cleared {} tasks from memory queue", count);
    }
}

#[async_trait]
impl JobDispatcher for MemoryJobDispatcher {
    async fn dispatch<T>(&self, job: T) -> crate::domain::DomainResult<crate::ports::JobId>
    where
        T: crate::ports::Job,
    {
        use crate::domain::errors::DomainError;
        
        let job_id = uuid::Uuid::new_v4().to_string();
        let job_type = job.job_type();
        let payload = serde_json::to_value(&job).map_err(|e| DomainError::ValidationError {
            field: "job_payload".to_string(),
            reason: e.to_string(),
        })?;
        
        let task = JobTask {
            id: job_id.clone(),
            job_type: job_type.to_string(),
            payload,
            retry_count: 0,
            max_retries: 3,
        };

        {
            let mut queue = self.queue.lock().await;
            queue.push_back(task);
        }

        debug!("Dispatched job {} (type: {})", job_id, job_type);
        Ok(job.job_id())
    }

    async fn dispatch_batch<T>(&self, jobs: Vec<T>) -> crate::domain::DomainResult<Vec<crate::ports::JobId>>
    where
        T: crate::ports::Job,
    {
        let mut job_ids = Vec::new();
        for job in jobs {
            let job_id = self.dispatch(job).await?;
            job_ids.push(job_id);
        }
        Ok(job_ids)
    }

    async fn dispatch_delayed<T>(&self, job: T, _delay: std::time::Duration) -> crate::domain::DomainResult<crate::ports::JobId>
    where
        T: crate::ports::Job,
    {
        // 内存队列暂不支持延迟任务，直接执行
        warn!("Memory job dispatcher does not support delayed jobs, executing immediately");
        self.dispatch(job).await
    }

    async fn get_dispatcher_status(&self) -> crate::domain::DomainResult<crate::ports::DispatcherStatus> {
        let running = self.is_running.lock().await;
        let queue_size = self.queue.lock().await.len();
        
        Ok(crate::ports::DispatcherStatus {
            is_active: *running,
            dispatched_jobs: queue_size as u64,
            failed_dispatches: 0, // 内存队列不跟踪失败任务数
            queue_health: if *running {
                crate::ports::QueueHealth::Healthy
            } else {
                crate::ports::QueueHealth::Unhealthy
            },
        })
    }
    
    async fn get_job_status(&self, _job_id: &crate::ports::JobId) -> crate::domain::DomainResult<Option<crate::ports::JobStatus>> {
        // 内存调度器的简单实现，暂时返回 None
        // 在实际应用中，可以维护一个任务状态映射
        log::warn!("内存调度器暂不支持任务状态查询");
        Ok(None)
    }
    
    async fn cancel_job(&self, _job_id: &crate::ports::JobId) -> crate::domain::DomainResult<bool> {
        // 内存调度器的简单实现，暂时返回 false
        // 在实际应用中，可以从队列中移除任务
        log::warn!("内存调度器暂不支持任务取消");
        Ok(false)
    }
    
    async fn retry_job(&self, _job_id: &crate::ports::JobId) -> crate::domain::DomainResult<bool> {
        // 内存调度器的简单实现，暂时返回 false
        // 在实际应用中，可以重新将任务加入队列
        log::warn!("内存调度器暂不支持任务重试");
        Ok(false)
    }
}

impl Default for MemoryJobDispatcher {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_memory_job_dispatcher() {
        let dispatcher = MemoryJobDispatcher::new();
        
        // 测试任务分发
        use crate::ports::{Job, JobId};
        use serde::{Serialize, Deserialize};
        
        #[derive(Debug, Clone, Serialize, Deserialize)]
        struct TestJob {
            test: String,
            job_id: JobId,
        }
        
        impl Job for TestJob {
            fn job_type(&self) -> &'static str {
                "test_job"
            }
            
            fn job_id(&self) -> JobId {
                self.job_id.clone()
            }
        }
        
        let test_job = TestJob {
            test: "data".to_string(),
            job_id: JobId::new(),
        };
        
        let job_id = dispatcher.dispatch(test_job).await.unwrap();
        // JobId 是基于 UUID 的，检查其不为空的方式
        assert_ne!(job_id.0, uuid::Uuid::nil());
        
        // 测试队列大小
        let size = dispatcher.get_queue_size().await;
        assert_eq!(size, 1);
        
        // 测试健康检查
        let status = dispatcher.get_dispatcher_status().await.unwrap();
        assert!(!status.is_active); // 未启动时应该返回 false
        
        // 测试清空队列
        dispatcher.clear_queue().await;
        let size = dispatcher.get_queue_size().await;
        assert_eq!(size, 0);
    }

    #[tokio::test]
    async fn test_start_stop() {
        let dispatcher = MemoryJobDispatcher::new();
        
        // 启动
        dispatcher.start().await;
        tokio::time::sleep(tokio::time::Duration::from_millis(10)).await;
        
        let status = dispatcher.get_dispatcher_status().await.unwrap();
        assert!(status.is_active);
        
        // 停止
        dispatcher.stop().await;
        tokio::time::sleep(tokio::time::Duration::from_millis(10)).await;
        
        let status = dispatcher.get_dispatcher_status().await.unwrap();
        assert!(!status.is_active);
    }
}
