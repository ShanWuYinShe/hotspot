use std::collections::HashMap;

use chrono::{DateTime, Utc};
use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use serde::{Deserialize, Serialize};
use sqlx::{FromRow, PgPool};

use crate::error::Result;
use crate::types::enums::GroupRole;
use crate::utils::pagination::{decode_cursor, encode_cursor, Paginated};

/// 用户加入群组的基础信息（用于 user/groups 列表）
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct UserJoinedGroupInfoRow {
    pub group_id: i64,
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub members_count: i32,
    pub has_password: bool,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
    pub joined_at: DateTime<Utc>, // From group_members
}

/// 基础群组成员Row (对应数据库表)
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupMemberRow {
    pub group_id: i64,
    pub user_id: i64,
    pub nickname: Option<String>, // Custom nickname in group
    pub role: GroupRole,
    pub is_muted: Option<bool>,
    pub muted_until: Option<DateTime<Utc>>,
    pub joined_at: DateTime<Utc>,
    pub inviter_id: Option<i64>,
}

/// 包含用户详情的群组成员Row (用于列表展示)
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupMemberDetailRow {
    pub group_id: i64,
    pub user_id: i64,
    pub group_nickname: Option<String>, // From group_members.nickname
    pub role: GroupRole,
    pub is_muted: Option<bool>,
    pub muted_until: Option<DateTime<Utc>>,
    pub joined_at: DateTime<Utc>,
    pub inviter_id: Option<i64>,

    // Joined fields from users table
    pub user_nickname: Option<String>, // From users.nickname
    pub avatar_url: Option<String>,
    pub user_type: Option<crate::types::enums::UserType>,
}

#[derive(Debug, Clone, FromRow)]
pub struct GroupAdminMemberRow {
    pub group_id: i64,
    pub user_id: i64,
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
    async fn invalidate_member_cache(&self, group_id: i64, user_id: i64) {
        let mut conn = self.redis.clone();
        let member_key = format!("group:member:{}:{}", group_id, user_id);
        let role_key = format!("group:member:role:{}:{}", group_id, user_id);
        let count_key = format!("group:members:count:{}", group_id);
        let _: () = conn.del(&member_key).await.unwrap_or(());
        let _: () = conn.del(&role_key).await.unwrap_or(());
        let _: () = conn.del(&count_key).await.unwrap_or(());
    }

    /// 添加成员
    pub async fn add_member(
        &self,
        group_id: i64,
        user_id: i64,
        role: GroupRole,
        inviter_id: Option<i64>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO group_members (group_id, user_id, role, inviter_id)
            VALUES ($1, $2, $3, $4)
            "#,
            group_id,
            user_id,
            role as GroupRole,
            inviter_id
        )
        .execute(&self.pool)
        .await?;

        let role_str = role.to_string();
        // 写入缓存 (1 = is_member)
        let mut conn = self.redis.clone();
        let key = format!("group:member:{}:{}", group_id, user_id);
        let _: () = conn.set_ex(&key, 1, 86400).await.unwrap_or(()); // 24小时
        let role_key = format!("group:member:role:{}:{}", group_id, user_id);
        let _: () = conn.set_ex(&role_key, role_str, 1800).await.unwrap_or(());
        let count_key = format!("group:members:count:{}", group_id);
        let _: () = conn.del(&count_key).await.unwrap_or(());

        Ok(())
    }

    /// 按用户ID和群组ID查找成员
    pub async fn find_member(&self, group_id: i64, user_id: i64) -> Result<Option<GroupMemberRow>> {
        let row = sqlx::query_as!(
            GroupMemberRow,
            r#"
            SELECT group_id, user_id, nickname, role as "role!: GroupRole", 
                   is_muted, muted_until, joined_at, inviter_id
            FROM group_members
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 更新成员角色
    pub async fn update_role(&self, group_id: i64, user_id: i64, role: GroupRole) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET role = $3
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id,
            role as GroupRole
        )
        .execute(&self.pool)
        .await?;

        self.invalidate_member_cache(group_id, user_id).await;
        Ok(())
    }

    /// 移除成员 (直接DELETE，不是软删除)
    pub async fn remove_member(&self, group_id: i64, user_id: i64) -> Result<()> {
        sqlx::query!(
            "DELETE FROM group_members WHERE group_id = $1 AND user_id = $2",
            group_id,
            user_id
        )
        .execute(&self.pool)
        .await?;

        self.invalidate_member_cache(group_id, user_id).await;
        Ok(())
    }

    /// 检查用户是否为群成员
    pub async fn is_member(&self, group_id: i64, user_id: i64) -> Result<bool> {
        // 先查Redis
        let mut conn = self.redis.clone();
        let key = format!("group:member:{}:{}", group_id, user_id);
        let cached: Option<i32> = conn.get(&key).await.ok();

        if let Some(val) = cached {
            return Ok(val == 1);
        }

        // 缓存未命中，查数据库
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM group_members
                WHERE group_id = $1 AND user_id = $2
            ) as "exists!"
            "#,
            group_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await?;

        // 写入缓存
        let val = if result.exists { 1 } else { 0 };
        let _: () = conn.set_ex(&key, val, 86400).await.unwrap_or(());

        Ok(result.exists)
    }

    /// 获取成员角色
    pub async fn get_member_role(&self, group_id: i64, user_id: i64) -> Result<Option<GroupRole>> {
        // 先查Redis
        let mut conn = self.redis.clone();
        let key = format!("group:member:role:{}:{}", group_id, user_id);
        let cached: Option<String> = conn.get(&key).await.ok();

        if let Some(role_str) = cached {
            return Ok(role_str.parse::<GroupRole>().ok());
        }

        // 缓存未命中，查数据库
        let row = sqlx::query!(
            r#"
            SELECT role as "role!: GroupRole"
            FROM group_members
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        if let Some(r) = &row {
            let role_str = r.role.to_string();
            let _: () = conn.set_ex(&key, role_str, 1800).await.unwrap_or(());
        }

        Ok(row.map(|r| r.role))
    }

    /// 批量获取多个群组的成员数量
    /// 返回 HashMap<group_id, count>，仅包含有成员的群组（空群组不会出现在 map 中，视为 0）
    pub async fn batch_count_members(&self, group_ids: &[i64]) -> Result<HashMap<i64, i64>> {
        if group_ids.is_empty() {
            return Ok(HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT group_id as "group_id!", COUNT(*) as "count!"
            FROM group_members
            WHERE group_id = ANY($1)
            GROUP BY group_id
            "#,
            group_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.group_id, r.count)).collect())
    }

    /// 获取群组成员数量
    pub async fn count_members(&self, group_id: i64) -> Result<i64> {
        // 先查Redis
        let mut conn = self.redis.clone();
        let key = format!("group:members:count:{}", group_id);
        let cached: Option<i64> = conn.get(&key).await.ok();

        if let Some(count) = cached {
            return Ok(count);
        }

        // 缓存未命中，查数据库
        let result = sqlx::query!(
            "SELECT COUNT(*) as \"count!\" FROM group_members WHERE group_id = $1",
            group_id
        )
        .fetch_one(&self.pool)
        .await?;

        let count = result.count;
        let _: () = conn.set_ex(&key, count, 1800).await.unwrap_or(());

        Ok(count)
    }

    /// 列出群组成员（带用户详情）
    pub async fn list_members(
        &self,
        group_id: i64,
        limit: usize,
        cursor: Option<&str>,
    ) -> Paginated<GroupMemberDetailRow> {
        let cursor_time = cursor.and_then(|c| decode_cursor::<DateTime<Utc>>(c).ok());

        let mut rows = if let Some(time) = cursor_time {
            sqlx::query_as!(
                GroupMemberDetailRow,
                r#"
                SELECT 
                    gm.group_id, gm.user_id, gm.nickname as group_nickname, 
                    gm.role as "role!: GroupRole", gm.is_muted, gm.muted_until, 
                    gm.joined_at, gm.inviter_id,
                    u.nickname as user_nickname, u.avatar_url,
                    u.user_type as "user_type: crate::types::enums::UserType"
                FROM group_members gm
                LEFT JOIN users u ON gm.user_id = u.user_id
                WHERE gm.group_id = $1 AND gm.joined_at > $2
                ORDER BY gm.joined_at ASC
                LIMIT $3
                "#,
                group_id,
                time,
                (limit + 1) as i64
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                GroupMemberDetailRow,
                r#"
                SELECT 
                    gm.group_id, gm.user_id, gm.nickname as group_nickname, 
                    gm.role as "role!: GroupRole", gm.is_muted, gm.muted_until, 
                    gm.joined_at, gm.inviter_id,
                    u.nickname as user_nickname, u.avatar_url,
                    u.user_type as "user_type: crate::types::enums::UserType"
                FROM group_members gm
                LEFT JOIN users u ON gm.user_id = u.user_id
                WHERE gm.group_id = $1
                ORDER BY gm.joined_at ASC
                LIMIT $2
                "#,
                group_id,
                (limit + 1) as i64
            )
            .fetch_all(&self.pool)
            .await?
        };

        let has_more = rows.len() > limit;
        if has_more {
            rows.truncate(limit);
        }
        let next_cursor = if has_more {
            rows.last()
                .map(|r| encode_cursor(&r.joined_at))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }

    /// 搜索群组成员（按昵称关键字，可选按角色筛选）
    pub async fn search_members(
        &self,
        group_id: i64,
        keyword: &str,
        role: Option<GroupRole>,
        limit: usize,
        cursor: Option<&str>,
    ) -> Paginated<GroupMemberDetailRow> {
        let keyword_pattern = format!("%{}%", keyword);
        let cursor_time = cursor.and_then(|c| decode_cursor::<DateTime<Utc>>(c).ok());
        let role_str = role.map(|r| r.to_string());

        let mut rows = match (cursor_time, &role_str) {
            (Some(time), Some(r)) => {
                sqlx::query_as!(
                    GroupMemberDetailRow,
                    r#"
                    SELECT 
                        gm.group_id, gm.user_id, gm.nickname as group_nickname, 
                        gm.role as "role!: GroupRole", gm.is_muted, gm.muted_until, 
                        gm.joined_at, gm.inviter_id,
                        u.nickname as user_nickname, u.avatar_url,
                        u.user_type as "user_type: crate::types::enums::UserType"
                    FROM group_members gm
                    LEFT JOIN users u ON gm.user_id = u.user_id
                    WHERE gm.group_id = $1
                      AND (COALESCE(u.nickname, '') ILIKE $2 OR COALESCE(gm.nickname, '') ILIKE $2)
                      AND gm.role::text = $3
                      AND gm.joined_at > $4
                    ORDER BY gm.joined_at ASC
                    LIMIT $5
                    "#,
                    group_id,
                    &keyword_pattern,
                    r.as_str(),
                    time,
                    (limit + 1) as i64
                )
                .fetch_all(&self.pool)
                .await?
            }
            (Some(time), None) => {
                sqlx::query_as!(
                    GroupMemberDetailRow,
                    r#"
                    SELECT 
                        gm.group_id, gm.user_id, gm.nickname as group_nickname, 
                        gm.role as "role!: GroupRole", gm.is_muted, gm.muted_until, 
                        gm.joined_at, gm.inviter_id,
                        u.nickname as user_nickname, u.avatar_url,
                        u.user_type as "user_type: crate::types::enums::UserType"
                    FROM group_members gm
                    LEFT JOIN users u ON gm.user_id = u.user_id
                    WHERE gm.group_id = $1
                      AND (COALESCE(u.nickname, '') ILIKE $2 OR COALESCE(gm.nickname, '') ILIKE $2)
                      AND gm.joined_at > $3
                    ORDER BY gm.joined_at ASC
                    LIMIT $4
                    "#,
                    group_id,
                    &keyword_pattern,
                    time,
                    (limit + 1) as i64
                )
                .fetch_all(&self.pool)
                .await?
            }
            (None, Some(r)) => {
                sqlx::query_as!(
                    GroupMemberDetailRow,
                    r#"
                    SELECT 
                        gm.group_id, gm.user_id, gm.nickname as group_nickname, 
                        gm.role as "role!: GroupRole", gm.is_muted, gm.muted_until, 
                        gm.joined_at, gm.inviter_id,
                        u.nickname as user_nickname, u.avatar_url,
                        u.user_type as "user_type: crate::types::enums::UserType"
                    FROM group_members gm
                    LEFT JOIN users u ON gm.user_id = u.user_id
                    WHERE gm.group_id = $1
                      AND (COALESCE(u.nickname, '') ILIKE $2 OR COALESCE(gm.nickname, '') ILIKE $2)
                      AND gm.role::text = $3
                    ORDER BY gm.joined_at ASC
                    LIMIT $4
                    "#,
                    group_id,
                    &keyword_pattern,
                    r.as_str(),
                    (limit + 1) as i64
                )
                .fetch_all(&self.pool)
                .await?
            }
            (None, None) => {
                sqlx::query_as!(
                    GroupMemberDetailRow,
                    r#"
                    SELECT 
                        gm.group_id, gm.user_id, gm.nickname as group_nickname, 
                        gm.role as "role!: GroupRole", gm.is_muted, gm.muted_until, 
                        gm.joined_at, gm.inviter_id,
                        u.nickname as user_nickname, u.avatar_url,
                        u.user_type as "user_type: crate::types::enums::UserType"
                    FROM group_members gm
                    LEFT JOIN users u ON gm.user_id = u.user_id
                    WHERE gm.group_id = $1
                      AND (COALESCE(u.nickname, '') ILIKE $2 OR COALESCE(gm.nickname, '') ILIKE $2)
                    ORDER BY gm.joined_at ASC
                    LIMIT $3
                    "#,
                    group_id,
                    &keyword_pattern,
                    (limit + 1) as i64
                )
                .fetch_all(&self.pool)
                .await?
            }
        };

        let has_more = rows.len() > limit;
        if has_more {
            rows.truncate(limit);
        }
        let next_cursor = if has_more {
            rows.last()
                .map(|r| encode_cursor(&r.joined_at))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }

    /// 获取用户加入的群组列表
    pub async fn list_user_groups(
        &self,
        user_id: i64,
        limit: usize,
        cursor: Option<&str>,
    ) -> Paginated<UserJoinedGroupInfoRow> {
        let cursor_time = cursor.and_then(|c| decode_cursor::<DateTime<Utc>>(c).ok());

        let mut rows = if let Some(time) = cursor_time {
            sqlx::query_as!(
                UserJoinedGroupInfoRow,
                r#"
                SELECT 
                    g.group_id, g.name, g.description, g.avatar_url,
                    COALESCE(gs.member_count, 0)::int as "members_count!",
                    (g.metadata->>'join_password' IS NOT NULL) as "has_password!",
                    ST_Y(g.location::geometry) as "latitude?",
                    ST_X(g.location::geometry) as "longitude?",
                    g.location_name as "location_name?",
                    gm.joined_at
                FROM group_members gm
                INNER JOIN groups g ON gm.group_id = g.group_id
                LEFT JOIN group_stats gs ON g.group_id = gs.group_id
                WHERE gm.user_id = $1 AND g.dismissed_at IS NULL
                  AND gm.joined_at > $2
                ORDER BY gm.joined_at DESC
                LIMIT $3
                "#,
                user_id,
                time,
                (limit + 1) as i64
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                UserJoinedGroupInfoRow,
                r#"
                SELECT 
                    g.group_id, g.name, g.description, g.avatar_url,
                    COALESCE(gs.member_count, 0)::int as "members_count!",
                    (g.metadata->>'join_password' IS NOT NULL) as "has_password!",
                    ST_Y(g.location::geometry) as "latitude?",
                    ST_X(g.location::geometry) as "longitude?",
                    g.location_name as "location_name?",
                    gm.joined_at
                FROM group_members gm
                INNER JOIN groups g ON gm.group_id = g.group_id
                LEFT JOIN group_stats gs ON g.group_id = gs.group_id
                WHERE gm.user_id = $1 AND g.dismissed_at IS NULL
                ORDER BY gm.joined_at DESC
                LIMIT $2
                "#,
                user_id,
                (limit + 1) as i64
            )
            .fetch_all(&self.pool)
            .await?
        };

        let has_more = rows.len() > limit;
        if has_more {
            rows.truncate(limit);
        }
        let next_cursor = if has_more {
            rows.last()
                .map(|r| encode_cursor(&r.joined_at))
                .transpose()?
        } else {
            None
        };

        Ok((rows, next_cursor))
    }

    pub async fn list_admin_members(&self, group_ids: &[i64]) -> Result<HashMap<i64, Vec<i64>>> {
        if group_ids.is_empty() {
            return Ok(HashMap::new());
        }

        let rows = sqlx::query_as!(
            GroupAdminMemberRow,
            r#"
            SELECT group_id, user_id
            FROM group_members
            WHERE group_id = ANY($1)
              AND role IN ('owner'::group_role, 'admin'::group_role)
            "#,
            group_ids
        )
        .fetch_all(&self.pool)
        .await?;

        let mut map: HashMap<i64, Vec<i64>> = HashMap::new();
        for row in rows {
            map.entry(row.group_id).or_default().push(row.user_id);
        }

        Ok(map)
    }

    /// 设置成员禁言
    pub async fn set_mute(
        &self,
        group_id: i64,
        user_id: i64,
        is_muted: bool,
        muted_until: Option<DateTime<Utc>>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET is_muted = $3, muted_until = $4
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id,
            is_muted,
            muted_until
        )
        .execute(&self.pool)
        .await?;

        self.invalidate_member_cache(group_id, user_id).await;
        Ok(())
    }

    /// 列出群组中所有管理员的用户ID（owner + admin）
    pub async fn list_admin_user_ids(&self, group_id: i64) -> Result<Vec<i64>> {
        let rows = sqlx::query!(
            r#"
            SELECT user_id
            FROM group_members
            WHERE group_id = $1 AND role IN ('owner', 'admin')
            "#,
            group_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| r.user_id).collect())
    }

    /// 批量获取用户在多个群组中的角色
    /// 返回 HashMap<group_id, GroupRole>，仅包含用户确实加入了的群组
    pub async fn batch_get_member_roles(
        &self,
        group_ids: &[i64],
        user_id: i64,
    ) -> Result<std::collections::HashMap<i64, GroupRole>> {
        if group_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query!(
            r#"
            SELECT group_id as "group_id!", role as "role!: GroupRole"
            FROM group_members
            WHERE group_id = ANY($1) AND user_id = $2
            "#,
            group_ids,
            user_id
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.group_id, r.role)).collect())
    }

    /// 批量查询用户在多个群组中的成员状态
    /// 返回 HashMap<group_id, GroupMemberRow>，仅包含用户确实加入了的群组
    pub async fn batch_find_member_status(
        &self,
        user_id: i64,
        group_ids: &[i64],
    ) -> Result<std::collections::HashMap<i64, GroupMemberRow>> {
        if group_ids.is_empty() {
            return Ok(std::collections::HashMap::new());
        }

        let rows = sqlx::query_as!(
            GroupMemberRow,
            r#"
            SELECT group_id, user_id, nickname, role as "role!: GroupRole",
                   is_muted, muted_until, joined_at, inviter_id
            FROM group_members
            WHERE user_id = $1 AND group_id = ANY($2)
            "#,
            user_id,
            group_ids
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(rows.into_iter().map(|r| (r.group_id, r)).collect())
    }
}
