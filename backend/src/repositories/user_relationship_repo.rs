use sqlx::PgPool;

use crate::{
    error::{AppError, Result},
    models::user_relationship::UserRelationship,
    types::enums::{RelationshipType, RelationshipStatus},
};

pub struct UserRelationshipRepository {
    pool: PgPool,
}

impl UserRelationshipRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建关系
    pub async fn create(
        &self,
        from_user_id: i64,
        to_user_id: i64,
        relationship_type: RelationshipType,
        status: RelationshipStatus,
    ) -> Result<i64> {
        let result = sqlx::query!(
            r#"
            INSERT INTO user_relationships (from_user_id, to_user_id, relationship_type, status)
            VALUES ($1, $2, $3::relationship_type, $4::relationship_status)
            RETURNING relationship_id
            "#,
            from_user_id,
            to_user_id,
            relationship_type as RelationshipType,
            status as RelationshipStatus
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.relationship_id)
    }

    /// 查找关系
    pub async fn find_relationship(
        &self,
        from_user_id: i64,
        to_user_id: i64,
        relationship_type: RelationshipType,
    ) -> Result<Option<UserRelationship>> {
        let relationship = sqlx::query_as!(
            UserRelationship,
            r#"
            SELECT relationship_id, from_user_id, to_user_id, relationship_type as "relationship_type: RelationshipType", status as "status: RelationshipStatus", remark, created_at, updated_at
            FROM user_relationships
            WHERE from_user_id = $1 AND to_user_id = $2 AND relationship_type = $3::relationship_type
            "#,
            from_user_id,
            to_user_id,
            relationship_type as RelationshipType
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(relationship)
    }

    /// 查找双向关系（好友）
    pub async fn find_mutual_relationship(
        &self,
        user1_id: i64,
        user2_id: i64,
        relationship_type: RelationshipType,
    ) -> Result<Option<UserRelationship>> {
        let relationship = sqlx::query_as!(
            UserRelationship,
            r#"
            SELECT relationship_id, from_user_id, to_user_id, relationship_type as "relationship_type: RelationshipType", status as "status: RelationshipStatus", remark, created_at, updated_at
            FROM user_relationships
            WHERE ((from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1))
              AND relationship_type = $3::relationship_type
            ORDER BY created_at DESC
            LIMIT 1
            "#,
            user1_id,
            user2_id,
            relationship_type as RelationshipType
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(relationship)
    }

    /// 更新关系状态
    pub async fn update_status(&self, relationship_id: i64, status: RelationshipStatus) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE user_relationships
            SET status = $2::relationship_status, updated_at = NOW()
            WHERE relationship_id = $1
            "#,
            relationship_id,
            status as RelationshipStatus
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 删除关系
    pub async fn delete(&self, relationship_id: i64) -> Result<()> {
        sqlx::query!(
            "DELETE FROM user_relationships WHERE relationship_id = $1",
            relationship_id
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 删除用户之间的关系
    pub async fn delete_by_users(
        &self,
        from_user_id: i64,
        to_user_id: i64,
        relationship_type: RelationshipType,
    ) -> Result<()> {
        sqlx::query!(
            "DELETE FROM user_relationships WHERE from_user_id = $1 AND to_user_id = $2 AND relationship_type = $3::relationship_type",
            from_user_id,
            to_user_id,
            relationship_type as RelationshipType
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 获取好友列表
    pub async fn list_friends(
        &self,
        user_id: i64,
        status: Option<RelationshipStatus>,
        limit: i32,
        cursor: Option<i64>,
    ) -> Result<(Vec<UserRelationship>, Option<String>)> {
        let relationships = if let Some(status) = status {
            sqlx::query_as!(
                UserRelationship,
                r#"
                SELECT relationship_id, from_user_id, to_user_id, relationship_type as "relationship_type: RelationshipType", status as "status: RelationshipStatus", remark, created_at, updated_at
                FROM user_relationships
                WHERE (from_user_id = $1 OR to_user_id = $1)
                  AND relationship_type = 'friend'::relationship_type
                  AND status = $2::relationship_status
                  AND ($3::BIGINT IS NULL OR relationship_id > $3::BIGINT)
                ORDER BY relationship_id
                LIMIT $4
                "#,
                user_id,
                status as RelationshipStatus,
                cursor,
                limit as i64
            )
            .fetch_all(&self.pool)
            .await?
        } else {
            sqlx::query_as!(
                UserRelationship,
                r#"
                SELECT relationship_id, from_user_id, to_user_id, relationship_type as "relationship_type: RelationshipType", status as "status: RelationshipStatus", remark, created_at, updated_at
                FROM user_relationships
                WHERE (from_user_id = $1 OR to_user_id = $1)
                  AND relationship_type = 'friend'::relationship_type
                  AND ($2::BIGINT IS NULL OR relationship_id > $2::BIGINT)
                ORDER BY relationship_id
                LIMIT $3
                "#,
                user_id,
                cursor,
                limit as i64
            )
            .fetch_all(&self.pool)
            .await?
        };

        let next_cursor = if relationships.len() == limit as usize {
            relationships.last().map(|r| r.relationship_id.to_string())
        } else {
            None
        };

        Ok((relationships, next_cursor))
    }

    /// 获取关注列表
    pub async fn list_following(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<i64>,
    ) -> Result<(Vec<UserRelationship>, Option<String>)> {
        let relationships = sqlx::query_as!(
            UserRelationship,
            r#"
            SELECT relationship_id, from_user_id, to_user_id, relationship_type as "relationship_type: RelationshipType", status as "status: RelationshipStatus", remark, created_at, updated_at
            FROM user_relationships
            WHERE from_user_id = $1
              AND relationship_type = 'following'::relationship_type
              AND ($2::BIGINT IS NULL OR relationship_id > $2::BIGINT)
            ORDER BY relationship_id
            LIMIT $3
            "#,
            user_id,
            cursor,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = if relationships.len() == limit as usize {
            relationships.last().map(|r| r.relationship_id.to_string())
        } else {
            None
        };

        Ok((relationships, next_cursor))
    }

    /// 获取粉丝列表
    pub async fn list_followers(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<i64>,
    ) -> Result<(Vec<UserRelationship>, Option<String>)> {
        let relationships = sqlx::query_as!(
            UserRelationship,
            r#"
            SELECT relationship_id, from_user_id, to_user_id, relationship_type as "relationship_type: RelationshipType", status as "status: RelationshipStatus", remark, created_at, updated_at
            FROM user_relationships
            WHERE to_user_id = $1
              AND relationship_type = 'following'::relationship_type
              AND ($2::BIGINT IS NULL OR relationship_id > $2::BIGINT)
            ORDER BY relationship_id
            LIMIT $3
            "#,
            user_id,
            cursor,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = if relationships.len() == limit as usize {
            relationships.last().map(|r| r.relationship_id.to_string())
        } else {
            None
        };

        Ok((relationships, next_cursor))
    }

    /// 获取拉黑列表
    pub async fn list_blocked(
        &self,
        user_id: i64,
        limit: i32,
        cursor: Option<i64>,
    ) -> Result<(Vec<UserRelationship>, Option<String>)> {
        let relationships = sqlx::query_as!(
            UserRelationship,
            r#"
            SELECT relationship_id, from_user_id, to_user_id, relationship_type as "relationship_type: RelationshipType", status as "status: RelationshipStatus", remark, created_at, updated_at
            FROM user_relationships
            WHERE from_user_id = $1
              AND relationship_type = 'blocked'::relationship_type
              AND ($2::BIGINT IS NULL OR relationship_id > $2::BIGINT)
            ORDER BY relationship_id
            LIMIT $3
            "#,
            user_id,
            cursor,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await?;

        let next_cursor = if relationships.len() == limit as usize {
            relationships.last().map(|r| r.relationship_id.to_string())
        } else {
            None
        };

        Ok((relationships, next_cursor))
    }

    /// 设置好友备注
    pub async fn set_remark(&self, relationship_id: i64, remark: &str) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE user_relationships
            SET remark = $2, updated_at = NOW()
            WHERE relationship_id = $1
            "#,
            relationship_id,
            remark
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 检查是否被拉黑
    pub async fn is_blocked(&self, from_user_id: i64, to_user_id: i64) -> Result<bool> {
        let result = sqlx::query!(
            r#"
            SELECT EXISTS(
                SELECT 1 FROM user_relationships
                WHERE from_user_id = $1 AND to_user_id = $2 AND relationship_type = 'blocked'::relationship_type
            ) as "exists!"
            "#,
            from_user_id,
            to_user_id
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(result.exists)
    }
}
