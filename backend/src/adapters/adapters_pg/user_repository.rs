use crate::domain::{DomainError, DomainResult, User, UserDomainError, UserType};
// 移除未使用的导入
use crate::ports::UserRepositoryPort;
/// 用户仓储PostgreSQL适配器
///
/// 实现用户数据的PostgreSQL存储和查询
use async_trait::async_trait;
use sqlx::PgPool;


/// PostgreSQL用户仓储实现
#[derive(Clone)]
pub struct PgUserRepository {
    pool: PgPool,
}

impl PgUserRepository {
    /// 创建新的PostgreSQL用户仓储
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl UserRepositoryPort for PgUserRepository {
    async fn find_by_id(&self, user_id: i64) -> DomainResult<Option<User>> {
        let user_row = sqlx::query!(
            r#"
            SELECT id, user_id, login_id, nickname, password_hash, expires_at, 
                   created_at, updated_at, last_active_at, email, avatar_url
            FROM users
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e)
        })?;

        if let Some(user_row) = user_row {
            // 判断用户类型：有expires_at的是临时用户
            let user_type = if user_row.expires_at.is_some() {
                UserType::Temporary
            } else {
                UserType::Regular
            };

            Ok(Some(User {
                id: user_row.id,
                user_id: user_row.user_id,
                login_id: user_row.login_id,
                nickname: user_row.nickname,
                password_hash: user_row.password_hash,
                user_type,
                email: user_row.email,
                avatar_url: user_row.avatar_url,
                expires_at: user_row.expires_at,
                created_at: user_row.created_at,
                updated_at: user_row.updated_at,
                last_active_at: user_row.last_active_at,
            }))
        } else {
            Ok(None)
        }
    }

    async fn find_by_login_id(&self, login_id: &str) -> DomainResult<Option<User>> {
        let user_row = sqlx::query!(
            r#"
            SELECT id, user_id, login_id, nickname, password_hash, expires_at, 
                   created_at, updated_at, last_active_at, email, avatar_url
            FROM users
            WHERE login_id = $1
            "#,
            login_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e)
        })?;

        if let Some(user_row) = user_row {
            // 判断用户类型：有expires_at的是临时用户
            let user_type = if user_row.expires_at.is_some() {
                UserType::Temporary
            } else {
                UserType::Regular
            };

            Ok(Some(User {
                id: user_row.id,
                user_id: user_row.user_id,
                login_id: user_row.login_id,
                nickname: user_row.nickname,
                password_hash: user_row.password_hash,
                user_type,
                email: user_row.email,
                avatar_url: user_row.avatar_url,
                expires_at: user_row.expires_at,
                created_at: user_row.created_at,
                updated_at: user_row.updated_at,
                last_active_at: user_row.last_active_at,
            }))
        } else {
            Ok(None)
        }
    }

    async fn create_user(&self, user: &User) -> DomainResult<User> {
        // 开始事务
        let mut tx = self
            .pool
            .begin()
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Failed to start transaction: {}", e),
            })?;

        // 插入用户记录
        let user_row = sqlx::query!(
            r#"
            INSERT INTO users (user_id, login_id, nickname, password_hash, expires_at, 
                             email, avatar_url, created_at, updated_at, last_active_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
            RETURNING id, user_id, login_id, nickname, password_hash, expires_at, 
                     email, avatar_url, created_at, updated_at, last_active_at
            "#,
            user.user_id,
            user.login_id,
            user.nickname,
            user.password_hash,
            user.expires_at,
            user.email,
            user.avatar_url,
            user.created_at,
            user.updated_at,
            user.last_active_at
        )
        .fetch_one(&mut *tx)
        .await
        .map_err(|e| {
            // 处理唯一约束违反错误
            if let sqlx::Error::Database(db_err) = &e {
                if db_err.constraint().is_some() {
                    return DomainError::User(UserDomainError::LoginIdAlreadyExists {
                        login_id: user.login_id.clone().unwrap_or_default(),
                    });
                }
            }
            DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e)
            }
        })?;

        // 提交事务
        tx.commit()
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Failed to commit transaction: {}", e),
            })?;

        // 返回创建的用户
        Ok(User {
            id: user_row.id,
            user_id: user_row.user_id,
            login_id: user_row.login_id,
            nickname: user_row.nickname,
            password_hash: user_row.password_hash,
            user_type: user.user_type.clone(),
            email: user_row.email,
            avatar_url: user_row.avatar_url,
            expires_at: user_row.expires_at,
            created_at: user_row.created_at,
            updated_at: user_row.updated_at,
            last_active_at: user_row.last_active_at,
        })
    }

    async fn update_user(&self, user: &User) -> DomainResult<User> {
        // 开始事务
        let mut tx = self
            .pool
            .begin()
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Failed to start transaction: {}", e),
            })?;

        // 更新用户记录
        let user_row = sqlx::query!(
            r#"
            UPDATE users
            SET login_id = $2, nickname = $3, password_hash = $4, email = $5, 
                avatar_url = $6, expires_at = $7, updated_at = $8, last_active_at = $9
            WHERE user_id = $1
            RETURNING id, user_id, login_id, nickname, password_hash, expires_at, 
                     email, avatar_url, created_at, updated_at, last_active_at
            "#,
            user.user_id,
            user.login_id,
            user.nickname,
            user.password_hash,
            user.email,
            user.avatar_url,
            user.expires_at,
            user.updated_at,
            user.last_active_at
        )
        .fetch_optional(&mut *tx)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e)
        })?;

        if let Some(user_row) = user_row {


            // 提交事务
            tx.commit()
                .await
                .map_err(|e| DomainError::BusinessRuleViolation {
                    rule: format!("Failed to commit transaction: {}", e),
                })?;

            // 判断用户类型
            let user_type = if user_row.expires_at.is_some() {
                UserType::Temporary
            } else {
                UserType::Regular
            };

            Ok(User {
                id: user_row.id,
                user_id: user_row.user_id,
                login_id: user_row.login_id,
                nickname: user_row.nickname,
                password_hash: user_row.password_hash,
                user_type,
                email: user_row.email,
                avatar_url: user_row.avatar_url,
                expires_at: user_row.expires_at,
                created_at: user_row.created_at,
                updated_at: user_row.updated_at,
                last_active_at: user_row.last_active_at,
            })
        } else {
            Err(DomainError::User(UserDomainError::UserNotFound {
                user_id: user.user_id,
            }))
        }
    }

    async fn delete_user(&self, user_id: i64) -> DomainResult<()> {
        let rows_affected = sqlx::query!("DELETE FROM users WHERE user_id = $1", user_id)
            .execute(&self.pool)
            .await
            .map_err(|e| DomainError::BusinessRuleViolation {
                rule: format!("Database error: {}", e),
            })?
            .rows_affected();

        if rows_affected == 0 {
            Err(DomainError::User(UserDomainError::UserNotFound { user_id }))
        } else {
            Ok(())
        }
    }

    async fn find_nearby_users(
        &self,
        latitude: f64,
        longitude: f64,
        radius_km: f64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<User>> {
        // 使用PostGIS扩展进行地理位置查询
        let rows = sqlx::query!(
            r#"
            SELECT u.id, u.user_id, u.login_id, u.nickname, u.password_hash, u.expires_at,
                   u.email, u.avatar_url, u.created_at, u.updated_at, u.last_active_at,
                   ST_Distance(
                       ST_Point(COALESCE(ul.longitude, 0.0), COALESCE(ul.latitude, 0.0))::geography,
                       ST_Point($2, $1)::geography
                   ) / 1000.0 as distance_km
            FROM users u
            LEFT JOIN user_locations ul ON u.user_id = ul.user_id
            WHERE ul.latitude IS NOT NULL
              AND ul.longitude IS NOT NULL
              AND ($4 = 0 OR u.user_id < $4)
              AND ST_DWithin(
                  ST_Point(COALESCE(ul.longitude, 0.0), COALESCE(ul.latitude, 0.0))::geography,
                  ST_Point($2, $1)::geography,
                  $3 * 1000
              )
            ORDER BY u.user_id DESC
            LIMIT $5
            "#,
            latitude,
            longitude,
            radius_km as i32, // 类型转换
            cursor as i32,
            limit as i64 // 类型转换
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let users = rows
            .into_iter()
            .map(|r| {
                let user_type = if r.expires_at.is_some() {
                    UserType::Temporary
                } else {
                    UserType::Regular
                };

                User {
                    id: r.id,
                    user_id: r.user_id,
                    login_id: r.login_id,
                    nickname: r.nickname,
                    password_hash: r.password_hash,
                    user_type,
                    email: r.email,
                    avatar_url: r.avatar_url,
                    expires_at: r.expires_at,
                    created_at: r.created_at,
                    updated_at: r.updated_at,
                    last_active_at: r.last_active_at,
                }
            })
            .collect();

        Ok(users)
    }

    async fn search_users_by_nickname(
        &self,
        keyword: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<User>> {
        let search_pattern = format!("%{}%", keyword);

        let user_rows = sqlx::query!(
            r#"
            SELECT id, user_id, login_id, nickname, password_hash, expires_at, 
                   email, avatar_url, created_at, updated_at, last_active_at
            FROM users
            WHERE nickname ILIKE $1
              AND ($2 = 0 OR user_id < $2)
            ORDER BY user_id DESC
            LIMIT $3
            "#,
            search_pattern,
            (cursor as i32),
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e)
        })?;

        let users = user_rows
            .into_iter()
            .map(|user_row| {
                let user_type = if user_row.expires_at.is_some() {
                    UserType::Temporary
                } else {
                    UserType::Regular
                };

                User {
                    id: user_row.id,
                    user_id: user_row.user_id,
                    login_id: user_row.login_id,
                    nickname: user_row.nickname,
                    password_hash: user_row.password_hash,
                    user_type,
                    email: user_row.email,
                    avatar_url: user_row.avatar_url,
                    expires_at: user_row.expires_at,
                    created_at: user_row.created_at,
                    updated_at: user_row.updated_at,
                    last_active_at: user_row.last_active_at,
                }
            })
            .collect();

        Ok(users)
    }

    async fn get_total_user_count(&self) -> DomainResult<i64> {
        let count = sqlx::query_scalar!(
            r#"
            SELECT COUNT(*) FROM users
            "#
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;
        Ok(count.unwrap_or(0))
    }

    async fn get_user_count_by_id_range(&self, start_id: i64, end_id: i64) -> DomainResult<i64> {
        let count = sqlx::query_scalar!(
            r#"
            SELECT COUNT(*) FROM users WHERE user_id BETWEEN $1 AND $2
            "#,
            start_id,
            end_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;
        Ok(count.unwrap_or(0))
    }
    async fn get_max_user_id_by_range(
        &self,
        start_id: i64,
        end_id: i64,
    ) -> DomainResult<Option<i64>> {
        let max_id = sqlx::query_scalar!(
            r#"
            SELECT MAX(user_id) FROM users WHERE user_id BETWEEN $1 AND $2
            "#,
            start_id,
            end_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;
        Ok(max_id)
    }

    async fn find_user_location(&self, user_id: i64) -> DomainResult<Option<crate::domain::UserLocation>> {
        let location_row = sqlx::query!(
            r#"
            SELECT id as "id!", user_id as "user_id!", latitude as "latitude!", 
                   longitude as "longitude!", location_name as "location_name!", updated_at as "updated_at!"
            FROM user_locations
            WHERE user_id = $1
            ORDER BY updated_at DESC
            LIMIT 1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        if let Some(row) = location_row {
            Ok(Some(crate::domain::UserLocation {
                id: row.id,
                user_id: row.user_id,
                latitude: row.latitude,
                longitude: row.longitude,
                location_name: row.location_name,
                updated_at: row.updated_at,
            }))
        } else {
            Ok(None)
        }
    }

    async fn upsert_user_location(&self, user_location: &crate::domain::UserLocation) -> DomainResult<crate::domain::UserLocation> {
        let location_row = sqlx::query!(
            r#"
            INSERT INTO user_locations (user_id, latitude, longitude, location_name, updated_at)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (user_id) 
            DO UPDATE SET 
                latitude = EXCLUDED.latitude,
                longitude = EXCLUDED.longitude,
                location_name = EXCLUDED.location_name,
                updated_at = EXCLUDED.updated_at
            RETURNING id as "id!", user_id as "user_id!", latitude as "latitude!", 
                      longitude as "longitude!", location_name as "location_name!", updated_at as "updated_at!"
            "#,
            user_location.user_id,
            user_location.latitude,
            user_location.longitude,
            user_location.location_name,
            user_location.updated_at
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        Ok(crate::domain::UserLocation {
            id: location_row.id,
            user_id: location_row.user_id,
            latitude: location_row.latitude,
            longitude: location_row.longitude,
            location_name: location_row.location_name,
            updated_at: location_row.updated_at,
        })
    }

    async fn find_nearby_users_with_distance(
        &self,
        latitude: f64,
        longitude: f64,
        radius_km: f64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<crate::application::NearbyUser>> {
        let rows = sqlx::query!(
            r#"
            SELECT u.user_id as "user_id!", u.nickname as "nickname!", u.created_at as "created_at!",
                   ul.latitude as "latitude!", ul.longitude as "longitude!", ul.location_name as "location_name!",
                   ST_Distance(
                       ST_Point(ul.longitude, ul.latitude)::geography,
                       ST_Point($2, $1)::geography
                   ) / 1000.0 as "distance_km!"
            FROM users u
            INNER JOIN user_locations ul ON u.user_id = ul.user_id
            WHERE ($4 = 0 OR u.user_id < $4)
              AND ST_DWithin(
                  ST_Point(ul.longitude, ul.latitude)::geography,
                  ST_Point($2, $1)::geography,
                  $3 * 1000
              )
            ORDER BY u.user_id DESC
            LIMIT $5
            "#,
            latitude,
            longitude,
            radius_km as f32,
            cursor as i32,
            limit as i64
        )
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::BusinessRuleViolation {
            rule: format!("Database error: {}", e),
        })?;

        let nearby_users = rows
            .into_iter()
            .map(|r| crate::application::NearbyUser {
                user_id: r.user_id,
                nickname: r.nickname,
                location: crate::application::UserLocation {
                    latitude: r.latitude,
                    longitude: r.longitude,
                    address: Some(r.location_name),
                },
                distance_km: r.distance_km,
                created_at: r.created_at,
            })
            .collect();

        Ok(nearby_users)
    }
}


