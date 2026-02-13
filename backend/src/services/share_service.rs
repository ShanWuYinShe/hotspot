//! 分享服务
//!
//! 负责分享的业务逻辑

use crate::dto::request::{
    CreateShareRequest, GetShareRequest, MySharesRequest, RevokeShareRequest,
};
use crate::dto::response::{PaginatedData, Share, UserInfo};
use crate::error::AppError;
use crate::repositories::share_repo::ShareRepository;
use crate::types::enums::{ShareType, UserType};
use crate::utils::{id_gen, validation::Validator};
use sqlx::PgPool;
use uuid::Uuid;

pub struct ShareService;

impl ShareService {
    /// 创建分享
    pub async fn create_share(
        pool: &PgPool,
        user_id: i64,
        req: CreateShareRequest,
    ) -> Result<Share, AppError> {
        // 验证 payload
        Self::validate_payload(&req.share_type, &req.payload)?;

        // 生成唯一的分享码
        let share_code = Self::generate_unique_share_code(pool).await?;

        // 创建分享
        let share_id = ShareRepository::create(
            pool,
            &share_code,
            req.share_type.as_str(),
            user_id,
            &req.payload,
        )
        .await?;

        // 获取创建者信息
        let creator = sqlx::query!(
            r#"
            SELECT user_id, nickname, avatar_url, user_type as "user_type: UserType"
            FROM users
            WHERE user_id = $1 AND status = 'active'::user_status
            "#,
            user_id
        )
        .fetch_optional(pool)
        .await
        .map_err(|e| AppError::internal(e.to_string()))?
        .ok_or(AppError::NotFound("用户不存在".to_string()))
        .map(|row| UserInfo {
            user_id: row.user_id,
            nickname: row.nickname.unwrap_or_else(|| "未知用户".to_string()),
            avatar_url: row.avatar_url,
        })?;

        Ok(Share {
            share_id: share_id.to_string(),
            share_code,
            share_type: req.share_type,
            creator,
            payload: req.payload,
            created_at: chrono::Utc::now().to_rfc3339(),
        })
    }

    /// 获取分享详情
    pub async fn get_share(pool: &PgPool, req: GetShareRequest) -> Result<Share, AppError> {
        // 获取分享
        let db_share = ShareRepository::get_by_code(pool, &req.share_code)
            .await?
            .ok_or(AppError::NotFound("分享不存在或已删除".to_string()))?;

        // 获取创建者信息
        let creator = sqlx::query!(
            r#"
            SELECT user_id, nickname, avatar_url, user_type as "user_type: UserType"
            FROM users
            WHERE user_id = $1 AND status = 'active'::user_status
            "#,
            db_share.creator_id
        )
        .fetch_optional(pool)
        .await
        .map_err(|e| AppError::internal(e.to_string()))?
        .ok_or(AppError::NotFound("用户不存在".to_string()))
        .map(|row| UserInfo {
            user_id: row.user_id,
            nickname: row.nickname.unwrap_or_else(|| "未知用户".to_string()),
            avatar_url: row.avatar_url,
        })?;

        // 解析 share_type
        let share_type = db_share
            .share_type
            .parse::<ShareType>()
            .map_err(|_| AppError::NotFound("无效的分享类型".to_string()))?;

        Ok(Share {
            share_id: db_share.share_id.to_string(),
            share_code: db_share.share_code,
            share_type,
            creator,
            payload: db_share.payload,
            created_at: db_share.created_at.to_rfc3339(),
        })
    }

    /// 撤销分享（按 share_id）
    pub async fn revoke_share(
        pool: &PgPool,
        user_id: i64,
        req: RevokeShareRequest,
    ) -> Result<(), AppError> {
        let share_id = Uuid::parse_str(&req.share_id)
            .map_err(|_| AppError::BadRequest("无效的 share_id".to_string()))?;

        let share = ShareRepository::get_by_id(pool, share_id)
            .await?
            .ok_or(AppError::NotFound("分享不存在或已删除".to_string()))?;

        if share.creator_id != user_id {
            return Err(AppError::Forbidden("无权撤销他人的分享".to_string()));
        }

        ShareRepository::revoke_by_id(pool, share_id)
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;

        Ok(())
    }

    /// 我的分享列表（分页）
    pub async fn my_shares(
        pool: &PgPool,
        user_id: i64,
        req: MySharesRequest,
    ) -> Result<PaginatedData<Share>, AppError> {
        let limit = Validator::validate_limit(req.limit, 50)? as i64;

        let (rows, next_cursor) =
            ShareRepository::list_by_creator(pool, user_id, limit, req.cursor.as_deref())
                .await
                .map_err(|e| AppError::internal(e.to_string()))?;

        let mut items = Vec::with_capacity(rows.len());
        for row in rows {
            let creator_row = sqlx::query!(
                r#"
                SELECT user_id, nickname, avatar_url, user_type as "user_type: UserType"
                FROM users
                WHERE user_id = $1 AND status = 'active'::user_status
                "#,
                row.creator_id
            )
            .fetch_optional(pool)
            .await
            .map_err(|e| AppError::internal(e.to_string()))?
            .ok_or(AppError::NotFound("用户不存在".to_string()))?;

            let creator = UserInfo {
                user_id: creator_row.user_id,
                nickname: creator_row
                    .nickname
                    .unwrap_or_else(|| "未知用户".to_string()),
                avatar_url: creator_row.avatar_url,
            };

            let share_type = row
                .share_type
                .parse::<ShareType>()
                .map_err(|_| AppError::NotFound("无效的分享类型".to_string()))?;

            items.push(Share {
                share_id: row.share_id.to_string(),
                share_code: row.share_code,
                share_type,
                creator,
                payload: row.payload,
                created_at: row.created_at.to_rfc3339(),
            });
        }

        Ok(PaginatedData {
            items,
            next_cursor,
        })
    }

    /// 验证 payload 的有效性
    fn validate_payload(
        share_type: &ShareType,
        payload: &serde_json::Value,
    ) -> Result<(), AppError> {
        match share_type {
            ShareType::ReferencePoint => {
                // 验证位置分享的 payload
                let latitude = payload
                    .get("latitude")
                    .and_then(|v| v.as_f64())
                    .ok_or(AppError::OutOfRange("缺少 latitude 字段".to_string()))?;

                let longitude = payload
                    .get("longitude")
                    .and_then(|v| v.as_f64())
                    .ok_or(AppError::OutOfRange("缺少 longitude 字段".to_string()))?;

                let location_name = payload
                    .get("location_name")
                    .and_then(|v| v.as_str())
                    .ok_or(AppError::OutOfRange("缺少 location_name 字段".to_string()))?;

                if !(-90.0..=90.0).contains(&latitude) {
                    return Err(AppError::OutOfRange(
                        "纬度必须在 -90 到 90 之间".to_string(),
                    ));
                }
                if !(-180.0..=180.0).contains(&longitude) {
                    return Err(AppError::OutOfRange(
                        "经度必须在 -180 到 180 之间".to_string(),
                    ));
                }
                if location_name.is_empty() || location_name.len() > 100 {
                    return Err(AppError::OutOfRange(
                        "位置名称长度必须在 1-255 之间".to_string(),
                    ));
                }

                Ok(())
            }
            ShareType::Moment => {
                // TODO: 验证动态分享的 payload
                Err(AppError::NotFound("动态分享暂未实现".to_string()))
            }
            ShareType::Group => {
                // TODO: 验证群组分享的 payload
                Err(AppError::NotFound("群组分享暂未实现".to_string()))
            }
            ShareType::User => {
                // TODO: 验证用户分享的 payload
                Err(AppError::NotFound("用户分享暂未实现".to_string()))
            }
        }
    }

    /// 生成唯一的分享码
    async fn generate_unique_share_code(pool: &PgPool) -> Result<String, AppError> {
        const MAX_ATTEMPTS: u32 = 10;

        for _ in 0..MAX_ATTEMPTS {
            let code = id_gen::generate_share_code();

            // 检查是否已存在
            if !ShareRepository::exists_by_code(pool, &code).await? {
                return Ok(code);
            }
        }

        Err(AppError::BusinessRuleViolation(
            "无法生成唯一的分享码，请稍后重试".to_string(),
        ))
    }
}
