//! 位置相关的任务定义
use serde::{Deserialize, Serialize};

/// 签到索引更新任务数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckinIndexJob {
    pub checkin_id: String,
    pub user_id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub timestamp: i64,
}

/// 附近签到通知任务数据
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NearbyCheckinNotificationJob {
    pub checkin_id: String,
    pub creator_id: i64,
    pub creator_nickname: String,
    pub nearby_user_ids: Vec<i64>,
    pub latitude: f64,
    pub longitude: f64,
    pub content: String,
    pub timestamp: i64,
}
