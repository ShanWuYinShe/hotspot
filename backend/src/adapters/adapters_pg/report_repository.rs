/// PostgreSQL 举报仓储实现
/// 
/// 功能职责：
/// - 实现举报记录的数据库操作
/// - 管理举报处理流程
/// - 提供举报统计和查询
/// - 支持管理员审核功能

use async_trait::async_trait;
use sqlx::{PgPool, Row};
use std::sync::Arc;
use tracing::{debug, error, info, warn};
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

/// 举报状态
#[derive(Debug, Clone, PartialEq)]
pub enum ReportStatus {
    Pending,
    Reviewing,
    Resolved,
    Dismissed,
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
            DomainError::DatabaseError { message: e.to_string() }
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
             WHERE reporter_id = $1 AND target_type = $2 AND target_id = $3 
             AND status IN ('pending', 'investigating')",
            reporter_id,
            target_type_str,
            target_id
        )
        .fetch_optional(&mut *tx)
        .await
        .map_err(|e| {
            error!("检查重复举报失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
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
                    DomainError::DatabaseError { message: e.to_string() }
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
            ReportReason::HateSpeech => "hate_speech",
            ReportReason::Violence => "violence",
            ReportReason::Nudity => "nudity",
            ReportReason::FakeInfo => "fake_info",
            ReportReason::Copyright => "copyright",
            ReportReason::Privacy => "privacy",
            ReportReason::FakeAccount => "fake_account",
            ReportReason::Other => "other",
        };
        
        sqlx::query!(
            "INSERT INTO reports (report_id, reporter_id, target_type, target_id, target_user_id, reason, description, status, created_at, updated_at)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
            report_id,
            reporter_id,
            target_type_str,
            target_id,
            target_user_id,
            reason_str,
            description,
            "pending",
            created_at,
            created_at
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("创建举报记录失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
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
            "SELECT r.report_id, r.reporter_id, r.target_type, r.target_id, r.target_user_id, 
                    r.reason, r.description, r.status, r.handled_by, r.handled_at, r.resolution, r.created_at
             FROM reports r
             WHERE r.reporter_id = $1 
             AND r.id > $2
             ORDER BY r.id ASC
             LIMIT $3",
            user_id,
            cursor,
            limit
        )
        .fetch_all(&*self.pool)
        .await
        .map_err(|e| {
            error!("查询用户举报记录失败: {}", e);
            DomainError::DatabaseError { message: e.to_string() }
        })?;
        
        let mut reports = Vec::new();
        for row in rows {
            let target_type = match row.target_type.as_str() {
                "user" => ReportTargetType::User,
                "group" => ReportTargetType::Group,
                "message" => ReportTargetType::Message,
                "moment" => ReportTargetType::Moment,
                "comment" => ReportTargetType::Comment,
                _ => {
                    warn!("未知的举报目标类型: {}", row.target_type);
                    continue;
                }
            };
            
            let reason = match row.reason.as_str() {
                "spam" => ReportReason::Spam,
                "harassment" => ReportReason::Harassment,
                "hate_speech" => ReportReason::HateSpeech,
                "violence" => ReportReason::Violence,
                "nudity" => ReportReason::Nudity,
                "fake_info" => ReportReason::FakeInfo,
                "copyright" => ReportReason::Copyright,
                "privacy" => ReportReason::Privacy,
                "fake_account" => ReportReason::FakeAccount,
                "other" => ReportReason::Other,
                _ => {
                    warn!("未知的举报原因: {}", row.reason);
                    continue;
                }
            };
            
            let status = match row.status.as_str() {
                "pending" => ReportStatus::Pending,
                "reviewing" => ReportStatus::Reviewing,
                "resolved" => ReportStatus::Resolved,
                "dismissed" => ReportStatus::Dismissed,
                _ => {
                    warn!("未知的举报状态: {}", row.status);
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
        
        // TODO: 实现管理员举报列表查询
        // 1. 查询reports表
        // 2. 根据状态和类型过滤
        // 3. 支持游标分页
        // 4. 关联相关信息
        
        Ok(vec![])
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
        
        // TODO: 实现举报处理逻辑
        // 1. 验证举报记录存在
        // 2. 验证处理人权限
        // 3. 根据处理动作执行相应操作
        // 4. 更新举报状态和处理信息
        // 5. 记录处理日志
        // 6. 通知举报者处理结果
        
        let handled_at = chrono::Utc::now();
        let status = match action.as_str() {
            "resolve" => ReportStatus::Resolved,
            "dismiss" => ReportStatus::Dismissed,
            _ => ReportStatus::Reviewing,
        };
        
        Ok(Report {
            report_id: report_id.clone(),
            reporter_id: 0, // TODO: 从数据库查询获取
            reported_user_id: None,
            target_type: ReportTargetType::User, // TODO: 从数据库查询获取
            target_id: "".to_string(), // TODO: 从数据库查询获取
            reason: ReportReason::Other, // TODO: 从数据库查询获取
            description: None,
            status,
            handled_by: Some(handler_id),
            handled_at: Some(handled_at),
            handler_note: note,
            created_at: chrono::Utc::now(), // TODO: 从数据库查询获取
        })
    }
    
    /// 根据ID查找举报
    async fn find_by_id(&self, report_id: String) -> DomainResult<Option<Report>> {
        debug!("查找举报记录: {}", report_id);
        
        // TODO: 实现根据ID查找举报
        // 1. 查询reports表
        // 2. 返回举报记录或None
        
        Ok(None)
    }
    
    /// 检查重复举报
    async fn check_duplicate_report(
        &self,
        reporter_id: i64,
        target_type: ReportTargetType,
        target_id: String,
    ) -> DomainResult<bool> {
        debug!("检查重复举报: 举报者={}, 目标类型={:?}, 目标ID={}", reporter_id, target_type, target_id);
        
        // TODO: 实现重复举报检查
        // 1. 查询是否已有相同的举报记录
        // 2. 考虑时间窗口（如24小时内）
        // 3. 返回是否重复
        
        Ok(false)
    }
}

impl Clone for PgReportRepository {
    fn clone(&self) -> Self {
        Self {
            pool: Arc::clone(&self.pool),
        }
    }
}
