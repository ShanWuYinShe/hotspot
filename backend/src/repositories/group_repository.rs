//! PostgreSQL实现的群组仓库

use sqlx::PgPool;
use uuid::Uuid;
use chrono::{DateTime, Utc};
use tracing::error;
use serde::{Deserialize, Serialize};

use crate::repositories::{Repository, GroupRepository, RepositoryResult, RepositoryError};
use crate::services::group::{GroupSearchResult};

// =============== 查询结果结构体定义 ===============

/// 活跃用户统计查询结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActiveUserStatsRow {
    pub nickname: String,
    pub message_count: i64,
}

/// 最近消息摘要查询结果
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecentMessageSummaryRow {
    pub nickname: String,
    pub content: String,
    pub created_at: DateTime<Utc>,
}

/// PostgreSQL实现的群组仓库
#[derive(Debug)]
pub struct PgGroupRepository {
    /// 数据库连接池
    pool: PgPool,
}

impl PgGroupRepository {
    /// 创建新的PostgreSQL群组仓库
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

impl Repository for PgGroupRepository {
    fn name(&self) -> &str {
        "pg_group_repository"
    }
}

impl GroupRepository for PgGroupRepository {
    #[cfg(feature = "group_digest_scheduler")]
    async fn find_inactive_members(
        &self,
        group_id: Uuid,
        min_inactivity_days: i32
    ) -> RepositoryResult<Vec<(i64, String)>> {
        let min_joined_at = Utc::now() - Duration::days(min_inactivity_days as i64);
        let members = sqlx::query!(
            r#"
            SELECT gm.user_id, u.nickname as "nickname!"
            FROM group_members gm
            JOIN users u ON gm.user_id = u.user_id
            WHERE gm.group_id = $1
            AND gm.joined_at < $2
            "#,
            group_id,
            min_joined_at
        )
        .fetch_all(&self.pool)
        .await
        .map_err(RepositoryError::Database)?
        .into_iter()
        .map(|row| (row.user_id, row.nickname))
        .collect();
        Ok(members)
    }
    
    #[cfg(feature = "group_digest_scheduler")]
    async fn generate_message_digest(
        &self,
        group_id: Uuid,
        since_timestamp: i64
    ) -> RepositoryResult<String> {
        let since_datetime = DateTime::<Utc>::from_timestamp(since_timestamp, 0)
            .unwrap_or_else(|| Utc::now() - Duration::days(7));
        let new_message_count_fut = {
            let pool = self.pool.clone();
            async move {
                sqlx::query_scalar!(
                    r#"
                    SELECT COUNT(*) as "count!"
                    FROM group_messages
                    WHERE group_id = $1 AND created_at > $2
                    "#,
                    group_id,
                    since_datetime
                )
                .fetch_one(&pool)
                .await
                .map_err(RepositoryError::Database)
            }
        };
        let active_users_count_fut = {
            let pool = self.pool.clone();
            async move {
                sqlx::query_scalar!(
                    r#"
                    SELECT COUNT(DISTINCT sender_id) as "count!"
                    FROM group_messages
                    WHERE group_id = $1 AND created_at > $2
                    "#,
                    group_id,
                    since_datetime
                )
                .fetch_one(&pool)
                .await
                .map_err(RepositoryError::Database)
            }
        };
        let most_active_users_fut = {
            let pool = self.pool.clone();
            async move {
                let rows = sqlx::query_as!(
                    ActiveUserStatsRow,
                    r#"
                    SELECT u.nickname as "nickname!", COUNT(*) as "message_count!"
                    FROM group_messages gm
                    JOIN users u ON gm.sender_id = u.user_id
                    WHERE gm.group_id = $1 AND gm.created_at > $2
                    GROUP BY u.nickname
                    ORDER BY COUNT(*) DESC
                    LIMIT 3
                    "#,
                    group_id,
                    since_datetime
                )
                .fetch_all(&pool)
                .await
                .map_err(RepositoryError::Database)?;
                let most_active_users = rows.into_iter()
                    .filter_map(|row| {
                        Some((row.nickname.clone(), row.message_count))
                    })
                    .collect::<Vec<_>>();
                Ok(most_active_users)
            }
        };
        let recent_messages_fut = {
            let pool = self.pool.clone();
            async move {
                let rows = sqlx::query_as!(
                    RecentMessageSummaryRow,
                    r#"
                    SELECT u.nickname as "nickname!", gm.content, gm.created_at
                    FROM group_messages gm
                    JOIN users u ON gm.sender_id = u.user_id
                    WHERE gm.group_id = $1 AND gm.created_at > $2
                    ORDER BY gm.created_at DESC
                    LIMIT 5
                    "#,
                    group_id,
                    since_datetime
                )
                .fetch_all(&pool)
                .await
                .map_err(RepositoryError::Database)?;
                let recent_messages = rows.into_iter()
                    .filter_map(|row| {
                        let nickname = row.nickname.clone();
                        let content = if row.content.len() > 30 {
                            format!("{}...", &row.content[0..30])
                        } else {
                            row.content
                        };
                        Some(format!("- {} ({}): {}", 
                            nickname, 
                            row.created_at.format("%m-%d %H:%M"), 
                            content
                        ))
                    })
                    .collect::<Vec<_>>();
                Ok(recent_messages)
            }
        };
        let (new_message_count, active_users_count, most_active_users, recent_messages) = tokio::try_join!(
            new_message_count_fut,
            active_users_count_fut,
            most_active_users_fut,
            recent_messages_fut
        )?;
        let mut digest = format!(
            "在你不活跃的这段时间，群组中有 {} 条新消息，来自 {} 位成员。\n\n", 
            new_message_count, 
            active_users_count
        );
        if !most_active_users.is_empty() {
            digest.push_str("活跃成员：\n");
            for (i, (nickname, count)) in most_active_users.iter().enumerate() {
                digest.push_str(&format!("{}. {} ({} 条消息)\n", i+1, nickname, count));
            }
            digest.push_str("\n");
        }
        if !recent_messages.is_empty() {
            digest.push_str("最新消息预览：\n");
            for message in recent_messages {
                digest.push_str(&format!("{}\n", message));
            }
        }
        Ok(digest)
    }
    #[cfg(feature = "group_digest_scheduler")]
    async fn find_groups_needing_digest(
        &self,
        activity_days: i32,
        _inactivity_days: i32,
        limit: i32
    ) -> RepositoryResult<Vec<(Uuid, String, DateTime<Utc>)>> {
        let active_date = Utc::now() - Duration::days(activity_days as i64);
        let groups = sqlx::query!(
            r#"
            SELECT DISTINCT g.group_id, g.name, g.last_active_at
            FROM groups g
            JOIN group_members gm ON g.group_id = gm.group_id
            WHERE 
                g.last_active_at > $1
                AND EXISTS (
                    SELECT 1 
                    FROM group_members gm2 
                    WHERE gm2.group_id = g.group_id 
                    AND gm2.notification_level != 'muted'
                )
            LIMIT $2
            "#,
            active_date,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(RepositoryError::Database)?
        .into_iter()
        .map(|row| (row.group_id, row.name, row.last_active_at))
        .collect();
        Ok(groups)
    }

    async fn check_group_exists(&self, group_id: Uuid) -> RepositoryResult<bool> {
        let exists = sqlx::query_scalar!(
            r#"SELECT EXISTS(SELECT 1 FROM groups WHERE group_id = $1) as "exists!""#,
            group_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("查询群组是否存在失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok(exists)
    }

    async fn check_user_is_group_owner(&self, group_id: Uuid, user_id: i64) -> RepositoryResult<bool> {
        let is_owner = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM groups 
                WHERE group_id = $1 AND owner_id = $2
            ) as "exists!""#,
            group_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("查询用户是否为群组所有者失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok(is_owner)
    }

    async fn get_group_password_hash(&self, group_id: Uuid) -> RepositoryResult<Option<String>> {
        let password_hash = sqlx::query_scalar!(
            "SELECT password_hash FROM groups WHERE group_id = $1",
            group_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| {
            error!("查询群组密码哈希失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok(password_hash.flatten())
    }

    async fn verify_group_password(&self, group_id: Uuid, password: &str) -> RepositoryResult<bool> {
        let password_hash = match self.get_group_password_hash(group_id).await? {
            Some(hash) => hash,
            None => return Ok(true),
        };
        use argon2::{Argon2, PasswordHash, PasswordVerifier};
        let parsed_hash = PasswordHash::new(&password_hash).map_err(|e| {
            error!("解析密码哈希失败: {:?}", e);
            RepositoryError::Other(format!("密码验证失败: {}", e))
        })?;
        let result = Argon2::default()
            .verify_password(password.as_bytes(), &parsed_hash)
            .is_ok();
        Ok(result)
    }

    #[cfg(feature = "group_digest_scheduler")]
    async fn count_group_admins(&self, group_id: Uuid) -> RepositoryResult<i64> {
        use crate::models::group::GroupRole;
        let count = sqlx::query_scalar!(
            "SELECT COUNT(*) FROM group_members WHERE group_id = $1 AND role = $2",
            group_id,
            GroupRole::Admin as GroupRole
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("查询群组管理员数量失败: {:?}", e);
            RepositoryError::Database(e)
        })?
        .unwrap_or(0);
        Ok(count)
    }

    async fn leave_group(&self, group_id: Uuid, user_id: i64) -> RepositoryResult<bool> {
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开启事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        sqlx::query!(
            r#"DELETE FROM group_members WHERE group_id = $1 AND user_id = $2"#,
            group_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("移除群组成员失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok(true)
    }

    async fn check_user_in_group(&self, group_id: Uuid, user_id: i64) -> RepositoryResult<bool> {
        let is_member = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM group_members 
                WHERE group_id = $1 AND user_id = $2
            ) as "exists!""#,
            group_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("检查用户是否在群组中失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok(is_member)
    }
    
    async fn check_user_is_group_admin(&self, group_id: Uuid, user_id: i64) -> RepositoryResult<bool> {
        if self.check_user_is_group_owner(group_id, user_id).await? {
            return Ok(true);
        }
        let is_admin = sqlx::query_scalar!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM group_members 
                WHERE group_id = $1 AND user_id = $2 AND role = $3
            ) as "exists!""#,
            group_id,
            user_id,
            crate::models::group::GroupRole::Admin as crate::models::group::GroupRole
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("检查用户是否是群组管理员失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok(is_admin)
    }
    
    async fn check_group_need_password(&self, group_id: Uuid) -> RepositoryResult<bool> {
        let need_password = sqlx::query_scalar!(
            r#"SELECT (password_hash IS NOT NULL) as "need_password!" FROM groups WHERE group_id = $1"#,
            group_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("检查群组密码需求失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok(need_password)
    }
    
    #[cfg(feature = "group_digest_scheduler")]
    async fn count_group_members(&self, group_id: Uuid) -> RepositoryResult<i64> {
        let count = sqlx::query_scalar!(
            "SELECT COUNT(*) FROM group_members WHERE group_id = $1",
            group_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("查询群组成员数量失败: {:?}", e);
            RepositoryError::Database(e)
        })?
        .unwrap_or(0);
        Ok(count)
    }
    
    async fn hash_password(&self, password: &str) -> RepositoryResult<String> {
        use argon2::{
            Argon2, PasswordHasher,
            password_hash::{SaltString, rand_core::OsRng},
        };
        let salt = SaltString::generate(&mut OsRng);
        let password_hash = Argon2::default()
            .hash_password(password.as_bytes(), &salt)
            .map_err(|e| {
                error!("密码加密失败: {:?}", e);
                RepositoryError::Other("密码处理失败".to_string())
            })?
            .to_string();
        Ok(password_hash)
    }
    
    async fn create_group(
        &self,
        name: &str,
        description: Option<&str>,
        owner_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str,
        password: Option<&str>
    ) -> RepositoryResult<Uuid> {
        let group_id = Uuid::new_v4();
        let now = chrono::Utc::now();
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开启事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        let password_hash = if let Some(pwd) = password {
            Some(self.hash_password(pwd).await?)
        } else {
            None
        };
        sqlx::query!(
            r#"INSERT INTO groups (group_id, name, description, owner_id, latitude, longitude, location_name, password_hash, created_at, updated_at, last_active_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)"#,
            group_id,
            name,
            description,
            owner_id,
            latitude,
            longitude,
            location_name,
            password_hash,
            now,
            now,
            now
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("创建群组失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        sqlx::query!(
            r#"INSERT INTO group_members (group_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4)"#,
            group_id,
            owner_id,
            crate::models::group::GroupRole::Owner as crate::models::group::GroupRole,
            now
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("添加群主为成员失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok(group_id)
    }
    
    async fn join_group(
        &self,
        group_id: Uuid,
        user_id: i64
    ) -> RepositoryResult<(bool, String)> {
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开启事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        sqlx::query!(
            r#"INSERT INTO group_members (group_id, user_id, role, joined_at) VALUES ($1, $2, $3, $4)"#,
            group_id,
            user_id,
            crate::models::group::GroupRole::Member as crate::models::group::GroupRole,
            chrono::Utc::now(),
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("添加群组成员失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok((true, crate::models::constants::group_roles::MEMBER.to_string()))
    }
    
    // 踢出成员方法
    async fn kick_member(
        &self,
        group_id: Uuid,
        user_id: i64,
        admin_id: i64
    ) -> RepositoryResult<bool> {
        if admin_id == user_id {
            return Err(RepositoryError::InvalidArgument("无法踢出自己，请使用离开群组".to_string()));
        }
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开启事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        sqlx::query!(
            r#"DELETE FROM group_members WHERE group_id = $1 AND user_id = $2"#,
            group_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("移除群组成员失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok(true)
    }
    async fn transfer_ownership(
        &self,
        group_id: Uuid,
        new_owner_id: i64,
        current_owner_id: i64
    ) -> RepositoryResult<bool> {
        let mut tx = self.pool.begin().await.map_err(|e| {
            error!("开启事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        sqlx::query!(
            r#"UPDATE groups SET owner_id = $1, updated_at = $2 WHERE group_id = $3"#,
            new_owner_id,
            chrono::Utc::now(),
            group_id
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("更新群主失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        sqlx::query!(
            r#"UPDATE group_members SET role = CASE WHEN user_id = $1 THEN $4 WHEN user_id = $2 THEN $5 ELSE role END WHERE group_id = $3 AND user_id IN ($1, $2)"#,
            new_owner_id,
            current_owner_id,
            group_id,
            crate::models::group::GroupRole::Owner as crate::models::group::GroupRole,
            crate::models::group::GroupRole::Member as crate::models::group::GroupRole
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| {
            error!("更新成员角色失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        tx.commit().await.map_err(|e| {
            error!("提交事务失败: {:?}", e);
            RepositoryError::Database(e)
        })?;
        Ok(true)
    }
    
    // 获取群组成员列表
    async fn get_group_members(
        &self,
        group_id: Uuid,
        cursor: i64,
        limit: i64
    ) -> RepositoryResult<(Vec<(i64, i64, String, DateTime<Utc>, String, DateTime<Utc>)>, bool, Option<i64>)> {
        struct GroupMemberRow {
            id: i64,
            user_id: i64,
            nickname: String,
            last_active_at: DateTime<Utc>,
            role: String,
            joined_at: DateTime<Utc>,
        }
        
        let rows = sqlx::query_as!(
            GroupMemberRow,
            r#"
            SELECT 
                gm.id,
                gm.user_id,
                COALESCE(u.nickname, '未知用户') as "nickname!",
                u.last_active_at,
                gm.role::text as "role!",
                gm.joined_at as "joined_at!"
            FROM group_members gm
            JOIN users u ON gm.user_id = u.user_id
            WHERE gm.group_id = $1
            AND ($2::bigint = 0 OR gm.id < $2::bigint)
            ORDER BY gm.id DESC
            LIMIT $3
            "#,
            group_id,
            cursor,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("获取群组成员失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        let has_more = rows.len() > limit as usize;
        let members: Vec<(i64, i64, String, DateTime<Utc>, String, DateTime<Utc>)> = rows.iter()
            .take(if has_more { limit as usize } else { rows.len() })
            .map(|row| (
                row.id,
                row.user_id,
                row.nickname.clone(),
                row.last_active_at,
                row.role.clone(),
                row.joined_at
            ))
            .collect();

        let next_cursor = if has_more && !members.is_empty() {
            Some(members.last().unwrap().0)
        } else {
            None
        };

        Ok((members, has_more, next_cursor))
    }
    
    // 根据位置搜索群组
    async fn search_group_by_location(
        &self,
        user_id: i64,
        radius: f64,
        cursor: i64,
        limit: i64
    ) -> RepositoryResult<(Vec<GroupSearchResult>, bool, Option<i64>)> {
        use crate::utils::constants::{DEFAULT_LATITUDE, DEFAULT_LONGITUDE};
        
        let rows = sqlx::query_as!(
            GroupSearchResult,
            r#"
            WITH nearby_groups AS (
                SELECT 
                    g.id,
                    g.group_id,
                    g.name,
                    g.description,
                    g.owner_id,
                    COALESCE(u.nickname, '') as creator_name,
                    g.created_at,
                    g.updated_at as last_active_at,
                    g.latitude,
                    g.longitude,
                    g.location_name,
                    COALESCE((SELECT COUNT(*) FROM group_members WHERE group_id = g.group_id), 0) as member_count,
                    COALESCE(ST_Distance(
                        ST_MakePoint(g.longitude, g.latitude)::geography,
                        ST_MakePoint(COALESCE(ul.longitude, $3), COALESCE(ul.latitude, $4))::geography
                    ), 0.0) as distance,
                    (g.password_hash IS NOT NULL) as is_password_required,
                    (EXISTS(SELECT 1 FROM group_members WHERE group_id = g.group_id AND user_id = $1)) as is_member,
                    COALESCE((SELECT role::text FROM group_members WHERE group_id = g.group_id AND user_id = $1), '') as user_role
                FROM groups g
                JOIN users u ON g.owner_id = u.user_id
                LEFT JOIN user_locations ul ON ul.user_id = $1
                WHERE ($5::bigint = 0 OR g.id < $5::bigint)
            )
            SELECT 
                id,
                group_id,
                name, 
                description, 
                owner_id, 
                creator_name as "creator_name!", 
                created_at, 
                last_active_at, 
                latitude, 
                longitude, 
                location_name, 
                member_count as "member_count!", 
                distance as "distance!", 
                is_password_required as "is_password_required!", 
                is_member as "is_member!", 
                user_role as "user_role!"
            FROM nearby_groups
            WHERE distance <= $2
            ORDER BY id DESC
            LIMIT $6
            "#,
            user_id,
            radius,
            DEFAULT_LONGITUDE,
            DEFAULT_LATITUDE,
            cursor,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("查询附近群组失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        let has_more = rows.len() > limit as usize;
        let rows_len = rows.len();
        let groups: Vec<GroupSearchResult> = rows.into_iter()
            .take(if has_more { limit as usize } else { rows_len })
            .collect();

        let next_cursor = if has_more && !groups.is_empty() {
            Some(groups.last().unwrap().id)
        } else {
            None
        };

        Ok((groups, has_more, next_cursor))
    }
    
    // 根据名称搜索群组
    async fn search_group_by_name(
        &self,
        user_id: i64,
        keyword: &str,
        cursor: i64,
        limit: i64
    ) -> RepositoryResult<(Vec<GroupSearchResult>, bool, Option<i64>)> {
        use crate::utils::constants::{DEFAULT_LATITUDE, DEFAULT_LONGITUDE};
        
        let rows = sqlx::query_as!(
            GroupSearchResult,
            r#"
            WITH search_groups AS (
                SELECT 
                    g.id,
                    g.group_id,
                    g.name,
                    g.description,
                    g.owner_id,
                    COALESCE(u.nickname, '') as creator_name,
                    g.created_at,
                    g.updated_at as last_active_at,
                    g.latitude,
                    g.longitude,
                    g.location_name,
                    COALESCE((SELECT COUNT(*) FROM group_members WHERE group_id = g.group_id), 0) as member_count,
                    COALESCE(ST_Distance(
                        ST_MakePoint(g.longitude, g.latitude)::geography,
                        ST_MakePoint(COALESCE(ul.longitude, $4), COALESCE(ul.latitude, $5))::geography
                    ), 0.0) as distance,
                    (g.password_hash IS NOT NULL) as is_password_required,
                    (EXISTS(SELECT 1 FROM group_members WHERE group_id = g.group_id AND user_id = $1)) as is_member,
                    COALESCE((SELECT role::text FROM group_members WHERE group_id = g.group_id AND user_id = $1), '') as user_role
                FROM groups g
                JOIN users u ON g.owner_id = u.user_id
                LEFT JOIN user_locations ul ON ul.user_id = $1
                WHERE g.name ILIKE $2 AND ($3::bigint = 0 OR g.id < $3::bigint)
            )
            SELECT 
                id,
                group_id,
                name, 
                description, 
                owner_id, 
                creator_name as "creator_name!", 
                created_at, 
                last_active_at, 
                latitude, 
                longitude, 
                location_name, 
                member_count as "member_count!", 
                distance as "distance!", 
                is_password_required as "is_password_required!", 
                is_member as "is_member!", 
                user_role as "user_role!"
            FROM search_groups
            ORDER BY id DESC
            LIMIT $6
            "#,
            user_id,
            format!("%{}%", keyword),
            cursor,
            DEFAULT_LONGITUDE,
            DEFAULT_LATITUDE,
            limit + 1
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("搜索群组失败: {:?}", e);
            RepositoryError::Database(e)
        })?;

        let has_more = rows.len() > limit as usize;
        let rows_len = rows.len();
        let groups: Vec<GroupSearchResult> = rows.into_iter()
            .take(if has_more { limit as usize } else { rows_len })
            .collect();

        let next_cursor = if has_more && !groups.is_empty() {
            Some(groups.last().unwrap().id)
        } else {
            None
        };

        Ok((groups, has_more, next_cursor))
    }
    
    // 根据ID获取群组
    async fn search_group_by_id(
        &self,
        user_id: i64,
        group_id: Uuid
    ) -> RepositoryResult<GroupSearchResult> {
        use crate::utils::constants::{DEFAULT_LATITUDE, DEFAULT_LONGITUDE};
        
        let row = sqlx::query_as!(
            GroupSearchResult,
            r#"
            SELECT 
                g.id,
                g.group_id,
                g.name,
                g.description,
                g.owner_id,
                COALESCE(u.nickname, '') as creator_name,
                g.created_at,
                g.updated_at as last_active_at,
                g.latitude,
                g.longitude,
                g.location_name,
                COALESCE((SELECT COUNT(*) FROM group_members WHERE group_id = g.group_id), 0) as member_count,
                COALESCE(ST_Distance(
                    ST_MakePoint(g.longitude, g.latitude)::geography,
                    ST_MakePoint(COALESCE(ul.longitude, $3), COALESCE(ul.latitude, $4))::geography
                ), 0.0) as distance,
                (g.password_hash IS NOT NULL) as is_password_required,
                (EXISTS(SELECT 1 FROM group_members WHERE group_id = g.group_id AND user_id = $1)) as is_member,
                COALESCE((SELECT role::text FROM group_members WHERE group_id = g.group_id AND user_id = $1), '') as user_role
            FROM groups g
            JOIN users u ON g.owner_id = u.user_id
            LEFT JOIN user_locations ul ON ul.user_id = $1
            WHERE g.group_id = $2
            "#,
            user_id,
            group_id,
            DEFAULT_LONGITUDE,
            DEFAULT_LATITUDE
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("查询群组失败: {:?}", e);
            match e {
                sqlx::Error::RowNotFound => RepositoryError::NotFound,
                _ => RepositoryError::Database(e)
            }
        })?;

        Ok(row)
    }

    async fn update_member_last_active(
        &self,
        _group_id: Uuid,
        _user_id: i64,
        _timestamp: DateTime<Utc>
    ) -> RepositoryResult<()> {
        Ok(())
    }

    /// 获取用户加入的群组列表
    async fn get_user_groups(
        &self,
        user_id: i64,
        limit: Option<i64>,
        offset: Option<i64>
    ) -> RepositoryResult<Vec<GroupSearchResult>> {
        let results = sqlx::query_as!(
            GroupSearchResult,
            r#"
            SELECT 
                g.id,
                g.group_id,
                g.name,
                g.description,
                g.owner_id,
                COALESCE(u.nickname, '') as "creator_name!",
                g.created_at,
                g.updated_at as last_active_at,
                g.latitude,
                g.longitude,
                g.location_name,
                COALESCE((SELECT COUNT(*) FROM group_members WHERE group_id = g.group_id), 0) as "member_count!",
                0.0::double precision as "distance!",
                (g.password_hash IS NOT NULL) as "is_password_required!",
                true as "is_member!",
                COALESCE((SELECT role::text FROM group_members WHERE group_id = g.group_id AND user_id = $1), 'member') as "user_role!"
            FROM groups g
            JOIN users u ON g.owner_id = u.user_id
            INNER JOIN group_members gm ON g.group_id = gm.group_id
            WHERE gm.user_id = $1
            ORDER BY g.created_at DESC
            LIMIT $2 OFFSET $3
            "#,
            user_id,
            limit.unwrap_or(20),
            offset.unwrap_or(0)
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("Failed to get user groups: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(results)
    }

    /// 获取用户群组总数
    async fn count_user_groups(&self, user_id: i64) -> RepositoryResult<i64> {
        let count = sqlx::query_scalar!(
            r#"
            SELECT COUNT(*)
            FROM groups g
            INNER JOIN group_members gm ON g.group_id = gm.group_id
            WHERE gm.user_id = $1
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| {
            error!("Failed to count user groups: {:?}", e);
            RepositoryError::Database(e)
        })?
        .unwrap_or(0);

        Ok(count)
    }
    
    /// 获取用户创建的群组列表
    async fn get_owned_groups(
        &self,
        user_id: i64,
        limit: Option<i64>,
        offset: Option<i64>
    ) -> RepositoryResult<Vec<GroupSearchResult>> {
        let results = sqlx::query_as!(
            GroupSearchResult,
            r#"
            SELECT 
                g.id,
                g.group_id,
                g.name,
                g.description,
                g.owner_id,
                COALESCE(u.nickname, '') as "creator_name!",
                g.created_at,
                g.updated_at as last_active_at,
                g.latitude,
                g.longitude,
                g.location_name,
                COALESCE((SELECT COUNT(*) FROM group_members WHERE group_id = g.group_id), 0) as "member_count!",
                0.0::double precision as "distance!",
                (g.password_hash IS NOT NULL) as "is_password_required!",
                (EXISTS(SELECT 1 FROM group_members WHERE group_id = g.group_id AND user_id = $1)) as "is_member",
                'owner'::text as user_role
            FROM groups g
            JOIN users u ON g.owner_id = u.user_id
            WHERE g.owner_id = $1
            ORDER BY g.created_at DESC
            LIMIT $2 OFFSET $3
            "#,
            user_id,
            limit.unwrap_or(20),
            offset.unwrap_or(0)
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| {
            error!("Failed to get owned groups: {:?}", e);
            RepositoryError::Database(e)
        })?;

        Ok(results)
    }
}
