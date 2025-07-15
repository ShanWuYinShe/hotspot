use std::sync::Arc;
use tracing::error;
use uuid::Uuid;

use crate::{
    repositories::{
        checkin_repository::CheckinQuery,
        RepositoryError
    },
    utils::{response::AppError, result_ext::ResultExt},
};

#[derive(Debug, Clone)]
pub struct CheckinService {
    repo: Arc<crate::repositories::PgCheckinRepository>,
}

impl CheckinService {
    pub fn new(db: crate::config::db::Database) -> Self {
        let repo = Arc::new(crate::repositories::PgCheckinRepository::new(db.pg_pool.clone()));
        Self { repo }
    }

    // =============== 验证函数 ===============

    // 检查签到是否存在
    #[allow(dead_code)]
    pub async fn check_checkin_exists(&self, checkin_id: Uuid) -> Result<bool, AppError> {
        self.repo
            .check_checkin_exists(checkin_id)
            .await
            .map_repository_error_with_msg("查询签到", "签到不存在")
    }

    // 检查签到所有权
    #[allow(dead_code)]
    pub async fn check_checkin_ownership(
        &self,
        checkin_id: Uuid,
        user_id: i64,
    ) -> Result<bool, AppError> {
        self.repo
            .check_checkin_ownership(checkin_id, user_id)
            .await
            .map_repository_error_with_msg("查询签到所有权", "签到不存在")
    }

    // 检查用户是否已点赞签到
    #[allow(dead_code)]
    pub async fn check_checkin_liked(
        &self,
        checkin_id: Uuid,
        user_id: i64,
    ) -> Result<bool, AppError> {
        self.repo.check_checkin_liked(checkin_id, user_id).await.map_err(|e| {
            error!("查询签到点赞状态失败: {:?}", e);
            match e {
                RepositoryError::NotFound => AppError::NotFound("签到不存在".to_string()),
                _ => AppError::InternalServerError("查询签到点赞失败".to_string()),
            }
        })
    }

    // =============== 操作函数 ===============

    // 创建签到
    pub async fn create_checkin(
        &self,
        user_id: i64,
        description: &str,
        latitude: f64,
        longitude: f64,
        location_name: &str,
        tags: &[String],
    ) -> Result<String, AppError> {
        self.repo.create_checkin(user_id, description, latitude, longitude, location_name, tags).await.map_err(|e| {
            error!("创建签到失败: {:?}", e);
            AppError::InternalServerError("创建签到失败".to_string())
        })
    }

    // 点赞签到
    pub async fn like_checkin(
        &self,
        _user_id: i64,
        _checkin_id: &str,
    ) -> Result<(), AppError> {
        // 这里应该实现点赞逻辑，暂时返回Ok
        Ok(())
    }

    // 取消点赞
    pub async fn unlike_checkin(
        &self,
        checkin_id: &str,
        user_id: i64,
    ) -> Result<(), AppError> {
        // 转换签到ID
        let uuid_checkin_id = Uuid::parse_str(checkin_id).map_err(|e| {
            error!("解析签到ID失败: {:?}", e);
            AppError::BadRequest("签到ID格式错误".to_string())
        })?;

        // 检查是否已点赞
        if !self.repo.check_checkin_liked(uuid_checkin_id, user_id).await.map_err(|e| {
            error!("查询签到点赞状态失败: {:?}", e);
            AppError::InternalServerError("查询签到点赞状态失败".to_string())
        })? {
            return Ok(()); // 没有点赞过，直接返回成功
        }

        // 执行取消点赞
        self.repo.remove_checkin_like(uuid_checkin_id, user_id).await.map_err(|e| {
            error!("移除签到点赞失败: {:?}", e);
            AppError::InternalServerError("移除签到点赞失败".to_string())
        })?;

        Ok(())
    }

    // 获取签到详情
    pub async fn search_checkin_by_id(
        &self,
        user_id: i64,
        checkin_id: &str,
    ) -> Result<CheckinQuery, AppError> {
        // 使用仓库方法获取签到信息
        self.repo.search_checkin_by_id(user_id, checkin_id).await.map_err(|e| {
            error!("查询签到失败: {:?}", e);
            match e {
                RepositoryError::NotFound => AppError::NotFound("签到不存在".to_string()),
                _ => AppError::InternalServerError("查询签到失败".to_string())
            }
        })
    }

    // 获取用户签到历史
    pub async fn search_checkins_by_user(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i64,
    ) -> Result<Vec<CheckinQuery>, AppError> {
        self.repo.search_checkins_by_user(user_id, cursor, limit).await.map_err(|e| {
            error!("查询用户签到历史失败: {:?}", e);
            AppError::InternalServerError("查询签到历史失败".to_string())
        })
    }

    // 搜索附近签到
    pub async fn search_checkins_by_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        radius: f64,
        cursor: i64,
        limit: i64,
    ) -> Result<Vec<CheckinQuery>, AppError> {
        self.repo.search_checkins_by_location(user_id, latitude, longitude, radius, cursor, limit).await.map_err(|e| {
            error!("查询附近签到失败: {:?}", e);
            AppError::InternalServerError("查询附近签到失败".to_string())
        })
    }

    // 根据标签搜索签到
    pub async fn search_checkins_by_tags(
        &self,
        user_id: i64,
        tags: &[String],
        cursor: i64,
        limit: i64,
    ) -> Result<Vec<CheckinQuery>, AppError> {
        self.repo.search_checkins_by_tags(user_id, tags, cursor, limit).await.map_err(|e| {
            error!("搜索签到失败: {:?}", e);
            AppError::InternalServerError("搜索签到失败".to_string())
        })
    }

    // 删除签到记录
    pub async fn delete_checkin_record(&self, checkin_id: Uuid) -> Result<(), AppError> {
        self.repo.delete_checkin_record(checkin_id).await.map_err(|e| {
            error!("删除签到失败: {:?}", e);
            AppError::InternalServerError("删除签到失败".to_string())
        })
    }

    // 添加签到点赞
    #[allow(dead_code)]
    pub async fn add_checkin_like(&self, checkin_id: Uuid, user_id: i64) -> Result<(), AppError> {
        self.repo.add_checkin_like(checkin_id, user_id).await.map_err(|e| {
            error!("添加签到点赞失败: {:?}", e);
            AppError::InternalServerError("添加签到点赞失败".to_string())
        })
    }

    // 移除签到点赞
    #[allow(dead_code)]
    pub async fn remove_checkin_like(
        &self,
        checkin_id: Uuid,
        user_id: i64,
    ) -> Result<(), AppError> {
        self.repo.remove_checkin_like(checkin_id, user_id).await.map_err(|e| {
            error!("移除签到点赞失败: {:?}", e);
            AppError::InternalServerError("移除签到点赞失败".to_string())
        })
    }
}
