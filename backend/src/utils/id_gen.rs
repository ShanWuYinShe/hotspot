//! ID生成器
//!
//! 提供分布式唯一 ID 与分享码生成功能。

use once_cell::sync::Lazy;
use rand::Rng;
use redis::aio::ConnectionManager;
use std::env;
use std::process;
use std::sync::atomic::{AtomicU16, Ordering};
use std::sync::Mutex;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tokio::task::JoinHandle;
use tracing::{error, info, warn};
use uuid::Uuid;

const NODE_ID_BITS: u8 = 10;
const SEQUENCE_BITS: u8 = 12;
const NODE_ID_SHIFT: u8 = SEQUENCE_BITS;
const TIMESTAMP_SHIFT: u8 = NODE_ID_BITS + SEQUENCE_BITS;
const MAX_NODE_ID: u16 = (1 << NODE_ID_BITS) - 1;
const SEQUENCE_MASK: u16 = (1 << SEQUENCE_BITS) - 1;
const SHARE_CODE_LENGTH: usize = 12;
const SHARE_CODE_CHARSET: &[u8] = b"ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
// 2025-01-01T00:00:00Z，自定义 epoch（毫秒）
const CUSTOM_EPOCH: i64 = 1_735_689_600_000;
const REDIS_NODE_ID_ALLOCATE_ENV: &str = "ID_NODE_ID_ALLOCATE_WITH_REDIS";
const REDIS_NODE_ID_PREFIX_ENV: &str = "ID_NODE_ID_REDIS_KEY_PREFIX";
const REDIS_NODE_ID_OWNER_ENV: &str = "ID_NODE_ID_REDIS_OWNER";
const REDIS_NODE_ID_TTL_SECS_ENV: &str = "ID_NODE_ID_REDIS_LEASE_TTL_SECS";
const REDIS_NODE_ID_RENEW_INTERVAL_ENV: &str = "ID_NODE_ID_REDIS_RENEW_INTERVAL_SECS";
const REDIS_NODE_ID_FATAL_ON_LOSS_ENV: &str = "ID_NODE_ID_REDIS_FATAL_ON_LOSS";
const DEFAULT_REDIS_NODE_ID_PREFIX: &str = "hotspot:id_node:lease";
const DEFAULT_REDIS_NODE_ID_TTL_SECS: u64 = 90;
const DEFAULT_REDIS_NODE_ID_RENEW_INTERVAL_SECS: u64 = 30;

static ID_GENERATOR: Lazy<SnowflakeGenerator> = Lazy::new(|| {
    let node_id = parse_node_id_from_env();
    SnowflakeGenerator::new(node_id)
});

fn parse_bool_flag(raw: &str) -> Option<bool> {
    match raw.trim().to_ascii_lowercase().as_str() {
        "1" | "true" | "yes" | "on" => Some(true),
        "0" | "false" | "no" | "off" => Some(false),
        _ => None,
    }
}

fn strict_node_id_mode_enabled() -> bool {
    if let Ok(raw) = env::var("ID_NODE_ID_STRICT") {
        return match parse_bool_flag(&raw) {
            Some(enabled) => enabled,
            None => {
                warn!(
                    raw_value = %raw,
                    "ID_NODE_ID_STRICT 无法解析为布尔值，按 false 处理"
                );
                false
            }
        };
    }

    ["APP_ENV", "NODE_ENV", "RUST_ENV"]
        .iter()
        .filter_map(|key| env::var(key).ok())
        .map(|v| v.trim().to_ascii_lowercase())
        .any(|v| matches!(v.as_str(), "prod" | "production"))
}

fn redis_node_id_mode_enabled() -> bool {
    match env::var(REDIS_NODE_ID_ALLOCATE_ENV) {
        Ok(raw) => match parse_bool_flag(&raw) {
            Some(enabled) => enabled,
            None => {
                warn!(
                    raw_value = %raw,
                    "ID_NODE_ID_ALLOCATE_WITH_REDIS 无法解析为布尔值，按 false 处理"
                );
                false
            }
        },
        Err(_) => false,
    }
}

fn parse_node_id_from_env() -> u16 {
    let strict_mode = strict_node_id_mode_enabled();
    let redis_mode = redis_node_id_mode_enabled();
    match env::var("ID_NODE_ID") {
        Ok(raw) => match raw.parse::<u16>() {
            Ok(node_id) if node_id <= MAX_NODE_ID => node_id,
            Ok(node_id) => {
                if strict_mode && !redis_mode {
                    panic!("ID_NODE_ID 超出范围：{node_id}，允许范围 0..={MAX_NODE_ID}");
                }
                warn!(
                    id_node_id = node_id,
                    max_node_id = MAX_NODE_ID,
                    "ID_NODE_ID 超出范围，将按上限截断"
                );
                MAX_NODE_ID
            }
            Err(_) => {
                if strict_mode && !redis_mode {
                    panic!("ID_NODE_ID 非法：{raw}，生产环境必须配置合法整数");
                }
                warn!(raw_value = %raw, "ID_NODE_ID 不是有效整数，使用默认值 0");
                0
            }
        },
        Err(_) => {
            if strict_mode && !redis_mode {
                panic!("ID_NODE_ID 未配置，生产环境必须显式配置避免分布式冲突");
            }
            warn!("ID_NODE_ID 未配置，使用默认值 0；多实例部署请显式设置避免冲突");
            0
        }
    }
}

struct SnowflakeState {
    last_timestamp: i64,
    sequence: u16,
}

struct SnowflakeGenerator {
    node_id: AtomicU16,
    state: Mutex<SnowflakeState>,
}

impl SnowflakeGenerator {
    fn new(node_id: u16) -> Self {
        Self {
            node_id: AtomicU16::new(node_id.min(MAX_NODE_ID)),
            state: Mutex::new(SnowflakeState {
                last_timestamp: -1,
                sequence: 0,
            }),
        }
    }

    fn current_millis() -> i64 {
        SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_else(|_| Duration::from_secs(0))
            .as_millis() as i64
    }

    fn next_id(&self) -> i64 {
        let mut state = self.state.lock().expect("snowflake generator poisoned");
        let mut timestamp = Self::current_millis().max(CUSTOM_EPOCH);

        // 时钟回拨时采用逻辑时钟，避免在互斥锁内 sleep 阻塞并发请求。
        if timestamp < state.last_timestamp {
            timestamp = state.last_timestamp;
        }

        if timestamp == state.last_timestamp {
            state.sequence = (state.sequence + 1) & SEQUENCE_MASK;
            if state.sequence == 0 {
                // 同毫秒序列号耗尽，推进到逻辑下一毫秒，避免阻塞等待物理时钟。
                timestamp = state.last_timestamp + 1;
            }
        } else {
            state.sequence = 0;
        }

        state.last_timestamp = timestamp;
        let node_id = self.node_id.load(Ordering::Relaxed);
        let timestamp_part = timestamp.saturating_sub(CUSTOM_EPOCH) << TIMESTAMP_SHIFT;
        let node_part = (node_id as i64) << NODE_ID_SHIFT;
        let sequence_part = state.sequence as i64;

        timestamp_part | node_part | sequence_part
    }

    fn set_node_id(&self, node_id: u16) {
        self.node_id
            .store(node_id.min(MAX_NODE_ID), Ordering::Relaxed);
    }
}

/// 生成全局唯一的 i64 ID（用于 user_id 等字段）
pub fn next_id() -> i64 {
    ID_GENERATOR.next_id()
}

fn parse_u64_env(key: &str, default: u64) -> u64 {
    match env::var(key) {
        Ok(raw) => match raw.parse::<u64>() {
            Ok(value) => value,
            Err(_) => {
                warn!(env_key = key, raw_value = %raw, default, "环境变量解析失败，使用默认值");
                default
            }
        },
        Err(_) => default,
    }
}

fn parse_fatal_on_lease_loss() -> bool {
    match env::var(REDIS_NODE_ID_FATAL_ON_LOSS_ENV) {
        Ok(raw) => match parse_bool_flag(&raw) {
            Some(enabled) => enabled,
            None => {
                warn!(
                    raw_value = %raw,
                    "ID_NODE_ID_REDIS_FATAL_ON_LOSS 无法解析为布尔值，按 true 处理"
                );
                true
            }
        },
        Err(_) => true,
    }
}

fn build_default_lease_owner() -> String {
    let host = env::var("HOSTNAME")
        .ok()
        .filter(|v| !v.trim().is_empty())
        .unwrap_or_else(|| "unknown-host".to_string());
    format!("{}:{}:{}", host, process::id(), Uuid::now_v7())
}

async fn acquire_node_id_via_redis(
    conn: &mut ConnectionManager,
    key_prefix: &str,
    owner_token: &str,
    ttl_secs: u64,
) -> Result<Option<u16>, redis::RedisError> {
    for node_id in 0..=MAX_NODE_ID {
        let lease_key = format!("{key_prefix}:{node_id}");
        let acquired: Option<String> = redis::cmd("SET")
            .arg(&lease_key)
            .arg(owner_token)
            .arg("NX")
            .arg("EX")
            .arg(ttl_secs)
            .query_async(conn)
            .await?;
        if acquired.is_some() {
            return Ok(Some(node_id));
        }
    }
    Ok(None)
}

fn spawn_node_id_renew_task(
    mut conn: ConnectionManager,
    lease_key: String,
    owner_token: String,
    ttl_secs: u64,
    renew_interval_secs: u64,
    node_id: u16,
    fatal_on_lease_loss: bool,
) -> JoinHandle<()> {
    tokio::spawn(async move {
        let renew_script = redis::Script::new(
            r#"
            if redis.call("GET", KEYS[1]) == ARGV[1] then
                return redis.call("EXPIRE", KEYS[1], ARGV[2])
            else
                return 0
            end
            "#,
        );
        let mut ticker = tokio::time::interval(Duration::from_secs(renew_interval_secs));
        let mut consecutive_failures: u64 = 0;
        loop {
            ticker.tick().await;
            let renewed: Result<i32, redis::RedisError> = renew_script
                .key(&lease_key)
                .arg(&owner_token)
                .arg(ttl_secs)
                .invoke_async(&mut conn)
                .await;
            match renewed {
                Ok(1) => {
                    consecutive_failures = 0;
                }
                Ok(_) => {
                    error!(
                        node_id,
                        lease_key = %lease_key,
                        "node_id Redis 租约已丢失，停止当前实例避免 ID 冲突"
                    );
                    if fatal_on_lease_loss {
                        process::exit(1);
                    }
                    break;
                }
                Err(err) => {
                    consecutive_failures = consecutive_failures.saturating_add(1);
                    warn!(
                        node_id,
                        lease_key = %lease_key,
                        failures = consecutive_failures,
                        error = %err,
                        "node_id Redis 租约续租失败"
                    );
                    if consecutive_failures.saturating_mul(renew_interval_secs) >= ttl_secs {
                        error!(
                            node_id,
                            ttl_secs,
                            renew_interval_secs,
                            "node_id Redis 租约续租连续失败超过 TTL，停止当前实例避免 ID 冲突"
                        );
                        if fatal_on_lease_loss {
                            process::exit(1);
                        }
                        break;
                    }
                }
            }
        }
    })
}

/// 启动时按需启用 Redis node_id 租约分配
///
/// 通过 `ID_NODE_ID_ALLOCATE_WITH_REDIS=true` 开启。
/// 开启后将自动：
/// - 扫描并抢占一个空闲 node_id（0..=1023）
/// - 覆盖本地 node_id 配置
/// - 后台周期续租，防止分布式节点冲突
pub async fn init_node_id_with_redis(redis_url: &str) -> Result<Option<u16>, String> {
    if !redis_node_id_mode_enabled() {
        return Ok(None);
    }

    let key_prefix = env::var(REDIS_NODE_ID_PREFIX_ENV)
        .ok()
        .filter(|v| !v.trim().is_empty())
        .unwrap_or_else(|| DEFAULT_REDIS_NODE_ID_PREFIX.to_string());
    let owner_token = env::var(REDIS_NODE_ID_OWNER_ENV)
        .ok()
        .filter(|v| !v.trim().is_empty())
        .unwrap_or_else(build_default_lease_owner);
    let ttl_secs = parse_u64_env(REDIS_NODE_ID_TTL_SECS_ENV, DEFAULT_REDIS_NODE_ID_TTL_SECS);
    if ttl_secs == 0 {
        return Err(format!("{REDIS_NODE_ID_TTL_SECS_ENV} 必须大于 0"));
    }

    let mut renew_interval_secs = parse_u64_env(
        REDIS_NODE_ID_RENEW_INTERVAL_ENV,
        DEFAULT_REDIS_NODE_ID_RENEW_INTERVAL_SECS,
    );
    if renew_interval_secs == 0 {
        return Err(format!("{REDIS_NODE_ID_RENEW_INTERVAL_ENV} 必须大于 0"));
    }
    if renew_interval_secs >= ttl_secs {
        let adjusted = (ttl_secs / 3).max(1);
        warn!(
            renew_interval_secs,
            ttl_secs, adjusted, "续租间隔不应大于等于 TTL，已自动下调"
        );
        renew_interval_secs = adjusted;
    }

    let fatal_on_lease_loss = parse_fatal_on_lease_loss();
    let client =
        redis::Client::open(redis_url).map_err(|err| format!("Redis 客户端初始化失败: {err}"))?;
    let mut conn = ConnectionManager::new(client)
        .await
        .map_err(|err| format!("Redis 连接建立失败: {err}"))?;
    let node_id = acquire_node_id_via_redis(&mut conn, &key_prefix, &owner_token, ttl_secs)
        .await
        .map_err(|err| format!("Redis 分配 node_id 失败: {err}"))?
        .ok_or_else(|| format!("node_id 分配失败，租约池已耗尽 (0..={MAX_NODE_ID})"))?;

    ID_GENERATOR.set_node_id(node_id);
    info!(
        node_id,
        key_prefix = %key_prefix,
        ttl_secs,
        renew_interval_secs,
        "通过 Redis 分配 node_id 成功"
    );

    let lease_key = format!("{key_prefix}:{node_id}");
    let renew_conn = conn.clone();
    let _renew_task = spawn_node_id_renew_task(
        renew_conn,
        lease_key,
        owner_token,
        ttl_secs,
        renew_interval_secs,
        node_id,
        fatal_on_lease_loss,
    );

    Ok(Some(node_id))
}

/// 生成12位分享码
pub fn generate_share_code() -> String {
    let mut rng = rand::rng();

    (0..SHARE_CODE_LENGTH)
        .map(|_| {
            let idx = rng.random_range(0..SHARE_CODE_CHARSET.len());
            SHARE_CODE_CHARSET[idx] as char
        })
        .collect()
}

/// 规范化并校验分享码（大小写不敏感，统一转大写）
pub fn normalize_share_code(raw: &str) -> Option<String> {
    let normalized = raw.trim().to_ascii_uppercase();
    if normalized.len() != SHARE_CODE_LENGTH {
        return None;
    }
    if normalized.bytes().all(|c| SHARE_CODE_CHARSET.contains(&c)) {
        Some(normalized)
    } else {
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use redis::AsyncCommands;

    #[test]
    fn next_id_should_be_unique_and_strictly_increasing() {
        let generator = SnowflakeGenerator::new(1);
        let mut last = generator.next_id();
        for _ in 0..10_000 {
            let id = generator.next_id();
            assert!(id > last);
            last = id;
        }
    }

    #[test]
    fn normalize_share_code_accepts_valid_lowercase_input() {
        let normalized = normalize_share_code("ab2cd3ef4gh5");
        assert_eq!(normalized.as_deref(), Some("AB2CD3EF4GH5"));
    }

    #[test]
    fn normalize_share_code_rejects_invalid_format() {
        assert!(normalize_share_code("ABC-123").is_none());
        assert!(normalize_share_code("ABCD0EFGHJKL").is_none());
        assert!(normalize_share_code("SHORT").is_none());
    }

    #[test]
    fn parse_bool_flag_should_support_common_values() {
        assert_eq!(parse_bool_flag("true"), Some(true));
        assert_eq!(parse_bool_flag("ON"), Some(true));
        assert_eq!(parse_bool_flag("0"), Some(false));
        assert_eq!(parse_bool_flag("off"), Some(false));
        assert_eq!(parse_bool_flag("unknown"), None);
    }

    async fn connect_test_redis() -> Option<ConnectionManager> {
        let redis_url =
            env::var("TEST_REDIS_URL").unwrap_or_else(|_| "redis://127.0.0.1:6379/1".to_string());
        let client = match redis::Client::open(redis_url.clone()) {
            Ok(client) => client,
            Err(err) => {
                eprintln!("skip redis test: invalid TEST_REDIS_URL, err={err}");
                return None;
            }
        };
        let mut conn = match ConnectionManager::new(client).await {
            Ok(conn) => conn,
            Err(err) => {
                eprintln!("skip redis test: redis unavailable, err={err}");
                return None;
            }
        };
        let ping: redis::RedisResult<String> = redis::cmd("PING").query_async(&mut conn).await;
        if ping.is_err() {
            eprintln!("skip redis test: redis ping failed");
            return None;
        }
        Some(conn)
    }

    #[tokio::test]
    async fn redis_lease_should_allocate_distinct_node_ids() {
        let mut conn = match connect_test_redis().await {
            Some(conn) => conn,
            None => return,
        };

        let key_prefix = format!("hotspot:test:id-node:{}", Uuid::now_v7());
        let owner1 = format!("owner-{}", Uuid::now_v7());
        let owner2 = format!("owner-{}", Uuid::now_v7());
        let ttl_secs = 20;

        let first = acquire_node_id_via_redis(&mut conn, &key_prefix, &owner1, ttl_secs)
            .await
            .expect("first lease allocate should succeed")
            .expect("first lease should return node_id");
        let second = acquire_node_id_via_redis(&mut conn, &key_prefix, &owner2, ttl_secs)
            .await
            .expect("second lease allocate should succeed")
            .expect("second lease should return node_id");

        assert_ne!(first, second);

        let _: usize = conn.del(format!("{key_prefix}:{first}")).await.unwrap_or(0);
        let _: usize = conn
            .del(format!("{key_prefix}:{second}"))
            .await
            .unwrap_or(0);
    }
}
