//! 用户相关数据仓库实现

use sqlx::PgPool;
use tracing::error;
use chrono::{DateTime, Utc, NaiveTime};

use crate::repositories::{Repository, RepositoryError, RepositoryResult};
use crate::utils::constants::{DEFAULT_LATITUDE, DEFAULT_LONGITUDE, DEFAULT_LOCATION_NAME};
use crate::utils::{UserIdPartition, UserType};
use crate::models::user_settings::{UserSettings, NotificationLevel};

/// 用户基本信息查询结果
#[derive(Debug, Clone)]
pub struct UserInfoRow {
    pub id: i64,
    pub nickname: String,
    #[allow(dead_code)]
    pub created_at: DateTime<Utc>,
}

/// 用户仓库接口
pub trait UserRepository: Repository {
    async fn get_user_info(&self, user_id: i64) -> RepositoryResult<UserInfoRow>;
    async fn get_auth_info(&self, user_id: i64) -> RepositoryResult<crate::services::user::AuthUserRow>;
    async fn find_user_friends(&self, user_id: i64) -> RepositoryResult<Vec<UserInfoRow>>;
    async fn is_login_id_taken(&self, login_id: &str) -> RepositoryResult<bool>;
    async fn is_password_set(&self, user_id: i64) -> RepositoryResult<bool>;
    async fn hash_password(&self, password: &str) -> RepositoryResult<String>;
    #[allow(dead_code)]
    async fn update_user_presence(&self, user_id: i64, latitude: f64, longitude: f64, location_name: &str) -> RepositoryResult<()>;
    /// 更新用户位置
    async fn update_user_location(
        &self, 
        user_id: i64, 
        latitude: f64, 
        longitude: f64, 
        location_name: &str
    ) -> RepositoryResult<()>;
    
    /// 获取用户登录信息
    async fn get_user_login_info(&self, login_id: &str) -> RepositoryResult<(i64, String)>;
    
    /// 获取正式用户密码哈希（正式用户必须有密码）
    async fn get_user_password_hash(&self, user_id: i64) -> RepositoryResult<String>;
    
    /// 获取密码哈希（通过login_id，可能返回空）
    async fn get_password_hash(&self, login_id: &str) -> RepositoryResult<Option<String>>;
    
    /// 更新用户最后活跃时间
    async fn update_last_active(&self, login_id: &str) -> RepositoryResult<()>;
    
    /// 判断用户是否为临时用户
    async fn is_temp_user(&self, user_id: i64) -> RepositoryResult<bool>;
    
    /// 验证密码
    async fn verify_password(&self, hashed_password: &str, password: &str) -> RepositoryResult<bool>;
    
    /// 更新用户密码
    async fn update_user_password(&self, user_id: i64, password_hash: &str) -> RepositoryResult<()>;
    
    /// 获取用户昵称
    async fn get_user_nickname(&self, user_id: i64) -> RepositoryResult<String>;
    /// 获取用户当前位置信息（纬度，经度）
    async fn get_user_location(&self, user_id: i64) -> RepositoryResult<Option<(f64, f64)>>;
    
    /// 创建新用户并返回用户ID（正式用户必须有密码）
    async fn create_user(
        &self, 
        login_id: &str, 
        nickname: &str, 
        password_hash: &str
    ) -> RepositoryResult<i64>;
    
    /// 创建临时用户并返回用户ID
    async fn create_temp_user(
        &self, 
        temp_login_id: &str, 
        nickname: &str
    ) -> RepositoryResult<i64>;
    
    /// 设置用户初始位置
    async fn set_initial_user_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str
    ) -> RepositoryResult<()>;

    /// 根据ID获取用户信息
    async fn get_user_by_id(
        &self,
        user_id: i64,
        current_user_id: i64,
    ) -> RepositoryResult<crate::services::user::UserRow>;
    
    /// 查找附近的用户
    async fn find_nearby_users(
        &self,
        longitude: f64,
        latitude: f64,
        radius: f64,
        cursor: i64,
        limit: i64,
        current_user_id: i64,
    ) -> RepositoryResult<(Vec<crate::services::user::UserRow>, bool, Option<i64>)>;
    
    /// 根据昵称搜索用户
    async fn find_users_by_name(
        &self,
        nickname: &str,
        cursor: i64,
        limit: i64,
        current_user_id: i64,
    ) -> RepositoryResult<(Vec<crate::services::user::UserRow>, bool, Option<i64>)>;
    
    /// 用户统计相关方法
    async fn get_total_user_count(&self) -> RepositoryResult<i64>;
    async fn get_user_count_by_type(&self, user_type: UserType) -> RepositoryResult<i64>;
    #[allow(dead_code)]
    async fn get_user_count_by_id_range(&self, start_id: i64, end_id: i64) -> RepositoryResult<i64>;
    
    /// 获取群组成员ID列表（用于通知系统）
    #[allow(dead_code)]
    async fn get_group_member_ids(&self, group_id: uuid::Uuid) -> RepositoryResult<Vec<i64>>;
    
    // =============== 用户资料和设置管理 ===============
    
    /// 更新用户昵称
    async fn update_nickname(&self, user_id: i64, nickname: &str) -> RepositoryResult<()>;
    
    /// 升级临时用户为正式用户，返回新的用户ID
    async fn upgrade_temp_user(&self, user_id: i64, login_id: &str, password_hash: &str) -> RepositoryResult<i64>;
    
    /// 获取用户设置
    async fn get_user_settings(&self, user_id: i64) -> RepositoryResult<crate::models::user_settings::UserSettings>;
    
    /// 更新用户设置
    async fn update_user_settings(&self, user_id: i64, settings: crate::models::api::user::UserSettingsRequest) -> RepositoryResult<()>;
    
    /// 删除用户账户
    async fn delete_user(&self, user_id: i64) -> RepositoryResult<()>;
}

/// PostgreSQL实现的用户仓库
#[derive(Debug)]
pub struct PgUserRepository {
    /// 数据库连接池
    pool: PgPool,
}

impl PgUserRepository {
    /// 创建新的PostgreSQL用户仓库
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

impl Repository for PgUserRepository {
    fn name(&self) -> &str {
        "PgUserRepository"
    }
}

impl PgUserRepository {
    /// 内部方法：为用户创建默认设置
    async fn create_user_settings_internal(&self, user_id: i64) -> RepositoryResult<()> {
        sqlx::query!(
            r#"
            INSERT INTO user_settings (
                user_id, 
                notification_level,
                message_notifications,
                group_notifications,
                system_notifications,
                location_notifications,
                quiet_hours_enabled,
                quiet_hours_start,
                quiet_hours_end,
                notification_radius_km,
                sound_enabled,
                vibration_enabled
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
            ON CONFLICT (user_id) DO NOTHING
            "#,
            user_id,
            NotificationLevel::All as NotificationLevel,
            true,
            true,
            true,
            true,
            false,
            NaiveTime::from_hms_opt(22, 0, 0),
            NaiveTime::from_hms_opt(8, 0, 0),
            5.0,
            true,
            true
        )
        .execute(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;
        
        Ok(())
    }
}

impl UserRepository for PgUserRepository {
    async fn get_user_info(&self, user_id: i64) -> RepositoryResult<UserInfoRow> {
        #[derive(sqlx::FromRow)]
        struct UserInfoQuery {
            id: i64,
            nickname: String,
            created_at: chrono::DateTime<Utc>,
        }
        
        let row = sqlx::query_as!(
            UserInfoQuery,
            "SELECT id, nickname, created_at
             FROM users
             WHERE id = $1",
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| match e {
            sqlx::Error::RowNotFound => RepositoryError::NotFound,
            err => RepositoryError::Database(err)
        })?;
        
        Ok(UserInfoRow {
            id: row.id,
            nickname: row.nickname,
            created_at: row.created_at,
        })
    }
    
    async fn get_auth_info(&self, user_id: i64) -> RepositoryResult<crate::services::user::AuthUserRow> {
        
        let user = sqlx::query_as!(
            crate::services::user::AuthUserRow,
            r#"
            SELECT 
                user_id as "user_id!",
                nickname as "nickname!"
            FROM users
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(RepositoryError::Database)?
        .ok_or(RepositoryError::NotFound)?;

        Ok(user)
    }
    
    async fn find_user_friends(&self, _user_id: i64) -> RepositoryResult<Vec<UserInfoRow>> {
        // 简化：好友功能暂未实现，返回空列表
        Ok(vec![])
    }
    
    async fn is_login_id_taken(&self, login_id: &str) -> RepositoryResult<bool> {
        let exists = sqlx::query_scalar!(
            "SELECT EXISTS(SELECT 1 FROM users WHERE login_id = $1)",
            login_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        Ok(exists.unwrap_or(false))
    }

    async fn is_password_set(&self, user_id: i64) -> RepositoryResult<bool> {
        #[derive(sqlx::FromRow)]
        struct PasswordSetRow {
            has_password: Option<bool>,
        }
        
        let result = sqlx::query_as!(
            PasswordSetRow,
            r#"SELECT password_hash IS NOT NULL AS has_password FROM users WHERE user_id = $1"#,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| match e {
            sqlx::Error::RowNotFound => RepositoryError::NotFound,
            err => RepositoryError::Database(err),
        })?;
        
        Ok(result.has_password.unwrap_or(false))
    }
    
    async fn hash_password(&self, password: &str) -> RepositoryResult<String> {
        use argon2::{
            Argon2, PasswordHasher,
            password_hash::{SaltString, rand_core::OsRng},
        };
        
        let salt = SaltString::generate(&mut OsRng);
        let password_hash = Argon2::default()
            .hash_password(password.as_bytes(), &salt)
            .map_err(|e| {
                error!("密码加密失败: {:?}", e);
                RepositoryError::Other("密码处理失败".to_string())
            })?
            .to_string();

        Ok(password_hash)
    }

    async fn update_user_presence(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str,
    ) -> RepositoryResult<()> {
        // 开启事务
        let mut tx = self.pool.begin().await.map_err(RepositoryError::Database)?;

        // 更新用户最后活跃时间
        sqlx::query!(
            r#"
            UPDATE users 
            SET last_active_at = $1
            WHERE user_id = $2
            "#,
            Utc::now(),
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;

        // 更新用户位置
        sqlx::query!(
            r#"
            INSERT INTO user_locations (user_id, latitude, longitude, location_name, updated_at)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (user_id) DO UPDATE
            SET latitude = EXCLUDED.latitude,
                longitude = EXCLUDED.longitude,
                location_name = EXCLUDED.location_name,
                updated_at = EXCLUDED.updated_at
            "#,
            user_id,
            latitude,
            longitude,
            location_name,
            Utc::now()
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;

        // 提交事务
        tx.commit().await.map_err(RepositoryError::Database)?;

        Ok(())
    }
    
    async fn update_user_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str,
    ) -> RepositoryResult<()> {
        sqlx::query!(
            r#"
            INSERT INTO user_locations (user_id, latitude, longitude, location_name, updated_at)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (user_id) DO UPDATE
            SET latitude = EXCLUDED.latitude,
                longitude = EXCLUDED.longitude,
                location_name = EXCLUDED.location_name,
                updated_at = EXCLUDED.updated_at
            "#,
            user_id,
            latitude,
            longitude,
            location_name,
            Utc::now()
        )
        .execute(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        Ok(())
    }

    async fn get_user_login_info(
        &self,
        login_id: &str,
    ) -> RepositoryResult<(i64, String)> {
        #[derive(sqlx::FromRow)]
        struct UserLoginRow {
            user_id: i64,
            nickname: String,
        }

        let user = sqlx::query_as!(
            UserLoginRow,
            r#"
            SELECT user_id as "user_id!", nickname as "nickname!"
            FROM users
            WHERE login_id = $1
            "#,
            login_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| match e {
            sqlx::Error::RowNotFound => RepositoryError::NotFound,
            err => RepositoryError::Database(err),
        })?;

        Ok((user.user_id, user.nickname))
    }
    
    async fn get_password_hash(&self, login_id: &str) -> RepositoryResult<Option<String>> {
        let password_hash = sqlx::query_scalar!(
            r#"
            SELECT password_hash
            FROM users
            WHERE login_id = $1
            "#,
            login_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| match e {
            sqlx::Error::RowNotFound => RepositoryError::NotFound,
            err => RepositoryError::Database(err),
        })?;

        Ok(password_hash)
    }

    async fn update_last_active(&self, login_id: &str) -> RepositoryResult<()> {
        sqlx::query!(
            r#"
            UPDATE users
            SET last_active_at = $1
            WHERE login_id = $2
            "#,
            Utc::now(),
            login_id
        )
        .execute(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        Ok(())
    }

    async fn is_temp_user(&self, user_id: i64) -> RepositoryResult<bool> {
        // 使用新的ID分区系统判断用户类型
        Ok(UserIdPartition::is_user_type(user_id, UserType::Temporary))
    }
    
    async fn verify_password(&self, hashed_password: &str, password: &str) -> RepositoryResult<bool> {
        use argon2::{
            Argon2, PasswordHash, PasswordVerifier,
        };
        
        let parsed_hash = PasswordHash::new(hashed_password)
            .map_err(|_| RepositoryError::Other("密码哈希格式错误".to_string()))?;
            
        let is_valid = Argon2::default()
            .verify_password(password.as_bytes(), &parsed_hash)
            .map_or(false, |_| true);
            
        Ok(is_valid)
    }
    
    async fn update_user_password(&self, user_id: i64, password_hash: &str) -> RepositoryResult<()> {
        sqlx::query!(
            r#"UPDATE users SET password_hash = $1, updated_at = $2 WHERE user_id = $3"#,
            password_hash,
            Utc::now(),
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;
        
        Ok(())
    }
    
    async fn get_user_nickname(&self, user_id: i64) -> RepositoryResult<String> {
        let nickname = sqlx::query_scalar!(
            r#"
            SELECT nickname
            FROM users
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(RepositoryError::Database)?
        .ok_or(RepositoryError::NotFound)?;
        
        Ok(nickname)
    }
    
    async fn get_user_location(&self, user_id: i64) -> RepositoryResult<Option<(f64, f64)>> {
        #[derive(sqlx::FromRow)]
        struct LocRow { latitude: Option<f64>, longitude: Option<f64> }
        let row = sqlx::query_as!(
            LocRow,
            r#"
            SELECT latitude, longitude
            FROM user_locations
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;
        Ok(row.and_then(|r| match (r.latitude, r.longitude) { (Some(lat), Some(lon)) => Some((lat, lon)), _ => None }))
    }
    
    /// 创建新用户并返回用户ID（正式用户必须有密码）
    async fn create_user(
        &self, 
        login_id: &str, 
        nickname: &str, 
        password_hash: &str
    ) -> RepositoryResult<i64> {
        // 开启事务
        let mut tx = self.pool.begin().await.map_err(RepositoryError::Database)?;
        
        // 获取正式用户类型的当前最大ID
        let (min_id, max_id) = UserIdPartition::get_type_range(UserType::Regular);
        let current_max = sqlx::query_scalar!(
            r#"
            SELECT COALESCE(MAX(user_id), $1) 
            FROM users 
            WHERE user_id BETWEEN $1 AND $2
            "#,
            min_id,
            max_id
        )
        .fetch_one(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?
        .unwrap_or(min_id);
        
        // 获取下一个可用的序列号
        let next_sequence = UserIdPartition::next_sequence_for_type(UserType::Regular, current_max)
            .map_err(|e| RepositoryError::Other(e))?;
        
        // 生成新的用户ID
        let user_id = UserIdPartition::generate_user_id(UserType::Regular, next_sequence)
            .map_err(|e| RepositoryError::Other(e))?;
        
        // 创建正式用户
        sqlx::query!(
            r#"
            INSERT INTO users (user_id, login_id, nickname, password_hash, created_at, updated_at, last_active_at)
            VALUES ($1, $2, $3, $4, $5, $5, $5)
            "#,
            user_id,
            login_id,
            nickname,
            password_hash,
            Utc::now()
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 提交事务
        tx.commit().await.map_err(RepositoryError::Database)?;
        
        Ok(user_id)
    }
    
    /// 创建临时用户并返回用户ID
    async fn create_temp_user(
        &self, 
        temp_login_id: &str, 
        nickname: &str
    ) -> RepositoryResult<i64> {
        // 开启事务
        let mut tx = self.pool.begin().await.map_err(RepositoryError::Database)?;
        
        // 获取临时用户类型的当前最大ID
        let (min_id, max_id) = UserIdPartition::get_type_range(UserType::Temporary);
        let current_max = sqlx::query_scalar!(
            r#"
            SELECT COALESCE(MAX(user_id), $1) 
            FROM users 
            WHERE user_id BETWEEN $1 AND $2
            "#,
            min_id,
            max_id
        )
        .fetch_one(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?
        .unwrap_or(min_id);
        
        // 获取下一个可用的序列号
        let next_sequence = UserIdPartition::next_sequence_for_type(UserType::Temporary, current_max)
            .map_err(|e| RepositoryError::Other(e))?;
        
        // 生成新的用户ID
        let user_id = UserIdPartition::generate_user_id(UserType::Temporary, next_sequence)
            .map_err(|e| RepositoryError::Other(e))?;
        
        // 创建临时用户（无密码）
        sqlx::query!(
            r#"
            INSERT INTO users (user_id, login_id, nickname, created_at, updated_at, last_active_at)
            VALUES ($1, $2, $3, $4, $4, $4)
            "#,
            user_id,
            temp_login_id,
            nickname,
            Utc::now()
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 提交事务
        tx.commit().await.map_err(RepositoryError::Database)?;
        
        Ok(user_id)
    }
    
    /// 设置用户初始位置
    async fn set_initial_user_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str
    ) -> RepositoryResult<()> {
        sqlx::query!(
            r#"
            INSERT INTO user_locations (user_id, latitude, longitude, location_name, updated_at)
            VALUES ($1, $2, $3, $4, $5)
            "#,
            user_id,
            latitude,
            longitude,
            location_name,
            Utc::now()
        )
        .execute(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;
        
        Ok(())
    }

    async fn get_user_by_id(
        &self,
        user_id: i64,
        current_user_id: i64,
    ) -> RepositoryResult<crate::services::user::UserRow> {
        
        let user = sqlx::query_as!(
            crate::services::user::UserRow,
            r#"
            SELECT 
                u.user_id,
                u.nickname as "nickname!",
                u.last_active_at as last_active,
                COALESCE(ul.latitude, $3) as "latitude!: f64",
                COALESCE(ul.longitude, $4) as "longitude!: f64",
                COALESCE(ul.location_name, $5) as "location_name!: String",
                COALESCE(
                    ST_Distance(
                        ST_MakePoint(ul.longitude, ul.latitude)::geography,
                        ST_MakePoint(COALESCE(ul2.longitude, $4), COALESCE(ul2.latitude, $3))::geography
                    ),
                    0.0
                ) as "distance!: f64",
                u.last_active_at as "last_active_at!"
            FROM users u
            LEFT JOIN user_locations ul ON u.user_id = ul.user_id
            LEFT JOIN user_locations ul2 ON ul2.user_id = $1
            WHERE u.user_id = $2
            "#,
            current_user_id,
            user_id,
            DEFAULT_LATITUDE,
            DEFAULT_LONGITUDE,
            DEFAULT_LOCATION_NAME
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| match e {
            sqlx::Error::RowNotFound => RepositoryError::NotFound,
            err => RepositoryError::Database(err),
        })?;

        Ok(user)
    }
    
    async fn find_nearby_users(
        &self,
        longitude: f64,
        latitude: f64,
        radius: f64,
        cursor: i64,
        limit: i64,
        current_user_id: i64,
    ) -> RepositoryResult<(Vec<crate::services::user::UserRow>, bool, Option<i64>)> {
        
        // 查询附近用户
        let user_rows = sqlx::query_as!(
            crate::services::user::UserRow,
            r#"
            SELECT 
                u.user_id,
                u.nickname as "nickname!",
                u.last_active_at as "last_active!",
                COALESCE(ul.latitude, $1) as "latitude!: f64",
                COALESCE(ul.longitude, $2) as "longitude!: f64",
                COALESCE(ul.location_name, '') as "location_name!: String",
                COALESCE(
                    ST_Distance(
                        ST_MakePoint(COALESCE(ul.longitude, $2), COALESCE(ul.latitude, $1))::geography,
                        ST_MakePoint($2, $1)::geography
                    ),
                    0.0
                ) as "distance!: f64",
                u.last_active_at as "last_active_at!"
            FROM users u
            LEFT JOIN user_locations ul ON u.user_id = ul.user_id
            WHERE 
                u.user_id != $4 AND
                ($5 = 0 OR u.user_id < $5) AND
                COALESCE(
                    ST_Distance(
                        ST_MakePoint(COALESCE(ul.longitude, $2), COALESCE(ul.latitude, $1))::geography,
                        ST_MakePoint($2, $1)::geography
                    ),
                    0.0
                ) <= $3
            ORDER BY u.user_id DESC
            LIMIT $6
            "#,
            latitude,
            longitude,
            radius,
            current_user_id,
            cursor as i32,
            (limit + 1) as i32
        )
        .fetch_all(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        // 检查是否还有更多记录
        let has_more = user_rows.len() > limit as usize;
        let next_cursor = if has_more && !user_rows.is_empty() {
            user_rows.last().map(|row| row.user_id)
        } else {
            None
        };

        // 移除额外查询的记录
        let result_rows = if has_more {
            user_rows[0..user_rows.len() - 1].to_vec()
        } else {
            user_rows
        };

        Ok((result_rows, has_more, next_cursor))
    }
    
    async fn find_users_by_name(
        &self,
        nickname: &str,
        cursor: i64,
        limit: i64,
        current_user_id: i64,
    ) -> RepositoryResult<(Vec<crate::services::user::UserRow>, bool, Option<i64>)> {
        
        // 添加通配符
        let search_pattern = format!("%{}%", nickname);
        
        // 查询符合条件的用户
        let user_rows = sqlx::query_as!(
            crate::services::user::UserRow,
            r#"
            SELECT 
                u.user_id,
                u.nickname as "nickname!",
                u.last_active_at as "last_active!",
                COALESCE(ul.latitude, $3) as "latitude!: f64",
                COALESCE(ul.longitude, $4) as "longitude!: f64",
                COALESCE(ul.location_name, $5) as "location_name!: String",
                COALESCE(
                    ST_Distance(
                        ST_MakePoint(ul.longitude, ul.latitude)::geography,
                        ST_MakePoint(COALESCE(ul2.longitude, $4), COALESCE(ul2.latitude, $3))::geography
                    ),
                    0.0
                ) as "distance!: f64",
                u.last_active_at as "last_active_at!"
            FROM users u
            LEFT JOIN user_locations ul ON u.user_id = ul.user_id
            LEFT JOIN user_locations ul2 ON ul2.user_id = $6
            WHERE 
                u.user_id != $6 AND
                ($7 = 0 OR u.user_id < $7) AND
                u.nickname ILIKE $1
            ORDER BY u.user_id DESC
            LIMIT $2
            "#,
            search_pattern,
            (limit + 1) as i32,
            DEFAULT_LATITUDE,
            DEFAULT_LONGITUDE,
            DEFAULT_LOCATION_NAME,
            current_user_id,
            cursor as i32
        )
        .fetch_all(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 检查是否还有更多记录
        let has_more = user_rows.len() > limit as usize;
        let next_cursor = if has_more && !user_rows.is_empty() {
            user_rows.last().map(|row| row.user_id)
        } else {
            None
        };
        
        // 移除额外查询的记录
        let result_rows = if has_more {
            user_rows[0..user_rows.len() - 1].to_vec()
        } else {
            user_rows
        };
        
        Ok((result_rows, has_more, next_cursor))
    }
    
    async fn get_user_password_hash(&self, user_id: i64) -> RepositoryResult<String> {
        let password_hash = sqlx::query_scalar!(
            r#"
            SELECT password_hash
            FROM users
            WHERE user_id = $1 AND password_hash IS NOT NULL
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(|e| match e {
            sqlx::Error::RowNotFound => RepositoryError::NotFound,
            err => RepositoryError::Database(err),
        })?
        .ok_or(RepositoryError::NotFound)?;

        Ok(password_hash)
    }

    async fn get_total_user_count(&self) -> RepositoryResult<i64> {
        let count = sqlx::query_scalar!("SELECT COUNT(*) FROM users")
            .fetch_one(&self.pool)
            .await
            .map_err(RepositoryError::Database)?;

        Ok(count.unwrap_or(0))
    }

    async fn get_user_count_by_type(&self, user_type: UserType) -> RepositoryResult<i64> {
        let (min_id, max_id) = UserIdPartition::get_type_range(user_type);
        
        let count = sqlx::query_scalar!(
            "SELECT COUNT(*) FROM users WHERE user_id BETWEEN $1 AND $2",
            min_id,
            max_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        Ok(count.unwrap_or(0))
    }

    async fn get_user_count_by_id_range(&self, start_id: i64, end_id: i64) -> RepositoryResult<i64> {
        let count = sqlx::query_scalar!(
            "SELECT COUNT(*) FROM users WHERE user_id BETWEEN $1 AND $2",
            start_id,
            end_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        Ok(count.unwrap_or(0))
    }
    
    async fn get_group_member_ids(&self, group_id: uuid::Uuid) -> RepositoryResult<Vec<i64>> {
        let member_ids = sqlx::query_scalar!(
            r#"
            SELECT ARRAY_AGG(user_id) 
            FROM group_members 
            WHERE group_id = $1
            "#,
            group_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(RepositoryError::Database)?
        .unwrap_or_else(|| vec![]);

        Ok(member_ids)
    }

    // =============== 用户资料和设置管理实现 ===============

    async fn update_nickname(&self, user_id: i64, nickname: &str) -> RepositoryResult<()> {
        sqlx::query!(
            "UPDATE users SET nickname = $1, updated_at = NOW() WHERE user_id = $2",
            nickname,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        Ok(())
    }

    async fn upgrade_temp_user(&self, user_id: i64, login_id: &str, password_hash: &str) -> RepositoryResult<i64> {
        // 开启事务
        let mut tx = self.pool.begin().await.map_err(RepositoryError::Database)?;
        
        // 1. 验证是否为临时用户
        let user_type = UserIdPartition::parse_user_id(user_id)
            .map_err(|e| RepositoryError::InvalidArgument(format!("无效的用户ID: {}", e)))?
            .0;
            
        if user_type != UserType::Temporary {
            return Err(RepositoryError::InvalidArgument("只能升级临时用户".to_string()));
        }
        
        // 2. 获取临时用户的当前信息
        let temp_user = sqlx::query!(
            "SELECT nickname, email, avatar_url, created_at FROM users WHERE user_id = $1",
            user_id
        )
        .fetch_one(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 3. 生成新的正式用户ID
        let (min_id, max_id) = UserIdPartition::get_type_range(UserType::Regular);
        let current_max = sqlx::query_scalar!(
            r#"
            SELECT COALESCE(MAX(user_id), $1) as "max_id!"
            FROM users 
            WHERE user_id BETWEEN $1 AND $2
            "#,
            min_id,
            max_id
        )
        .fetch_one(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        let new_user_id = if current_max < max_id {
            current_max + 1
        } else {
            return Err(RepositoryError::InvalidArgument("正式用户ID已达到上限".to_string()));
        };
        
        // 4. 创建新的正式用户记录
        sqlx::query!(
            r#"
            INSERT INTO users (user_id, login_id, nickname, password_hash, email, avatar_url, created_at, updated_at, last_active_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
            "#,
            new_user_id,
            login_id,
            temp_user.nickname,
            password_hash,
            temp_user.email,
            temp_user.avatar_url,
            temp_user.created_at
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 5. 迁移用户位置信息
        sqlx::query!(
            r#"
            UPDATE user_locations 
            SET user_id = $1 
            WHERE user_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 6. 迁移用户设置
        sqlx::query!(
            r#"
            UPDATE user_settings 
            SET user_id = $1 
            WHERE user_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 7. 迁移群组成员关系
        sqlx::query!(
            r#"
            UPDATE group_members 
            SET user_id = $1 
            WHERE user_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 8. 迁移私聊消息记录
        sqlx::query!(
            r#"
            UPDATE direct_messages 
            SET sender_id = $1 
            WHERE sender_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        sqlx::query!(
            r#"
            UPDATE direct_messages 
            SET recipient_id = $1 
            WHERE recipient_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 迁移群组消息记录
        sqlx::query!(
            r#"
            UPDATE group_messages 
            SET sender_id = $1 
            WHERE sender_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 9. 迁移动态记录
        sqlx::query!(
            r#"
            UPDATE moments 
            SET user_id = $1 
            WHERE user_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 10. 迁移好友关系
        sqlx::query!(
            r#"
            UPDATE friendships 
            SET requester_id = $1 
            WHERE requester_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        sqlx::query!(
            r#"
            UPDATE friendships 
            SET addressee_id = $1 
            WHERE addressee_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 11. 迁移会话记录
        sqlx::query!(
            r#"
            UPDATE conversations 
            SET user1_id = $1 
            WHERE user1_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        sqlx::query!(
            r#"
            UPDATE conversations 
            SET user2_id = $1 
            WHERE user2_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 12. 迁移消息已读状态
        sqlx::query!(
            r#"
            UPDATE user_message_read_status 
            SET user_id = $1 
            WHERE user_id = $2
            "#,
            new_user_id,
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 13. 删除旧的临时用户记录
        sqlx::query!(
            "DELETE FROM users WHERE user_id = $1",
            user_id
        )
        .execute(&mut *tx)
        .await
        .map_err(RepositoryError::Database)?;
        
        // 提交事务
        tx.commit().await.map_err(RepositoryError::Database)?;

        Ok(new_user_id)
    }

    async fn get_user_settings(&self, user_id: i64) -> RepositoryResult<crate::models::user_settings::UserSettings> {
        let settings = sqlx::query_as!(
            UserSettings,
            r#"
            SELECT 
                id,
                user_id,
                notification_level as "notification_level: NotificationLevel",
                message_notifications,
                group_notifications,
                system_notifications,
                location_notifications,
                quiet_hours_enabled,
                quiet_hours_start,
                quiet_hours_end,
                notification_radius_km,
                sound_enabled,
                vibration_enabled
            FROM user_settings 
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        match settings {
            Some(settings) => Ok(settings),
            None => {
                // 如果不存在设置，创建默认设置
                self.create_user_settings_internal(user_id).await?;
                // 再次查询返回新创建的设置
                sqlx::query_as!(
                    UserSettings,
                    r#"
                    SELECT 
                        id,
                        user_id,
                        notification_level as "notification_level: NotificationLevel",
                        message_notifications,
                        group_notifications,
                        system_notifications,
                        location_notifications,
                        quiet_hours_enabled,
                        quiet_hours_start,
                        quiet_hours_end,
                        notification_radius_km,
                        sound_enabled,
                        vibration_enabled
                    FROM user_settings 
                    WHERE user_id = $1
                    "#,
                    user_id
                )
                .fetch_one(&self.pool)
                .await
                .map_err(RepositoryError::Database)
            }
        }
    }

    async fn update_user_settings(&self, user_id: i64, settings: crate::models::api::user::UserSettingsRequest) -> RepositoryResult<()> {
        use crate::models::user_settings::NotificationLevel;
        use chrono::NaiveTime;

        // 将可选字符串级别映射为枚举的 Option
        let level_opt: Option<NotificationLevel> = settings.notification_level.as_deref().map(|level| match level {
            "all" => NotificationLevel::All,
            "mention" => NotificationLevel::Mention,
            "important" => NotificationLevel::Important,
            "muted" => NotificationLevel::Muted,
            _ => NotificationLevel::All,
        });

        // 解析时间字符串为 NaiveTime（支持 HH:MM 或 HH:MM:SS）
        let parse_time = |s: &str| {
            NaiveTime::parse_from_str(s, "%H:%M:%S")
                .or_else(|_| NaiveTime::parse_from_str(s, "%H:%M"))
                .ok()
        };
        let quiet_start_opt: Option<NaiveTime> = settings
            .quiet_hours_start
            .as_deref()
            .and_then(parse_time);
        let quiet_end_opt: Option<NaiveTime> = settings
            .quiet_hours_end
            .as_deref()
            .and_then(parse_time);

        sqlx::query!(
            r#"
            UPDATE user_settings
            SET
                notification_level       = COALESCE($1, notification_level),
                message_notifications    = COALESCE($2, message_notifications),
                group_notifications      = COALESCE($3, group_notifications),
                system_notifications     = COALESCE($4, system_notifications),
                location_notifications   = COALESCE($5, location_notifications),
                quiet_hours_enabled      = COALESCE($6, quiet_hours_enabled),
                quiet_hours_start        = COALESCE($7, quiet_hours_start),
                quiet_hours_end          = COALESCE($8, quiet_hours_end),
                notification_radius_km   = COALESCE($9, notification_radius_km),
                sound_enabled            = COALESCE($10, sound_enabled),
                vibration_enabled        = COALESCE($11, vibration_enabled)
            WHERE user_id = $12
            "#,
            level_opt as Option<NotificationLevel>,
            settings.message_notifications,
            settings.group_notifications,
            settings.system_notifications,
            settings.location_notifications,
            settings.quiet_hours_enabled,
            quiet_start_opt,
            quiet_end_opt,
            settings.notification_radius_km,
            settings.sound_enabled,
            settings.vibration_enabled,
            user_id
        )
        .execute(&self.pool)
        .await
        .map_err(RepositoryError::Database)?;

        Ok(())
    }

    async fn delete_user(&self, user_id: i64) -> RepositoryResult<()> {
        // 在事务中删除用户及相关数据
        let mut tx = self.pool.begin().await.map_err(RepositoryError::Database)?;

        // 删除用户设置
        sqlx::query!("DELETE FROM user_settings WHERE user_id = $1", user_id)
            .execute(&mut *tx)
            .await
            .map_err(RepositoryError::Database)?;

        // 删除群组成员关系
        sqlx::query!("DELETE FROM group_members WHERE user_id = $1", user_id)
            .execute(&mut *tx)
            .await
            .map_err(RepositoryError::Database)?;

        // 删除私聊消息
        sqlx::query!("DELETE FROM direct_messages WHERE sender_id = $1 OR recipient_id = $1", user_id)
            .execute(&mut *tx)
            .await
            .map_err(RepositoryError::Database)?;

        // 删除群聊消息
        sqlx::query!("DELETE FROM group_messages WHERE sender_id = $1", user_id)
            .execute(&mut *tx)
            .await
            .map_err(RepositoryError::Database)?;

        // 删除动态记录
        sqlx::query!("DELETE FROM moments WHERE user_id = $1", user_id)
            .execute(&mut *tx)
            .await
            .map_err(RepositoryError::Database)?;

        // 删除会话记录
        sqlx::query!("DELETE FROM conversations WHERE user1_id = $1 OR user2_id = $1", user_id)
            .execute(&mut *tx)
            .await
            .map_err(RepositoryError::Database)?;

        // 最后删除用户
        sqlx::query!("DELETE FROM users WHERE user_id = $1", user_id)
            .execute(&mut *tx)
            .await
            .map_err(RepositoryError::Database)?;

        tx.commit().await.map_err(RepositoryError::Database)?;
        Ok(())
    }
}
