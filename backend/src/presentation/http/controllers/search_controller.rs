/// 搜索控制器
/// 
/// 处理全局搜索相关的HTTP请求
use axum::{extract::State, response::Response, Json};
use serde::Deserialize;
use serde_json::json;
use tracing::{info, warn, error};
use std::sync::Arc;

use crate::{
    application::{SearchApplicationService, SearchByGlobalQuery, SearchType},
    presentation::http::controllers::{Controller, ok_response, map_domain_error_to_http_response},
    presentation::http::auth::CurrentUser,
};

/// 全局搜索请求参数
#[derive(Debug, Deserialize)]
pub struct SearchByGlobalRequest {
    /// 搜索关键词
    pub q: String,
    /// 搜索类型
    pub type_: Option<String>,
    /// 分页游标
    pub cursor: Option<i64>,
    /// 每页数量
    pub limit: Option<i32>,
    /// 用户位置（用于地理位置相关的搜索排序）
    pub lat: Option<f64>,
    pub lng: Option<f64>,
}

/// 搜索建议请求参数
#[derive(Debug, Deserialize)]
pub struct SearchBySuggestionsRequest {
    /// 搜索关键词
    pub q: String,
    /// 建议数量限制
    pub limit: Option<i32>,
}

/// 搜索控制器
#[derive(Clone)]
pub struct SearchController {
    search_service: Arc<dyn SearchApplicationService>,
}

impl SearchController {
    /// 创建新的搜索控制器
    pub fn new(search_service: Arc<dyn SearchApplicationService>) -> Self {
        Self { search_service }
    }
}

impl Controller for SearchController {
    fn name(&self) -> &'static str {
        "SearchController"
    }
}

impl SearchController {
    /// 全局搜索
    pub async fn search_by_global(
        State(controller): State<SearchController>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(params): Json<SearchByGlobalRequest>,
    ) -> Response {
        info!("用户 {} 执行全局搜索: '{}'", current_user.user_id, params.q);
        
        // 解析搜索类型
        let search_type = match params.type_.as_deref() {
            Some("users") => Some(SearchType::Users),
            Some("groups") => Some(SearchType::Groups),
            Some("moments") => Some(SearchType::Moments),
            Some("all") | None => Some(SearchType::All),
            Some(unknown) => {
                warn!("未知的搜索类型: {}", unknown);
                Some(SearchType::All)
            }
        };
        
        // 构建搜索查询
        let search_query = SearchByGlobalQuery {
            query: params.q.clone(),
            search_type,
            pagination: crate::application::queries::PaginationQuery {
                cursor: params.cursor,
                limit: params.limit,
            },
            latitude: params.lat,
            longitude: params.lng,
        };
        
        // 执行搜索
        match controller.search_service.global_search(search_query, current_user.user_id).await {
            Ok(result) => {
                info!("搜索成功");
                
                let response = json!({
                    "code": 0,
                    "msg": "搜索成功",
                    "data": result
                });
                
                ok_response(response)
            }
            Err(e) => {
                error!("搜索失败: {:?}", e);
                
                // 直接使用统一的错误处理
                
                map_domain_error_to_http_response(e)
            }
        }
    }
    
    /// 搜索建议（快速搜索）
    pub async fn search_by_suggestions(
        State(controller): State<SearchController>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(params): Json<SearchBySuggestionsRequest>,
    ) -> Response {
        info!("用户 {} 请求搜索建议: '{}'", current_user.user_id, params.q);
        
        // 如果查询太短，返回空建议
        if params.q.trim().len() < 2 {
            let response = json!({
                "code": 0,
                "msg": "查询太短",
                "data": {
                    "suggestions": []
                }
            });
            return ok_response(response);
        }
        
        // 构建限制性搜索查询（只返回前5个结果）
        let search_query = SearchByGlobalQuery {
            query: params.q.clone(),
            search_type: Some(SearchType::All),
            pagination: crate::application::queries::PaginationQuery {
                cursor: None,
                limit: params.limit.or(Some(5)), // 默认返回5个建议
            },
            latitude: None, // 建议搜索不需要位置信息
            longitude: None,
        };
        
        match controller.search_service.global_search(search_query, current_user.user_id).await {
            Ok(result) => {
                // 简化建议格式，直接返回搜索结果
                let response = json!({
                    "code": 0,
                    "msg": "获取建议成功",
                    "data": {
                        "suggestions": result
                    }
                });
                
                ok_response(response)
            }
            Err(e) => {
                warn!("获取搜索建议失败: {:?}", e);
                
                // 搜索建议失败时返回空结果，不影响用户体验
                let response = json!({
                    "code": 0,
                    "msg": "暂无建议",
                    "data": {
                        "suggestions": []
                    }
                });
                
                ok_response(response)
            }
        }
    }
}
