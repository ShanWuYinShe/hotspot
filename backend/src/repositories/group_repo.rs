use sqlx::PgPool;
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::group::Group,
};

pub struct GroupRepository<'a> {
    db: &'a PgPool,
}

impl<'a> GroupRepository<'a> {
    pub fn new(db: &'a PgPool) -> Self {
        Self { db }
    }

    /// 创建群组
    pub async fn create(&self, group: &Group) -> Result<Uuid> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO groups (
                group_id, name, description, group_type, owner_id, 
                avatar_url, latitude, longitude, max_members, password_hash
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
            RETURNING group_id
            "#,
            group.group_id,
            group.name,
            group.description,
            group.group_type,
            group.owner_id,
            group.avatar_url,
            group.latitude,
            group.longitude,
            group.max_members,
            group.password_hash,
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.group_id)
    }

    /// 根据ID查找群组
    pub async fn find_by_id(&self, group_id: Uuid) -> Result<Option<Group>> {
        let group = sqlx::query_as!(
            Group,
            r#"
            SELECT * FROM groups WHERE group_id = $1 AND deleted_at IS NULL
            "#,
            group_id
        )
        .fetch_optional(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(group)
    }

    /// 更新群组信息
    pub async fn update(&self, group: &Group) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE groups
            SET name = $2, description = $3, avatar_url = $4, 
                latitude = $5, longitude = $6, max_members = $7,
                password_hash = $8
            WHERE group_id = $1
            "#,
            group.group_id,
            group.name,
            group.description,
            group.avatar_url,
            group.latitude,
            group.longitude,
            group.max_members,
            group.password_hash,
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 软删除群组
    pub async fn soft_delete(&self, group_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE groups
            SET deleted_at = NOW()
            WHERE group_id = $1
            "#,
            group_id
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }

    /// 根据名称搜索群组
    pub async fn search_by_name(
        &self,
        name: &str,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<Group>> {
        let groups = sqlx::query_as!(
            Group,
            r#"
            SELECT * FROM groups
            WHERE name ILIKE $1 AND deleted_at IS NULL
            ORDER BY created_at DESC
            LIMIT $2 OFFSET $3
            "#,
            format!("%{}%", name),
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(groups)
    }

    /// 根据位置搜索群组（使用PostGIS）
    pub async fn search_by_location(
        &self,
        lat: f64,
        lon: f64,
        radius_meters: f64,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<(Group, f64)>> {
        let records = sqlx::query!(
            r#"
            SELECT 
                g.*,
                ST_Distance(
                    ST_MakePoint(g.longitude, g.latitude)::geography,
                    ST_MakePoint($2, $1)::geography
                ) as "distance!"
            FROM groups g
            WHERE 
                g.deleted_at IS NULL
                AND ST_DWithin(
                    ST_MakePoint(g.longitude, g.latitude)::geography,
                    ST_MakePoint($2, $1)::geography,
                    $3
                )
            ORDER BY distance ASC
            LIMIT $4 OFFSET $5
            "#,
            lat,
            lon,
            radius_meters,
            limit,
            offset
        )
        .fetch_all(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        let results = records
            .into_iter()
            .map(|rec| {
                let group = Group {
                    group_id: rec.group_id,
                    name: rec.name,
                    description: rec.description,
                    group_type: rec.group_type,
                    owner_id: rec.owner_id,
                    avatar_url: rec.avatar_url,
                    latitude: rec.latitude,
                    longitude: rec.longitude,
                    max_members: rec.max_members,
                    password_hash: rec.password_hash,
                    created_at: rec.created_at,
                    deleted_at: rec.deleted_at,
                };
                (group, rec.distance)
            })
            .collect();

        Ok(results)
    }

    /// 获取群组成员数量
    pub async fn get_member_count(&self, group_id: Uuid) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM group_members
            WHERE group_id = $1 AND status = 'active'
            "#,
            group_id
        )
        .fetch_one(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(rec.count)
    }

    /// 转移群主
    pub async fn transfer_ownership(&self, group_id: Uuid, new_owner_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE groups
            SET owner_id = $2
            WHERE group_id = $1
            "#,
            group_id,
            new_owner_id
        )
        .execute(self.db)
        .await
        .map_err(|e| AppError::DatabaseError(e.to_string()))?;

        Ok(())
    }
}
