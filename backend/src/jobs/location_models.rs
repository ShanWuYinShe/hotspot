//! 位置相关的任务定义
use serde::{Deserialize, Serialize};

/// 生活瞬间索引更新任务数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MomentIndexJob {
    pub moment_id: String,
    pub user_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub timestamp: i64,
}

//
