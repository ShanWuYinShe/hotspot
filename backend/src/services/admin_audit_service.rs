use chrono::{DateTime, Utc};
use serde_json::{Map, Value};
use std::collections::HashMap;
use uuid::Uuid;

use crate::{
    dto::{
        request::{
            AuditLogDetailRequest, AuditLogExportRequest, AuditLogExportStatusRequest,
            AuditLogListRequest,
        },
        response::{
            AdminBanRecord, AdminDismissalRecord, AuditLogDetail, AuditLogExportResult,
            AuditLogItem, PaginatedData, PaginationMeta, UserInfo,
        },
    },
    error::{AppError, Result},
    repositories::{
        admin_audit_repo::{
            AdminAuditRepository, AuditExportParams, AuditLogFilter, AuditLogRow, NewAuditLog,
        },
        user_repo::UserRepository,
    },
    services::media_service::MediaService,
    utils::validation::Validator,
};

const EXPORT_LIMIT: i64 = 5000;
const ALLOWED_AUDIT_ACTIONS: [&str; 11] = [
    "user.ban",
    "user.unban",
    "group.dismiss",
    "announcement.create",
    "announcement.delete",
    "role.create",
    "role.update",
    "role.delete",
    "permission.grant",
    "permission.revoke",
    "config.update",
];
const ALLOWED_AUDIT_TARGET_TYPES: [&str; 5] = ["user", "group", "announcement", "role", "config"];

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
        let limit = Validator::validate_limit(req.limit, 100)? as i64;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;
        let filter = AuditLogFilter {
            action: req.action,
            operator_id: req.operator_id,
            target_type: req.target_type,
            target_id: req.target_id,
            start_time: parse_time(req.start_time.as_deref(), "start_time")?,
            end_time: parse_time(req.end_time.as_deref(), "end_time")?,
        };

        let (rows, next_cursor) = self.repo.list_logs(filter, limit, cursor).await?;
        let items = self.map_rows_to_items(rows).await?;

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    pub async fn get_detail(&self, req: AuditLogDetailRequest) -> Result<AuditLogDetail> {
        let log_id = Uuid::parse_str(&req.log_id)
            .map_err(|_| AppError::InvalidFormat("log_id 格式错误".to_string()))?;
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
            target_type: row.target_type,
            target_id: row.target_id,
            target_name: row.target_name,
            details: redact_optional(row.details),
            ip_address: row.ip_address,
            user_agent: row.user_agent,
            created_at: row.created_at.to_rfc3339(),
            before_state: redact_optional(row.before_state),
            after_state: redact_optional(row.after_state),
        })
    }

    pub async fn export_logs(
        &self,
        _operator_id: i64,
        req: AuditLogExportRequest,
    ) -> Result<AuditLogExportResult> {
        validate_action(&req.action)?;
        validate_target_type(&req.target_type)?;
        let format = req.format.as_deref().unwrap_or("csv").to_lowercase();
        if format != "csv" && format != "json" {
            return Err(AppError::InvalidFormat(
                "format 仅支持 csv/json".to_string(),
            ));
        }

        let filter = AuditLogFilter {
            action: req.action.clone(),
            operator_id: req.operator_id,
            target_type: req.target_type.clone(),
            target_id: None,
            start_time: parse_time(req.start_time.as_deref(), "start_time")?,
            end_time: parse_time(req.end_time.as_deref(), "end_time")?,
        };

        let export_row = self
            .repo
            .create_export(AuditExportParams {
                status: "processing".to_string(),
                format: format.clone(),
                action: req.action,
                operator_id: req.operator_id,
                target_type: req.target_type,
                target_id: None,
                start_time: filter.start_time,
                end_time: filter.end_time,
            })
            .await?;

        let export_id = export_row.export_id;
        let rows = self.repo.list_logs_for_export(filter, EXPORT_LIMIT).await?;
        let items = self.map_rows_to_items(rows).await?;

        let content = if format == "json" {
            serde_json::to_string(&items).map_err(|e| AppError::InternalError(e.to_string()))?
        } else {
            build_csv(&items)
        };

        let key = format!("exports/admin_audit/{}.{}", export_id, format);
        let content_type = if format == "json" {
            "application/json"
        } else {
            "text/csv"
        };

        let upload_result = self
            .media_service
            .upload_bytes(&key, content_type, content.into_bytes())
            .await;

        if let Err(e) = upload_result {
            let _ = self
                .repo
                .update_export(export_id, "failed", None, None, Some(e.to_string()))
                .await;
            return Err(AppError::InternalError("导出失败".to_string()));
        }

        let expires_at = Utc::now() + chrono::Duration::hours(1);
        let export_row = self
            .repo
            .update_export(
                export_id,
                "completed",
                Some(key.clone()),
                Some(expires_at),
                None,
            )
            .await?;

        let download_url = self
            .media_service
            .sign_url_with_expiry(&key, std::time::Duration::from_secs(3600))
            .await?;
        Ok(AuditLogExportResult {
            export_id: export_row.export_id.to_string(),
            status: export_row.status,
            download_url: Some(download_url),
            expires_at: Some(expires_at.to_rfc3339()),
        })
    }

    pub async fn export_status(
        &self,
        req: AuditLogExportStatusRequest,
    ) -> Result<AuditLogExportResult> {
        let export_id = Uuid::parse_str(&req.export_id)
            .map_err(|_| AppError::InvalidFormat("export_id 格式错误".to_string()))?;
        let row = self
            .repo
            .get_export(export_id)
            .await?
            .ok_or_else(|| AppError::NotFound("导出任务不存在".to_string()))?;

        let mut download_url = None;
        if row.status == "completed" {
            if let Some(key) = row.download_key.as_deref() {
                if let Some(expires_at) = row.expires_at {
                    let ttl = expires_at.signed_duration_since(Utc::now());
                    if ttl.num_seconds() > 0 {
                        download_url = Some(
                            self.media_service
                                .sign_url_with_expiry(
                                    key,
                                    std::time::Duration::from_secs(ttl.num_seconds() as u64),
                                )
                                .await?,
                        );
                    }
                }
            }
        }

        Ok(AuditLogExportResult {
            export_id: row.export_id.to_string(),
            status: row.status,
            download_url,
            expires_at: row.expires_at.map(|t| t.to_rfc3339()),
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

    pub async fn get_group_dismissal_history(
        &self,
        group_id: &str,
    ) -> Result<Option<Vec<AdminDismissalRecord>>> {
        let rows = self.repo.list_group_dismiss_logs(group_id).await?;
        if rows.is_empty() {
            return Ok(None);
        }

        let operator_ids: Vec<i64> = rows.iter().map(|r| r.operator_id).collect();
        let operator_map = self.get_operator_map(&operator_ids).await;

        let mut records = Vec::new();
        for row in rows {
            let operator = operator_map
                .get(&row.operator_id)
                .cloned()
                .unwrap_or_else(|| fallback_user(row.operator_id));
            records.push(AdminDismissalRecord {
                dismissed_at: row.created_at.to_rfc3339(),
                dismissed_by: operator,
                reason: extract_reason(&row.details),
            });
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
                target_type: r.target_type,
                target_id: r.target_id,
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
                UserInfo {
                    user_id: profile.user_id,
                    nickname: profile.nickname,
                    avatar_url,
                    user_type: profile.user_type,
                },
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
    UserInfo {
        user_id,
        nickname: "未知用户".to_string(),
        avatar_url: None,
        user_type: crate::types::enums::UserType::Normal,
    }
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

fn build_csv(items: &[AuditLogItem]) -> String {
    let mut out = String::new();
    out.push_str("log_id,action,operator_id,operator_nickname,target_type,target_id,target_name,details,ip_address,user_agent,created_at\n");
    for item in items {
        let details = item
            .details
            .as_ref()
            .and_then(|v| serde_json::to_string(v).ok())
            .unwrap_or_default();
        let row = [
            item.log_id.as_str(),
            item.action.as_str(),
            item.operator.user_id.to_string().as_str(),
            item.operator.nickname.as_str(),
            item.target_type.as_str(),
            item.target_id.as_str(),
            item.target_name.as_deref().unwrap_or(""),
            details.as_str(),
            item.ip_address.as_deref().unwrap_or(""),
            item.user_agent.as_deref().unwrap_or(""),
            item.created_at.as_str(),
        ]
        .iter()
        .map(|v| csv_escape(v))
        .collect::<Vec<_>>()
        .join(",");
        out.push_str(&row);
        out.push('\n');
    }
    out
}

fn csv_escape(input: &str) -> String {
    let need_quote = input.contains(',') || input.contains('"') || input.contains('\n');
    if !need_quote {
        return input.to_string();
    }
    let escaped = input.replace('"', "\"\"");
    format!("\"{}\"", escaped)
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
