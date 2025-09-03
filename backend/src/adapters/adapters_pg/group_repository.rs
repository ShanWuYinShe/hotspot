use crate::domain::{DomainError, DomainResult, Group};
use crate::ports::{GroupMemberInfo, GroupRepositoryPort};
/// 群组仓储PostgreSQL适配器（待实现）
use async_trait::async_trait;
use sqlx::PgPool;
use uuid::Uuid;

/// PostgreSQL群组仓储实现
#[derive(Clone)]
pub struct PgGroupRepository {
    pool: PgPool,
}

impl PgGroupRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl GroupRepositoryPort for PgGroupRepository {
    async fn find_by_id(&self, group_id: i64) -> DomainResult<Option<Group>> {
        let row = sqlx::query!(
            r#"
            SELECT
                id as "id!",
                group_id as "group_id!",
                name as "name!",
                description,
                owner_id as "owner_id!",
                latitude as "latitude!",
                longitude as "longitude!",
                location_name as "location_name!",
                password_hash,
                created_at as "created_at!",
                updated_at as "updated_at!",
                last_active_at as "last_active_at!"
            FROM groups
            WHERE id = $1
            "#,
            group_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let g = row.map(|r| Group {
            id: r.id,
            group_id: r.group_id.to_string(),
            name: r.name,
            description: r.description,
            owner_id: r.owner_id,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            member_count: 1, // Default value, will be updated by migration
            max_members: 100, // Default value
            is_public: true, // Default value
            password_hash: r.password_hash,
            created_at: r.created_at,
            updated_at: r.updated_at,
            last_active_at: r.last_active_at,
        });
        Ok(g)
    }

    async fn create_group(&self, group: &Group) -> DomainResult<Group> {
        let row = sqlx::query!(
            r#"
            INSERT INTO groups (group_id, name, description, owner_id, latitude, longitude, location_name, password_hash, created_at, updated_at, last_active_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
            RETURNING
                id as "id!",
                group_id as "group_id!",
                name as "name!",
                description,
                owner_id as "owner_id!",
                latitude as "latitude!",
                longitude as "longitude!",
                location_name as "location_name!",
                password_hash,
                created_at as "created_at!",
                updated_at as "updated_at!",
                last_active_at as "last_active_at!"
            "#,
            Uuid::parse_str(&group.group_id).map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Invalid UUID: {}", e) })?,
            group.name,
            group.description,
            group.owner_id,
            group.latitude,
            group.longitude,
            group.location_name,
            group.password_hash,
            group.created_at,
            group.updated_at,
            group.last_active_at
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        Ok(Group {
            id: row.id,
            group_id: row.group_id.to_string(),
            name: row.name,
            description: row.description,
            owner_id: row.owner_id,
            latitude: row.latitude,
            longitude: row.longitude,
            location_name: row.location_name,
            member_count: group.member_count,
            max_members: group.max_members,
            is_public: group.is_public,
            password_hash: row.password_hash,
            created_at: row.created_at,
            updated_at: row.updated_at,
            last_active_at: row.last_active_at,
        })
    }

    async fn update_group(&self, group: &Group) -> DomainResult<Group> {
        let row = sqlx::query!(
            r#"
            UPDATE groups
            SET name = COALESCE($2, name),
                description = $3,
                latitude = $4,
                longitude = $5,
                location_name = COALESCE($6, location_name),
                updated_at = $7
            WHERE id = $1
            RETURNING
                id as "id!",
                group_id as "group_id!",
                name as "name!",
                description,
                owner_id as "owner_id!",
                latitude as "latitude!",
                longitude as "longitude!",
                location_name as "location_name!",
                password_hash,
                created_at as "created_at!",
                updated_at as "updated_at!",
                last_active_at as "last_active_at!"
            "#,
            group.id,
            Some(group.name.clone()),
            group.description.clone(),
            group.latitude,
            group.longitude,
            group.location_name.clone(),
            group.updated_at
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(Group {
            id: row.id,
            group_id: row.group_id.to_string(),
            name: row.name,
            description: row.description,
            owner_id: row.owner_id,
            latitude: row.latitude,
            longitude: row.longitude,
            location_name: row.location_name,
            member_count: group.member_count, // Keep existing value
            max_members: group.max_members, // Keep existing value
            is_public: group.is_public, // Keep existing value
            password_hash: row.password_hash,
            created_at: row.created_at,
            updated_at: row.updated_at,
            last_active_at: row.last_active_at,
        })
    }

    async fn delete_group(&self, group_id: i64) -> DomainResult<()> {
        let rows = sqlx::query!(r#"DELETE FROM groups WHERE id = $1"#, group_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
            .rows_affected();
        if rows == 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "Group not found".to_string(),
            });
        }
        Ok(())
    }

    async fn find_nearby_groups(
        &self,
        latitude: f64,
        longitude: f64,
        radius_km: f64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Group>> {
        let rows = sqlx::query!(
            r#"
            SELECT
                id as "id!",
                group_id as "group_id!",
                name as "name!",
                description,
                owner_id as "owner_id!",
                latitude as "latitude!",
                longitude as "longitude!",
                location_name as "location_name!",
                password_hash,
                created_at as "created_at!",
                updated_at as "updated_at!",
                last_active_at as "last_active_at!",
                ST_Distance(
                    geom,
                    ST_Point($2, $1)::geography
                ) / 1000.0 as distance_km
            FROM groups
            WHERE ($4::bigint = 0 OR id < $4::bigint)
              AND ST_DWithin(
                    geom,
                    ST_Point($2, $1)::geography,
                    $3 * 1000
              )
            ORDER BY id DESC
            LIMIT $5::bigint
            "#,
            latitude,
            longitude,
            (radius_km as i32),
            cursor,
            (limit as i64)
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let groups = rows
            .into_iter()
            .map(|r| Group {
                id: r.id,
                group_id: r.group_id.to_string(),
                name: r.name,
                description: r.description,
                owner_id: r.owner_id,
                latitude: r.latitude,
                longitude: r.longitude,
                location_name: r.location_name,
                member_count: 1, // Default value, will be updated by migration
                max_members: 100, // Default value
                is_public: true, // Default value
                password_hash: r.password_hash,
                created_at: r.created_at,
                updated_at: r.updated_at,
                last_active_at: r.last_active_at,
            })
            .collect();
        Ok(groups)
    }

    async fn search_groups_by_name(
        &self,
        _keyword: &str,
        _cursor: i64,
        _limit: i32,
    ) -> DomainResult<Vec<Group>> {
        let pattern = format!("%{}%", _keyword);
        let rows = sqlx::query!(
            r#"
            SELECT
                id as "id!",
                group_id as "group_id!",
                name as "name!",
                description,
                owner_id as "owner_id!",
                latitude as "latitude!",
                longitude as "longitude!",
                location_name as "location_name!",
                password_hash,
                created_at as "created_at!",
                updated_at as "updated_at!",
                last_active_at as "last_active_at!"
            FROM groups
            WHERE name ILIKE $1 AND ($2::bigint = 0 OR id < $2::bigint)
            ORDER BY id DESC
            LIMIT $3::bigint
            "#,
            pattern,
            _cursor,
            (_limit as i64)
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let groups = rows
            .into_iter()
            .map(|r| Group {
                id: r.id,
                group_id: r.group_id.to_string(),
                name: r.name,
                description: r.description,
                owner_id: r.owner_id,
                latitude: r.latitude,
                longitude: r.longitude,
                location_name: r.location_name,
                member_count: 1, // Default value, will be updated by migration
                max_members: 100, // Default value
                is_public: true, // Default value
                password_hash: r.password_hash,
                created_at: r.created_at,
                updated_at: r.updated_at,
                last_active_at: r.last_active_at,
            })
            .collect();
        Ok(groups)
    }

    // -------- 新增：成员管理与检查方法（使用 groups.id -> groups.group_id(UUID) 转换） --------

    async fn check_group_exists(&self, group_id: i64) -> DomainResult<bool> {
        let exists = sqlx::query_scalar!(
            r#"SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1) as "exists!""#,
            group_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;
        Ok(exists)
    }

    async fn check_user_in_group(&self, group_id: i64, user_id: i64) -> DomainResult<bool> {
        let group_uuid: Option<Uuid> =
            sqlx::query_scalar!(r#"SELECT group_id FROM groups WHERE id = $1"#, group_id)
                .fetch_optional(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("Database error: {}", e),
                })?;
        if let Some(gid) = group_uuid {
            let exists = sqlx::query_scalar!(
                r#"SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2) as "exists!""#,
                gid,
                user_id
            )
            .fetch_one(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;
            Ok(exists)
        } else {
            Ok(false)
        }
    }

    async fn check_user_is_group_owner(&self, group_id: i64, user_id: i64) -> DomainResult<bool> {
        let is_owner = sqlx::query_scalar!(
            r#"SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1 AND owner_id = $2) as "exists!""#,
            group_id,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;
        Ok(is_owner)
    }

    async fn check_user_is_group_admin(&self, group_id: i64, user_id: i64) -> DomainResult<bool> {
        // admin 或 owner 皆视为具备管理权限
        let group_uuid: Option<Uuid> =
            sqlx::query_scalar!(r#"SELECT group_id FROM groups WHERE id = $1"#, group_id)
                .fetch_optional(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("Database error: {}", e),
                })?;
        if let Some(gid) = group_uuid {
            let is_admin = sqlx::query_scalar!(
                r#"
                SELECT EXISTS(
                    SELECT 1 FROM group_members
                    WHERE group_id = $1 AND user_id = $2 AND role IN ('admin','owner')
                ) as "exists!"
                "#,
                gid,
                user_id
            )
            .fetch_one(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?;
            Ok(is_admin)
        } else {
            Ok(false)
        }
    }

    async fn join_group(
        &self,
        group_id: i64,
        user_id: i64,
        _password: Option<&str>,
    ) -> DomainResult<(bool, String)> {
        // 获取群组UUID
        let group_uuid: Uuid =
            sqlx::query_scalar!(r#"SELECT group_id FROM groups WHERE id = $1"#, group_id)
                .fetch_one(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("加入群组失败: {}", e),
                })?;

        // 插入成员（若已存在将因唯一约束报错）
        sqlx::query!(
            r#"INSERT INTO group_members (group_id, user_id, role, joined_at) VALUES ($1, $2, 'member', $3)"#,
            group_uuid,
            user_id,
            chrono::Utc::now()
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("加入群组失败: {}", e) })?;

        Ok((true, "member".to_string()))
    }

    async fn leave_group(&self, group_id: i64, user_id: i64) -> DomainResult<bool> {
        let group_uuid: Uuid =
            sqlx::query_scalar!(r#"SELECT group_id FROM groups WHERE id = $1"#, group_id)
                .fetch_one(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("离开群组失败: {}", e),
                })?;

        let _ = sqlx::query!(
            r#"DELETE FROM group_members WHERE group_id = $1 AND user_id = $2"#,
            group_uuid,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("离开群组失败: {}", e),
        })?
        .rows_affected();

        Ok(true)
    }

    async fn kick_member(
        &self,
        group_id: i64,
        target_user_id: i64,
        _admin_id: i64,
    ) -> DomainResult<bool> {
        let group_uuid: Uuid =
            sqlx::query_scalar!(r#"SELECT group_id FROM groups WHERE id = $1"#, group_id)
                .fetch_one(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("移除成员失败: {}", e),
                })?;

        let _ = sqlx::query!(
            r#"DELETE FROM group_members WHERE group_id = $1 AND user_id = $2"#,
            group_uuid,
            target_user_id
        )
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("移除成员失败: {}", e),
        })?
        .rows_affected();

        Ok(true)
    }

    async fn transfer_ownership(
        &self,
        group_id: i64,
        new_owner_id: i64,
        current_owner_id: i64,
    ) -> DomainResult<bool> {
        // 获取群UUID
        let group_uuid: Uuid =
            sqlx::query_scalar!(r#"SELECT group_id FROM groups WHERE id = $1"#, group_id)
                .fetch_one(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("转移所有权失败: {}", e),
                })?;

        let mut tx = self
            .pool
            .begin()
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("开启事务失败: {}", e),
            })?;

        // 更新群主
        sqlx::query!(
            r#"UPDATE groups SET owner_id = $1, updated_at = $2 WHERE id = $3"#,
            new_owner_id,
            chrono::Utc::now(),
            group_id
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("更新群主失败: {}", e),
        })?;

        // 更新成员角色：新群主->owner，旧群主->member（若非成员则插入）
        sqlx::query!(
            r#"
            INSERT INTO group_members (group_id, user_id, role, joined_at)
            VALUES ($1, $2, 'owner', $3)
            ON CONFLICT (group_id, user_id) DO UPDATE SET role = 'owner'
            ;
            "#,
            group_uuid,
            new_owner_id,
            chrono::Utc::now()
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("更新新群主角色失败: {}", e),
        })?;

        sqlx::query!(
            r#"
            INSERT INTO group_members (group_id, user_id, role, joined_at)
            VALUES ($1, $2, 'member', $3)
            ON CONFLICT (group_id, user_id) DO UPDATE SET role = 'member'
            ;
            "#,
            group_uuid,
            current_owner_id,
            chrono::Utc::now()
        )
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("降级旧群主角色失败: {}", e),
        })?;

        tx.commit()
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("提交事务失败: {}", e),
            })?;

        Ok(true)
    }

    async fn get_group_members(
        &self,
        group_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<(Vec<GroupMemberInfo>, bool, Option<i64>)> {
        let group_uuid: Uuid =
            sqlx::query_scalar!(r#"SELECT group_id FROM groups WHERE id = $1"#, group_id)
                .fetch_one(&self.pool)
                .await
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("获取群成员失败: {}", e),
                })?;

        // 取 limit + 1 以判断是否有更多
        let rows = sqlx::query!(
            r#"
            SELECT
                gm.id as "id!",
                gm.user_id as "user_id!",
                COALESCE(u.nickname, '未知用户') as "nickname!",
                u.last_active_at as "last_active_at!",
                gm.role::text as "role!",
                gm.joined_at as "joined_at!"
            FROM group_members gm
            JOIN users u ON gm.user_id = u.user_id
            WHERE gm.group_id = $1
              AND ($2::bigint = 0 OR gm.id < $2::bigint)
            ORDER BY gm.id DESC
            LIMIT $3::bigint
            "#,
            group_uuid,
            cursor,
            ((limit as i64) + 1)
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("获取群成员失败: {}", e),
        })?;

        let has_more = rows.len() as i32 > limit;
        let items = rows
            .into_iter()
            .take(limit as usize)
            .map(|r| GroupMemberInfo {
                member_id: r.id,
                user_id: r.user_id,
                nickname: r.nickname,
                role: r.role,
                last_active_at: r.last_active_at,
                joined_at: r.joined_at,
            })
            .collect::<Vec<_>>();

        let next_cursor = if has_more {
            items.last().map(|m| m.member_id)
        } else {
            None
        };

        Ok((items, has_more, next_cursor))
    }

    async fn find_user_groups(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Group>> {
        let rows = sqlx::query!(
            r#"
            SELECT g.id as "id!", g.group_id as "group_id!", g.name as "name!", 
                   g.description, g.owner_id as "owner_id!", g.latitude as "latitude!", 
                   g.longitude as "longitude!", g.location_name as "location_name!", 
                   g.member_count as "member_count!", g.max_members as "max_members!", 
                   g.is_public as "is_public!", g.created_at as "created_at!", 
                   g.updated_at as "updated_at!", g.last_active_at as "last_active_at!"
            FROM groups g
            INNER JOIN group_members gm ON g.group_id = gm.group_id
            WHERE gm.user_id = $1 AND ($2 = 0 OR g.id < $2)
            ORDER BY g.id DESC
            LIMIT $3
            "#,
            user_id,
            cursor as i64,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        let groups = rows.into_iter().map(|r| Group {
            id: r.id,
            group_id: r.group_id.to_string(),
            name: r.name,
            description: r.description,
            owner_id: r.owner_id,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            member_count: r.member_count,
            max_members: r.max_members,
            is_public: r.is_public,
            password_hash: None, // 不返回密码哈希
            created_at: r.created_at,
            updated_at: r.updated_at,
            last_active_at: r.last_active_at,
        }).collect();

        Ok(groups)
    }

    async fn find_owned_groups(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Group>> {
        let rows = sqlx::query!(
            r#"
            SELECT id as "id!", group_id as "group_id!", name as "name!", 
                   description, owner_id as "owner_id!", latitude as "latitude!", 
                   longitude as "longitude!", location_name as "location_name!", 
                   member_count as "member_count!", max_members as "max_members!", 
                   is_public as "is_public!", created_at as "created_at!", 
                   updated_at as "updated_at!", last_active_at as "last_active_at!"
            FROM groups
            WHERE owner_id = $1 AND ($2 = 0 OR id < $2)
            ORDER BY id DESC
            LIMIT $3
            "#,
            user_id,
            cursor as i64,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation { rule: format!("Database error: {}", e) })?;

        let groups = rows.into_iter().map(|r| Group {
            id: r.id,
            group_id: r.group_id.to_string(),
            name: r.name,
            description: r.description,
            owner_id: r.owner_id,
            latitude: r.latitude,
            longitude: r.longitude,
            location_name: r.location_name,
            member_count: r.member_count,
            max_members: r.max_members,
            is_public: r.is_public,
            password_hash: None, // 不返回密码哈希
            created_at: r.created_at,
            updated_at: r.updated_at,
            last_active_at: r.last_active_at,
        }).collect();

        Ok(groups)
    }
}
