use sqlx::{PgPool, FromRow};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::error::Result;
use crate::types::enums::MemberRole;

/// 群组成员Row - 数据库查询返回类型
#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct GroupMemberRow {
    pub id: i64,
    pub cursor_id: Uuid,
    pub group_chat_id: Uuid,
    pub user_id: i64,
    pub role: MemberRole,
    pub joined_at: DateTime<Utc>,
    pub muted_until: Option<DateTime<Utc>>,
    pub is_deleted: bool,
    pub deleted_at: Option<DateTime<Utc>>,
    
    // Joined fields
    pub nickname: Option<String>,
    pub avatar_url: Option<String>,
    pub is_muted: Option<bool>, // Computed field
    pub is_banned: Option<bool>, // Computed field (placeholder)
}

#[derive(Debug, Clone)]
pub struct GroupMemberRepository {
    pool: PgPool,
}

impl GroupMemberRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 添加成员
    pub async fn add_member(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
        role: MemberRole,
    ) -> Result<GroupMemberRow> {
        let row = sqlx::query_as!(
            GroupMemberRow,
            r#"
            WITH inserted AS (
                INSERT INTO group_members (group_chat_id, user_id, role)
                VALUES ($1, $2, $3::member_role)
                RETURNING id, cursor_id, group_chat_id, user_id, role, joined_at, muted_until, is_deleted, deleted_at
            )
            SELECT i.id, i.cursor_id, i.group_chat_id, i.user_id, i.role as "role: MemberRole", 
                   i.joined_at, i.muted_until, i.is_deleted, i.deleted_at,
                   u.nickname, u.avatar_url,
                   (i.muted_until > NOW()) as "is_muted",
                   FALSE as "is_banned"
            FROM inserted i
            LEFT JOIN users u ON i.user_id = u.id
            "#,
            group_chat_id,
            user_id,
            role as MemberRole
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(row)
    }

    /// 查找成员
    pub async fn find_member(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
    ) -> Result<Option<GroupMemberRow>> {
        let row = sqlx::query_as!(
            GroupMemberRow,
            r#"
            SELECT m.id, m.cursor_id, m.group_chat_id, m.user_id, m.role as "role: MemberRole", 
                   m.joined_at, m.muted_until, m.is_deleted, m.deleted_at,
                   u.nickname, u.avatar_url,
                   (m.muted_until > NOW()) as "is_muted",
                   FALSE as "is_banned"
            FROM group_members m
            LEFT JOIN users u ON m.user_id = u.user_id
            WHERE m.group_chat_id = $1 AND m.user_id = $2 AND m.is_deleted = FALSE
            "#,
            group_chat_id,
            user_id
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(row)
    }

    /// 查找成员（别名方法）
    pub async fn find_by_user_and_group(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
    ) -> Result<Option<GroupMemberRow>> {
        self.find_member(group_chat_id, user_id).await
    }

    /// 更新成员角色
    pub async fn update_role(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
        role: MemberRole,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET role = $3::member_role
            WHERE group_chat_id = $1 AND user_id = $2
            "#,
            group_chat_id,
            user_id,
            role as MemberRole
        )
        .execute(&self.pool)
        .await?;

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

        Ok(())
    }

    /// 获取群组成员数
    pub async fn count_members(&self, group_chat_id: Uuid) -> Result<i64> {
        let count = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM group_members
            WHERE group_chat_id = $1 AND is_deleted = FALSE
            "#,
            group_chat_id
        )
        .fetch_one(&self.pool)
        .await?
        .count;

        Ok(count)
    }

    /// 按群组和用户查找成员
    pub async fn find_by_group_and_user(
        &self,
        group_chat_id: Uuid,
        user_id: i64,
    ) -> Result<Option<GroupMemberRow>> {
        self.find_member(group_chat_id, user_id).await
    }

    /// 获取成员角色
    pub async fn get_member_role(&self, group_chat_id: Uuid, user_id: i64) -> Result<Option<MemberRole>> {
        let member = self.find_member(group_chat_id, user_id).await?;
        Ok(member.map(|m| m.role))
    }

    /// 检查用户是否是群组成员
    pub async fn is_member(&self, group_chat_id: Uuid, user_id: i64) -> Result<bool> {
        let member = self.find_member(group_chat_id, user_id).await?;
        Ok(member.is_some())
    }

    /// 创建成员（简化版add_member）
    pub async fn create(&self, group_chat_id: Uuid, user_id: i64, role: crate::types::enums::MemberRole) -> Result<GroupMemberRow> {
        self.add_member(group_chat_id, user_id, role).await
    }

    /// 将 UUID cursor 转换为自增主键
    async fn resolve_cursor(&self, cursor: Option<Uuid>) -> Result<i64> {
        match cursor {
            Some(c) => {
                sqlx::query_scalar!(
                    r#"SELECT id FROM group_members WHERE cursor_id = $1"#,
                    c
                )
                .fetch_optional(&self.pool)
                .await?
                .ok_or_else(|| crate::error::AppError::BadRequest("无效的 cursor".to_string()))
            }
            None => Ok(i64::MAX),
        }
    }

    /// 按群组ID查询成员列表
    pub async fn list_by_group(&self, group_chat_id: Uuid, limit: i32, cursor: Option<Uuid>) -> Result<Vec<GroupMemberRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let members = sqlx::query_as!(
            GroupMemberRow,
            r#"
            SELECT m.id, m.cursor_id, m.group_chat_id, m.user_id, m.role as "role: MemberRole", 
                   m.joined_at, m.muted_until, m.is_deleted, m.deleted_at,
                   u.nickname, u.avatar_url,
                   (m.muted_until > NOW()) as "is_muted",
                   FALSE as "is_banned"
            FROM group_members m
            LEFT JOIN users u ON m.user_id = u.id
            WHERE m.group_chat_id = $1 AND m.id < $2 AND m.is_deleted = FALSE
            ORDER BY m.id DESC
            LIMIT $3
            "#,
            group_chat_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(members)
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
        Ok(())
    }

    /// 禁言成员
    pub async fn mute_member(&self, group_chat_id: Uuid, user_id: i64, muted_until: chrono::DateTime<chrono::Utc>) -> Result<()> {
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

    /// 按用户ID查询所加入的群组
    pub async fn list_by_user(&self, user_id: i64, limit: i32, cursor: Option<Uuid>) -> Result<Vec<GroupMemberRow>> {
        let cursor_pk = self.resolve_cursor(cursor).await?;
        
        let members = sqlx::query_as!(
            GroupMemberRow,
            r#"
            SELECT m.id, m.cursor_id, m.group_chat_id, m.user_id, m.role as "role: MemberRole", 
                   m.joined_at, m.muted_until, m.is_deleted, m.deleted_at,
                   u.nickname, u.avatar_url,
                   (m.muted_until > NOW()) as "is_muted",
                   FALSE as "is_banned"
            FROM group_members m
            LEFT JOIN users u ON m.user_id = u.user_id
            WHERE m.user_id = $1 AND m.id < $2 AND m.is_deleted = FALSE
            ORDER BY m.id DESC
            LIMIT $3
            "#,
            user_id,
            cursor_pk,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;
        
        Ok(members)
    }
}
