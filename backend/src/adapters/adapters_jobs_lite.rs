/// 轻量级任务队列适配器
/// 
/// 基于内存的简单任务队列实现

use async_trait::async_trait;
use std::collections::{HashMap, VecDeque};
use std::sync::{Arc, RwLock};
// use tokio::sync::mpsc;
use crate::domain::{DomainResult, DomainError};
use crate::ports::{JobDispatcherPort, Job, JobId, JobStatus, QueueStats, JobPayload};

/// 轻量级任务队列实现
#[derive(Clone)]
pub struct LiteJobDispatcher {
    jobs: Arc<RwLock<HashMap<JobId, JobStatus>>>,
    queue: Arc<RwLock<VecDeque<String>>>, // 存储序列化的任务
    stats: Arc<RwLock<QueueStats>>,
}

impl LiteJobDispatcher {
    /// 创建新的轻量级任务调度器
    pub fn new() -> Self {
        Self {
            jobs: Arc::new(RwLock::new(HashMap::new())),
            queue: Arc::new(RwLock::new(VecDeque::new())),
            stats: Arc::new(RwLock::new(QueueStats {
                queued_jobs: 0,
                running_jobs: 0,
                completed_jobs: 0,
                failed_jobs: 0,
                avg_processing_time_ms: 0,
            })),
        }
    }
}

impl Default for LiteJobDispatcher {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait]
impl JobDispatcherPort for LiteJobDispatcher {
    async fn dispatch_job<T>(&self, job: Job<T>) -> DomainResult<JobId>
    where
        T: JobPayload + Send + Sync,
    {
        let job_id = job.id.clone();
        
        // 序列化任务
        let job_json = serde_json::to_string(&job)
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("任务序列化失败: {}", e),
            })?;
        
        // 添加到队列
        {
            let mut queue = self.queue.write().unwrap();
            let mut jobs = self.jobs.write().unwrap();
            let mut stats = self.stats.write().unwrap();
            
            queue.push_back(job_json);
            jobs.insert(job_id.clone(), JobStatus::Queued);
            stats.queued_jobs += 1;
        }
        
        log::info!("任务已加入队列: {}", job_id);
        Ok(job_id)
    }

    async fn get_job_status(&self, job_id: &JobId) -> DomainResult<Option<JobStatus>> {
        let jobs = self.jobs.read().unwrap();
        Ok(jobs.get(job_id).cloned())
    }

    async fn cancel_job(&self, job_id: &JobId) -> DomainResult<bool> {
        let mut jobs = self.jobs.write().unwrap();
        
        if let Some(status) = jobs.get(job_id) {
            match status {
                JobStatus::Queued => {
                    jobs.insert(job_id.clone(), JobStatus::Cancelled {
                        cancelled_at: chrono::Utc::now(),
                    });
                    Ok(true)
                }
                JobStatus::Running => {
                    // 运行中的任务无法立即取消，但可以标记
                    jobs.insert(job_id.clone(), JobStatus::Cancelled {
                        cancelled_at: chrono::Utc::now(),
                    });
                    Ok(true)
                }
                _ => Ok(false), // 已完成或已失败的任务无法取消
            }
        } else {
            Ok(false)
        }
    }

    async fn get_queue_stats(&self) -> DomainResult<QueueStats> {
        let stats = self.stats.read().unwrap();
        Ok(stats.clone())
    }

    async fn retry_job(&self, job_id: &JobId) -> DomainResult<bool> {
        let mut jobs = self.jobs.write().unwrap();
        
        if let Some(status) = jobs.get(job_id) {
            match status {
                JobStatus::Failed { .. } => {
                    jobs.insert(job_id.clone(), JobStatus::Queued);
                    Ok(true)
                }
                _ => Ok(false), // 只有失败的任务可以重试
            }
        } else {
            Ok(false)
        }
    }
}
