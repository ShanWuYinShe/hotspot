use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::group_member::GroupMember,
};

pub struct GroupMemberRepository<'a> {
    db: &'a PgPool,
}

impl<'a> GroupMemberRepository<'a> {
    pub fn new(db: &'a PgPool) -> Self {
        Self { db }
    }

    /// 添加成员
    pub async fn create(&self, member: &GroupMember) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO group_members (
                group_id, user_id, role, nickname, status, muted_until
            )
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id
            "#,
            member.group_id,
            member.user_id,
            member.role,
            member.nickname,
            member.status,
            member.muted_until,
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.id)
    }

    /// 查找成员
    pub async fn find_by_group_and_user(
        &self,
        group_id: Uuid,
        user_id: i64,
    ) -> Result<Option<GroupMember>> {
        let member = sqlx::query_as!(
            GroupMember,
            r#"
            SELECT * FROM group_members
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id
        )
        .fetch_optional(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(member)
    }

    /// 获取群组成员列表
    pub async fn list_by_group(
        &self,
        group_id: Uuid,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<GroupMember>> {
        let members = sqlx::query_as!(
            GroupMember,
            r#"
            SELECT * FROM group_members
            WHERE group_id = $1 AND status = 'active'
            ORDER BY joined_at DESC
            LIMIT $2 OFFSET $3
            "#,
            group_id,
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(members)
    }

    /// 获取用户加入的群组列表
    pub async fn list_by_user(
        &self,
        user_id: i64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<GroupMember>> {
        let members = sqlx::query_as!(
            GroupMember,
            r#"
            SELECT * FROM group_members
            WHERE user_id = $1 AND status = 'active'
            ORDER BY joined_at DESC
            LIMIT $2 OFFSET $3
            "#,
            user_id,
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(members)
    }

    /// 更新成员角色
    pub async fn update_role(&self, group_id: Uuid, user_id: i64, role: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET role = $3
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id,
            role
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 更新成员昵称
    pub async fn update_nickname(
        &self,
        group_id: Uuid,
        user_id: i64,
        nickname: Option<String>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET nickname = $3
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id,
            nickname
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 更新成员状态
    pub async fn update_status(&self, group_id: Uuid, user_id: i64, status: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET status = $3
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id,
            status
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 禁言成员
    pub async fn mute_member(
        &self,
        group_id: Uuid,
        user_id: i64,
        muted_until: chrono::NaiveDateTime,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET muted_until = $3
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id,
            muted_until
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 解除禁言
    pub async fn unmute_member(&self, group_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET muted_until = NULL
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 移除成员（软删除）
    pub async fn remove_member(&self, group_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET status = 'removed', left_at = NOW()
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 成员主动退出
    pub async fn leave_group(&self, group_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE group_members
            SET status = 'left', left_at = NOW()
            WHERE group_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 检查用户是否为群成员
    pub async fn is_member(&self, group_id: Uuid, user_id: i64) -> Result<bool> {
        let rec = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM group_members
                WHERE group_id = $1 AND user_id = $2 AND status = 'active'
            ) as "exists!"
            "#,
            group_id,
            user_id
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.exists)
    }

    /// 获取成员角色
    pub async fn get_member_role(&self, group_id: Uuid, user_id: i64) -> Result<Option<String>> {
        let rec = sqlx::query!(
            r#"
            SELECT role FROM group_members
            WHERE group_id = $1 AND user_id = $2 AND status = 'active'
            "#,
            group_id,
            user_id
        )
        .fetch_optional(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.map(|r| r.role))
    }

    /// 检查用户是否被禁言
    pub async fn is_muted(&self, group_id: Uuid, user_id: i64) -> Result<bool> {
        let rec = sqlx::query!(
            r#"
            SELECT muted_until FROM group_members
            WHERE group_id = $1 AND user_id = $2 AND status = 'active'
            "#,
            group_id,
            user_id
        )
        .fetch_optional(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        if let Some(member) = rec {
            if let Some(muted_until) = member.muted_until {
                return Ok(chrono::Utc::now().naive_utc() < muted_until);
            }
        }

        Ok(false)
    }
}
