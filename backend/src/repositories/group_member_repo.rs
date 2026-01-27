use chrono::{DateTime, Utc};
use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::GroupRole;
use crate::utils::pagination::{decode_cursor, encode_cursor, IdCursor};

/// 用户加入群组的基础信息（用于 user/groups 列表）
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserJoinedGroupInfoRow {
    /// 用于 keyset 分页的成员关系行 ID（group_members.id）
    pub member_row_id: i64,
    pub group_id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub members_count: i32,
    pub has_password: bool,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub distance: f64,
    pub bearing_degrees: f64,
}

/// 基础群组成员Row (对应数据库表)
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupMemberRow {
    pub id: i64,
    pub group_chat_id: Uuid,
    pub user_id: i64,
    pub role: GroupRole,
    pub joined_at: DateTime<Utc>,
    pub muted_until: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
}

/// 包含用户详情的群组成员Row (用于列表展示)
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupMemberDetailRow {
    pub id: i64,
    pub group_chat_id: Uuid,
    pub user_id: i64,
    pub role: GroupRole,
    pub joined_at: DateTime<Utc>,
    pub muted_until: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,

    // Joined fields from users table
    pub nickname: Option<String>,
    pub avatar_url: Option<String>,
    pub user_type: Option<crate::types::enums::UserType>,
    pub is_muted: Option<bool>, // Computed field
}

#[derive(Clone)]
pub struct GroupMemberRepository {
    pool: PgPool,
    redis: ConnectionManager,
}

impl GroupMemberRepository {
    pub fn new(pool: PgPool, redis: ConnectionManager) -> Self {
        Self { pool, redis }
    }

    /// 辅助：清除成员缓存
    async fn invalidate_member_cache(&self, group_chat_id: Uuid, user_id: i64) {
        let mut conn = self.redis.clone();
        let member_key = format!("group:member:{}:{}", group_chat_id, user_id);
        let role_key = format!("group:member:role:{}:{}", group_chat_id, user_id);
        let count_key = format!("group:members:count:{}", group_chat_id);
        let _: () = conn.del(&member_key).await.unwrap_or(());
        let _: () = conn.del(&role_key).await.unwrap_or(());
        let _: () = conn.del(&count_key).await.unwrap_or(());
    }

    /// 添加成员
    pub async fn add_member(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
        role: GroupRole,
    ) -> Result<()> {
        let role_str = role.to_string();
        sqlx::query!(
            r#"
            INSERT INTO group_members (group_chat_id, user_id, role)
            VALUES ($1, $2, $3::text::member_role)
            "#,
            group_chat_id,
            user_id,
            role_str
        )
        .execute(&self.pool)
        .await?;

        // 写入缓存 (1 = is_member)
        let mut conn = self.redis.clone();
        let key = format!("group:member:{}:{}", group_chat_id, user_id);
        let _: () = conn.set_ex(&key, 1, 86400).await.unwrap_or(()); // 24小时
        let role_key = format!("group:member:role:{}:{}", group_chat_id, user_id);
        let _: () = conn
            .set_ex(&role_key, role_str, 1800)
            .await
            .unwrap_or(());
        let count_key = format!("group:members:count:{}", group_chat_id);
        let _: () = conn.del(&count_key).await.unwrap_or(());

        Ok(())
    }

    /// 按用户ID和群组ID查找成员
    pub async fn find_member(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
    ) -> Result<Option<GroupMemberRow>> {
        let row = sqlx::query_as!(
            GroupMemberRow,
            r#"
            SELECT id, group_chat_id, user_id, role as "role: GroupRole", 
                   joined_at, muted_until, is_deleted, deleted_at
            FROM group_members
            WHERE group_chat_id = $1 AND user_id = $2 AND is_deleted = FALSE
            "#,
            group_chat_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 更新成员角色
    pub async fn update_role(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
        role: GroupRole,
    ) -> Result<()> {
        let role_str = role.to_string();
        sqlx::query!(
            r#"
            UPDATE group_members
            SET role = $3::text::member_role
            WHERE group_chat_id = $1 AND user_id = $2
            "#,
            group_chat_id,
            user_id,
            role_str
        )
        .execute(&self.pool)
        .await?;

        self.invalidate_member_cache(group_chat_id, user_id).await;

        Ok(())
    }

    /// 移除成员（软删除）
    pub async fn remove_member(&self, group_chat_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE group_chat_id = $1 AND user_id = $2
            "#,
            group_chat_id,
            user_id
        )
        .execute(&self.pool)
        .await?;

        self.invalidate_member_cache(group_chat_id, user_id).await;

        Ok(())
    }

    /// 获取成员角色
    pub async fn get_member_role(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
    ) -> Result<Option<GroupRole>> {
        let mut conn = self.redis.clone();
        let key = format!("group:member:role:{}:{}", group_chat_id, user_id);
        if let Ok(Some(val)) = conn.get::<_, Option<String>>(&key).await {
            if val == "none" {
                return Ok(None);
            }
            if let Ok(role) = GroupRole::from_str(&val) {
                return Ok(Some(role));
            }
        }

        let row = sqlx::query!(
            r#"
            SELECT role as "role: GroupRole"
            FROM group_members
            WHERE group_chat_id = $1 AND user_id = $2 AND is_deleted = FALSE
            "#,
            group_chat_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;
        if let Some(r) = row {
            let _: () = conn.set_ex(&key, r.role.to_string(), 1800).await.unwrap_or(());
            Ok(Some(r.role))
        } else {
            let _: () = conn.set_ex(&key, "none", 300).await.unwrap_or(());
            Ok(None)
        }
    }

    /// 检查用户是否是群组成员 (Redis 优先)
    pub async fn is_member(&self, group_chat_id: Uuid, user_id: i64) -> Result<bool> {
        let mut conn = self.redis.clone();
        let key = format!("group:member:{}:{}", group_chat_id, user_id);

        // 1. 查缓存
        if let Ok(Some(val)) = conn.get::<_, Option<i32>>(&key).await {
            return Ok(val == 1);
        }

        let result = sqlx::query!(
            r#"
            SELECT EXISTS (
                SELECT 1 FROM group_members
                WHERE group_chat_id = $1 AND user_id = $2 AND is_deleted = FALSE
            ) as "exists!"
            "#,
            group_chat_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        // 3. 写缓存 (如果是成员则缓存，非成员不缓存防穿透？或者也可以缓存 0)
        // 这里简单起见，如果是成员则缓存 1
        if result.exists {
            let _: () = conn.set_ex(&key, 1, 86400).await.unwrap_or(());
        }

        Ok(result.exists)
    }

    /// 获取群组成员数量
    pub async fn count_members(&self, group_chat_id: Uuid) -> Result<i64> {
        let mut conn = self.redis.clone();
        let key = format!("group:members:count:{}", group_chat_id);
        if let Ok(Some(val)) = conn.get::<_, Option<i64>>(&key).await {
            return Ok(val);
        }

        let result = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM group_members
            WHERE group_chat_id = $1 AND is_deleted = FALSE
            "#,
            group_chat_id
        )
        .fetch_one(&self.pool)
        .await?;
        let _: () = conn.set_ex(&key, result.count, 300).await.unwrap_or(());
        Ok(result.count)
    }

    /// 获取群组管理员列表（Owner + Admin）
    ///
    /// 用途：群组申请等“需要通知管理员”的场景，避免在事件处理器里做“全量广播”。
    pub async fn list_admin_user_ids(&self, group_chat_id: Uuid) -> Result<Vec<i64>> {
        let owner_role = GroupRole::Owner.to_string();
        let admin_role = GroupRole::Admin.to_string();

        let rows = sqlx::query!(
            r#"
            SELECT user_id
            FROM group_members
            WHERE group_chat_id = $1
              AND is_deleted = FALSE
              AND role::text IN ($2, $3)
            "#,
            group_chat_id,
            owner_role,
            admin_role
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.user_id).collect())
    }

    /// 获取群组所有成员 user_id
    pub async fn list_member_user_ids(&self, group_chat_id: Uuid) -> Result<Vec<i64>> {
        let rows = sqlx::query!(
            r#"
            SELECT user_id
            FROM group_members
            WHERE group_chat_id = $1
              AND is_deleted = FALSE
            "#,
            group_chat_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.user_id).collect())
    }

    /// 按群组ID查询成员列表 (需要用户详情)
    pub async fn list_by_group(
        &self,
        group_chat_id: Uuid,
        role: Option<&str>,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupMemberDetailRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let members = sqlx::query_as!(
            GroupMemberDetailRow,
            r#"
            SELECT m.id, m.group_chat_id, m.user_id, m.role as "role: GroupRole", 
                   m.joined_at, m.muted_until, m.is_deleted, m.deleted_at,
                   u.nickname, u.avatar_url, u.user_type as "user_type: crate::types::enums::UserType",
                   (m.muted_until > NOW()) as "is_muted"
            FROM group_members m
            LEFT JOIN users u ON m.user_id = u.user_id
            WHERE m.group_chat_id = $1
              AND ($4::text IS NULL OR m.role::text = $4)
              AND m.id < $2
              AND m.is_deleted = FALSE
            ORDER BY m.id DESC
            LIMIT $3
            "#,
            group_chat_id,
            last_id,
            limit as i64,
            role
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = members
            .last()
            .map(|last| encode_cursor(&last.id))
            .transpose()?;
        Ok((members, next_cursor))
    }

    /// 离开群组
    pub async fn leave_group(&self, group_chat_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE group_chat_id = $1 AND user_id = $2
            "#,
            group_chat_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        self.invalidate_member_cache(group_chat_id, user_id).await;
        Ok(())
    }

    /// 禁言成员
    pub async fn mute_member(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
        muted_until: chrono::DateTime<chrono::Utc>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET muted_until = $1
            WHERE group_chat_id = $2 AND user_id = $3
            "#,
            muted_until,
            group_chat_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 取消禁言成员
    pub async fn unmute_member(&self, group_chat_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET muted_until = NULL
            WHERE group_chat_id = $1 AND user_id = $2
            "#,
            group_chat_id,
            user_id
        )
        .execute(&self.pool)
        .await?;
        Ok(())
    }

    /// 按用户ID查询所加入的群组 (不需要用户详情，因为是查自己的群组)
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<String>,
    ) -> Result<(Vec<GroupMemberRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let members = sqlx::query_as!(
            GroupMemberRow,
            r#"
            SELECT id, group_chat_id, user_id, role as "role: GroupRole", 
                   joined_at, muted_until, is_deleted, deleted_at
            FROM group_members
            WHERE user_id = $1 AND id < $2 AND is_deleted = FALSE
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            last_id,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = members
            .last()
            .map(|last| encode_cursor(&last.id))
            .transpose()?;
        Ok((members, next_cursor))
    }

    /// 按用户ID查询所加入的“可公开展示”的群组信息（单条 JOIN SQL，避免 N+1）
    ///
    /// - 只返回：群组未删除/未解散 + 成员关系未删除 + (group_settings.is_searchable=true)
    /// - 使用 group_members.id 做 keyset 分页（与 list_by_user 保持一致的分页语义）
    pub async fn list_public_group_infos_by_user(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<String>,
        origin_latitude: f64,
        origin_longitude: f64,
    ) -> Result<(Vec<UserJoinedGroupInfoRow>, Option<String>)> {
        let last_id = if let Some(c) = cursor {
            decode_cursor::<IdCursor>(&c)?
        } else {
            i64::MAX
        };

        let rows = sqlx::query_as!(
            UserJoinedGroupInfoRow,
            r#"
            SELECT
                m.id as member_row_id,
                g.group_chat_id as "group_id!",
                g.name as "name!",
                g.description,
                g.avatar_url,
                COALESCE(cnt.member_count, 0)::int4 as "members_count!",
                (gs.password_hash IS NOT NULL) as "has_password!",
                COALESCE(g.latitude, 35.154922) as "latitude!",
                COALESCE(g.longitude, 112.682524) as "longitude!",
                COALESCE(g.location_name, '南天门') as "location_name!",
                ST_Distance(
                    COALESCE(g.location::geography, ST_SetSRID(ST_MakePoint(112.682524, 35.154922), 4326)::geography),
                    ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography
                ) as "distance!",
                COALESCE(degrees(ST_Azimuth(
                    ST_SetSRID(ST_MakePoint($4, $5), 4326),
                    COALESCE(g.location::geometry, ST_SetSRID(ST_MakePoint(112.682524, 35.154922), 4326))
                )), 0) as "bearing_degrees!"
            FROM group_members m
            JOIN group_chats g
              ON m.group_chat_id = g.group_chat_id
             AND g.is_deleted = FALSE
             AND g.is_dismissed = FALSE
            LEFT JOIN group_settings gs
              ON g.group_chat_id = gs.group_chat_id
            LEFT JOIN (
                SELECT group_chat_id, COUNT(*) as member_count
                FROM group_members
                WHERE is_deleted = FALSE
                GROUP BY group_chat_id
            ) cnt
              ON g.group_chat_id = cnt.group_chat_id
            WHERE m.user_id = $1
              AND m.id < $2
              AND m.is_deleted = FALSE
              AND COALESCE(gs.is_searchable, TRUE) = TRUE
            ORDER BY m.id DESC
            LIMIT $3
            "#,
            user_id,
            last_id,
            limit as i64,
            origin_longitude,
            origin_latitude
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = rows
            .last()
            .map(|last| encode_cursor(&last.member_row_id))
            .transpose()?;
        Ok((rows, next_cursor))
    }
}
