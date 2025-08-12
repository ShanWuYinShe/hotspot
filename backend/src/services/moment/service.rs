use std::sync::Arc;
use tracing::error;
use uuid::Uuid;

use crate::{
    repositories::{
        moment_repository::{MomentRow, NewMomentData},
        PgMomentRepository,
        RepositoryError
    },
    utils::error::AppError
};

pub struct MomentService {
    repo: Arc<PgMomentRepository>,
}

impl MomentService {
    pub fn new(repo: Arc<PgMomentRepository>) -> Self {
        Self { repo }
    }

    

    // 创建动态
    pub async fn create_moment(
        &self,
        user_id: i64,
        content: &str,
        latitude: f64,
        longitude: f64,
        location_name: &str,
        tags: &[String],
    ) -> Result<String, AppError> {
        let data = NewMomentData {
            user_id,
            content: content.to_string(),
            latitude,
            longitude,
            location_name: location_name.to_string(),
            tags: tags.to_vec(),
        };
        
        self.repo.create_moment(data).await.map_err(|e| {
            error!("创建动态失败: {:?}", e);
            AppError::InternalServerError("创建动态失败".to_string())
        })
    }

    // 点赞动态
    pub async fn like_moment(
        &self,
        moment_id: &str,
        user_id: i64,
    ) -> Result<(), AppError> {
        let _uuid_moment_id = moment_id.parse::<Uuid>().map_err(|_| {
            AppError::BadRequest("无效的动态ID格式".to_string())
        })?;

        self.repo.like_moment(moment_id, user_id).await.map_err(|e| {
            error!("添加动态点赞失败: {:?}", e);
            AppError::InternalServerError("添加动态点赞失败".to_string())
        })?;

        Ok(())
    }

    // 取消点赞动态
    pub async fn unlike_moment(
        &self,
        moment_id: &str,
        user_id: i64,
    ) -> Result<(), AppError> {
        let _uuid_moment_id = moment_id.parse::<Uuid>().map_err(|_| {
            AppError::BadRequest("无效的动态ID格式".to_string())
        })?;

        self.repo.unlike_moment(moment_id, user_id).await.map_err(|e| {
            error!("移除动态点赞失败: {:?}", e);
            AppError::InternalServerError("移除动态点赞失败".to_string())
        })?;

        Ok(())
    }

    // 获取动态详情
    pub async fn search_moment_by_id(
        &self,
        user_id: i64,
        moment_id: &str,
    ) -> Result<MomentRow, AppError> {
        // 使用仓库方法获取动态信息
        let result = self.repo.search_moment_by_id(moment_id, user_id).await.map_err(|e| {
            error!("查询动态失败: {:?}", e);
            match e {
                RepositoryError::NotFound => AppError::NotFound("动态不存在".to_string()),
                _ => AppError::InternalServerError("查询动态失败".to_string()),
            }
        })?;

        match result {
            Some(moment) => Ok(moment),
            None => Err(AppError::NotFound("动态不存在".to_string())),
        }
    }

    // 按用户获取动态列表
    pub async fn search_moments_by_user(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i64,
    ) -> Result<Vec<MomentRow>, AppError> {
        self.repo.search_moments_by_user(user_id, cursor, limit as i32).await.map_err(|e| {
            error!("查询用户动态列表失败: {:?}", e);
            AppError::InternalServerError("查询用户动态列表失败".to_string())
        })
    }

    // 按位置搜索动态
    pub async fn search_moments_by_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        radius: f64,
        cursor: i64,
        limit: i64,
    ) -> Result<Vec<MomentRow>, AppError> {
        self.repo.search_moments_by_location(user_id, latitude, longitude, radius, cursor, limit as i32).await.map_err(|e| {
            error!("按位置搜索动态失败: {:?}", e);
            AppError::InternalServerError("按位置搜索动态失败".to_string())
        })
    }

    // 按标签搜索动态
    pub async fn search_moments_by_tags(
        &self,
        user_id: i64,
        tags: &[String],
        cursor: i64,
        limit: i64,
    ) -> Result<Vec<MomentRow>, AppError> {
        self.repo.search_moments_by_tags(user_id, tags.to_vec(), cursor, limit as i32).await.map_err(|e| {
            error!("按标签搜索动态失败: {:?}", e);
            AppError::InternalServerError("按标签搜索动态失败".to_string())
        })
    }

    // 删除动态
    pub async fn delete_moment(&self, moment_id: &str) -> Result<(), AppError> {
        self.repo.delete_moment_record(moment_id).await.map_err(|e| {
            error!("删除动态失败: {:?}", e);
            AppError::InternalServerError("删除动态失败".to_string())
        })?;

        Ok(())
    }
}
