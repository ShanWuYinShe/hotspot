/// 群组数据访问层 - 三层数据结构实现
/// 
/// 架构说明：
/// 1. Row结构体：仅在此文件内使用，对应数据库字段
/// 2. Domain模型：返回给上层的业务模型
/// 3. 查询优化：只查询必要字段，使用sqlx!宏和非NULL标记

#[allow(dead_code, unused_variables)]
use sqlx::PgPool;
use chrono::{DateTime, Utc};
use crate::domain::{
    DomainError, DomainResult, GroupProfile, GroupAuth, GroupDetail, GroupSearchResult, UserProfile,
    Group, GroupId, UserId
};
use crate::application::queries::{PaginationQuery, LocationQuery};
use crate::application::handlers::{GroupView, UserView, PaginatedResult};
use crate::ports::repository_ports::GroupRepository;
use crate::application::services::message_application_service::GroupWithLastMessage;
use crate::ports::repositories::GroupStats;
use super::query_optimizer::PgQueryOptimizer;

// ===== PostgreSQL 枚举类型映射 =====

/// 群组成员角色枚举
#[derive(Debug, Clone, sqlx::Type)]
#[sqlx(type_name = "group_role")]
#[sqlx(rename_all = "lowercase")]
pub enum GroupRole {
    Owner,
    Admin,
    Member,
}

// ===== 数据库行结构体 (Repository层专用) =====

/// 群组资料查询的轻量级数据库行结构
#[derive(Debug, sqlx::FromRow)]
struct GroupProfileRow {
    group_id: String,
    name: String,
    description: Option<String>,
    owner_id: i64,
    latitude: Option<f64>,  // 修改为Option类型，因为查询中可能为NULL
    longitude: Option<f64>, // 修改为Option类型，因为查询中可能为NULL
    location_name: Option<String>, // 修改为Option类型
    member_count: i32,
    is_public: bool,
    last_active_at: Option<DateTime<Utc>>, // 修改为Option类型
}

/// 群组认证查询的数据库行结构（仅用于密码验证）
#[derive(Debug, sqlx::FromRow)]
struct GroupAuthRow {
    group_id: String,
    name: String,
    password_hash: Option<String>,
    is_public: bool,
    owner_id: i64,
}

/// 群组详细信息查询的数据库行结构
#[derive(Debug, sqlx::FromRow)]
struct GroupDetailRow {
    group_id: String,
    name: String,
    description: Option<String>,
    owner_id: i64,
    latitude: Option<f64>,  // 修改为Option类型
    longitude: Option<f64>, // 修改为Option类型
    location_name: Option<String>, // 修改为Option类型
    member_count: i32,
    max_members: i32,
    is_public: bool,
    created_at: Option<DateTime<Utc>>, // 修改为Option类型
    updated_at: Option<DateTime<Utc>>, // 修改为Option类型
    last_active_at: Option<DateTime<Utc>>, // 修改为Option类型
}

/// 群组搜索结果的数据库行结构
#[derive(Debug, sqlx::FromRow)]
struct GroupSearchRow {
    group_id: String,
    name: String,
    description: Option<String>,
    owner_id: i64,
    latitude: Option<f64>,  // 修改为Option类型
    longitude: Option<f64>, // 修改为Option类型
    location_name: Option<String>, // 修改为Option类型
    member_count: i32,
    is_public: bool,
    last_active_at: Option<DateTime<Utc>>, // 修改为Option类型
    distance: Option<f64>, // 仅地理位置搜索时有值
}

/// 用户资料查询的轻量级数据库行结构（用于成员列表）
#[derive(Debug, sqlx::FromRow)]
struct UserProfileRow {
    user_id: i64,
    nickname: String,
    avatar_url: Option<String>,
    last_active_at: DateTime<Utc>,
}

// ===== Row到Domain的转换实现 =====

impl GroupProfileRow {
    fn to_domain(self) -> GroupProfile {
        GroupProfile::new(
            self.group_id,
            self.name,
            self.description,
            self.owner_id,
            self.latitude.unwrap_or(0.0),  // 提供默认值
            self.longitude.unwrap_or(0.0), // 提供默认值
            self.location_name.unwrap_or_default(), // 提供默认值
            self.member_count,
            crate::domain::entities::group::GroupType::Normal, // 默认群组类型
            if self.is_public { crate::domain::entities::group::Visibility::Public } else { crate::domain::entities::group::Visibility::Private }, // 根据is_public设置可见性
            self.last_active_at.unwrap_or_else(|| chrono::Utc::now()), // 提供默认值
        )
    }
}

impl GroupAuthRow {
    fn to_domain(self) -> GroupAuth {
        GroupAuth::new(
            self.group_id,
            self.name,
            self.password_hash,
            crate::domain::entities::group::GroupType::Normal, // 默认群组类型
            if self.is_public { crate::domain::entities::group::Visibility::Public } else { crate::domain::entities::group::Visibility::Private }, // 根据is_public设置可见性
            self.owner_id,
        )
    }
}

impl GroupDetailRow {
    fn to_domain(self) -> GroupDetail {
        GroupDetail::new(
            self.group_id,
            self.name,
            self.description,
            self.owner_id,
            self.latitude.unwrap_or(0.0),  // 提供默认值
            self.longitude.unwrap_or(0.0), // 提供默认值
            self.location_name.unwrap_or_default(), // 提供默认值
            self.member_count,
            self.max_members,
            crate::domain::entities::group::GroupType::Normal, // 默认群组类型
            if self.is_public { crate::domain::entities::group::Visibility::Public } else { crate::domain::entities::group::Visibility::Private }, // 根据is_public设置可见性
            self.created_at.unwrap_or_else(|| chrono::Utc::now()), // 提供默认值
            self.updated_at.unwrap_or_else(|| chrono::Utc::now()), // 提供默认值
            self.last_active_at.unwrap_or_else(|| chrono::Utc::now()), // 提供默认值
        )
    }
}

impl GroupSearchRow {
    fn to_domain(self, is_password_required: bool, is_member: bool, user_role: Option<String>) -> GroupSearchResult {
        GroupSearchResult::new(
            self.group_id,
            self.name,
            self.description,
            self.owner_id,
            self.latitude.unwrap_or(0.0),  // 提供默认值
            self.longitude.unwrap_or(0.0), // 提供默认值
            self.location_name.unwrap_or_default(), // 提供默认值
            self.member_count,
            crate::domain::entities::group::GroupType::Normal, // 默认群组类型
            if self.is_public { crate::domain::entities::group::Visibility::Public } else { crate::domain::entities::group::Visibility::Private }, // 根据is_public设置可见性
            self.last_active_at.unwrap_or_else(|| chrono::Utc::now()), // 提供默认值
            self.distance,
            is_password_required,
            is_member,
            user_role,
        )
    }
}

impl UserProfileRow {
    fn to_domain(self) -> UserProfile {
        use crate::domain::entities::UserType;
        
        // 暂时将所有用户设为Regular类型
        let user_type = UserType::Regular;

        UserProfile::new(
            self.user_id,
            self.nickname,
            self.avatar_url,
            user_type,
            self.last_active_at,
        )
    }
}

// ===== Repository实现 =====

#[derive(Clone)]
pub struct PgGroupRepository {
    pool: PgPool,
    query_optimizer: PgQueryOptimizer,
}

impl PgGroupRepository {
    pub fn new(pool: PgPool) -> Self {
        // 强制启用查询优化器
        let query_optimizer = PgQueryOptimizer::new(pool.clone());
        Self { 
            pool,
            query_optimizer,
        }
    }
    
    /// 将 GroupDetail 转换为 Group 实体
    fn convert_detail_to_group(&self, detail: GroupDetail) -> DomainResult<Group> {
        use crate::domain::entities::group::*;
        use crate::domain::values::*;
        
        // 创建群组ID
        let group_id = GroupId::new(detail.group_id.clone())?;
        
        // 创建群组名称
        let group_name = GroupName::new(detail.name.clone())?;
        
        // 创建用户ID
        let owner_id = UserId::new(detail.owner_id)?;
        
        // 创建坐标
        let coordinates = Coordinates::new(detail.latitude, detail.longitude)?;
        
        // 创建群组位置
        let location = GroupLocation {
            coordinates,
            location_name: detail.location_name.clone(),
            radius: 1000.0, // 默认半径
        };
        
        // 创建群组类型和可见性
        let group_type = crate::domain::entities::group::GroupType::Normal; // 默认普通类型
        let visibility = detail.visibility; // 直接使用detail中的visibility
        
        // 创建群组实体
        let mut group = Group::new(
            group_id,
            group_name,
            owner_id,
            location,
            group_type,
            visibility,
        );
        
        // 设置其他字段
        group.description = detail.description;
        group.max_members = detail.max_members as u32;
        group.member_count = detail.member_count as u32;
        group.created_at = detail.created_at;
        group.last_active_at = detail.last_active_at;
        
        // 清空事件（这是从数据库加载的，不是新创建的）
        group.events.clear();
        
        Ok(group)
    }
    
    /// 执行查询并进行性能监控
    async fn execute_query_with_monitoring<F, T>(
        &self,
        operation_name: &str,
        query_fn: F,
    ) -> DomainResult<T>
    where
        F: std::future::Future<Output = DomainResult<T>>,
    {
        // 强制使用查询优化器进行监控和优化
        let params_summary = std::collections::HashMap::new();
        
        let (result, metrics) = self.query_optimizer.execute_with_monitoring(
            operation_name,
            params_summary,
            query_fn
        ).await?;
        
        // 记录位置查询监控（如果是位置相关查询）
        if operation_name.contains("location") || operation_name.contains("nearby") {
            let metrics_collector = crate::utils::metrics::get_app_metrics();
            if let Some(ref perf_metrics) = metrics {
                metrics_collector.record_location_query(perf_metrics.execution_time_ms as f64 / 1000.0);
            }
        }
        
        // 记录性能指标
        if let Some(ref perf_metrics) = metrics {
            if perf_metrics.execution_time_ms > 100 {
                tracing::warn!(
                    operation = operation_name,
                    duration_ms = perf_metrics.execution_time_ms,
                    query_type = perf_metrics.query_type,
                    used_index = perf_metrics.used_index,
                    repository = "group_repository",
                    "慢查询检测"
                );
            } else {
                tracing::debug!(
                    operation = operation_name,
                    duration_ms = perf_metrics.execution_time_ms,
                    query_type = perf_metrics.query_type,
                    used_index = perf_metrics.used_index,
                    repository = "group_repository",
                    "查询完成"
                );
            }
        }
        
        Ok(result)
    }

    /// 辅助方法：将字符串转换为 UUID
    fn parse_group_id(&self, group_id: &str) -> DomainResult<uuid::Uuid> {
        uuid::Uuid::parse_str(group_id)
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "Invalid group ID format".to_string(),
            })
    }

    /// 🔒 专门的密码验证方法：只在需要验证密码时查询密码哈希
    pub async fn find_group_password_hash(&self, group_id: &str) -> DomainResult<Option<String>> {
        let group_uuid = uuid::Uuid::parse_str(group_id)
            .map_err(|_| DomainError::BusinessRuleViolation {
                rule: "Invalid group ID format".to_string(),
            })?;
            
        let password_hash = sqlx::query_scalar!(
            r#"
            SELECT password_hash
            FROM groups
            WHERE group_id = $1
            "#,
            group_uuid.to_string()
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(password_hash.flatten())
    }
}

#[async_trait::async_trait]
impl GroupRepository for PgGroupRepository {
    /// 获取群组基础资料（轻量级）
    async fn find_group_profile(&self, group_id: &str) -> DomainResult<Option<GroupProfile>> {
        self.execute_query_with_monitoring("find_group_profile", async {
            let group_uuid = uuid::Uuid::parse_str(group_id)
                .map_err(|_| DomainError::BusinessRuleViolation {
                    rule: "Invalid group ID format".to_string(),
                })?;
            let group_row = sqlx::query_as!(
                GroupProfileRow,
                r#"
                SELECT group_id as "group_id!", name as "name!", description,
                       owner_id as "owner_id!", NULL::DOUBLE PRECISION as latitude, NULL::DOUBLE PRECISION as longitude,
                       NULL::TEXT as location_name, member_count as "member_count!",
                       is_public as "is_public!", updated_at as last_active_at
                FROM groups
                WHERE group_id = $1
                "#,
                group_uuid.to_string()
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            Ok(group_row.map(|row| row.to_domain()))
        }).await
    }

    /// 获取群组认证信息（仅用于密码验证）
    async fn find_group_auth(&self, group_id: &str) -> DomainResult<Option<GroupAuth>> {
        self.execute_query_with_monitoring("find_group_auth", async {
            let group_uuid = uuid::Uuid::parse_str(group_id)
                .map_err(|_| DomainError::BusinessRuleViolation {
                    rule: "Invalid group ID format".to_string(),
                })?;
            let group_row = sqlx::query_as!(
                GroupAuthRow,
                r#"
                SELECT group_id as "group_id!", name as "name!", password_hash,
                       is_public as "is_public!", owner_id as "owner_id!"
                FROM groups
                WHERE group_id = $1
                "#,
                group_uuid.to_string()
            )
            .fetch_optional(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;

            Ok(group_row.map(|row| row.to_domain()))
        }).await
    }

    /// 获取群组详细信息
    async fn find_group_detail(&self, group_id: &str) -> DomainResult<Option<GroupDetail>> {
        let group_uuid = self.parse_group_id(group_id)?;
        let group_row = sqlx::query_as!(
            GroupDetailRow,
            r#"
            SELECT group_id as "group_id!", name as "name!", description,
                   owner_id as "owner_id!", NULL::DOUBLE PRECISION as latitude, NULL::DOUBLE PRECISION as longitude,
                   NULL::TEXT as location_name, member_count as "member_count!",
                   max_members as "max_members!", is_public as "is_public!",
                   created_at as "created_at!", updated_at as "updated_at!", updated_at as last_active_at
            FROM groups
            WHERE group_id = $1
            "#,
            group_uuid.to_string()
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(group_row.map(|row| row.to_domain()))
    }

    /// 按名称搜索群组
    async fn search_groups_by_name(
        &self,
        keyword: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<GroupSearchResult>> {
        let search_pattern = format!("%{}%", keyword);

        let group_rows = sqlx::query_as!(
            GroupSearchRow,
            r#"
            SELECT group_id as "group_id!", name as "name!", description,
                   owner_id as "owner_id!", NULL::DOUBLE PRECISION as latitude, NULL::DOUBLE PRECISION as longitude,
                   NULL::TEXT as location_name, member_count as "member_count!",
                   is_public as "is_public!", updated_at as last_active_at,
                   NULL::DOUBLE PRECISION as distance
            FROM groups
            WHERE name ILIKE $1
              AND ($2 = 0 OR id < $2)
            ORDER BY id DESC
            LIMIT $3
            "#,
            search_pattern,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let mut results = Vec::new();
        for group_row in group_rows {
            // 检查是否需要密码（私有群组且有密码哈希）
            let is_password_required = if !group_row.is_public {
                self.find_group_password_hash(&group_row.group_id).await?.is_some()
            } else {
                false
            };

            results.push(group_row.to_domain(is_password_required, false, None));
        }

        Ok(results)
    }

    /// 按位置搜索附近群组
    async fn find_nearby_groups_with_distance(
        &self,
        _latitude: f64,
        _longitude: f64,
        _radius_meters: f64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<GroupSearchResult>> {
        let group_rows = sqlx::query_as!(
            GroupSearchRow,
            r#"
            SELECT 
                g.group_id as "group_id!",
                g.name as "name!",
                g.description,
                g.owner_id as "owner_id!",
                NULL::DOUBLE PRECISION as latitude,
                NULL::DOUBLE PRECISION as longitude,
                NULL::TEXT as location_name,
                g.member_count as "member_count!",
                g.is_public as "is_public!",
                g.updated_at as last_active_at,
                NULL::DOUBLE PRECISION as distance
            FROM groups g
            WHERE ($1 = 0 OR g.id < $1)
            ORDER BY g.id DESC
            LIMIT $2
            "#,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let mut results = Vec::new();
        for group_row in group_rows {
            let is_password_required = if !group_row.is_public {
                self.find_group_password_hash(&group_row.group_id).await?.is_some()
            } else {
                false
            };

            results.push(group_row.to_domain(is_password_required, false, None));
        }

        Ok(results)
    }

    /// 获取用户加入的群组列表（包含用户角色信息）
    async fn find_user_groups(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<GroupProfile>> {
        let group_rows = sqlx::query!(
            r#"
            SELECT g.group_id as "group_id!", g.name as "name!", g.description,
                   g.owner_id as "owner_id!", NULL::DOUBLE PRECISION as latitude, NULL::DOUBLE PRECISION as longitude,
                   NULL::TEXT as location_name, g.member_count as "member_count!",
                   g.is_public as "is_public!", g.updated_at as last_active_at,
                   gm.role::text as "user_role!"
            FROM groups g
            INNER JOIN group_members gm ON g.group_id = gm.group_id
            WHERE gm.user_id = $1
              AND ($2 = 0 OR g.id < $2)
            ORDER BY g.id DESC
            LIMIT $3
            "#,
            user_id,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let profiles: Vec<GroupProfile> = group_rows.into_iter().map(|row| {
            GroupProfile {
                group_id: row.group_id,
                name: row.name,
                description: row.description,
                owner_id: row.owner_id,
                latitude: row.latitude.unwrap_or(0.0),
                longitude: row.longitude.unwrap_or(0.0),
                location_name: row.location_name.unwrap_or_else(|| "未知位置".to_string()),
                member_count: row.member_count,
                group_type: crate::domain::entities::group::GroupType::Normal, // 默认群组类型
                visibility: if row.is_public { crate::domain::entities::group::Visibility::Public } else { crate::domain::entities::group::Visibility::Private },
                last_active_at: row.last_active_at.unwrap_or_else(chrono::Utc::now),
                // 新增：用户在群组中的角色
                user_role: Some(match row.user_role.as_str() {
                    "owner" => "owner".to_string(),
                    "admin" => "admin".to_string(),
                    "member" => "member".to_string(),
                    _ => "member".to_string(),
                }),
            }
        }).collect();

        Ok(profiles)
    }

    /// 获取用户拥有的群组列表
    async fn find_owned_groups(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<GroupProfile>> {
        let group_rows = sqlx::query_as!(
            GroupProfileRow,
            r#"
            SELECT group_id as "group_id!", name as "name!", description,
                   owner_id as "owner_id!", NULL::DOUBLE PRECISION as latitude, NULL::DOUBLE PRECISION as longitude,
                   NULL::TEXT as location_name, member_count as "member_count!",
                   is_public as "is_public!", updated_at as last_active_at
            FROM groups
            WHERE owner_id = $1
              AND ($2 = 0 OR id < $2)
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(group_rows.into_iter().map(|row| row.to_domain()).collect())
    }

    /// 创建新群组
    async fn create_group(
        &self,
        name: &str,
        description: Option<&str>,
        owner_id: i64,
        _latitude: f64,
        _longitude: f64,
        _location_name: &str,
        is_public: bool,
        password_hash: Option<&str>,
    ) -> DomainResult<GroupProfile> {
        let group_row = sqlx::query_as!(
            GroupProfileRow,
            r#"
            INSERT INTO groups (name, description, owner_id, is_public, password_hash)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING group_id as "group_id!", name as "name!", description,
                      owner_id as "owner_id!", NULL::DOUBLE PRECISION as latitude, NULL::DOUBLE PRECISION as longitude,
                      NULL::TEXT as location_name, member_count as "member_count!",
                      is_public as "is_public!", updated_at as last_active_at
            "#,
            name,
            description,
            owner_id,
            is_public,
            password_hash
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        // 自动将创建者加入群组
        self.add_group_member(&group_row.group_id, owner_id, "owner").await?;

        Ok(group_row.to_domain())
    }

    /// 更新群组信息
    async fn update_group(
        &self,
        group_id: &str,
        name: Option<&str>,
        description: Option<&str>,
    ) -> DomainResult<GroupProfile> {
        let group_uuid = self.parse_group_id(group_id)?;
        let group_row = sqlx::query_as!(
            GroupProfileRow,
            r#"
            UPDATE groups 
            SET name = COALESCE($2, name),
                description = COALESCE($3, description),
                updated_at = NOW()
            WHERE group_id = $1
            RETURNING group_id as "group_id!", name as "name!", description,
                      owner_id as "owner_id!", NULL::DOUBLE PRECISION as latitude, NULL::DOUBLE PRECISION as longitude,
                      NULL::TEXT as location_name, member_count as "member_count!",
                      is_public as "is_public!", updated_at as last_active_at
            "#,
            group_uuid.to_string(),
            name,
            description
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(group_row.to_domain())
    }

    /// 删除群组
    async fn delete_group(&self, group_id: &str) -> DomainResult<()> {
        let group_uuid = self.parse_group_id(group_id)?;
        sqlx::query!(
            r#"
            DELETE FROM groups WHERE group_id = $1
            "#,
            group_uuid.to_string()
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(())
    }

    /// 添加群组成员
    async fn add_group_member(&self, group_id: &str, user_id: i64, role: &str) -> DomainResult<()> {
        // 首先获取群组的数字ID
        let group_numeric_id = sqlx::query_scalar!(
            "SELECT id FROM groups WHERE group_id = $1",
            group_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: "Group not found".to_string(),
        })?;
            
        sqlx::query!(
            r#"
            INSERT INTO group_members (group_id, user_id, role)
            VALUES ($1, $2, $3)
            ON CONFLICT (group_id, user_id) DO NOTHING
            "#,
            group_numeric_id.unwrap(),
            user_id,
            match role {
                "owner" => GroupRole::Owner,
                "admin" => GroupRole::Admin,
                "member" => GroupRole::Member,
                _ => GroupRole::Member, // 默认为普通成员
            } as GroupRole
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(())
    }

    /// 移除群组成员
    async fn remove_group_member(&self, group_id: &str, user_id: i64) -> DomainResult<()> {
        // 首先获取群组的数字ID
        let group_numeric_id = sqlx::query_scalar!(
            "SELECT id FROM groups WHERE group_id = $1",
            group_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: "Group not found".to_string(),
        })?;
        
        sqlx::query!(
            r#"
            DELETE FROM group_members 
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_numeric_id.unwrap(),
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(())
    }

    /// 检查用户是否为群组成员
    async fn is_group_member(&self, group_id: &str, user_id: i64) -> DomainResult<bool> {
        // 首先获取群组的数字ID
        let group_numeric_id = sqlx::query_scalar!(
            "SELECT id FROM groups WHERE group_id = $1",
            group_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: "Group not found".to_string(),
        })?;
        
        let exists = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM group_members 
                WHERE group_id = $1 AND user_id = $2
            ) as "exists!"
            "#,
            group_numeric_id.unwrap(),
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(exists)
    }

    /// 获取用户在群组中的角色
    async fn get_user_role_in_group(&self, group_id: &str, user_id: i64) -> DomainResult<Option<String>> {
        // 首先获取群组的数字ID
        let group_numeric_id = sqlx::query_scalar!(
            "SELECT id FROM groups WHERE group_id = $1",
            group_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: "Group not found".to_string(),
        })?;
        
        let role: Option<GroupRole> = sqlx::query_scalar!(
            r#"
            SELECT role as "role: GroupRole"
            FROM group_members
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_numeric_id.unwrap(),
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(role.map(|r| match r {
            GroupRole::Owner => "owner".to_string(),
            GroupRole::Admin => "admin".to_string(),
            GroupRole::Member => "member".to_string(),
        }))
    }

    /// 转移群组所有权
    async fn transfer_group_ownership(&self, group_id: &str, new_owner_id: i64) -> DomainResult<()> {
        // 首先获取群组的数字ID
        let group_numeric_id = sqlx::query_scalar!(
            "SELECT id FROM groups WHERE group_id = $1",
            group_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: "Group not found".to_string(),
        })?;
        
        // 开启事务
        let mut tx = self.pool.begin().await.map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        // 更新群组所有者
        sqlx::query!(
            r#"
            UPDATE groups 
            SET owner_id = $2, updated_at = NOW()
            WHERE group_id = $1
            "#,
            group_id,
            new_owner_id
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        // 更新成员角色
        sqlx::query!(
            r#"
            UPDATE group_members 
            SET role = 'owner'
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_numeric_id.unwrap(),
            new_owner_id
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        tx.commit().await.map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(())
    }

    /// 获取群组成员列表
    async fn get_group_members(
        &self,
        group_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<UserProfile>> {
        // 首先获取群组的数字 ID
        let group_numeric_id = sqlx::query_scalar!(
            "SELECT id FROM groups WHERE group_id = $1",
            group_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: "Group not found".to_string(),
        })?;

        let user_rows = sqlx::query_as!(
            UserProfileRow,
            r#"
            SELECT u.id as "user_id!", u.nickname as "nickname!", 
                   u.avatar_url, COALESCE(u.updated_at, u.created_at) as "last_active_at!"
            FROM users u
            INNER JOIN group_members gm ON u.id = gm.user_id
            WHERE gm.group_id = $1
              AND ($2 = 0 OR u.id < $2)
            ORDER BY u.id DESC
            LIMIT $3
            "#,
            group_numeric_id.unwrap(),
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(user_rows.into_iter().map(|row| row.to_domain()).collect())
    }

    // === 缺失的 trait 方法实现 ===
    
    async fn save(&self, __group: &Group) -> DomainResult<()> {
        todo!("Implement save method")
    }

    async fn delete_by_id(&self, __group_id: GroupId) -> DomainResult<bool> {
        todo!("Implement delete_by_id method")
    }

    async fn find_by_id(&self, group_id: GroupId, viewer_user_id: Option<UserId>) -> DomainResult<Option<Group>> {
        let group_id_str = group_id.to_string();
        
        let group_row = sqlx::query_as!(
            GroupDetailRow,
            r#"
            SELECT g.group_id as "group_id!", g.name as "name!", g.description, 
                   g.owner_id as "owner_id!", g.latitude, g.longitude, 
                   g.location_name, COUNT(gm.user_id)::int as "member_count!",
                   g.max_members as "max_members!", 
                   g.is_public as "is_public!", 
                   g.created_at, g.updated_at, g.updated_at as "last_active_at"
            FROM groups g
            LEFT JOIN group_members gm ON g.group_id = gm.group_id
            WHERE g.group_id = $1
            GROUP BY g.id, g.group_id, g.name, g.description, g.owner_id, 
                     g.latitude, g.longitude, g.location_name, g.max_members,
                     g.is_public, g.created_at, g.updated_at
            "#,
            group_id_str
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        match group_row {
            Some(row) => {
                // 检查用户是否为成员（如果提供了viewer_user_id）
                let _is_member = if let Some(user_id) = viewer_user_id {
                    self.is_member(&group_id, &user_id).await.unwrap_or(false)
                } else {
                    false
                };
                
                // 返回一个简化的 Group 实体
                let group_detail = row.to_domain();
                let group = self.convert_detail_to_group(group_detail)?;
                Ok(Some(group))
            }
            None => Ok(None),
        }
    }

    async fn search_by_name(
        &self,
        query: &str,
        pagination: &PaginationQuery,
    ) -> DomainResult<PaginatedResult<GroupView>> {
        let limit = pagination.limit.unwrap_or(20).min(100) as i64;
        let cursor = pagination.cursor.unwrap_or(0);
        
        let group_rows = sqlx::query!(
            r#"
            SELECT g.group_id, g.name, g.description, 
                   g.owner_id, g.latitude, g.longitude, 
                   g.location_name, COUNT(gm.user_id)::int as member_count, 
                   (g.password_hash IS NOT NULL) as is_password_required, g.updated_at as last_active_at,
                   u.nickname as creator_name
            FROM groups g
            LEFT JOIN group_members gm ON g.group_id = gm.group_id
            LEFT JOIN users u ON g.owner_id = u.id
            WHERE g.name ILIKE $1
              AND ($2 = 0 OR g.id < $2)
            GROUP BY g.id, g.group_id, g.name, g.description, g.owner_id, 
                     g.latitude, g.longitude, g.location_name, g.password_hash, 
                     g.updated_at, u.nickname
            ORDER BY g.id DESC
            LIMIT $3
            "#,
            format!("%{}%", query),
            cursor as i32,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let has_more = group_rows.len() > limit as usize;
        let mut items = group_rows;
        if has_more {
            items.pop(); // 移除多查询的一条记录
        }

        let next_cursor = items.last().map(|_row| {
            // 简化处理，返回字符串格式的游标
            "0".to_string()
        });

        let group_views: Vec<GroupView> = items.into_iter()
            .map(|row| GroupView {
                group_id: row.group_id,
                name: row.name,
                description: row.description,
                owner_id: row.owner_id.to_string(),
                creator_name: row.creator_name.unwrap_or_else(|| "Unknown".to_string()),
                created_at: row.last_active_at.map(|dt| dt.to_rfc3339()).unwrap_or_else(|| "".to_string()),
                last_active_at: row.last_active_at.map(|dt| dt.to_rfc3339()).unwrap_or_else(|| "".to_string()),
                latitude: row.latitude,
                longitude: row.longitude,
                location_name: row.location_name.unwrap_or_else(|| "Unknown Location".to_string()),
                member_count: row.member_count.unwrap_or(0).max(0) as u32,
                is_password_required: row.is_password_required.unwrap_or(false),
                is_member: false, // 需要额外查询确定
                user_role: None,
                distance: None,
                avatar_url: None,
            })
            .collect();

        Ok(PaginatedResult {
            items: group_views,
            has_more,
            next_cursor,
        })
    }

    async fn search_nearby(
        &self,
        location: &LocationQuery,
        pagination: &PaginationQuery,
    ) -> DomainResult<PaginatedResult<GroupView>> {
        let limit = pagination.limit.unwrap_or(20).min(100) as i64;
        let cursor = pagination.cursor.unwrap_or(0);
        
        let group_rows = sqlx::query!(
            r#"
            SELECT g.group_id, g.name, g.description, g.owner_id, 
                   g.latitude, g.longitude, g.location_name, 
                   (g.password_hash IS NOT NULL) as is_password_required, g.updated_at as last_active_at,
                   COUNT(gm.user_id)::int as member_count,
                   u.nickname as creator_name,
                   ST_Distance(
                       ST_SetSRID(ST_MakePoint(g.longitude, g.latitude), 4326)::geography,
                       ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
                   ) as distance
            FROM groups g
            LEFT JOIN group_members gm ON g.group_id = gm.group_id
            LEFT JOIN users u ON g.owner_id = u.id
            WHERE g.latitude IS NOT NULL 
              AND g.longitude IS NOT NULL
              AND ST_DWithin(
                  ST_SetSRID(ST_MakePoint(g.longitude, g.latitude), 4326)::geography,
                  ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                  $3
              )
              AND ($4 = 0 OR g.id < $4)
            GROUP BY g.id, g.group_id, g.name, g.description, g.owner_id, 
                     g.latitude, g.longitude, g.location_name, g.password_hash, 
                     g.updated_at, u.nickname
            ORDER BY distance ASC, g.id DESC
            LIMIT $5
            "#,
            location.center.longitude.value(),
            location.center.latitude.value(),
            location.radius,
            cursor as i32,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let has_more = group_rows.len() > limit as usize;
        let mut items = group_rows;
        if has_more {
            items.pop(); // 移除多查询的一条记录
        }

        let next_cursor = items.last().map(|_| "0".to_string()); // 简化处理

        let group_views: Vec<GroupView> = items.into_iter()
            .map(|row| GroupView {
                group_id: row.group_id,
                name: row.name,
                description: row.description,
                owner_id: row.owner_id.to_string(),
                creator_name: row.creator_name.unwrap_or_else(|| "Unknown".to_string()),
                created_at: row.last_active_at.map(|dt| dt.to_rfc3339()).unwrap_or_else(|| "".to_string()),
                last_active_at: row.last_active_at.map(|dt| dt.to_rfc3339()).unwrap_or_else(|| "".to_string()),
                latitude: row.latitude,
                longitude: row.longitude,
                location_name: row.location_name.unwrap_or_else(|| "Unknown Location".to_string()),
                member_count: row.member_count.unwrap_or(0).max(0) as u32,
                is_password_required: row.is_password_required.unwrap_or(false),
                is_member: false, // 需要额外查询确定
                user_role: None,
                distance: row.distance,
                avatar_url: None,
            })
            .collect();

        Ok(PaginatedResult {
            items: group_views,
            has_more,
            next_cursor,
        })
    }

    async fn get_members(
        &self,
        group_id: GroupId,
        pagination: &PaginationQuery,
    ) -> DomainResult<PaginatedResult<UserView>> {
        let group_id_str = group_id.to_string();
        let limit = pagination.limit.unwrap_or(20).min(100) as i64;
        let cursor = pagination.cursor.unwrap_or(0);
        
        // 首先获取群组的数字ID
        let group_numeric_id = sqlx::query_scalar!(
            "SELECT id FROM groups WHERE group_id = $1",
            group_id_str
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: "Group not found".to_string(),
        })?;

        let member_rows = sqlx::query!(
            r#"
            SELECT u.id as user_id, u.login_id, u.nickname, u.avatar_url, u.bio,
                   COALESCE(u.updated_at, u.created_at) as last_active_at,
                   gm.role::text as member_role, gm.joined_at, u.created_at
            FROM users u
            INNER JOIN group_members gm ON u.id = gm.user_id
            WHERE gm.group_id = $1
              AND ($2 = 0 OR u.id < $2)
            ORDER BY u.id DESC
            LIMIT $3
            "#,
            group_numeric_id.unwrap(),
            cursor as i32,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let has_more = member_rows.len() > limit as usize;
        let mut items = member_rows;
        if has_more {
            items.pop(); // 移除多查询的一条记录
        }

        let next_cursor = items.last().map(|row| row.user_id.to_string());

        let user_views: Vec<UserView> = items.into_iter()
            .map(|row| UserView {
                user_id: row.user_id.to_string(),
                login_id: row.login_id,
                nickname: row.nickname.unwrap_or_else(|| "Unknown".to_string()),
                avatar_url: row.avatar_url,
                email: row.email,
                bio: row.bio,
                is_temp: row.expires_at.is_some(),
                created_at: row.created_at.map(|dt| dt.to_rfc3339()).unwrap_or_else(|| "".to_string()),
                last_active_at: row.last_active_at.map(|dt| dt.to_rfc3339()).unwrap_or_else(|| "".to_string()),
                latitude: None,
                longitude: None,
                location_name: None,
                distance: None,
            })
            .collect();

        Ok(PaginatedResult {
            items: user_views,
            has_more,
            next_cursor,
        })
    }

    async fn get_user_groups(
        &self,
        _user_id: UserId,
        _pagination: &PaginationQuery,
    ) -> DomainResult<PaginatedResult<GroupView>> {
        // TODO: 实现获取用户群组逻辑
        todo!("Implement get_user_groups method")
    }

    async fn get_user_owned_groups(
        &self,
        _user_id: UserId,
        _pagination: &PaginationQuery,
    ) -> DomainResult<PaginatedResult<GroupView>> {
        // TODO: 实现获取用户拥有的群组逻辑
        todo!("Implement get_user_owned_groups method")
    }

    async fn is_member(&self, group_id: &GroupId, user_id: &UserId) -> DomainResult<bool> {
        let group_id_str = group_id.to_string();
        let user_id_val = user_id.value();
        
        // 首先获取群组的数字ID
        let group_numeric_id = sqlx::query_scalar!(
            "SELECT id FROM groups WHERE group_id = $1",
            group_id_str
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?
        .ok_or_else(|| DomainError::BusinessRuleViolation {
            rule: "Group not found".to_string(),
        })?;

        let is_member = sqlx::query_scalar!(
            "SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2) as \"exists!\"",
            group_numeric_id.unwrap(),
            user_id_val
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(is_member)
    }

    // 重复的is_member方法已移除，使用上面的实现
    
    async fn get_group_stats(&self, _group_id: GroupId) -> DomainResult<GroupStats> {
        // TODO: 实现获取群组统计信息逻辑
        todo!("Implement get_group_stats method")
    }
    
    /// 根据用户ID查找群组列表
    async fn find_by_user_id(&self, user_id: &UserId) -> DomainResult<Vec<Group>> {
        // 简化实现，返回空列表
        Ok(vec![])
    }
    
    /// 根据用户ID查找群组列表（包含最后消息信息）
    async fn find_by_user_id_with_last_message(
        &self,
        user_id: &UserId,
        cursor: Option<&str>,
        limit: i32,
        include_archived: bool,
    ) -> DomainResult<PaginatedResult<GroupWithLastMessage>> {
        // 简化实现，返回空的分页结果
        Ok(PaginatedResult {
            items: vec![],
            has_more: false,
            next_cursor: None,
        })
    }
}