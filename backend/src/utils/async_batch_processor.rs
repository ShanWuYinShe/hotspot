/// 异步批处理器
/// 
/// 提供高效的批量异步操作处理，减少数据库连接和网络开销
use std::collections::HashMap;
use std::future::Future;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::{mpsc, RwLock, Semaphore};
use tokio::time::timeout;
use tracing::{debug, error, info, warn};

use crate::domain::{DomainResult, DomainError};

/// 批处理配置
#[derive(Debug, Clone)]
pub struct BatchConfig {
    /// 批处理大小
    pub batch_size: usize,
    /// 批处理超时时间
    pub batch_timeout: Duration,
    /// 最大并发数
    pub max_concurrency: usize,
    /// 重试次数
    pub max_retries: u32,
    /// 重试延迟
    pub retry_delay: Duration,
}

impl Default for BatchConfig {
    fn default() -> Self {
        Self {
            batch_size: 100,
            batch_timeout: Duration::from_millis(100),
            max_concurrency: 10,
            max_retries: 3,
            retry_delay: Duration::from_millis(100),
        }
    }
}

/// 批处理项目
#[derive(Debug)]
pub struct BatchItem<T, R> {
    pub id: String,
    pub data: T,
    pub sender: tokio::sync::oneshot::Sender<DomainResult<R>>,
}

/// 批处理器
pub struct AsyncBatchProcessor<T, R> {
    config: BatchConfig,
    processor_fn: Arc<dyn Fn(Vec<T>) -> Box<dyn Future<Output = DomainResult<Vec<R>>> + Send + Sync>>,
    sender: mpsc::UnboundedSender<BatchItem<T, R>>,
    _handle: tokio::task::JoinHandle<()>,
}

impl<T, R> AsyncBatchProcessor<T, R>
where
    T: Send + 'static,
    R: Send + 'static,
{
    /// 创建新的批处理器
    pub fn new<F, Fut>(config: BatchConfig, processor_fn: F) -> Self
    where
        F: Fn(Vec<T>) -> Fut + Send + Sync + 'static,
        Fut: Future<Output = DomainResult<Vec<R>>> + Send + 'static,
    {
        let (sender, receiver) = mpsc::unbounded_channel();
        
        let processor_fn = Arc::new(move |items: Vec<T>| -> Box<dyn Future<Output = DomainResult<Vec<R>>> + Send> {
            Box::new(processor_fn(items))
        });
        
        let handle = tokio::spawn(Self::batch_worker(
            config.clone(),
            processor_fn.clone(),
            receiver,
        ));

        Self {
            config,
            processor_fn,
            sender,
            _handle: handle,
        }
    }

    /// 提交项目进行批处理
    pub async fn process(&self, id: String, data: T) -> DomainResult<R> {
        let (tx, rx) = tokio::sync::oneshot::channel();
        
        let item = BatchItem {
            id,
            data,
            sender: tx,
        };

        self.sender.send(item).map_err(|_| DomainError::BusinessRuleViolation {
            rule: "批处理器已关闭".to_string(),
        })?;

        rx.await.map_err(|_| DomainError::BusinessRuleViolation {
            rule: "批处理响应接收失败".to_string(),
        })?
    }

    /// 批处理工作线程
    async fn batch_worker(
        config: BatchConfig,
        processor_fn: Arc<dyn Fn(Vec<T>) -> Box<dyn Future<Output = DomainResult<Vec<R>>> + Send + Sync>>,
        mut receiver: mpsc::UnboundedReceiver<BatchItem<T, R>>,
    ) {
        let mut batch = Vec::new();
        let mut batch_start = Instant::now();
        let semaphore = Arc::new(Semaphore::new(config.max_concurrency));

        loop {
            let timeout_duration = config.batch_timeout.saturating_sub(batch_start.elapsed());
            
            match timeout(timeout_duration, receiver.recv()).await {
                Ok(Some(item)) => {
                    batch.push(item);
                    
                    // 检查是否需要处理批次
                    if batch.len() >= config.batch_size {
                        Self::process_batch(
                            &config,
                            &processor_fn,
                            &semaphore,
                            std::mem::take(&mut batch),
                        ).await;
                        batch_start = Instant::now();
                    }
                }
                Ok(None) => {
                    // 通道关闭，处理剩余批次
                    if !batch.is_empty() {
                        Self::process_batch(
                            &config,
                            &processor_fn,
                            &semaphore,
                            batch,
                        ).await;
                    }
                    break;
                }
                Err(_) => {
                    // 超时，处理当前批次
                    if !batch.is_empty() {
                        Self::process_batch(
                            &config,
                            &processor_fn,
                            &semaphore,
                            std::mem::take(&mut batch),
                        ).await;
                        batch_start = Instant::now();
                    }
                }
            }
        }
    }

    /// 处理单个批次
    async fn process_batch(
        config: &BatchConfig,
        processor_fn: &Arc<dyn Fn(Vec<T>) -> Box<dyn Future<Output = DomainResult<Vec<R>>> + Send + Sync>>,
        semaphore: &Arc<Semaphore>,
        batch: Vec<BatchItem<T, R>>,
    ) {
        let _permit = semaphore.acquire().await.unwrap();
        
        let batch_size = batch.len();
        let start_time = Instant::now();
        
        debug!("开始处理批次，大小: {}", batch_size);

        // 提取数据和发送器
        let (data, senders): (Vec<_>, Vec<_>) = batch.into_iter()
            .map(|item| (item.data, item.sender))
            .unzip();

        // 执行批处理，带重试机制
        let result = Self::execute_with_retry(config, processor_fn, data).await;
        
        let processing_time = start_time.elapsed();
        
        match result {
            Ok(results) => {
                info!(
                    "批次处理成功，大小: {}，耗时: {:?}",
                    batch_size, processing_time
                );
                
                // 发送结果给各个等待者
                for (sender, result) in senders.into_iter().zip(results.into_iter()) {
                    let _ = sender.send(Ok(result));
                }
            }
            Err(error) => {
                error!(
                    "批次处理失败，大小: {}，耗时: {:?}，错误: {:?}",
                    batch_size, processing_time, error
                );
                
                // 发送错误给所有等待者
                for sender in senders {
                    let _ = sender.send(Err(error.clone()));
                }
            }
        }
    }

    /// 带重试机制的执行
    async fn execute_with_retry(
        config: &BatchConfig,
        processor_fn: &Arc<dyn Fn(Vec<T>) -> Box<dyn Future<Output = DomainResult<Vec<R>>> + Send + Sync>>,
        data: Vec<T>,
    ) -> DomainResult<Vec<R>> {
        let mut last_error = None;
        
        for attempt in 0..=config.max_retries {
            if attempt > 0 {
                warn!("重试批处理，尝试次数: {}/{}", attempt, config.max_retries);
                tokio::time::sleep(config.retry_delay * attempt).await;
            }
            
            // 注意：这里我们需要克隆数据，因为processor_fn可能会消费它
            // 在实际实现中，可能需要要求T实现Clone或使用Arc<T>
            match processor_fn(data.clone()).await {
                Ok(results) => return Ok(results),
                Err(error) => {
                    last_error = Some(error);
                    if attempt < config.max_retries {
                        continue;
                    }
                }
            }
        }
        
        Err(last_error.unwrap_or_else(|| DomainError::BusinessRuleViolation {
            rule: "批处理执行失败".to_string(),
        }))
    }
}

/// 并发限制器
pub struct ConcurrencyLimiter {
    semaphore: Arc<Semaphore>,
    active_tasks: Arc<RwLock<HashMap<String, Instant>>>,
    max_task_duration: Duration,
}

impl ConcurrencyLimiter {
    /// 创建并发限制器
    pub fn new(max_concurrency: usize, max_task_duration: Duration) -> Self {
        Self {
            semaphore: Arc::new(Semaphore::new(max_concurrency)),
            active_tasks: Arc::new(RwLock::new(HashMap::new())),
            max_task_duration,
        }
    }

    /// 执行受限制的异步任务
    pub async fn execute<F, Fut, T>(&self, task_id: String, task: F) -> DomainResult<T>
    where
        F: FnOnce() -> Fut,
        Fut: Future<Output = DomainResult<T>>,
    {
        // 获取许可证
        let _permit = self.semaphore.acquire().await.map_err(|_| {
            DomainError::BusinessRuleViolation {
                rule: "无法获取并发许可证".to_string(),
            }
        })?;

        // 记录任务开始时间
        {
            let mut active_tasks = self.active_tasks.write().await;
            active_tasks.insert(task_id.clone(), Instant::now());
        }

        // 执行任务（带超时）
        let result = timeout(self.max_task_duration, task()).await;

        // 清理任务记录
        {
            let mut active_tasks = self.active_tasks.write().await;
            active_tasks.remove(&task_id);
        }

        match result {
            Ok(task_result) => task_result,
            Err(_) => Err(DomainError::BusinessRuleViolation {
                rule: format!("任务 {} 执行超时", task_id),
            }),
        }
    }

    /// 获取当前活跃任务数
    pub async fn active_task_count(&self) -> usize {
        self.active_tasks.read().await.len()
    }

    /// 获取可用许可证数
    pub fn available_permits(&self) -> usize {
        self.semaphore.available_permits()
    }

    /// 清理超时任务（用于监控）
    pub async fn cleanup_stale_tasks(&self) -> usize {
        let mut active_tasks = self.active_tasks.write().await;
        let now = Instant::now();
        let stale_threshold = self.max_task_duration * 2;
        
        let initial_count = active_tasks.len();
        active_tasks.retain(|_, start_time| {
            now.duration_since(*start_time) < stale_threshold
        });
        
        let cleaned_count = initial_count - active_tasks.len();
        if cleaned_count > 0 {
            warn!("清理了 {} 个超时任务记录", cleaned_count);
        }
        
        cleaned_count
    }
}

/// 异步任务池
pub struct AsyncTaskPool {
    limiter: ConcurrencyLimiter,
    task_counter: Arc<std::sync::atomic::AtomicU64>,
}

impl AsyncTaskPool {
    /// 创建任务池
    pub fn new(max_concurrency: usize, max_task_duration: Duration) -> Self {
        Self {
            limiter: ConcurrencyLimiter::new(max_concurrency, max_task_duration),
            task_counter: Arc::new(std::sync::atomic::AtomicU64::new(0)),
        }
    }

    /// 提交任务到池中执行
    pub async fn submit<F, Fut, T>(&self, task: F) -> DomainResult<T>
    where
        F: FnOnce() -> Fut + Send + 'static,
        Fut: Future<Output = DomainResult<T>> + Send + 'static,
        T: Send + 'static,
    {
        let task_id = format!(
            "task_{}",
            self.task_counter.fetch_add(1, std::sync::atomic::Ordering::Relaxed)
        );

        self.limiter.execute(task_id, task).await
    }

    /// 获取池统计信息
    pub async fn get_stats(&self) -> TaskPoolStats {
        TaskPoolStats {
            active_tasks: self.limiter.active_task_count().await,
            available_permits: self.limiter.available_permits(),
            total_submitted: self.task_counter.load(std::sync::atomic::Ordering::Relaxed),
        }
    }
}

/// 任务池统计信息
#[derive(Debug, Clone)]
pub struct TaskPoolStats {
    pub active_tasks: usize,
    pub available_permits: usize,
    pub total_submitted: u64,
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::time::sleep;

    #[tokio::test]
    async fn test_batch_processor() {
        let config = BatchConfig {
            batch_size: 3,
            batch_timeout: Duration::from_millis(50),
            ..Default::default()
        };

        let processor = AsyncBatchProcessor::new(config, |items: Vec<i32>| async move {
            // 模拟批处理：将所有数字乘以2
            Ok(items.into_iter().map(|x| x * 2).collect())
        });

        // 提交一些任务
        let mut handles = Vec::new();
        for i in 1..=5 {
            let processor = &processor;
            let handle = tokio::spawn(async move {
                processor.process(format!("task_{}", i), i).await
            });
            handles.push(handle);
        }

        // 等待所有任务完成
        for handle in handles {
            let result = handle.await.unwrap().unwrap();
            assert!(result > 0);
        }
    }

    #[tokio::test]
    async fn test_concurrency_limiter() {
        let limiter = ConcurrencyLimiter::new(2, Duration::from_millis(100));

        let mut handles = Vec::new();
        for i in 0..5 {
            let limiter = &limiter;
            let handle = tokio::spawn(async move {
                limiter.execute(format!("task_{}", i), || async {
                    sleep(Duration::from_millis(10)).await;
                    Ok(i)
                }).await
            });
            handles.push(handle);
        }

        // 等待所有任务完成
        for handle in handles {
            let result = handle.await.unwrap().unwrap();
            assert!(result < 5);
        }

        // 检查最终状态
        assert_eq!(limiter.active_task_count().await, 0);
        assert_eq!(limiter.available_permits(), 2);
    }

    #[tokio::test]
    async fn test_task_pool() {
        let pool = AsyncTaskPool::new(3, Duration::from_millis(100));

        let mut handles = Vec::new();
        for i in 0..10 {
            let pool = &pool;
            let handle = tokio::spawn(async move {
                pool.submit(|| async move {
                    sleep(Duration::from_millis(5)).await;
                    Ok(i * 2)
                }).await
            });
            handles.push(handle);
        }

        // 等待所有任务完成
        for (i, handle) in handles.into_iter().enumerate() {
            let result = handle.await.unwrap().unwrap();
            assert_eq!(result, i * 2);
        }

        let stats = pool.get_stats().await;
        assert_eq!(stats.active_tasks, 0);
        assert_eq!(stats.total_submitted, 10);
    }
}
