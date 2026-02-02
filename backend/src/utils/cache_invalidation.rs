use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use tokio::sync::broadcast;

/// 缓存失效消息类型
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub enum InvalidationEvent {
    /// 会话失效 (User ID, Session ID)
    SessionRevoked { user_id: i64, session_id: String },
    /// 用户资料更新 (User ID)
    UserProfileUpdated { user_id: i64 },
    /// 群组信息更新
    GroupInfoUpdated { group_id: String },
    /// 群组成员变更
    GroupMemberChanged { group_id: String },
    /// 瞬间内容更新
    MomentUpdated { moment_id: String },
    /// 瞬间删除
    MomentDeleted { moment_id: String },
    /// 评论变更
    CommentChanged {
        moment_id: String,
        comment_id: String,
    },
}

impl InvalidationEvent {
    pub const CHANNEL_NAME: &'static str = "cache:invalidation";
}

/// 缓存失效广播器
#[derive(Clone)]
pub struct CacheInvalidator {
    redis: ConnectionManager,
}

impl CacheInvalidator {
    pub fn new(redis: ConnectionManager) -> Self {
        Self { redis }
    }

    /// 发布失效消息
    pub async fn publish(&self, event: InvalidationEvent) -> Result<(), redis::RedisError> {
        let message = serde_json::to_string(&event).unwrap_or_default();
        let mut conn = self.redis.clone();
        conn.publish::<_, _, ()>(InvalidationEvent::CHANNEL_NAME, message)
            .await?;
        Ok(())
    }
}

/// 连接并监听 Redis Pub/Sub（单次连接）
async fn connect_and_listen(
    redis_url: &str,
    tx: &broadcast::Sender<InvalidationEvent>,
) -> Result<(), redis::RedisError> {
    let client = redis::Client::open(redis_url)?;

    // 获取异步 Pub/Sub 连接
    let mut pubsub = client.get_async_pubsub().await?;

    pubsub.subscribe(InvalidationEvent::CHANNEL_NAME).await?;
    tracing::info!("Redis Pub/Sub 已连接并订阅");

    use futures_util::StreamExt;
    let mut stream = pubsub.on_message();

    while let Some(msg) = stream.next().await {
        let payload: String = match msg.get_payload() {
            Ok(p) => p,
            Err(e) => {
                tracing::warn!(error = %e, "Failed to get payload from redis message");
                continue;
            }
        };

        match serde_json::from_str::<InvalidationEvent>(&payload) {
            Ok(event) => {
                if tx.send(event).is_err() {
                    tracing::warn!("No receivers for invalidation event");
                }
            }
            Err(e) => {
                tracing::warn!(error = %e, "Failed to deserialize invalidation event");
            }
        }
    }

    Ok(())
}

/// 启动 Redis 订阅监听任务（带指数退避重连）
///
/// 收到 Redis 消息后，转发到内部的 broadcast channel，供各服务订阅处理
pub fn start_invalidation_listener(
    redis_url: String,
    tx: broadcast::Sender<InvalidationEvent>,
) -> tokio::task::JoinHandle<()> {
    tokio::spawn(async move {
        let mut retry_count = 0;
        const MAX_RETRIES: u32 = 10;

        loop {
            match connect_and_listen(&redis_url, &tx).await {
                Ok(_) => {
                    // 连接断开，重置重试计数
                    retry_count = 0;
                    tracing::warn!("Redis Pub/Sub 连接断开，将重连");
                }
                Err(e) => {
                    retry_count += 1;

                    if retry_count >= MAX_RETRIES {
                        tracing::error!("Redis Pub/Sub 重连失败次数过多，停止重试");
                        break;
                    }

                    // 指数退避，最大 60 秒
                    let backoff_secs = std::cmp::min(2_u64.pow(retry_count), 60);

                    tracing::error!(
                        error = %e,
                        retry_count,
                        backoff_secs,
                        "Redis Pub/Sub 连接失败，等待后重试"
                    );

                    tokio::time::sleep(std::time::Duration::from_secs(backoff_secs)).await;
                }
            }
        }

        tracing::error!("Redis Pub/Sub 监听器已停止");
    })
}
