/// PostgreSQL 举报仓储实现
/// 
/// 功能职责：
/// - 实现举报记录的数据库操作
/// - 管理举报处理流程
/// - 提供举报统计和查询
/// - 支持管理员审核功能

use async_trait::async_trait;
use sqlx::PgPool;
use std::sync::Arc;
use tracing::{debug, error, warn};
use uuid::Uuid;

use crate::domain::{DomainResult, DomainError};

/// 举报目标类型
#[derive(Debug, Clone, PartialEq)]
pub enum ReportTargetType {
    User,
    Moment,
    Comment,
    Message,
    Group,
}

impl ReportTargetType {
    pub fn as_str(&self) -> &str {
        match self {
            ReportTargetType::User => "user",
            ReportTargetType::Moment => "moment",
            ReportTargetType::Comment => "comment",
            ReportTargetType::Message => "message",
            ReportTargetType::Group => "group",
        }
    }
    
    pub fn from_str(s: &str) -> Result<Self, String> {
        match s {
            "user" => Ok(ReportTargetType::User),
            "moment" => Ok(ReportTargetType::Moment),
            "comment" => Ok(ReportTargetType::Comment),
            "message" => Ok(ReportTargetType::Message),
            "group" => Ok(ReportTargetType::Group),
            _ => Err(format!("Unknown report target type: {}", s)),
        }
    }
}

/// 举报原因
#[derive(Debug, Clone, PartialEq)]
pub enum ReportReason {
    Spam,
    Harassment,
    InappropriateContent,
    FakeAccount,
    Violence,
    Other,
}

impl ReportReason {
    pub fn as_str(&self) -> &str {
        match self {
            ReportReason::Spam => "spam",
            ReportReason::Harassment => "harassment",
            ReportReason::InappropriateContent => "inappropriate_content",
            ReportReason::FakeAccount => "fake_account",
            ReportReason::Violence => "violence",
            ReportReason::Other => "other",
        }
    }
    
    pub fn from_str(s: &str) -> Result<Self, String> {
        match s {
            "spam" => Ok(ReportReason::Spam),
            "harassment" => Ok(ReportReason::Harassment),
            "inappropriate_content" => Ok(ReportReason::InappropriateContent),
            "fake_account" => Ok(ReportReason::FakeAccount),
            "violence" => Ok(ReportReason::Violence),
            "other" => Ok(ReportReason::Other),
            _ => Err(format!("Unknown report reason: {}", s)),
        }
    }
}

/// 举报状态
#[derive(Debug, Clone, PartialEq)]
pub enum ReportStatus {
    Pending,
    Reviewing,
    Resolved,
    Dismissed,
}

impl ReportStatus {
    pub fn as_str(&self) -> &str {
        match self {
            ReportStatus::Pending => "pending",
            ReportStatus::Reviewing => "reviewing",
            ReportStatus::Resolved => "resolved",
            ReportStatus::Dismissed => "dismissed",
        }
    }
    
    pub fn from_str(s: &str) -> Result<Self, String> {
        match s {
            "pending" => Ok(ReportStatus::Pending),
            "reviewing" => Ok(ReportStatus::Reviewing),
            "resolved" => Ok(ReportStatus::Resolved),
            "dismissed" => Ok(ReportStatus::Dismissed),
            _ => Err(format!("Unknown report status: {}", s)),
        }
    }
}

/// 举报记录
#[derive(Debug, Clone)]
pub struct Report {
    pub report_id: String,
    pub reporter_id: i64,
    pub reported_user_id: Option<i64>,
    pub target_type: ReportTargetType,
    pub target_id: String,
    pub reason: ReportReason,
    pub description: Option<String>,
    pub status: ReportStatus,
    pub handled_by: Option<i64>,
    pub handled_at: Option<chrono::DateTime<chrono::Utc>>,
    pub handler_note: Option<String>,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

/// 举报仓储接口
#[async_trait]
pub trait ReportRepository: Send + Sync {
    /// 创建举报记录
    async fn create_report(
        &self,
        reporter_id: i64,
        target_type: ReportTargetType,
        target_id: String,
        reason: ReportReason,
        description: Option<String>,
    ) -> DomainResult<Report>;
    
    /// 获取用户的举报记录
    async fn get_user_reports(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Report>>;
    
    /// 获取举报列表（管理员）
    async fn get_reports(
        &self,
        status: Option<ReportStatus>,
        target_type: Option<ReportTargetType>,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Report>>;
    
    /// 处理举报
    async fn handle_report(
        &self,
        report_id: String,
        handler_id: i64,
        action: String,
        note: Option<String>,
    ) -> DomainResult<Report>;
    
    /// 根据ID查找举报
    async fn find_by_id(&self, report_id: String) -> DomainResult<Option<Report>>;
    
    /// 检查重复举报
    async fn check_duplicate_report(
        &self,
        reporter_id: i64,
        target_type: ReportTargetType,
        target_id: String,
    ) -> DomainResult<bool>;
}

/// PostgreSQL 举报仓储实现
pub struct PgReportRepository {
    pool: Arc<PgPool>,
}

impl PgReportRepository {
    /// 创建新的举报仓储
    pub fn new(pool: Arc<PgPool>) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl ReportRepository for PgReportRepository {
    /// 创建举报记录
    async fn create_report(
        &self,
        reporter_id: i64,
        target_type: ReportTargetType,
        target_id: String,
        reason: ReportReason,
        description: Option<String>,
    ) -> DomainResult<Report> {
        debug!("创建举报记录: 举报者={}, 目标类型={:?}, 目标ID={}", reporter_id, target_type, target_id);
        
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开始事务失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        // 检查是否重复举报（同一用户对同一目标的相同类型举报）
        let target_type_str = match target_type {
            ReportTargetType::User => "user",
            ReportTargetType::Group => "group", 
            ReportTargetType::Message => "message",
            ReportTargetType::Moment => "moment",
            ReportTargetType::Comment => "comment",
        };
        
        let existing = sqlx::query!(
            "SELECT report_id FROM reports 
             WHERE reporter_id = $1 AND target_type::text = $2 AND target_id = $3 
             AND status::text IN ('pending', 'investigating')",
            reporter_id,
            target_type_str,
            target_id
        )
        .fetch_optional(&mut *tx)
        .await
        .map_err(|e| {
            error!("检查重复举报失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        if existing.is_some() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "不能重复举报同一目标".to_string(),
            });
        }
        
        // 根据目标类型解析被举报用户ID
        let target_user_id = match target_type {
            ReportTargetType::User => {
                // 验证用户存在
                let user_exists = sqlx::query!(
                    "SELECT id FROM users WHERE id = $1",
                    target_id.parse::<i64>().unwrap_or(0)
                )
                .fetch_optional(&mut *tx)
                .await
                .map_err(|e| {
                    error!("验证用户存在失败: {}", e);
                    DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
                })?;
                
                if user_exists.is_none() {
                    return Err(DomainError::NotFound {
                        resource: "被举报用户".to_string(),
                    });
                }
                
                Some(target_id.parse::<i64>().unwrap_or(0))
            }
            _ => {
                // 对于其他类型（群组、消息、瞬间等），需要查询对应表获取用户ID
                // 这里暂时返回None，实际应该根据具体业务逻辑查询
                None
            }
        };
        
        // 创建举报记录
        let report_id = Uuid::new_v4().to_string();
        let created_at = chrono::Utc::now();
        
        // target_type_str已在上面定义
        
        let reason_str = match reason {
            ReportReason::Spam => "spam",
            ReportReason::Harassment => "harassment",
            ReportReason::InappropriateContent => "inappropriate_content",
            ReportReason::Violence => "violence",
            // 这些变体在枚举定义中不存在，使用Other代替
            // ReportReason::Nudity => "nudity",
            // ReportReason::FakeInfo => "fake_info", 
            // ReportReason::Copyright => "copyright",
            // ReportReason::Privacy => "privacy",
            ReportReason::FakeAccount => "fake_account",
            ReportReason::Other => "other",
        };
        
        sqlx::query(
            "INSERT INTO reports (report_id, reporter_id, target_type, target_id, target_user_id, reason, description, status, created_at, updated_at)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)"
        )
        .bind(&report_id)
        .bind(reporter_id)
        .bind(target_type_str)
        .bind(&target_id)
        .bind(target_user_id)
        .bind(reason_str)
        .bind(&description)
        .bind("pending")
        .bind(created_at)
        .bind(created_at)
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("创建举报记录失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        debug!("举报记录创建成功: {}", report_id);
        
        Ok(Report {
            report_id,
            reporter_id,
            reported_user_id: target_user_id,
            target_type,
            target_id,
            reason,
            description,
            status: ReportStatus::Pending,
            handled_by: None,
            handled_at: None,
            handler_note: None,
            created_at,
        })
    }
    
    /// 获取用户的举报记录
    async fn get_user_reports(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Report>> {
        debug!("获取用户 {} 的举报记录，游标: {}, 限制: {}", user_id, cursor, limit);
        
        let rows = sqlx::query!(
            "SELECT r.report_id, r.reporter_id, r.target_type::text as target_type, r.target_id, r.target_user_id, 
                    r.reason::text as reason, r.description, r.status::text as status, r.handled_by, r.handled_at, r.resolution, r.created_at
             FROM reports r
             WHERE r.reporter_id = $1 
             AND r.id > $2
             ORDER BY r.id ASC
             LIMIT $3",
            user_id,
            cursor,
            limit as i64
        )
        .fetch_all(&*self.pool)
        .await
        .map_err(|e| {
            error!("查询用户举报记录失败: {}", e);
            DomainError::DatabaseError { operation: "database_query".to_string(), details: e.to_string() }
        })?;
        
        let mut reports = Vec::new();
        for row in rows {
            let target_type = match row.target_type.as_deref() {
                Some("user") => ReportTargetType::User,
                Some("group") => ReportTargetType::Group,
                Some("message") => ReportTargetType::Message,
                Some("moment") => ReportTargetType::Moment,
                Some("comment") => ReportTargetType::Comment,
                _ => {
                    warn!("未知的举报目标类型: {:?}", row.target_type);
                    continue;
                }
            };
            
            let reason = match row.reason.as_deref() {
                Some("spam") => ReportReason::Spam,
                Some("harassment") => ReportReason::Harassment,
                Some("inappropriate_content") => ReportReason::InappropriateContent,
                Some("fake_account") => ReportReason::FakeAccount,
                Some("violence") => ReportReason::Violence,
                Some("other") => ReportReason::Other,
                _ => {
                    warn!("未知的举报原因: {:?}", row.reason);
                    continue;
                }
            };
            
            let status = match row.status.as_deref() {
                Some("pending") => ReportStatus::Pending,
                Some("reviewing") => ReportStatus::Reviewing,
                Some("resolved") => ReportStatus::Resolved,
                Some("dismissed") => ReportStatus::Dismissed,
                _ => {
                    warn!("未知的举报状态: {:?}", row.status);
                    continue;
                }
            };
            
            let report = Report {
                report_id: row.report_id,
                reporter_id: row.reporter_id,
                reported_user_id: row.target_user_id,
                target_type,
                target_id: row.target_id,
                reason,
                description: row.description,
                status,
                handled_by: row.handled_by,
                handled_at: row.handled_at,
                handler_note: row.resolution,
                created_at: row.created_at,
            };
            reports.push(report);
        }
        
        debug!("获取到 {} 个举报记录", reports.len());
        Ok(reports)
    }
    
    /// 获取举报列表（管理员）
    async fn get_reports(
        &self,
        status: Option<ReportStatus>,
        target_type: Option<ReportTargetType>,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Report>> {
        debug!("获取举报列表，状态过滤: {:?}, 类型过滤: {:?}", status, target_type);

        // 构建查询条件
        let status_filter = status.as_ref().map(|s| format!("'{}'", s.as_str())).unwrap_or("NULL".to_string());
        let type_filter = target_type.as_ref().map(|t| format!("'{}'", t.as_str())).unwrap_or("NULL".to_string());

        let _status_condition = if status.is_some() {
            format!("AND r.status::text = {}", status_filter)
        } else {
            String::new()
        };

        let _type_condition = if target_type.is_some() {
            format!("AND r.target_type::text = {}", type_filter)
        } else {
            String::new()
        };

        let status_filter = status.as_ref().map(|s| s.as_str()).unwrap_or("");
        let type_filter = target_type.as_ref().map(|t| t.as_str()).unwrap_or("");

        let rows = sqlx::query!(
            r#"
            SELECT
                r.report_id,
                r.reporter_id,
                r.target_user_id,
                r.target_type as "target_type!: String",
                r.target_id,
                r.reason as "reason!: String",
                r.description,
                r.status as "status!: String",
                r.handled_by,
                r.handled_at,
                r.resolution as handler_note,
                r.created_at as "created_at!"
            FROM reports r
            WHERE r.is_deleted = false
              AND ($3 = 0 OR r.id < $3)
              AND ($1 = '' OR r.status::text = $1)
              AND ($2 = '' OR r.target_type::text = $2)
            ORDER BY r.id DESC
            LIMIT $4
            "#,
            status_filter,
            type_filter,
            cursor as i64,
            limit as i64
        )
        .fetch_all(&*self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError {
            operation: "get_admin_reports".to_string(),
            details: e.to_string()
        })?;

        let reports: Vec<Report> = rows.into_iter().filter_map(|row| {
            // 解析目标类型
            let target_type = match ReportTargetType::from_str(&row.target_type) {
                Ok(t) => t,
                Err(_) => {
                    warn!("未知的举报目标类型: {}", row.target_type);
                    return None;
                }
            };
            
            // 解析举报原因
            let reason = match ReportReason::from_str(&row.reason) {
                Ok(r) => r,
                Err(_) => {
                    warn!("未知的举报原因: {}", row.reason);
                    return None;
                }
            };
            
            // 解析举报状态
            let status = match ReportStatus::from_str(&row.status) {
                Ok(s) => s,
                Err(_) => {
                    warn!("未知的举报状态: {}", row.status);
                    return None;
                }
            };
            
            Some(Report {
                report_id: row.report_id,
                reporter_id: row.reporter_id,
                reported_user_id: row.target_user_id,
                target_type,
                target_id: row.target_id,
                reason,
                description: row.description,
                status,
                handled_by: row.handled_by,
                handled_at: row.handled_at,
                handler_note: row.handler_note,
                created_at: row.created_at,
            })
        }).collect();

        Ok(reports)
    }
    
    /// 处理举报
    async fn handle_report(
        &self,
        report_id: String,
        handler_id: i64,
        action: String,
        note: Option<String>,
    ) -> DomainResult<Report> {
        debug!("处理举报 {}，处理人: {}, 动作: {}", report_id, handler_id, action);

        // 验证处理人权限（简化实现，实际应该检查管理员权限）
        // TODO: 验证处理人权限

        let handled_at = chrono::Utc::now();
        let status = match action.as_str() {
            "resolve" => ReportStatus::Resolved,
            "dismiss" => ReportStatus::Dismissed,
            "escalate" => ReportStatus::Reviewing,
            _ => ReportStatus::Reviewing,
        };

        // 获取当前举报记录
        let row = sqlx::query!(
            r#"
            SELECT
                r.report_id,
                r.reporter_id,
                r.target_user_id,
                r.target_type as "target_type!: String",
                r.target_id,
                r.reason as "reason!: String",
                r.description,
                r.status as "status!: String",
                r.handled_by,
                r.handled_at,
                r.resolution as handler_note,
                r.created_at as "created_at!"
            FROM reports r
            WHERE r.report_id = $1 AND r.is_deleted = false
            "#,
            report_id
        )
        .fetch_optional(&*self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError {
            operation: "handle_report".to_string(),
            details: e.to_string()
        })?;

        let report = match row {
            Some(row) => Report {
                report_id: row.report_id,
                reporter_id: row.reporter_id,
                reported_user_id: row.target_user_id,
                target_type: ReportTargetType::from_str(&row.target_type).map_err(|_| DomainError::DatabaseError {
                    operation: "ReportTargetType::from_str".to_string(),
                    details: format!("Invalid target_type: {}", row.target_type),
                })?,
                target_id: row.target_id,
                reason: ReportReason::from_str(&row.reason).map_err(|_| DomainError::DatabaseError {
                    operation: "ReportReason::from_str".to_string(),
                    details: format!("Invalid reason: {}", row.reason),
                })?,
                description: row.description,
                status: ReportStatus::from_str(&row.status).map_err(|_| DomainError::DatabaseError {
                    operation: "ReportStatus::from_str".to_string(),
                    details: format!("Invalid status: {}", row.status),
                })?,
                handled_by: row.handled_by,
                handled_at: row.handled_at,
                handler_note: row.handler_note,
                created_at: row.created_at,
            },
            None => {
                return Err(DomainError::NotFound {
                    resource: "Report".to_string()
                });
            }
        };

        // 更新举报状态
        let status_str = match status {
            ReportStatus::Pending => "pending",
            ReportStatus::Reviewing => "reviewing", 
            ReportStatus::Resolved => "resolved",
            ReportStatus::Dismissed => "dismissed",
        };
        
        sqlx::query!(
            r#"
            UPDATE reports
            SET
                status = $1::text::report_status_enum,
                handled_by = $2,
                handled_at = $3,
                resolution = $4
            WHERE report_id = $5 AND is_deleted = false
            "#,
            status_str,
            Some(handler_id),
            Some(handled_at),
            note,
            report_id
        )
        .execute(&*self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError {
            operation: "handle_report_update".to_string(),
            details: e.to_string()
        })?;

        // 返回更新后的举报记录
        Ok(Report {
            report_id: report.report_id,
            reporter_id: report.reporter_id,
            reported_user_id: report.reported_user_id,
            target_type: report.target_type,
            target_id: report.target_id,
            reason: report.reason,
            description: report.description,
            status, // 使用更新后的状态
            handled_by: Some(handler_id), // 使用更新后的处理人
            handled_at: Some(handled_at), // 使用更新后的处理时间
            handler_note: note, // 使用更新后的处理备注
            created_at: report.created_at,
        })
    }
    
    /// 根据ID查找举报
    async fn find_by_id(&self, report_id: String) -> DomainResult<Option<Report>> {
        debug!("查找举报记录: {}", report_id);

        let row = sqlx::query!(
            r#"
            SELECT
                r.report_id,
                r.reporter_id,
                r.target_user_id,
                r.target_type as "target_type!: String",
                r.target_id,
                r.reason as "reason!: String",
                r.description,
                r.status as "status!: String",
                r.handled_by,
                r.handled_at,
                r.resolution as handler_note,
                r.created_at as "created_at!"
            FROM reports r
            WHERE r.report_id = $1 AND r.is_deleted = false
            "#,
            report_id
        )
        .fetch_optional(&*self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError {
            operation: "find_by_id".to_string(),
            details: e.to_string()
        })?;

        let report = row.and_then(|row| {
            // 解析目标类型
            let target_type = match ReportTargetType::from_str(&row.target_type) {
                Ok(t) => t,
                Err(_) => {
                    warn!("未知的举报目标类型: {}", row.target_type);
                    return None;
                }
            };
            
            // 解析举报原因
            let reason = match ReportReason::from_str(&row.reason) {
                Ok(r) => r,
                Err(_) => {
                    warn!("未知的举报原因: {}", row.reason);
                    return None;
                }
            };
            
            // 解析举报状态
            let status = match ReportStatus::from_str(&row.status) {
                Ok(s) => s,
                Err(_) => {
                    warn!("未知的举报状态: {}", row.status);
                    return None;
                }
            };
            
            Some(Report {
                report_id: row.report_id,
                reporter_id: row.reporter_id,
                reported_user_id: row.target_user_id,
                target_type,
                target_id: row.target_id,
                reason,
                description: row.description,
                status,
                handled_by: row.handled_by,
                handled_at: row.handled_at,
                handler_note: row.handler_note,
                created_at: row.created_at,
            })
        });

        Ok(report)
    }
    
    /// 检查重复举报
    async fn check_duplicate_report(
        &self,
        reporter_id: i64,
        target_type: ReportTargetType,
        target_id: String,
    ) -> DomainResult<bool> {
        debug!("检查重复举报: 举报者={}, 目标类型={:?}, 目标ID={}", reporter_id, target_type, target_id);

        let target_type_str = target_type.as_str();

        let count = sqlx::query_scalar!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM reports r
            WHERE r.reporter_id = $1
              AND r.target_type::text = $2
              AND r.target_id = $3
              AND r.is_deleted = false
              AND r.status NOT IN ('resolved', 'dismissed')
            "#,
            reporter_id,
            target_type_str,
            target_id
        )
        .fetch_one(&*self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError {
            operation: "check_duplicate_report".to_string(),
            details: e.to_string()
        })?;

        Ok(count > 0)
    }
}

impl Clone for PgReportRepository {
    fn clone(&self) -> Self {
        Self {
            pool: Arc::clone(&self.pool),
        }
    }
}
