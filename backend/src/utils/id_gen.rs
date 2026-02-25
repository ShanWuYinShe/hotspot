//! ID生成器
//!
//! 提供分布式唯一 ID 与分享码生成功能。

use once_cell::sync::Lazy;
use rand::Rng;
use std::env;
use std::sync::Mutex;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

const NODE_ID_BITS: u8 = 10;
const SEQUENCE_BITS: u8 = 12;
const NODE_ID_SHIFT: u8 = SEQUENCE_BITS;
const TIMESTAMP_SHIFT: u8 = NODE_ID_BITS + SEQUENCE_BITS;
const MAX_NODE_ID: u16 = (1 << NODE_ID_BITS) - 1;
const SEQUENCE_MASK: u16 = (1 << SEQUENCE_BITS) - 1;
// 2025-01-01T00:00:00Z，自定义 epoch（毫秒）
const CUSTOM_EPOCH: i64 = 1_735_689_600_000;

static ID_GENERATOR: Lazy<SnowflakeGenerator> = Lazy::new(|| {
    let node_id = env::var("ID_NODE_ID")
        .ok()
        .and_then(|v| v.parse::<u16>().ok())
        .unwrap_or(0);
    SnowflakeGenerator::new(node_id)
});

struct SnowflakeState {
    last_timestamp: i64,
    sequence: u16,
}

struct SnowflakeGenerator {
    node_id: u16,
    state: Mutex<SnowflakeState>,
}

impl SnowflakeGenerator {
    fn new(node_id: u16) -> Self {
        Self {
            node_id: node_id.min(MAX_NODE_ID),
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

    fn wait_next_millis(last_timestamp: i64) -> i64 {
        let mut timestamp = Self::current_millis();
        while timestamp <= last_timestamp {
            std::thread::sleep(Duration::from_micros(100));
            timestamp = Self::current_millis();
        }
        timestamp
    }

    fn next_id(&self) -> i64 {
        let mut state = self.state.lock().expect("snowflake generator poisoned");
        let mut timestamp = Self::current_millis();

        if timestamp < state.last_timestamp {
            let wait_ms = (state.last_timestamp - timestamp) as u64;
            std::thread::sleep(Duration::from_millis(wait_ms));
            timestamp = Self::current_millis();
            if timestamp < state.last_timestamp {
                timestamp = state.last_timestamp;
            }
        }

        if timestamp == state.last_timestamp {
            state.sequence = (state.sequence + 1) & SEQUENCE_MASK;
            if state.sequence == 0 {
                timestamp = Self::wait_next_millis(state.last_timestamp);
            }
        } else {
            state.sequence = 0;
        }

        state.last_timestamp = timestamp;
        let timestamp_part = (timestamp - CUSTOM_EPOCH) << TIMESTAMP_SHIFT;
        let node_part = (self.node_id as i64) << NODE_ID_SHIFT;
        let sequence_part = state.sequence as i64;

        timestamp_part | node_part | sequence_part
    }
}

/// 生成全局唯一的 i64 ID（用于 user_id 等字段）
pub fn next_id() -> i64 {
    ID_GENERATOR.next_id()
}

/// 生成12位分享码
pub fn generate_share_code() -> String {
    const CHARSET: &[u8] = b"ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
    let mut rng = rand::rng();

    (0..12)
        .map(|_| {
            let idx = rng.random_range(0..CHARSET.len());
            CHARSET[idx] as char
        })
        .collect()
}
