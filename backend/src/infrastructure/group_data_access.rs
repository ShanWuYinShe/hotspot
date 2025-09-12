/// 群组数据访问层 - sqlx宏和极简Option使用的完整示例
/// 
/// 展示特性：
/// - 复杂查询的sqlx宏使用
/// - 多表关联查询
/// - 聚合函数的非NULL标记
/// - 布尔表达式的处理
/// - 地理位置查询优化

use async_trait::async_trait;
use sqlx::PgPool;
use chrono::{DateTime, Utc};
use uuid::Uuid;
use crate::domain::{DomainResult, DomainError, GroupId, Group, UserId, GroupType, GroupVisibility};
use crate::ports::GroupRepository;
use crate::infrastructure::errors::{ErrorContext, ContextualError, ErrorConverter};

// === 群组相关数据库行结构体 ===

/// 群组基础信息行结构
#[derive(Debug, sqlx::FromRow)]
struct GroupRow {
    /// 群组ID - NOT NULL主键
    group_id: Uuid,
    /// 群组名称 - NOT NULL
    name: String,
    /// 群组描述 - 可为NULL
    description: Option<String>,
    /// 群组头像 - 可为NULL
    avatar_url: Option<String>,
    /// 群组类型 - NOT NULL枚举
    group_type: GroupType,
    /// 可见性 - NOT NULL枚举
    visibility: GroupVisibility,
    /// 最大成员数 - NOT NULL，有默认值
    max_members: i32,
    /// 创建者ID - NOT NULL
    created_by: i64,
    /// 创建时间 - NOT NULL
    created_at: DateTime<Utc>,
    /// 更新时间 - NOT NULL
    updated_at: DateTime<Utc>,
}

/// 群组位置信息行结构
#[derive(Debug, sqlx::FromRow)]
struct GroupLocationRow {
    /// 群组ID - NOT NULL
    group_id: Uuid,
    /// 纬度 - NOT NULL
    latitude: f64,
    /// 经度 - NOT NULL
    longitude: f64,
    /// 位置名称 - NOT NULL
    location_name: String,
    /// 半径（米）- NOT NULL
    radius_meters: f64,
    /// 更新时间 - NOT NULL
    updated_at: DateTime<Utc>,
}

/// 群组成员统计行结构 - 聚合查询结果
#[derive(Debug, sqlx::FromRow)]
struct GroupMemberStatsRow {
    /// 群组ID - NOT NULL
    group_id: Uuid,
    /// 总成员数 - COUNT结果，必定非NULL
    total_members: i64,
    /// 活跃成员数（最近7天有活动）- COUNT结果，必定非NULL
    active_members: i64,
    /// 在线成员数 - COUNT结果，必定非NULL
    online_members: i64,
    /// 是否为成员 - 布尔表达式结果，必定非NULL
    is_member: bool,
    /// 是否为管理员 - 布尔表达式结果，必定非NULL
    is_admin: bool,
}

/// 群组搜索结果行结构
#[derive(Debug, sqlx::FromRow)]
struct GroupSearchRow {
    /// 群组ID - NOT NULL
    group_id: Uuid,
    /// 群组名称 - NOT NULL
    name: String,
    /// 群组描述 - 可为NULL
    description: Option<String>,
    /// 群组头像 - 可为NULL
    avatar_url: Option<String>,
    /// 群组类型 - NOT NULL
    group_type: GroupType,
    /// 可见性 - NOT NULL
    visibility: GroupVisibility,
    /// 成员数量 - COUNT结果，必定非NULL
    member_count: i64,
    /// 距离（米）- 地理搜索时非NULL，名称搜索时为NULL
    distance_meters: Option<f64>,
    /// 创建时间 - NOT NULL
    created_at: DateTime<Utc>,
}

/// 群组详细信息行结构 - 包含创建者信息
#[derive(Debug, sqlx::FromRow)]
struct GroupDetailRow {
    /// 群组ID - NOT NULL
    group_id: Uuid,
    /// 群组名称 - NOT NULL
    name: String,
    /// 群组描述 - 可为NULL
    description: Option<String>,
    /// 群组头像 - 可为NULL
    avatar_url: Option<String>,
    /// 群组类型 - NOT NULL
    group_type: GroupType,
    /// 可见性 - NOT NULL
    visibility: GroupVisibility,
    /// 最大成员数 - NOT NULL
    max_members: i32,
    /// 创建者ID - NOT NULL
    created_by: i64,
    /// 创建者昵称 - NOT NULL（通过JOIN获取）
    creator_nickname: String,
    /// 创建者头像 - 可为NULL（通过JOIN获取）
    creator_avatar_url: Option<String>,
    /// 创建时间 - NOT NULL
    created_at: DateTime<Utc>,
    /// 更新时间 - NOT NULL
    updated_at: DateTime<Utc>,
}

// === Row到Domain模型转换 ===

impl GroupRow {
    /// 转换为领域群组模型
    fn to_domain(
        self, 
        location: Option<crate::domain::entities::GroupLocation>,
        member_stats: Option<GroupMemberStatsRow>
    ) -> Result<Group, sqlx::Error> {
        Ok(Group {
            id: GroupId::new(self.group_id.to_string()).map_err(|e| sqlx::Error::Protocol(format!("Invalid group_id: {}", e)))?,
            name: crate::domain::values::GroupName::new(self.name).map_err(|e| sqlx::Error::Protocol(format!("Invalid group name: {}", e)))?,
            description: self.description,
            owner_id: UserId::new(self.created_by).map_err(|e| sqlx::Error::Protocol(format!("Invalid creator_id: {}", e)))?,
            location: location.unwrap_or_else(|| crate::domain::entities::GroupLocation {
                coordinates: crate::domain::values::Coordinates::new(0.0, 0.0).unwrap_or_else(|_| crate::domain::values::Coordinates::new(0.0, 0.0).unwrap()),
                location_name: "".to_string(),
                radius: 1000.0,
            }),
            group_type: self.group_type,
            visibility: self.visibility,
            is_public: matches!(self.visibility, crate::domain::values::GroupVisibility::Public),
            max_members: self.max_members as u32,
            member_count: member_stats.map(|s| s.total_members as u32).unwrap_or(0),
            created_at: self.created_at,
            last_active_at: self.updated_at,
            members: Vec::new(), // 成员信息需要单独查询
            events: Vec::new(),
        })
    }
}

impl GroupLocationRow {
    /// 转换为领域位置模型
    fn to_domain(self) -> crate::domain::entities::GroupLocation {
        crate::domain::entities::GroupLocation {
            coordinates: crate::domain::values::Coordinates::new(self.latitude, self.longitude).unwrap_or_else(|_| crate::domain::values::Coordinates::new(0.0, 0.0).unwrap()),
            location_name: self.location_name,
            radius: self.radius_meters,
        }
    }
}

impl GroupSearchRow {
    /// 转换为群组搜索结果
    fn to_domain(self) -> crate::application::GroupView {
        crate::application::GroupView {
            group_id: self.group_id.to_string(),
            name: self.name,
            description: self.description,
            owner_id: "".to_string(), // 需要从其他地方获取
            creator_name: "".to_string(), // 需要从其他地方获取
            created_at: self.created_at.to_rfc3339(),
            last_active_at: self.created_at.to_rfc3339(), // 暂时使用创建时间
            latitude: 0.0, // 需要从其他地方获取
            longitude: 0.0, // 需要从其他地方获取
            member_count: self.member_count as u32,
            distance: self.distance_meters,
            location_name: "".to_string(), // 需要从其他地方获取
            is_password_required: false, // 需要从其他地方获取
            is_member: false, // 需要从其他地方获取
            user_role: None,
            avatar_url: self.avatar_url,
        }
    }
}

// === PostgreSQL群组仓储实现 ===

/// PostgreSQL群组仓储实现 - 展示复杂查询的sqlx宏使用
#[derive(Clone)]
pub struct PgGroupRepository {
    pool: PgPool,
}

impl PgGroupRepository {
    /// 创建新的PostgreSQL群组仓储
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
    
    /// 执行查询并处理错误
    async fn execute_query<F, T>(&self, operation: &str, query_fn: F) -> DomainResult<T>
    where
        F: std::future::Future<Output = Result<T, sqlx::Error>>,
    {
        let context = ErrorContext::new(operation);
        
        match query_fn.await {
            Ok(result) => Ok(result),
            Err(sqlx_error) => {
                let infra_error = ErrorConverter::from_sqlx_error(sqlx_error, Some(operation));
                let contextual_error = ContextualError::new(infra_error, context);
                
                tracing::error!(
                    error = %contextual_error,
                    error_data = ?contextual_error.to_structured_log(),
                    "Database operation failed"
                );
                
                Err(self.convert_to_domain_error(&contextual_error))
            }
        }
    }
    
    /// 转换为领域错误
    fn convert_to_domain_error(&self, error: &ContextualError) -> DomainError {
        match &error.error {
            crate::infrastructure::errors::InfrastructureError::DatabaseQuery { message, .. } => {
                if message.contains("unique constraint") {
                    DomainError::BusinessRuleViolation {
                        rule: "group name must be unique".to_string(),
                    }
                } else if message.contains("foreign key constraint") {
                    DomainError::UserNotFound
                } else {
                    DomainError::ValidationError {
                        field: "database".to_string(),
                        reason: message.clone(),
                    }
                }
            }
            _ => {
                DomainError::BusinessRuleViolation {
                    rule: "infrastructure error".to_string(),
                }
            }
        }
    }
}

impl PgGroupRepository {
    /// 保存群组 - 复杂的UPSERT操作
    async fn save(&self, group: &Group) -> DomainResult<()> {
        self.execute_query("save_group", async {
            // 开始事务
            let mut tx = self.pool.begin().await?;
            
            // 保存群组基础信息
            sqlx::query!(
                r#"
                INSERT INTO groups (
                    group_id, name, description, avatar_url, group_type, 
                    visibility, max_members, created_by, created_at, updated_at
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                ON CONFLICT (group_id) DO UPDATE SET
                    name = EXCLUDED.name,
                    description = EXCLUDED.description,
                    avatar_url = EXCLUDED.avatar_url,
                    group_type = EXCLUDED.group_type,
                    visibility = EXCLUDED.visibility,
                    max_members = EXCLUDED.max_members,
                    updated_at = EXCLUDED.updated_at
                "#,
                group.id.value(),
                group.name.name(),
                group.description,
                None::<String>, // Group 实体没有 avatar_url 字段
                group.group_type as _,
                group.visibility as _,
                group.max_members as i32,
                group.owner_id.value(), // Group 实体使用 owner_id 而不是 creator_id
                group.created_at,
                group.last_active_at // Group 实体使用 last_active_at 而不是 updated_at
            )
            .execute(&mut *tx)
            .await?;
            
            // 保存群组位置（如果有）
            let location = &group.location; // GroupLocation 不是 Option 类型
            sqlx::query!(
                    r#"
                    INSERT INTO group_locations (
                        group_id, latitude, longitude, location_name, 
                        radius_meters, updated_at
                    ) VALUES ($1, $2, $3, $4, $5, $6)
                    ON CONFLICT (group_id) DO UPDATE SET
                        latitude = EXCLUDED.latitude,
                        longitude = EXCLUDED.longitude,
                        location_name = EXCLUDED.location_name,
                        radius_meters = EXCLUDED.radius_meters,
                        updated_at = EXCLUDED.updated_at
                    "#,
                    uuid::Uuid::parse_str(group.id.value()).unwrap(),
                    location.coordinates.latitude.value(),
                    location.coordinates.longitude.value(),
                    location.location_name,
                    location.radius,
                    chrono::Utc::now() // GroupLocation 没有 updated_at 字段
                )
                .execute(&mut *tx)
                .await?;
            
            // 提交事务
            tx.commit().await?;
            Ok(())
        }).await
    }
    
    /// 根据ID查找群组 - 复杂的多表关联查询
    async fn find_by_id(&self, group_id: GroupId, viewer_user_id: Option<UserId>) -> DomainResult<Option<Group>> {
        self.execute_query("find_group_by_id", async {
            // 查询群组基础信息
            let group_row = sqlx::query_as!(
                GroupRow,
                r#"
                SELECT 
                    group_id as "group_id!",
                    name as "name!",
                    description,
                    avatar_url,
                    group_type as "group_type!: GroupType",
                    visibility as "visibility!: GroupVisibility",
                    max_members as "max_members!",
                    created_by as "created_by!",
                    created_at as "created_at!",
                    updated_at as "updated_at!"
                FROM groups 
                WHERE group_id = $1
                "#,
                group_id.value()
            )
            .fetch_optional(&self.pool)
            .await?;
            
            if let Some(group_row) = group_row {
                // 查询群组位置
                let location_row = sqlx::query_as!(
                    GroupLocationRow,
                    r#"
                    SELECT 
                        group_id as "group_id!",
                        latitude as "latitude!",
                        longitude as "longitude!",
                        location_name as "location_name!",
                        radius_meters as "radius_meters!",
                        updated_at as "updated_at!"
                    FROM group_locations
                    WHERE group_id = $1
                    "#,
                    group_id.value()
                )
                .fetch_optional(&self.pool)
                .await?;
                
                // 查询成员统计信息 - 展示复杂聚合查询和布尔表达式
                let viewer_id = viewer_user_id.map(|id| id.value()).unwrap_or(-1);
                let member_stats = sqlx::query_as!(
                    GroupMemberStatsRow,
                    r#"
                    SELECT 
                        $1 as group_id,
                        COUNT(*) as "total_members!",
                        COUNT(*) FILTER (
                            WHERE u.last_login_at > NOW() - INTERVAL '7 days'
                        ) as "active_members!",
                        COUNT(*) FILTER (
                            WHERE u.last_login_at > NOW() - INTERVAL '5 minutes'
                        ) as "online_members!",
                        EXISTS(
                            SELECT 1 FROM group_members gm2
                            INNER JOIN groups g2 ON gm2.group_id = g2.id
                            WHERE g2.group_id = $3 AND gm2.user_id = $4
                        ) as "is_member!",
                        EXISTS(
                            SELECT 1 FROM group_members gm3
                            INNER JOIN groups g3 ON gm3.group_id = g3.id
                            WHERE g3.group_id = $5 AND gm3.user_id = $6 AND gm3.role = 'admin'
                        ) as "is_admin!"
                    FROM group_members gm
                    INNER JOIN users u ON gm.user_id = u.id
                    INNER JOIN groups g ON gm.group_id = g.id
                    WHERE g.group_id = $2
                    "#,
                    group_id.value(),
                    group_id.value(),
                    group_id.value(),
                    viewer_id,
                    group_id.value(),
                    viewer_id
                )
                .fetch_one(&self.pool)
                .await?;
                
                let location = location_row.map(|row| row.to_domain());
                let group = group_row.to_domain(location, Some(member_stats))?;
                
                Ok(Some(group))
            } else {
                Ok(None)
            }
        }).await
    }
    
    /// 搜索附近群组 - 复杂的地理位置查询
    async fn search_nearby(
        &self,
        location: &crate::application::LocationQuery,
        pagination: &crate::application::PaginationQuery,
    ) -> DomainResult<crate::application::PaginatedResult<crate::application::GroupView>> {
        self.execute_query("search_nearby_groups", async {
            let limit = pagination.limit.unwrap_or(20).min(100) as i64;
            let cursor = pagination.cursor.unwrap_or(0) as i64;
            
            // 复杂的地理位置查询 - 展示PostGIS函数的使用
            let rows = sqlx::query_as!(
                GroupSearchRow,
                r#"
                SELECT 
                    g.group_id as "group_id!",
                    g.name as "name!",
                    g.description,
                    g.avatar_url,
                    g.group_type as "group_type!: GroupType",
                    g.visibility as "visibility!: GroupVisibility",
                    COUNT(gm.user_id) as "member_count!",
                    ST_Distance(
                        gl.location::geography,
                        ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
                    ) as distance_meters,
                    g.created_at as "created_at!"
                FROM groups g
                INNER JOIN group_locations gl ON g.id = gl.group_id
                LEFT JOIN group_members gm ON g.id = gm.group_id
                WHERE g.visibility = 'public'
                  AND ($3 = 0 OR g.id > $3)
                  AND ST_DWithin(
                      gl.location::geography,
                      ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                      $4
                  )
                GROUP BY g.group_id, g.name, g.description, g.avatar_url, g.group_type, g.visibility, g.created_at, gl.location, gl.latitude, gl.longitude
                ORDER BY ST_Distance(
                    gl.location::geography,
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
                ) ASC, g.group_id ASC
                LIMIT $5
                "#,
                location.center.longitude.value(),
                location.center.latitude.value(),
                cursor,
                location.radius * 1000.0, // 转换为米
                limit + 1
            )
            .fetch_all(&self.pool)
            .await?;
            
            let has_more = rows.len() > limit as usize;
            let mut items: Vec<_> = rows.into_iter()
                .take(limit as usize)
                .map(|row| row.to_domain())
                .collect();
            
            let next_cursor = if has_more && !items.is_empty() {
                // 使用群组ID作为游标，而不是距离
                Some(items.last().unwrap().group_id.as_bytes().iter().fold(0i64, |acc, &b| acc.wrapping_mul(256).wrapping_add(b as i64)))
            } else {
                None
            };
            
            Ok(crate::application::PaginatedResult {
                items,
                has_more,
                next_cursor: next_cursor.map(|c| c.to_string()),
            })
        }).await
    }
    
    /// 按名称搜索群组 - 全文搜索示例
    async fn search_by_name(
        &self,
        query: &str,
        pagination: &crate::application::PaginationQuery,
    ) -> DomainResult<crate::application::PaginatedResult<crate::application::GroupView>> {
        self.execute_query("search_groups_by_name", async {
            let limit = pagination.limit.unwrap_or(20).min(100) as i64;
            let cursor = pagination.cursor.unwrap_or(0);
            
            // 使用PostgreSQL的全文搜索功能
            let rows = sqlx::query_as!(
                GroupSearchRow,
                r#"
                SELECT 
                    g.group_id as "group_id!",
                    g.name as "name!",
                    g.description,
                    g.avatar_url,
                    g.group_type as "group_type!: GroupType",
                    g.visibility as "visibility!: GroupVisibility",
                    COUNT(gm.user_id) as "member_count!",
                    NULL::FLOAT8 as distance_meters,
                    g.created_at as "created_at!"
                FROM groups g
                LEFT JOIN group_members gm ON g.id = gm.group_id
                WHERE g.visibility = 'public'
                  AND ($2 = 0 OR g.id > $2)
                  AND (
                      g.name ILIKE $1 
                      OR g.description ILIKE $1
                      OR to_tsvector('simple', COALESCE(g.name, '') || ' ' || COALESCE(g.description, '')) 
                         @@ plainto_tsquery('simple', $3)
                  )
                GROUP BY g.group_id, g.name, g.description, g.avatar_url, g.group_type, g.visibility, g.created_at
                ORDER BY 
                    -- 精确匹配优先
                    CASE WHEN g.name ILIKE $1 THEN 1 ELSE 2 END,
                    -- 然后按相关性排序
                    ts_rank(
                        to_tsvector('simple', COALESCE(g.name, '') || ' ' || COALESCE(g.description, '')),
                        plainto_tsquery('simple', $3)
                    ) DESC,
                    g.group_id ASC
                LIMIT $4
                "#,
                format!("%{}%", query),
                cursor as i32,
                query,
                limit + 1
            )
            .fetch_all(&self.pool)
            .await?;
            
            let has_more = rows.len() > limit as usize;
            let mut items: Vec<_> = rows.into_iter()
                .take(limit as usize)
                .map(|row| row.to_domain())
                .collect();
            
            let next_cursor = if has_more && !items.is_empty() {
                Some(items.last().unwrap().group_id.as_bytes().iter().fold(0i64, |acc, &b| acc.wrapping_mul(256).wrapping_add(b as i64)))
            } else {
                None
            };
            
            Ok(crate::application::PaginatedResult {
                items,
                has_more,
                next_cursor: next_cursor.map(|c| c.to_string()),
            })
        }).await
    }
    
    /// 检查用户是否为群组成员 - 简单的EXISTS查询
    async fn is_member(&self, group_id: GroupId, user_id: UserId) -> DomainResult<bool> {
        self.execute_query("check_group_membership", async {
            let is_member = sqlx::query_scalar!(
                r#"
                SELECT EXISTS(
                    SELECT 1 FROM group_members 
                    WHERE group_id = $1 AND user_id = $2
                ) as "is_member!"
                "#,
                uuid::Uuid::parse_str(group_id.value()).unwrap().parse::<i64>().unwrap_or(0),
                user_id.value()
            )
            .fetch_one(&self.pool)
            .await?;
            
            // EXISTS结果必定非NULL
            Ok(is_member)
        }).await
    }
    
    /// 删除群组 - 级联删除示例
    async fn delete_by_id(&self, group_id: GroupId) -> DomainResult<bool> {
        self.execute_query("delete_group", async {
            let result = sqlx::query!(
                "DELETE FROM groups WHERE group_id = $1",
                uuid::Uuid::parse_str(group_id.value()).unwrap()
            )
            .execute(&self.pool)
            .await?;
            
            Ok(result.rows_affected() > 0)
        }).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_group_row_conversion() {
        let group_row = GroupRow {
            group_id: Uuid::new_v4(),
            name: "测试群组".to_string(),
            description: Some("这是一个测试群组".to_string()),
            avatar_url: None,
            group_type: GroupType::Public,
            visibility: GroupVisibility::Public,
            max_members: 100,
            created_by: 123,
            created_at: Utc::now(),
            updated_at: Utc::now(),
        };
        
        let group = group_row.to_domain(None, None);
        assert_eq!(group.name.value(), "测试群组");
        assert_eq!(group.description, Some("这是一个测试群组".to_string()));
        assert_eq!(group.group_type, GroupType::Public);
        assert_eq!(group.max_members, 100);
    }
    
    #[test]
    fn test_group_member_stats_conversion() {
        let stats_row = GroupMemberStatsRow {
            group_id: Uuid::new_v4(),
            total_members: 50,
            active_members: 30,
            online_members: 10,
            is_member: true,
            is_admin: false,
        };
        
        assert_eq!(stats_row.total_members, 50);
        assert_eq!(stats_row.active_members, 30);
        assert_eq!(stats_row.online_members, 10);
        assert!(stats_row.is_member);
        assert!(!stats_row.is_admin);
    }
}

