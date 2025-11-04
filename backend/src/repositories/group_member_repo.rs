use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::group_member::GroupMember,
    types::enums::MemberRole,
};

pub struct GroupMemberRepository {
    db: PgPool,
}

impl GroupMemberRepository {
    pub fn new(db: PgPool) -> Self {
        Self { db }
    }

    /// 添加成员
    pub async fn create(&self, member: &GroupMember) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO room_members (
                room_id, user_id, role, nickname, is_banned, is_muted, muted_until, unread_count, last_read_at
            )
            VALUES ($1, $2, $3::member_role, $4, $5, $6, $7, $8, $9)
            RETURNING id
            "#,
            member.group_id,
            member.user_id,
            member.role as MemberRole,
            member.nickname,
            member.is_banned,
            member.is_muted,
            member.muted_until,
            member.unread_count,
            member.last_read_at,
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.id)
    }

    /// 添加成员（别名方法）
    pub async fn add_member(&self, member: &GroupMember) -> Result<i64> {
        self.create(member).await
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
            SELECT id, room_id as group_id, user_id, role as "role: MemberRole", nickname, is_banned,
                   is_muted, muted_until, joined_at, unread_count, last_read_at, is_deleted, deleted_at
            FROM room_members
            WHERE room_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id
        )
        .fetch_optional(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(member)
    }

    /// 查找成员（别名方法）
    pub async fn find_by_room_and_user(
        &self,
        room_id: Uuid,
        user_id: i64,
    ) -> Result<Option<GroupMember>> {
        self.find_by_group_and_user(room_id, user_id).await
    }

    /// 获取群组成员列表
    /// 获取群组成员列表（游标分页）
    /// cursor: user_id (BIGINT字符串)
    pub async fn list_by_group(
        &self,
        group_id: Uuid,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<GroupMember>> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let user_id_val: i64 = cursor_str.parse()
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM room_members WHERE room_id = $1 AND user_id = $2 AND is_deleted = FALSE",
                group_id,
                user_id_val
            )
            .fetch_optional(&self.db)
            .await
            .map_err(|e| AppError::from(e))?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用自增id进行游标分页
        let members = sqlx::query_as!(
            GroupMember,
            r#"
            SELECT id, room_id as group_id, user_id, role as "role: MemberRole", nickname, is_banned,
                   is_muted, muted_until, joined_at, unread_count, last_read_at, is_deleted, deleted_at
            FROM room_members
            WHERE room_id = $1 
            AND is_deleted = FALSE
            AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id DESC
            LIMIT $3
            "#,
            group_id,
            cursor_id,
            limit
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

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
            SELECT id, room_id as group_id, user_id, role as "role: MemberRole", nickname, is_banned,
                   is_muted, muted_until, joined_at, unread_count, last_read_at, is_deleted, deleted_at
            FROM room_members
            WHERE user_id = $1 AND is_deleted = FALSE
            ORDER BY joined_at DESC
            LIMIT $2 OFFSET $3
            "#,
            user_id,
            limit,
            offset
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(members)
    }

    /// 更新成员角色
    pub async fn update_role(&self, group_id: Uuid, user_id: i64, role: MemberRole) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE room_members
            SET role = $3::member_role
            WHERE room_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id,
            role as MemberRole
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

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
            UPDATE room_members
            SET nickname = $3
            WHERE room_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id,
            nickname
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 更新成员封禁状态
    pub async fn update_ban_status(&self, group_id: Uuid, user_id: i64, is_banned: bool) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE room_members
            SET is_banned = $3
            WHERE room_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id,
            is_banned
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 禁言成员
    pub async fn mute_member(
        &self,
        group_id: Uuid,
        user_id: i64,
        muted_until: chrono::DateTime<chrono::Utc>,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE room_members
            SET muted_until = $3
            WHERE room_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id,
            muted_until
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 解除禁言
    pub async fn unmute_member(&self, group_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE room_members
            SET muted_until = NULL
            WHERE room_id = $1 AND user_id = $2
            "#,
            group_id,
            user_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 移除成员（软删除）
    pub async fn remove_member(&self, group_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE room_members
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE room_id = $1 AND user_id = $2 AND is_deleted = FALSE
            "#,
            group_id,
            user_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 成员主动退出（软删除）
    pub async fn leave_group(&self, group_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE room_members
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE room_id = $1 AND user_id = $2 AND is_deleted = FALSE
            "#,
            group_id,
            user_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 检查用户是否为群成员
    pub async fn is_member(&self, group_id: Uuid, user_id: i64) -> Result<bool> {
        let rec = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM room_members
                WHERE room_id = $1 AND user_id = $2 AND is_deleted = FALSE
            ) as "exists!"
            "#,
            group_id,
            user_id
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.exists)
    }

    /// 获取成员角色
    pub async fn get_member_role(&self, group_id: Uuid, user_id: i64) -> Result<Option<MemberRole>> {
        let rec = sqlx::query!(
            r#"
            SELECT role as "role: MemberRole" FROM room_members
            WHERE room_id = $1 AND user_id = $2 AND is_deleted = FALSE
            "#,
            group_id,
            user_id
        )
        .fetch_optional(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.map(|r| r.role))
    }

    /// 检查用户是否被禁言
    pub async fn is_muted(&self, group_id: Uuid, user_id: i64) -> Result<bool> {
        let rec = sqlx::query!(
            r#"
            SELECT muted_until FROM room_members
            WHERE room_id = $1 AND user_id = $2 AND is_deleted = FALSE
            "#,
            group_id,
            user_id
        )
        .fetch_optional(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        if let Some(member) = rec {
            if let Some(muted_until) = member.muted_until {
                return Ok(chrono::Utc::now() < muted_until);
            }
        }

        Ok(false)
    }

    /// 增加未读计数
    pub async fn increment_unread(&self, room_id: Uuid, user_id: i64, count: i32) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE room_members
            SET unread_count = unread_count + $3
            WHERE room_id = $1 AND user_id = $2 AND is_deleted = FALSE
            "#,
            room_id,
            user_id,
            count
        )
        .execute(&self.db)
        .await?;

        Ok(())
    }

    /// 清空未读计数
    pub async fn clear_unread(&self, room_id: Uuid, user_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE room_members
            SET unread_count = 0, last_read_at = NOW()
            WHERE room_id = $1 AND user_id = $2 AND is_deleted = FALSE
            "#,
            room_id,
            user_id
        )
        .execute(&self.db)
        .await?;

        Ok(())
    }

    /// 获取未读计数
    pub async fn get_unread_count(&self, room_id: Uuid, user_id: i64) -> Result<i32> {
        let rec = sqlx::query!(
            r#"
            SELECT unread_count FROM room_members
            WHERE room_id = $1 AND user_id = $2 AND is_deleted = FALSE
            "#,
            room_id,
            user_id
        )
        .fetch_optional(&self.db)
        .await?;

        Ok(rec.map(|r| r.unread_count).unwrap_or(0))
    }

    /// 统计用户所有聊天室的未读总数
    pub async fn get_total_unread(&self, user_id: i64) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COALESCE(SUM(unread_count), 0) as "total!"
            FROM room_members
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_one(&self.db)
        .await?;

        Ok(rec.total)
    }

    /// 获取成员数量
    pub async fn count_members(&self, room_id: Uuid) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM room_members
            WHERE room_id = $1 AND is_deleted = FALSE
            "#,
            room_id
        )
        .fetch_one(&self.db)
        .await?;

        Ok(rec.count)
    }
}
