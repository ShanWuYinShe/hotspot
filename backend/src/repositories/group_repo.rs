use sqlx::{PgPool, Row};
use uuid::Uuid;

use crate::{
    error::{AppError, Result},
    models::group::Group,
    types::enums::RoomType,
};

pub struct GroupRepository {
    db: PgPool,
}

impl GroupRepository {
    pub fn new(db: PgPool) -> Self {
        Self { db }
    }

    /// 创建群组
    pub async fn create(&self, group: &Group) -> Result<Uuid> {
        let rec = sqlx::query!(
            r#"
            INSERT INTO chat_rooms (
                room_id, name, description, room_type, owner_id, 
                avatar_url, max_members, password_hash,
                latitude, longitude, location_name
            )
            VALUES ($1, $2, $3, $4::room_type, $5, $6, $7, $8, $9, $10, $11)
            RETURNING room_id
            "#,
            group.group_id,
            group.name,
            group.description,
            group.group_type as RoomType,
            group.owner_id,
            group.avatar_url,
            group.max_members,
            group.password_hash,
            group.latitude,
            group.longitude,
            group.location_name,
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.room_id)
    }

    /// 根据ID查找群组
    pub async fn find_by_id(&self, group_id: Uuid) -> Result<Option<Group>> {
        let group = sqlx::query_as!(
            Group,
            r#"
            SELECT id, room_id as group_id, room_type as "group_type: RoomType", name, description, avatar_url, 
                   owner_id, password_hash, max_members, 
                   COALESCE(latitude, 35.154922) as "latitude!", 
                   COALESCE(longitude, 112.682524) as "longitude!", 
                   COALESCE(location_name, '南天门') as "location_name!",
                   created_at, last_message_at as last_active_at, 
                   is_deleted, deleted_at
            FROM chat_rooms 
            WHERE room_id = $1 
              AND room_type IN ('public_group'::room_type, 'private_group'::room_type)
              AND is_deleted = FALSE
            "#,
            group_id
        )
        .fetch_optional(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(group)
    }

    /// 更新群组信息
    pub async fn update(&self, group: &Group) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE chat_rooms
            SET name = $2, description = $3, avatar_url = $4, 
                max_members = $5, password_hash = $6,
                latitude = $7, longitude = $8, location_name = $9
            WHERE room_id = $1 AND room_type IN ('public_group'::room_type, 'private_group'::room_type)
            "#,
            group.group_id,
            group.name,
            group.description,
            group.avatar_url,
            group.max_members,
            group.password_hash,
            group.latitude,
            group.longitude,
            group.location_name,
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 软删除群组
    pub async fn soft_delete(&self, group_id: Uuid) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE chat_rooms
            SET is_deleted = TRUE, deleted_at = NOW()
            WHERE room_id = $1 AND room_type IN ('public_group'::room_type, 'private_group'::room_type)
            "#,
            group_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }

    /// 根据名称搜索群组
    /// 按名称搜索群组（游标分页）
    /// cursor: room_id (UUID字符串)
    pub async fn search_by_name(
        &self,
        name: &str,
        cursor: Option<String>,
        limit: i64,
    ) -> Result<Vec<Group>> {
        // 1. 解析cursor获取自增主键id
        let cursor_id = if let Some(cursor_str) = cursor {
            let room_uuid = uuid::Uuid::parse_str(&cursor_str)
                .map_err(|_| AppError::BadRequest("无效的游标".to_string()))?;
            
            let result = sqlx::query!(
                "SELECT id FROM chat_rooms WHERE room_id = $1 AND is_deleted = FALSE",
                room_uuid
            )
            .fetch_optional(&self.db)
            .await
            .map_err(|e| AppError::from(e))?;
            
            result.map(|r| r.id)
        } else {
            None
        };

        // 2. 使用自增id进行游标分页
        let groups = sqlx::query_as!(
            Group,
            r#"
            SELECT id, room_id as group_id, room_type as "group_type: RoomType", name, description, avatar_url, 
                   owner_id, password_hash, max_members,
                   COALESCE(latitude, 35.154922) as "latitude!", 
                   COALESCE(longitude, 112.682524) as "longitude!", 
                   COALESCE(location_name, '南天门') as "location_name!",
                   created_at, last_message_at as last_active_at, 
                   is_deleted, deleted_at
            FROM chat_rooms
            WHERE name ILIKE $1 
              AND room_type IN ('public_group'::room_type, 'private_group'::room_type)
              AND is_deleted = FALSE
              AND ($2::BIGINT IS NULL OR id > $2::BIGINT)
            ORDER BY id DESC
            LIMIT $3
            "#,
            format!("%{}%", name),
            cursor_id,
            limit
        )
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

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
        let records = sqlx::query(
            r#"
            WITH latest_locations AS (
                SELECT DISTINCT ON (room_id) 
                    room_id,
                    location,
                    latitude,
                    longitude,
                    location_name
                FROM room_location_history
                ORDER BY room_id, created_at DESC
            )
            SELECT 
                cr.id, cr.room_id, cr.room_type, cr.name, cr.description, cr.avatar_url,
                cr.owner_id, cr.password_hash, cr.max_members, 
                COALESCE(cr.latitude, ll.latitude, 35.154922) as latitude,
                COALESCE(cr.longitude, ll.longitude, 112.682524) as longitude,
                COALESCE(cr.location_name, ll.location_name, '南天门') as location_name,
                cr.created_at, cr.last_message_at, 
                cr.is_deleted, cr.deleted_at,
                ST_Distance(
                    ll.location,
                    ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography
                ) as distance
            FROM chat_rooms cr
            INNER JOIN latest_locations ll ON cr.room_id = ll.room_id
            WHERE 
                cr.room_type IN ('public_group'::room_type, 'private_group'::room_type)
                AND cr.is_deleted = FALSE
                AND ST_DWithin(
                    ll.location,
                    ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
                    $3
                )
            ORDER BY distance ASC
            LIMIT $4 OFFSET $5
            "#
        )
        .bind(lat)
        .bind(lon)
        .bind(radius_meters)
        .bind(limit)
        .bind(offset)
        .fetch_all(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        let results = records
            .iter()
            .map(|rec| {
                let group = Group {
                    id: rec.get("id"),
                    group_id: rec.get("room_id"),
                    group_type: rec.get("room_type"),
                    name: rec.get("name"),
                    description: rec.get("description"),
                    avatar_url: rec.get("avatar_url"),
                    owner_id: rec.get("owner_id"),
                    max_members: rec.get("max_members"),
                    password_hash: rec.get("password_hash"),
                    latitude: rec.get("latitude"),
                    longitude: rec.get("longitude"),
                    location_name: rec.get("location_name"),
                    created_at: rec.get("created_at"),
                    last_active_at: rec.get("last_message_at"),
                    is_deleted: rec.get("is_deleted"),
                    deleted_at: rec.get("deleted_at"),
                };
                let distance: f64 = rec.get("distance");
                (group, distance)
            })
            .collect();

        Ok(results)
    }

    /// 获取群组成员数量
    pub async fn get_member_count(&self, group_id: Uuid) -> Result<i64> {
        let rec = sqlx::query!(
            r#"
            SELECT COUNT(*) as "count!"
            FROM room_members
            WHERE room_id = $1 AND is_deleted = FALSE
            "#,
            group_id
        )
        .fetch_one(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(rec.count)
    }

    /// 转移群主
    pub async fn transfer_ownership(&self, group_id: Uuid, new_owner_id: i64) -> Result<()> {
        sqlx::query!(
            r#"
            UPDATE chat_rooms
            SET owner_id = $2
            WHERE room_id = $1 AND room_type IN ('public_group'::room_type, 'private_group'::room_type)
            "#,
            group_id,
            new_owner_id
        )
        .execute(&self.db)
        .await
        .map_err(|e| AppError::from(e))?;

        Ok(())
    }
}
