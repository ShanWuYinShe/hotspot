use chrono::{DateTime, Utc};
use serde_json::{Map, Value};
use std::collections::HashMap;

use crate::{
    dto::{
        request::{AuditLogDetailRequest, AuditLogListRequest},
        response::{AdminBanRecord, AuditLogDetail, AuditLogItem, PaginatedData, UserInfo},
    },
    error::{AppError, Result},
    repositories::{
        admin_audit_repo::{AdminAuditRepository, AuditLogFilter, AuditLogRow, NewAuditLog},
        user_repo::UserRepository,
    },
    services::media_service::MediaService,
    utils::validation::Validator,
};

const ALLOWED_AUDIT_ACTIONS: [&str; 15] = [
    "user.ban",
    "user.unban",
    "group.dismiss",
    "group.member_mute",
    "group.member_ban",
    "group.member_remove",
    "announcement.create",
    "announcement.delete",
    "role.create",
    "role.update",
    "role.delete",
    "permission.grant",
    "permission.revoke",
    "config.update",
    "tag.delete",
];
const ALLOWED_AUDIT_TARGET_TYPES: [&str; 6] =
    ["user", "group", "announcement", "role", "config", "tag"];

#[derive(Debug, Clone)]
pub struct AuditLogInput {
    pub action: String,
    pub operator_id: i64,
    pub target_type: String,
    pub target_id: String,
    pub target_name: Option<String>,
    pub details: Option<Value>,
    pub before_state: Option<Value>,
    pub after_state: Option<Value>,
    pub ip_address: Option<String>,
    pub user_agent: Option<String>,
}

impl AuditLogInput {
    /// 构建针对用户操作的审计日志输入
    pub fn user_action(
        operator_id: i64,
        action: &str,
        target_user_id: i64,
        details: Option<Value>,
    ) -> Self {
        Self {
            action: action.to_string(),
            operator_id,
            target_type: "user".to_string(),
            target_id: target_user_id.to_string(),
            target_name: None,
            details,
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        }
    }

    /// 构建针对群组操作的审计日志输入
    pub fn group_action(
        operator_id: i64,
        action: &str,
        group_id: impl std::fmt::Display,
        details: Option<Value>,
    ) -> Self {
        Self {
            action: action.to_string(),
            operator_id,
            target_type: "group".to_string(),
            target_id: group_id.to_string(),
            target_name: None,
            details,
            before_state: None,
            after_state: None,
            ip_address: None,
            user_agent: None,
        }
    }
}

#[derive(Clone)]
pub struct AdminAuditService {
    repo: AdminAuditRepository,
    user_repo: UserRepository,
    media_service: MediaService,
}

impl AdminAuditService {
    pub fn new(
        repo: AdminAuditRepository,
        user_repo: UserRepository,
        media_service: MediaService,
    ) -> Self {
        Self {
            repo,
            user_repo,
            media_service,
        }
    }

    pub async fn record(&self, input: AuditLogInput) -> Result<()> {
        let new_log = NewAuditLog {
            action: input.action,
            operator_id: input.operator_id,
            target_type: input.target_type,
            target_id: input.target_id,
            target_name: input.target_name,
            details: input.details,
            before_state: input.before_state,
            after_state: input.after_state,
            ip_address: input.ip_address,
            user_agent: input.user_agent,
        };
        let _ = self.repo.create_log(new_log).await?;
        Ok(())
    }

    pub async fn list_logs(&self, req: AuditLogListRequest) -> Result<PaginatedData<AuditLogItem>> {
        validate_action(&req.action)?;
        validate_target_type(&req.target_type)?;
        let limit = Validator::validate_limit(req.limit, crate::constants::MAX_PAGE_LIMIT)? as i64;
        let cursor = req.cursor.clone();
        let operator_id_filter = req
            .operator_id
            .as_ref()
            .map(|id| Validator::parse_user_id(id))
            .transpose()?;
        let filter = AuditLogFilter {
            action: req.action,
            operator_id: operator_id_filter,
            target_type: req.target_type,
            target_id: req.target_id,
            start_time: parse_time(req.start_time.as_deref(), "start_time")?,
            end_time: parse_time(req.end_time.as_deref(), "end_time")?,
        };

        let (rows, next_cursor) = self.repo.list_logs(filter, limit, cursor).await?;
        let items = self.map_rows_to_items(rows).await?;

        Ok(PaginatedData { items, next_cursor })
    }

    pub async fn get_detail(&self, req: AuditLogDetailRequest) -> Result<AuditLogDetail> {
        let log_id = Validator::parse_uuid(&req.log_id, "log_id")?;
        let row = self
            .repo
            .get_detail(log_id)
            .await?
            .ok_or_else(|| AppError::NotFound("日志不存在".to_string()))?;

        let operator = self.get_operator_info(row.operator_id).await;
        Ok(AuditLogDetail {
            log_id: row.log_id.to_string(),
            action: row.action,
            operator,
            target_type: row.target_type.unwrap_or_default(),
            target_id: row.target_id.unwrap_or_default(),
            target_name: row.target_name,
            details: redact_optional(row.details),
            ip_address: row.ip_address,
            user_agent: row.user_agent,
            created_at: row.created_at.to_rfc3339(),
            before_state: redact_optional(row.before_state),
            after_state: redact_optional(row.after_state),
        })
    }

    pub async fn get_user_ban_history(&self, user_id: i64) -> Result<Option<Vec<AdminBanRecord>>> {
        let rows = self.repo.list_user_ban_logs(&user_id.to_string()).await?;
        if rows.is_empty() {
            return Ok(None);
        }

        let operator_ids: Vec<i64> = rows.iter().map(|r| r.operator_id).collect();
        let operator_map = self.get_operator_map(&operator_ids).await;

        let mut records: Vec<AdminBanRecord> = Vec::new();
        for row in rows {
            let operator = operator_map
                .get(&row.operator_id)
                .cloned()
                .unwrap_or_else(|| fallback_user(row.operator_id));
            if row.action == "user.ban" {
                records.push(AdminBanRecord {
                    banned_at: row.created_at.to_rfc3339(),
                    banned_by: operator,
                    reason: extract_reason(&row.details),
                    duration_days: extract_duration_days(&row.details),
                    unbanned_at: None,
                    unbanned_by: None,
                });
            } else if row.action == "user.unban" {
                if let Some(last) = records.iter_mut().rev().find(|r| r.unbanned_at.is_none()) {
                    last.unbanned_at = Some(row.created_at.to_rfc3339());
                    last.unbanned_by = Some(operator);
                } else {
                    records.push(AdminBanRecord {
                        banned_at: row.created_at.to_rfc3339(),
                        banned_by: operator.clone(),
                        reason: None,
                        duration_days: None,
                        unbanned_at: Some(row.created_at.to_rfc3339()),
                        unbanned_by: Some(operator),
                    });
                }
            }
        }

        Ok(Some(records))
    }

    async fn map_rows_to_items(&self, rows: Vec<AuditLogRow>) -> Result<Vec<AuditLogItem>> {
        let operator_ids: Vec<i64> = rows.iter().map(|r| r.operator_id).collect();
        let operator_map = self.get_operator_map(&operator_ids).await;

        Ok(rows
            .into_iter()
            .map(|r| AuditLogItem {
                log_id: r.log_id.to_string(),
                action: r.action,
                operator: operator_map
                    .get(&r.operator_id)
                    .cloned()
                    .unwrap_or_else(|| fallback_user(r.operator_id)),
                target_type: r.target_type.unwrap_or_default(),
                target_id: r.target_id.unwrap_or_default(),
                target_name: r.target_name,
                details: redact_optional(r.details),
                ip_address: r.ip_address,
                user_agent: r.user_agent,
                created_at: r.created_at.to_rfc3339(),
            })
            .collect())
    }

    async fn get_operator_map(&self, operator_ids: &[i64]) -> HashMap<i64, UserInfo> {
        let mut map = HashMap::new();
        let profiles = self
            .user_repo
            .find_profiles(operator_ids)
            .await
            .unwrap_or_default();
        for profile in profiles {
            let avatar_url = self
                .media_service
                .sign_url_if_path(profile.avatar_url.as_deref())
                .await;
            map.insert(
                profile.user_id,
                UserInfo::new(
                    profile.user_id,
                    profile.nickname.unwrap_or_default(),
                    avatar_url,
                ),
            );
        }
        map
    }

    async fn get_operator_info(&self, operator_id: i64) -> UserInfo {
        let mut map = self.get_operator_map(&[operator_id]).await;
        map.remove(&operator_id)
            .unwrap_or_else(|| fallback_user(operator_id))
    }
}

fn parse_time(input: Option<&str>, field: &str) -> Result<Option<DateTime<Utc>>> {
    let Some(raw) = input else {
        return Ok(None);
    };
    let parsed = DateTime::parse_from_rfc3339(raw)
        .map_err(|_| AppError::InvalidFormat(format!("{} 格式错误", field)))?;
    Ok(Some(parsed.with_timezone(&Utc)))
}

fn fallback_user(user_id: i64) -> UserInfo {
    UserInfo::new(user_id, crate::constants::UNKNOWN_USER_NICKNAME, None)
}

fn redact_optional(value: Option<Value>) -> Option<Value> {
    value.map(|v| redact_value(&v))
}

fn redact_value(value: &Value) -> Value {
    match value {
        Value::Object(map) => {
            let mut result = Map::new();
            for (k, v) in map {
                if is_sensitive_key(k) {
                    result.insert(k.clone(), Value::String("[REDACTED]".to_string()));
                } else {
                    result.insert(k.clone(), redact_value(v));
                }
            }
            Value::Object(result)
        }
        Value::Array(items) => Value::Array(items.iter().map(redact_value).collect()),
        _ => value.clone(),
    }
}

fn is_sensitive_key(key: &str) -> bool {
    let lower = key.to_lowercase();
    matches!(
        lower.as_str(),
        "password"
            | "password_hash"
            | "secret"
            | "token"
            | "api_key"
            | "credit_card"
            | "ssn"
            | "id_card"
    ) || lower.ends_with("_secret")
        || lower.ends_with("_key")
        || lower.ends_with("_token")
}

fn validate_action(action: &Option<String>) -> Result<()> {
    let Some(action) = action else {
        return Ok(());
    };
    if !ALLOWED_AUDIT_ACTIONS.contains(&action.as_str()) {
        return Err(AppError::InvalidFormat("action 无效".to_string()));
    }
    Ok(())
}

fn validate_target_type(target_type: &Option<String>) -> Result<()> {
    let Some(target_type) = target_type else {
        return Ok(());
    };
    if !ALLOWED_AUDIT_TARGET_TYPES.contains(&target_type.as_str()) {
        return Err(AppError::InvalidFormat("target_type 无效".to_string()));
    }
    Ok(())
}

fn extract_reason(details: &Option<Value>) -> Option<String> {
    details
        .as_ref()
        .and_then(|v| v.get("reason"))
        .and_then(|v| v.as_str())
        .map(|s| s.to_string())
}

fn extract_duration_days(details: &Option<Value>) -> Option<i32> {
    details
        .as_ref()
        .and_then(|v| v.get("duration_days"))
        .and_then(|v| v.as_i64())
        .map(|v| v as i32)
}
