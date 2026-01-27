use serde::{Deserialize, Serialize};

/// 举报目标 ID：当 target_type = user 时为 number，其它为 string
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(untagged)]
pub enum ReportTargetId {
    String(String),
    Number(i64),
}

/// 通用 ID 类型：string | number
pub type StringOrNumberId = ReportTargetId;

impl std::fmt::Display for ReportTargetId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ReportTargetId::String(value) => write!(f, "{}", value),
            ReportTargetId::Number(value) => write!(f, "{}", value),
        }
    }
}
