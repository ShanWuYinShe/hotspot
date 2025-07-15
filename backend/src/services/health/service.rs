//! 健康检查服务
use std::sync::Arc;


use crate::{
    utils::response::AppError,
    utils::result_ext::ResultExt,
};

/// 健康检查服务
#[derive(Debug, Clone)]
pub struct HealthService {
    repo: Arc<crate::repositories::PgHealthRepository>,
}

impl HealthService {
    /// 创建新的健康检查服务
    pub fn new(repo: Arc<crate::repositories::PgHealthRepository>) -> Self {
        Self { repo }
    }
    
    /// 检查数据库连接
    pub async fn check_database(&self) -> Result<bool, AppError> {
        self.repo.check_database().await.map_repository_error("数据库健康检查")
    }
    
    /// 检查Redis连接
    pub async fn check_redis(&self) -> Result<bool, AppError> {
        self.repo.check_redis().await.map_repository_error("Redis健康检查")
    }
}
