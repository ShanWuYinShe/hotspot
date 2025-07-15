//! 异步任务管理模块
use lapin::{
    options::*, types::{FieldTable, AMQPValue}, Channel, Connection, ConnectionProperties, Consumer, BasicProperties,
};
use lapin::Result as LapinResult;
use serde::{Deserialize, Serialize};
use std::{future::Future, sync::Arc};
use tokio::sync::Mutex;
use tracing::{debug, error, info, warn};

/// 任务优先级
#[derive(Debug, Clone, Copy)]
pub enum JobPriority {
    Normal = 5,
}

impl Default for JobPriority {
    fn default() -> Self {
        Self::Normal
    }
}

/// 任务定义
#[derive(Debug, Clone)]
pub struct Job<T>
where
    T: Clone + Serialize + for<'a> Deserialize<'a>,
{
    /// 任务ID
    pub id: String,
    /// 任务名称
    pub name: String,
    /// 任务数据
    pub data: T,
    /// 重试次数
    pub retries: u8,
    /// 延迟执行时间（毫秒）
    pub delay_ms: u64,
    /// 最大重试次数
    pub max_retries: u8,
    /// 优先级
    pub priority: JobPriority,
}

impl<T> Serialize for Job<T>
where
    T: Clone + Serialize + for<'a> Deserialize<'a>,
{
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut state = serializer.serialize_struct("Job", 6)?;
        state.serialize_field("id", &self.id)?;
        state.serialize_field("name", &self.name)?;
        state.serialize_field("data", &self.data)?;
        state.serialize_field("retries", &self.retries)?;
        state.serialize_field("delay_ms", &self.delay_ms)?;
        state.serialize_field("max_retries", &self.max_retries)?;
        state.end()
    }
}

impl<'de, T> Deserialize<'de> for Job<T>
where
    T: Clone + Serialize + for<'a> Deserialize<'a>,
{
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        use serde::de::{self, MapAccess, Visitor};
        use std::fmt;

        struct JobVisitor<T>(std::marker::PhantomData<T>);

        impl<'de, T> Visitor<'de> for JobVisitor<T>
        where
            T: Clone + Serialize + for<'a> Deserialize<'a>,
        {
            type Value = Job<T>;

            fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
                formatter.write_str("struct Job")
            }

            fn visit_map<V>(self, mut map: V) -> Result<Job<T>, V::Error>
            where
                V: MapAccess<'de>,
            {
                let mut id = None;
                let mut name = None;
                let mut data = None;
                let mut retries = None;
                let mut delay_ms = None;
                let mut max_retries = None;

                while let Some(key) = map.next_key()? {
                    match key {
                        "id" => {
                            if id.is_some() {
                                return Err(de::Error::duplicate_field("id"));
                            }
                            id = Some(map.next_value()?);
                        }
                        "name" => {
                            if name.is_some() {
                                return Err(de::Error::duplicate_field("name"));
                            }
                            name = Some(map.next_value()?);
                        }
                        "data" => {
                            if data.is_some() {
                                return Err(de::Error::duplicate_field("data"));
                            }
                            data = Some(map.next_value()?);
                        }
                        "retries" => {
                            if retries.is_some() {
                                return Err(de::Error::duplicate_field("retries"));
                            }
                            retries = Some(map.next_value()?);
                        }
                        "delay_ms" => {
                            if delay_ms.is_some() {
                                return Err(de::Error::duplicate_field("delay_ms"));
                            }
                            delay_ms = Some(map.next_value()?);
                        }
                        "max_retries" => {
                            if max_retries.is_some() {
                                return Err(de::Error::duplicate_field("max_retries"));
                            }
                            max_retries = Some(map.next_value()?);
                        }
                        _ => {
                            let _ = map.next_value::<serde::de::IgnoredAny>()?;
                        }
                    }
                }

                let id = id.ok_or_else(|| de::Error::missing_field("id"))?;
                let name = name.ok_or_else(|| de::Error::missing_field("name"))?;
                let data = data.ok_or_else(|| de::Error::missing_field("data"))?;
                let retries = retries.unwrap_or(0);
                let delay_ms = delay_ms.unwrap_or(0);
                let max_retries = max_retries.unwrap_or(3);

                Ok(Job {
                    id,
                    name,
                    data,
                    retries,
                    delay_ms,
                    max_retries,
                    priority: JobPriority::default(),
                })
            }
        }

        const FIELDS: &[&str] = &["id", "name", "data", "retries", "delay_ms", "max_retries"];
        deserializer.deserialize_struct("Job", FIELDS, JobVisitor(std::marker::PhantomData))
    }
}

#[allow(dead_code)]
fn default_max_retries() -> u8 {
    3
}

impl<T> Job<T> 
where
    T: Clone + Serialize + for<'a> Deserialize<'a>,
{
    pub fn new(name: impl Into<String>, data: T) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            name: name.into(),
            data,
            retries: 0,
            delay_ms: 0,
            max_retries: 3,
            priority: JobPriority::default(),
        }
    }

    pub fn with_priority(mut self, priority: JobPriority) -> Self {
        self.priority = priority;
        self
    }

    pub fn with_delay(mut self, delay_ms: u64) -> Self {
        self.delay_ms = delay_ms;
        self
    }

    #[allow(dead_code)]
    pub fn with_max_retries(mut self, max_retries: u8) -> Self {
        self.max_retries = max_retries;
        self
    }
}

/// 任务队列客户端
pub struct JobQueue {
    /// RabbitMQ连接
    #[allow(dead_code)]
    connection: Connection,
    /// 通道
    channel: Mutex<Option<Channel>>,
}

impl JobQueue {
    /// 创建新的任务队列
    pub async fn new(amqp_url: &str) -> LapinResult<Self> {
        let connection = Connection::connect(
            amqp_url,
            ConnectionProperties::default(),
        )
        .await?;

        let channel = connection.create_channel().await?;
        
        // 声明队列并确保队列存在
        Self::declare_queues(&channel).await?;

        Ok(Self {
            connection,
            channel: Mutex::new(Some(channel)),
        })
    }

    async fn declare_queues(channel: &Channel) -> LapinResult<()> {
        // 声明普通任务队列
        channel
            .queue_declare(
                "tasks",
                QueueDeclareOptions {
                    durable: true,
                    ..Default::default()
                },
                FieldTable::default(),
            )
            .await?;

        // 声明延迟队列（使用死信交换机实现）
        channel
            .exchange_declare(
                "tasks.delay",
                lapin::ExchangeKind::Direct,
                ExchangeDeclareOptions {
                    durable: true,
                    ..Default::default()
                },
                FieldTable::default(),
            )
            .await?;

        // 声明死信交换机
        channel
            .exchange_declare(
                "tasks.dlx",
                lapin::ExchangeKind::Direct,
                ExchangeDeclareOptions {
                    durable: true,
                    ..Default::default()
                },
                FieldTable::default(),
            )
            .await?;

        // 声明延迟队列
        let mut args = FieldTable::default();
        args.insert(
            "x-dead-letter-exchange".into(),
            AMQPValue::LongString("tasks.dlx".into()),
        );
        args.insert(
            "x-dead-letter-routing-key".into(),
            AMQPValue::LongString("tasks".into()),
        );

        channel
            .queue_declare(
                "tasks.delayed",
                QueueDeclareOptions {
                    durable: true,
                    ..Default::default()
                },
                args,
            )
            .await?;

        // 绑定延迟队列到延迟交换机
        channel
            .queue_bind(
                "tasks.delayed",
                "tasks.delay",
                "delayed",
                QueueBindOptions::default(),
                FieldTable::default(),
            )
            .await?;

        // 绑定普通队列到死信交换机
        channel
            .queue_bind(
                "tasks",
                "tasks.dlx",
                "tasks",
                QueueBindOptions::default(),
                FieldTable::default(),
            )
            .await?;

        Ok(())
    }

    /// 发送任务到队列
    pub async fn enqueue<T>(&self, job: Job<T>) -> LapinResult<()> 
    where
        T: Clone + Serialize + for<'a> Deserialize<'a>,
    {
        let channel_guard = self.channel.lock().await;
        let channel = channel_guard.as_ref().expect("Channel should be available");

        let payload = serde_json::to_vec(&job).expect("Failed to serialize job");

        if job.delay_ms > 0 {
            // 使用延迟队列
            let mut headers = FieldTable::default();
            headers.insert("x-delay".into(), job.delay_ms.into());

            channel
                .basic_publish(
                    "tasks.delay",
                    "delayed",
                    BasicPublishOptions::default(),
                    &payload,
                    BasicProperties::default()
                        .with_priority(job.priority as u8)
                        .with_headers(headers),
                )
                .await?;
        } else {
            // 直接发送到普通队列
            channel
                .basic_publish(
                    "",
                    "tasks",
                    BasicPublishOptions::default(),
                    &payload,
                    BasicProperties::default().with_priority(job.priority as u8),
                )
                .await?;
        }

        debug!("任务已入队: {}", job.name);
        Ok(())
    }

    /// 注册任务处理器
    pub async fn register_handler<T, F, Fut>(
        &self,
        handler: F,
    ) -> LapinResult<Consumer>
    where
        T: Clone + Serialize + for<'a> Deserialize<'a> + Send + 'static,
        F: Fn(Job<T>) -> Fut + Send + Sync + 'static,
        Fut: Future<Output = LapinResult<()>> + Send + 'static,
    {
        let channel_guard = self.channel.lock().await;
        let channel = channel_guard.as_ref().expect("Channel should be available");

        let consumer = channel
            .basic_consume(
                "tasks",
                &format!("consumer-{}", uuid::Uuid::new_v4()),
                BasicConsumeOptions::default(),
                FieldTable::default(),
            )
            .await?;

        let channel_clone = channel.clone();
        let consumer_clone = consumer.clone();
        
        tokio::spawn(async move {
            info!("任务处理器已启动");
            
            let handler = Arc::new(handler);
            
            consumer_clone.set_delegate(move |delivery: Result<Option<lapin::message::Delivery>, lapin::Error>| {
                let handler = handler.clone();
                let channel = channel_clone.clone();
                
                async move {
                    let delivery = match delivery {
                        Ok(Some(delivery)) => delivery,
                        Ok(None) => return,
                        Err(e) => {
                            error!("接收任务失败: {:?}", e);
                            return;
                        }
                    };

                    match serde_json::from_slice::<Job<T>>(&delivery.data) {
                        Ok(job) => {
                            debug!("处理任务: {} (ID: {})", job.name, job.id);
                            
                            match handler(job.clone()).await {
                                Ok(_) => {
                                    info!("任务完成: {} (ID: {})", job.name, job.id);
                                    let _ = channel
                                        .basic_ack(delivery.delivery_tag, BasicAckOptions::default())
                                        .await;
                                }
                                Err(e) => {
                                    error!("任务执行失败 {} (ID: {}): {:?}", job.name, job.id, e);
                                    
                                    if job.retries < job.max_retries {
                                        // 重试任务
                                        warn!("重试任务 {} (ID: {}), 次数: {}/{}", 
                                            job.name, job.id, job.retries + 1, job.max_retries);
                                            
                                        let mut retry_job = job;
                                        retry_job.retries += 1;
                                        // 添加指数退避延迟
                                        retry_job.delay_ms = 1000 * (2_u64.pow(retry_job.retries as u32));
                                        
                                        let retry_payload = serde_json::to_vec(&retry_job)
                                            .expect("Failed to serialize retry job");
                                            
                                        // 返回任务到延迟队列
                                        let mut headers = FieldTable::default();
                                        headers.insert("x-delay".into(), retry_job.delay_ms.into());
                                        
                                        if let Err(e) = channel
                                            .basic_publish(
                                                "tasks.delay",
                                                "delayed",
                                                BasicPublishOptions::default(),
                                                &retry_payload,
                                                BasicProperties::default()
                                                    .with_priority(retry_job.priority as u8)
                                                    .with_headers(headers),
                                            )
                                            .await
                                        {
                                            error!("重试任务发布失败: {:?}", e);
                                        }
                                    }
                                    
                                    let _ = channel
                                        .basic_ack(delivery.delivery_tag, BasicAckOptions::default())
                                        .await;
                                }
                            }
                        }
                        Err(e) => {
                            error!("任务反序列化失败: {:?}", e);
                            let _ = channel
                                .basic_nack(
                                    delivery.delivery_tag,
                                    BasicNackOptions::default(),
                                )
                                .await;
                        }
                    }
                }
            });
        });

        Ok(consumer)
    }
}
