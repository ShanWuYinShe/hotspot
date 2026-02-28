//! 分享服务
//!
//! 负责分享的业务逻辑

use crate::dto::request::{
    CreateShareRequest, GetShareRequest, RevokeShareRequest, ShareListRequest,
};
use crate::dto::response::{
    PaginatedData, ShareCreated, ShareDetail, ShareListItem, SharePreview, UserInfo,
};
use crate::error::AppError;
use crate::repositories::{
    group_repo::GroupRepository,
    moment_repo::{MomentPreview, MomentRepository},
    share_repo::{ShareListRow, ShareRepository},
    user_repo::UserRepository,
};
use crate::services::media_service::MediaService;
use crate::types::enums::ShareType;
use crate::utils::{
    cache::{cache_keys, CacheManager, RateLimiter},
    id_gen,
    pagination::{decode_cursor, encode_cursor, paginate_with_limit_plus_one},
    validation::Validator,
};
use chrono::{DateTime, Utc};
use redis::aio::ConnectionManager;
use serde::{Deserialize, Serialize};
use sqlx::PgPool;
use std::{collections::HashMap, sync::Arc};
use tracing::{debug, info, warn};
use uuid::Uuid;

#[derive(Clone)]
pub struct ShareService {
    db: PgPool,
    redis: ConnectionManager,
    media_service: Arc<MediaService>,
    cache_manager: CacheManager,
    moment_repo: MomentRepository,
    user_repo: UserRepository,
    group_repo: GroupRepository,
}

impl ShareService {
    pub fn new(
        db: PgPool,
        redis: ConnectionManager,
        media_service: Arc<MediaService>,
        cache_manager: CacheManager,
        moment_repo: MomentRepository,
        user_repo: UserRepository,
        group_repo: GroupRepository,
    ) -> Self {
        Self {
            db,
            redis,
            media_service,
            cache_manager,
            moment_repo,
            user_repo,
            group_repo,
        }
    }

    /// 创建分享
    pub async fn create_share(
        &self,
        user_id: i64,
        req: CreateShareRequest,
    ) -> Result<ShareCreated, AppError> {
        let started_at = std::time::Instant::now();
        info!(
            user_id,
            share_type = ?req.share_type,
            has_expires = req.expires_in_seconds.is_some(),
            "share.create started"
        );
        // 验证 payload 格式
        Self::validate_payload(&req.share_type, &req.payload)?;

        // 验证目标资源是否存在
        self.verify_share_target(&req.share_type, &req.payload)
            .await?;

        // 计算过期时间（expires_in_seconds 须为正整数且不超过系统允许上限）
        const MAX_EXPIRES_SECONDS: u32 = 86400 * 365; // 1 年
        let expires_at = if let Some(secs) = req.expires_in_seconds {
            if secs == 0 {
                return Err(AppError::OutOfRange(
                    "expires_in_seconds 须为正整数".to_string(),
                ));
            }
            if secs > MAX_EXPIRES_SECONDS {
                return Err(AppError::OutOfRange(
                    "expires_in_seconds 超过系统允许上限".to_string(),
                ));
            }
            Some(chrono::Utc::now() + chrono::Duration::seconds(secs as i64))
        } else {
            None
        };

        // 直接插入并处理唯一键冲突重试，避免“先查后写”的并发竞态窗口。
        let (share_id, share_code) = Self::create_share_with_unique_code(
            &self.db,
            req.share_type.as_str(),
            user_id,
            &req.payload,
            expires_at.clone(),
        )
        .await?;
        info!(
            user_id,
            share_id = %share_id,
            share_type = ?req.share_type,
            elapsed_ms = started_at.elapsed().as_millis(),
            "share.create completed"
        );

        Ok(ShareCreated {
            share_id: share_id.to_string(),
            share_code,
            expires_at: expires_at.map(|t| t.to_rfc3339()),
        })
    }

    /// 获取分享详情
    pub async fn get_share(
        &self,
        req: GetShareRequest,
        user_id: Option<i64>,
        client_ip: &str,
    ) -> Result<ShareDetail, AppError> {
        self.enforce_get_rate_limit(user_id, client_ip).await?;
        let normalized_share_code =
            id_gen::normalize_share_code(&req.share_code).ok_or_else(|| {
                warn!(
                    share_code = %req.share_code,
                    user_id = ?user_id,
                    client_ip = client_ip,
                    "share get failed: invalid share_code format"
                );
                AppError::BadRequest("share_code 格式错误".to_string())
            })?;
        // 获取分享
        let db_share = ShareRepository::get_by_code(&self.db, &normalized_share_code)
            .await?
            .ok_or_else(|| {
                warn!(
                    share_code = %normalized_share_code,
                    user_id = ?user_id,
                    client_ip = client_ip,
                    "share get failed: not found or revoked"
                );
                AppError::NotFound("分享不存在或已删除".to_string())
            })?;

        // 获取创建者信息
        let creator_row = sqlx::query!(
            r#"
            SELECT user_id, nickname, avatar_url
            FROM users
            WHERE user_id = $1 AND status != 'banned' AND deleted_at IS NULL
            "#,
            db_share.creator_id
        )
        .fetch_optional(&self.db)
        .await
        .map_err(|e| AppError::internal(e.to_string()))?
        .ok_or(AppError::NotFound("用户不存在".to_string()))?;

        let creator = {
            let avatar_url = self
                .media_service
                .sign_url_if_path(creator_row.avatar_url.as_deref())
                .await;
            UserInfo::new(
                creator_row.user_id,
                creator_row
                    .nickname
                    .unwrap_or_else(|| crate::constants::UNKNOWN_USER_NICKNAME.to_string()),
                avatar_url,
            )
        };

        // 解析 share_type
        let share_type = db_share
            .share_type
            .parse::<ShareType>()
            .map_err(|_| AppError::NotFound("无效的分享类型".to_string()))?;

        let share_base = crate::dto::response::ShareBase {
            share_id: db_share.share_id.to_string(),
            share_code: db_share.share_code,
            share_type,
            creator,
            created_at: db_share.created_at.to_rfc3339(),
            expires_at: db_share.expires_at.map(|t| t.to_rfc3339()),
        };

        Ok(crate::dto::response::ShareDetail {
            share: share_base,
            payload: db_share.payload,
            preview: None, // 已撤销、已过期或目标不可访问时不返回 preview
        })
    }

    async fn enforce_get_rate_limit(
        &self,
        user_id: Option<i64>,
        client_ip: &str,
    ) -> Result<(), AppError> {
        const IP_LIMIT: i64 = 10;
        const USER_LIMIT: i64 = 30;
        const WINDOW_SECS: i64 = 60;
        let limiter = RateLimiter::new(self.redis.clone());

        if limiter
            .is_rate_limited(
                &format!("share_get:ip:{}", client_ip),
                IP_LIMIT,
                WINDOW_SECS,
            )
            .await
            .unwrap_or(false)
        {
            tracing::warn!(ip = client_ip, "share get rate limit (ip)");
            return Err(AppError::ShareRateLimited {
                message: "分享查看请求过于频繁，请稍后再试".to_string(),
                retry_after: WINDOW_SECS as u32,
            });
        }

        if let Some(uid) = user_id {
            if limiter
                .is_rate_limited(&format!("share_get:user:{}", uid), USER_LIMIT, WINDOW_SECS)
                .await
                .unwrap_or(false)
            {
                tracing::warn!(user_id = uid, "share get rate limit (user)");
                return Err(AppError::ShareRateLimited {
                    message: "分享查看请求过于频繁，请稍后再试".to_string(),
                    retry_after: WINDOW_SECS as u32,
                });
            }
        }

        Ok(())
    }

    /// 撤销分享（按 share_id）
    pub async fn revoke_share(
        &self,
        user_id: i64,
        req: RevokeShareRequest,
    ) -> Result<(), AppError> {
        let started_at = std::time::Instant::now();
        let share_id = Validator::parse_uuid(&req.share_id, "share_id")?;
        info!(user_id, share_id = %share_id, "share.revoke started");

        let share = ShareRepository::get_by_id(&self.db, share_id)
            .await?
            .ok_or(AppError::NotFound("分享不存在或已删除".to_string()))?;

        if share.creator_id != user_id {
            warn!(
                user_id,
                share_id = %share_id,
                creator_id = share.creator_id,
                "share.revoke rejected: not creator"
            );
            return Err(AppError::Forbidden("无权撤销他人的分享".to_string()));
        }

        if share.is_revoked {
            info!(
                user_id,
                share_id = %share_id,
                elapsed_ms = started_at.elapsed().as_millis(),
                "share.revoke completed (noop_already_revoked)"
            );
            return Ok(());
        }

        ShareRepository::revoke_by_id(&self.db, share_id)
            .await
            .map_err(|e| AppError::internal(e.to_string()))?;
        info!(
            user_id,
            share_id = %share_id,
            elapsed_ms = started_at.elapsed().as_millis(),
            "share.revoke completed"
        );

        Ok(())
    }

    /// 获取当前用户的分享列表
    pub async fn list_my_shares(
        &self,
        user_id: i64,
        req: ShareListRequest,
    ) -> Result<PaginatedData<ShareListItem>, AppError> {
        let limit =
            Validator::validate_limit(req.pagination.as_ref().and_then(|p| p.limit), 100)? as i32;
        let cursor_val = req
            .pagination
            .as_ref()
            .and_then(|p| p.cursor.as_deref())
            .map(decode_cursor::<ShareListCursor>)
            .transpose()?;

        let share_type_filter = req.share_type.map(|ty| ty.to_string());
        let rows = ShareRepository::list_by_creator(
            &self.db,
            user_id,
            share_type_filter.as_deref(),
            cursor_val.map(|c| (c.created_at, c.share_id)),
            (limit + 1) as i64,
        )
        .await?;

        let encode_cursor_fn = |row: &ShareListRow| -> Result<String, AppError> {
            encode_cursor(&ShareListCursor {
                created_at: row.created_at,
                share_id: row.share_id,
            })
        };

        let (rows, next_cursor) = paginate_with_limit_plus_one(rows, limit, encode_cursor_fn)?;

        let mut share_records = Vec::with_capacity(rows.len());
        let mut moment_targets: HashMap<Uuid, Uuid> = HashMap::new();
        let mut moment_ids = Vec::new();
        let mut user_targets: HashMap<Uuid, i64> = HashMap::new();
        let mut user_ids = Vec::new();
        let mut group_targets: HashMap<Uuid, i64> = HashMap::new();
        let mut group_ids = Vec::new();
        let mut location_previews: HashMap<Uuid, SharePreview> = HashMap::new();

        for row in rows {
            let share_type = row
                .share_type
                .parse::<ShareType>()
                .map_err(|_| AppError::InvalidFormat("share_type 数据损坏".to_string()))?;

            match share_type {
                ShareType::Moment => {
                    if let Some(moment_id) = row
                        .payload
                        .get("shared_moment_id")
                        .and_then(|v| v.as_str())
                        .and_then(|s| Uuid::parse_str(s).ok())
                    {
                        if moment_targets.insert(row.share_id, moment_id).is_none() {
                            moment_ids.push(moment_id);
                        }
                    }
                }
                ShareType::User => {
                    if let Some(user_id_str) = row
                        .payload
                        .get("shared_user_id")
                        .and_then(|v| v.as_str())
                        .and_then(|s| s.parse::<i64>().ok())
                    {
                        if user_targets.insert(row.share_id, user_id_str).is_none() {
                            user_ids.push(user_id_str);
                        }
                    }
                }
                ShareType::Group => {
                    if let Some(group_id) = row
                        .payload
                        .get("shared_group_id")
                        .and_then(|v| v.as_str())
                        .and_then(|s| s.parse::<i64>().ok())
                    {
                        if group_targets.insert(row.share_id, group_id).is_none() {
                            group_ids.push(group_id);
                        }
                    }
                }
                ShareType::Location => {
                    if let Some(preview) = Self::build_location_preview(&row.payload) {
                        location_previews.insert(row.share_id, preview);
                    }
                }
            }

            share_records.push((row, share_type));
        }

        let mut moment_preview_map = HashMap::new();
        if !moment_ids.is_empty() {
            let previews_by_moment = self.load_moment_previews(&moment_ids).await?;
            for (share_id, moment_id) in &moment_targets {
                if let Some(preview) = previews_by_moment.get(moment_id) {
                    moment_preview_map.insert(*share_id, preview.clone());
                }
            }
        }

        let mut user_preview_map = HashMap::new();
        if !user_ids.is_empty() {
            user_preview_map = self.load_user_previews(&user_ids).await?;
        }

        let mut group_preview_map = HashMap::new();
        if !group_ids.is_empty() {
            group_preview_map = self.load_group_previews(&group_ids).await?;
        }

        let mut items = Vec::with_capacity(share_records.len());
        for (row, share_type) in share_records {
            let preview = match share_type {
                ShareType::Moment => moment_preview_map.get(&row.share_id).cloned(),
                ShareType::User => user_targets
                    .get(&row.share_id)
                    .and_then(|id| user_preview_map.get(id))
                    .cloned(),
                ShareType::Group => group_targets
                    .get(&row.share_id)
                    .and_then(|id| group_preview_map.get(id))
                    .cloned(),
                ShareType::Location => location_previews.get(&row.share_id).cloned(),
            };

            items.push(ShareListItem {
                share_id: row.share_id.to_string(),
                share_code: row.share_code,
                share_type,
                created_at: row.created_at.to_rfc3339(),
                expires_at: row.expires_at.map(|t| t.to_rfc3339()),
                is_revoked: row.is_revoked,
                preview,
            });
        }

        Ok(PaginatedData { items, next_cursor })
    }
    fn build_moment_preview(data: &MomentPreview) -> SharePreview {
        let title = data
            .description
            .as_deref()
            .map(Self::trim_preview_text)
            .unwrap_or_else(|| "动态分享".to_string());
        SharePreview {
            title,
            subtitle: data.location_name.clone(),
            cover_url: None,
        }
    }

    fn build_location_preview(payload: &serde_json::Value) -> Option<SharePreview> {
        let title = payload.get("location_name")?.as_str()?.to_string();
        let latitude = payload.get("latitude")?.as_f64()?;
        let longitude = payload.get("longitude")?.as_f64()?;
        Some(SharePreview {
            title,
            subtitle: Some(format!("{latitude:.4}, {longitude:.4}")),
            cover_url: None,
        })
    }

    fn trim_preview_text(text: &str) -> String {
        const MAX_LEN: usize = 40;
        let trimmed = text.trim();
        if trimmed.chars().count() <= MAX_LEN {
            return trimmed.to_string();
        }
        let mut shortened = String::with_capacity(MAX_LEN + 3);
        for (idx, ch) in trimmed.chars().enumerate() {
            if idx >= MAX_LEN {
                break;
            }
            shortened.push(ch);
        }
        shortened.push('…');
        shortened
    }

    async fn load_moment_previews(
        &self,
        moment_ids: &[Uuid],
    ) -> Result<HashMap<Uuid, SharePreview>, AppError> {
        let mut preview_map = HashMap::new();
        if moment_ids.is_empty() {
            return Ok(preview_map);
        }

        let mut cache_hits = 0usize;
        let mut cache_null_hits = 0usize;
        let mut cache_misses = 0usize;
        let mut db_loaded = 0usize;
        let mut missing = Vec::new();
        for moment_id in moment_ids {
            let key = moment_id.to_string();
            match self
                .get_cached_preview(cache_keys::SHARE_MOMENT_PREVIEW, &key)
                .await
            {
                Some(Some(preview)) => {
                    cache_hits += 1;
                    preview_map.insert(*moment_id, preview);
                }
                Some(None) => {
                    cache_null_hits += 1;
                }
                None => {
                    cache_misses += 1;
                    missing.push((*moment_id, key));
                }
            }
        }

        if !missing.is_empty() {
            let ids: Vec<Uuid> = missing.iter().map(|(id, _)| *id).collect();
            let db_previews = self.moment_repo.get_previews(&ids).await?;
            for (moment_id, key) in missing {
                if let Some(data) = db_previews.get(&moment_id) {
                    let preview = Self::build_moment_preview(data);
                    self.cache_preview_value(cache_keys::SHARE_MOMENT_PREVIEW, &key, &preview)
                        .await;
                    preview_map.insert(moment_id, preview);
                    db_loaded += 1;
                    db_loaded += 1;
                } else {
                    self.cache_preview_miss(cache_keys::SHARE_MOMENT_PREVIEW, &key)
                        .await;
                }
            }
            debug!(
                target: "share_service",
                preview_type = "moment",
                cache_hits,
                cache_null_hits,
                cache_misses,
                db_loaded,
                "share preview cache stats"
            );
        } else {
            debug!(
                target: "share_service",
                preview_type = "moment",
                cache_hits,
                cache_null_hits,
                cache_misses,
                db_loaded = 0,
                "share preview cache stats"
            );
        }

        Ok(preview_map)
    }

    async fn load_user_previews(
        &self,
        user_ids: &[i64],
    ) -> Result<HashMap<i64, SharePreview>, AppError> {
        let mut preview_map = HashMap::new();
        if user_ids.is_empty() {
            return Ok(preview_map);
        }

        let mut cache_hits = 0usize;
        let mut cache_null_hits = 0usize;
        let mut cache_misses = 0usize;
        let mut db_loaded = 0usize;
        let mut missing = Vec::new();
        for user_id in user_ids {
            let key = user_id.to_string();
            match self
                .get_cached_preview(cache_keys::SHARE_USER_PREVIEW, &key)
                .await
            {
                Some(Some(preview)) => {
                    cache_hits += 1;
                    preview_map.insert(*user_id, preview);
                }
                Some(None) => {
                    cache_null_hits += 1;
                }
                None => {
                    cache_misses += 1;
                    missing.push((*user_id, key));
                }
            }
        }

        if !missing.is_empty() {
            let ids: Vec<i64> = missing.iter().map(|(id, _)| *id).collect();
            let profiles = self.user_repo.find_profiles(&ids).await?;
            let profile_map: HashMap<i64, SharePreview> = profiles
                .into_iter()
                .map(|profile| {
                    let title = profile.nickname.clone().unwrap_or_else(|| {
                        if profile.username.is_empty() {
                            "用户".to_string()
                        } else {
                            profile.username.clone()
                        }
                    });
                    (
                        profile.user_id,
                        SharePreview {
                            title,
                            subtitle: profile.bio.clone(),
                            cover_url: None,
                        },
                    )
                })
                .collect();

            for (user_id, key) in missing {
                if let Some(preview) = profile_map.get(&user_id) {
                    self.cache_preview_value(cache_keys::SHARE_USER_PREVIEW, &key, preview)
                        .await;
                    preview_map.insert(user_id, preview.clone());
                    db_loaded += 1;
                } else {
                    self.cache_preview_miss(cache_keys::SHARE_USER_PREVIEW, &key)
                        .await;
                }
            }
            debug!(
                target: "share_service",
                preview_type = "user",
                cache_hits,
                cache_null_hits,
                cache_misses,
                db_loaded,
                "share preview cache stats"
            );
        } else {
            debug!(
                target: "share_service",
                preview_type = "user",
                cache_hits,
                cache_null_hits,
                cache_misses,
                db_loaded,
                "share preview cache stats"
            );
        }

        Ok(preview_map)
    }

    async fn load_group_previews(
        &self,
        group_ids: &[i64],
    ) -> Result<HashMap<i64, SharePreview>, AppError> {
        let mut preview_map = HashMap::new();
        if group_ids.is_empty() {
            return Ok(preview_map);
        }

        let mut cache_hits = 0usize;
        let mut cache_null_hits = 0usize;
        let mut cache_misses = 0usize;
        let mut db_loaded = 0usize;
        let mut missing = Vec::new();
        for group_id in group_ids {
            let key = group_id.to_string();
            match self
                .get_cached_preview(cache_keys::SHARE_GROUP_PREVIEW, &key)
                .await
            {
                Some(Some(preview)) => {
                    cache_hits += 1;
                    preview_map.insert(*group_id, preview);
                }
                Some(None) => {
                    cache_null_hits += 1;
                }
                None => {
                    cache_misses += 1;
                    missing.push((*group_id, key));
                }
            }
        }

        if !missing.is_empty() {
            let ids: Vec<i64> = missing.iter().map(|(id, _)| *id).collect();
            let groups = self.group_repo.list_by_ids(&ids).await?;
            let group_map: HashMap<i64, SharePreview> = groups
                .into_iter()
                .map(|group| {
                    (
                        group.group_id,
                        SharePreview {
                            title: group.name.clone(),
                            subtitle: group.description.clone(),
                            cover_url: group.avatar_url.clone(),
                        },
                    )
                })
                .collect();

            for (group_id, key) in missing {
                if let Some(preview) = group_map.get(&group_id) {
                    self.cache_preview_value(cache_keys::SHARE_GROUP_PREVIEW, &key, preview)
                        .await;
                    preview_map.insert(group_id, preview.clone());
                    db_loaded += 1;
                } else {
                    self.cache_preview_miss(cache_keys::SHARE_GROUP_PREVIEW, &key)
                        .await;
                }
            }
            debug!(
                target: "share_service",
                preview_type = "group",
                cache_hits,
                cache_null_hits,
                cache_misses,
                db_loaded,
                "share preview cache stats"
            );
        } else {
            debug!(
                target: "share_service",
                preview_type = "group",
                cache_hits,
                cache_null_hits,
                cache_misses,
                db_loaded,
                "share preview cache stats"
            );
        }

        Ok(preview_map)
    }

    async fn get_cached_preview(&self, prefix: &str, key: &str) -> Option<Option<SharePreview>> {
        match self.cache_manager.get::<SharePreview>(prefix, key).await {
            Ok(value) => value,
            Err(err) => {
                warn!(target: "share_service", error = %err, "failed to fetch cached preview");
                None
            }
        }
    }

    async fn cache_preview_value(&self, prefix: &str, key: &str, preview: &SharePreview) {
        if let Err(err) = self.cache_manager.set(prefix, key, preview).await {
            warn!(target: "share_service", error = %err, "failed to cache share preview");
        }
    }

    async fn cache_preview_miss(&self, prefix: &str, key: &str) {
        if let Err(err) = self.cache_manager.set_null(prefix, key).await {
            warn!(target: "share_service", error = %err, "failed to cache empty share preview");
        }
    }

    /// 验证 payload 的有效性
    fn validate_payload(
        share_type: &ShareType,
        payload: &serde_json::Value,
    ) -> Result<(), AppError> {
        match share_type {
            ShareType::Location => {
                let shared_location = payload
                    .get("shared_location")
                    .ok_or(AppError::MissingParameter("shared_location".to_string()))?;

                let point = shared_location
                    .get("point")
                    .ok_or(AppError::MissingParameter(
                        "shared_location.point".to_string(),
                    ))?;

                let lat =
                    point
                        .get("lat")
                        .and_then(|v| v.as_f64())
                        .ok_or(AppError::MissingParameter(
                            "shared_location.point.lat".to_string(),
                        ))?;

                let lng =
                    point
                        .get("lng")
                        .and_then(|v| v.as_f64())
                        .ok_or(AppError::MissingParameter(
                            "shared_location.point.lng".to_string(),
                        ))?;

                let location_name = shared_location
                    .get("location_name")
                    .and_then(|v| v.as_str())
                    .ok_or(AppError::MissingParameter(
                        "shared_location.location_name".to_string(),
                    ))?;

                if !(-90.0..=90.0).contains(&lat) {
                    return Err(AppError::OutOfRange(
                        "纬度必须在 -90 到 90 之间".to_string(),
                    ));
                }
                if !(-180.0..=180.0).contains(&lng) {
                    return Err(AppError::OutOfRange(
                        "经度必须在 -180 到 180 之间".to_string(),
                    ));
                }
                if location_name.is_empty() || location_name.len() > 100 {
                    return Err(AppError::OutOfRange(
                        "位置名称长度必须在 1-100 之间".to_string(),
                    ));
                }

                Ok(())
            }
            ShareType::Moment => {
                // 验证动态分享的 payload: 需要 shared_moment_id
                let moment_id_str = payload
                    .get("shared_moment_id")
                    .and_then(|v| v.as_str())
                    .ok_or(AppError::MissingParameter("shared_moment_id".to_string()))?;

                // 验证是合法的 UUID
                Validator::parse_uuid(moment_id_str, "shared_moment_id")?;

                Ok(())
            }
            ShareType::Group => {
                // 验证群组分享的 payload: 需要 shared_group_id
                let group_id_str = payload
                    .get("shared_group_id")
                    .and_then(|v| v.as_str())
                    .ok_or(AppError::MissingParameter("shared_group_id".to_string()))?;

                // 验证是合法的数字 ID
                Validator::parse_id(group_id_str, "shared_group_id")?;

                Ok(())
            }
            ShareType::User => {
                // 验证用户分享的 payload: 需要 shared_user_id
                let user_id_str = payload
                    .get("shared_user_id")
                    .and_then(|v| v.as_str())
                    .ok_or(AppError::MissingParameter("shared_user_id".to_string()))?;

                // 验证是合法的数字 ID
                Validator::parse_id(user_id_str, "shared_user_id")?;

                Ok(())
            }
        }
    }

    /// 验证分享目标资源是否存在（格式验证已由 validate_payload 完成）
    async fn verify_share_target(
        &self,
        share_type: &ShareType,
        payload: &serde_json::Value,
    ) -> Result<(), AppError> {
        match share_type {
            ShareType::Location => Ok(()),
            ShareType::Moment => {
                let moment_id_str = payload
                    .get("shared_moment_id")
                    .and_then(|v| v.as_str())
                    .ok_or_else(|| AppError::BadRequest("缺少 shared_moment_id".to_string()))?;
                let moment_id = Validator::parse_uuid(moment_id_str, "shared_moment_id")?;
                self.moment_repo
                    .find_by_id(moment_id)
                    .await?
                    .ok_or(AppError::NotFound("分享的动态不存在".to_string()))?;
                Ok(())
            }
            ShareType::Group => {
                let group_id_str = payload
                    .get("shared_group_id")
                    .and_then(|v| v.as_str())
                    .ok_or_else(|| AppError::BadRequest("缺少 shared_group_id".to_string()))?;
                let group_id = Validator::parse_id(group_id_str, "shared_group_id")?;
                self.group_repo
                    .find_by_id(group_id)
                    .await?
                    .ok_or(AppError::NotFound("分享的群组不存在".to_string()))?;
                Ok(())
            }
            ShareType::User => {
                let user_id_str = payload
                    .get("shared_user_id")
                    .and_then(|v| v.as_str())
                    .ok_or_else(|| AppError::BadRequest("缺少 shared_user_id".to_string()))?;
                let user_id = Validator::parse_id(user_id_str, "shared_user_id")?;
                self.user_repo
                    .find_by_id(user_id)
                    .await?
                    .ok_or(AppError::NotFound("分享的用户不存在".to_string()))?;
                Ok(())
            }
        }
    }

    /// 创建分享并处理 share_code 并发冲突重试
    async fn create_share_with_unique_code(
        pool: &PgPool,
        share_type: &str,
        creator_id: i64,
        payload: &serde_json::Value,
        expires_at: Option<DateTime<Utc>>,
    ) -> Result<(Uuid, String), AppError> {
        const MAX_ATTEMPTS: u32 = 10;
        const UNIQUE_VIOLATION_CODE: &str = "23505";

        for attempt in 1..=MAX_ATTEMPTS {
            let code = id_gen::generate_share_code();

            match ShareRepository::create(
                pool,
                &code,
                share_type,
                creator_id,
                payload,
                expires_at.clone(),
            )
            .await
            {
                Ok(share_id) => return Ok((share_id, code)),
                Err(AppError::Sqlx(sqlx::Error::Database(db_err))) => {
                    if matches!(db_err.code().as_deref(), Some(UNIQUE_VIOLATION_CODE)) {
                        debug!(attempt, "share_code 插入冲突，重试生成");
                        continue;
                    }
                    return Err(AppError::Sqlx(sqlx::Error::Database(db_err)));
                }
                Err(err) => return Err(err),
            }
        }

        warn!(
            max_attempts = MAX_ATTEMPTS,
            "share_code 在重试后仍冲突，可能存在异常并发或随机源问题"
        );
        Err(AppError::BusinessRuleViolation(
            "无法生成唯一的分享码，请稍后重试".to_string(),
        ))
    }
}

#[derive(Debug, Serialize, Deserialize)]
struct ShareListCursor {
    created_at: DateTime<Utc>,
    share_id: uuid::Uuid,
}
