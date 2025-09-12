/// 活动控制器
/// 
/// 处理系统活动数据相关的HTTP请求
use axum::{extract::Query, response::{Response, IntoResponse}, Json};
use serde::Deserialize;
use serde_json::json;
use tracing::{info, warn, error};

use crate::{
    application::{ActivityApplicationService, GetRecentActivitiesQuery, ActivityType},
    presentation::http::controllers::{Controller, ok_response, map_domain_error_to_http_response},
    presentation::http::auth::CurrentUser,
};

/// 获取最近活动请求参数
#[derive(Debug, Deserialize)]
pub struct GetRecentActivitiesRequest {
    /// 分页游标
    pub cursor: Option<i64>,
    /// 每页数量
    pub limit: Option<i32>,
    /// 活动类型过滤（逗号分隔）
    pub types: Option<String>,
    /// 时间范围（小时）
    pub hours: Option<i32>,
}

/// 活动控制器
#[derive(Clone)]
pub struct ActivityController<U, G, M> 
where
    U: crate::ports::UserRepository,
    G: crate::ports::GroupRepository,
    M: crate::ports::MomentRepository,
{
    activity_service: ActivityApplicationService<U, G, M>,
}

impl<U, G, M> ActivityController<U, G, M>
where
    U: crate::ports::UserRepository,
    G: crate::ports::GroupRepository,
    M: crate::ports::MomentRepository,
{
    /// 创建新的活动控制器
    pub fn new(activity_service: ActivityApplicationService<U, G, M>) -> Self {
        Self { activity_service }
    }
}

impl<U, G, M> Controller for ActivityController<U, G, M>
where
    U: crate::ports::UserRepository,
    G: crate::ports::GroupRepository,
    M: crate::ports::MomentRepository,
{
    fn name(&self) -> &'static str {
        "ActivityController"
    }
}

impl<U, G, M> ActivityController<U, G, M>
where
    U: crate::ports::UserRepository,
    G: crate::ports::GroupRepository,
    M: crate::ports::MomentRepository,
{
    /// 获取最近活动
    pub async fn get_recent_activities(
        axum::extract::State(controller): axum::extract::State<ActivityController<U, G, M>>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Query(params): Query<GetRecentActivitiesRequest>,
    ) -> Response {
        info!("用户 {} 请求最近活动", current_user.user_id);
        
        // 解析活动类型过滤
        let activity_types = if let Some(types_str) = params.types {
            let types: Result<Vec<ActivityType>, _> = types_str
                .split(',')
                .map(|s| match s.trim() {
                    "user_registered" => Ok(ActivityType::UserRegistered),
                    "group_created" => Ok(ActivityType::GroupCreated),
                    "group_joined" => Ok(ActivityType::GroupJoined),
                    "moment_created" => Ok(ActivityType::MomentCreated),
                    "moment_liked" => Ok(ActivityType::MomentLiked),
                    "location_updated" => Ok(ActivityType::LocationUpdated),
                    _ => Err(format!("未知的活动类型: {}", s)),
                })
                .collect();
            
            match types {
                Ok(types) if !types.is_empty() => Some(types),
                Ok(_) => None,
                Err(e) => {
                    warn!("解析活动类型失败: {}", e);
                    let error_response = json!({
                        "code": 1000,
                        "msg": format!("无效的活动类型: {}", e),
                        "data": null
                    });
                    return (axum::http::StatusCode::OK, Json(error_response)).into_response();
                }
            }
        } else {
            None
        };
        
        // 构建查询
        let query = GetRecentActivitiesQuery {
            pagination: crate::application::queries::PaginationQuery {
                cursor: params.cursor,
                limit: params.limit.map(|l| l as usize),
            },
            activity_types,
            hours_range: params.hours,
        };
        
        // 执行查询
        match controller.activity_service.get_recent_activities(query).await {
            Ok(result) => {
                info!("获取活动成功: 找到{}个活动", result.activities.len());
                
                // 转换为前端需要的格式
                let frontend_activities: Vec<serde_json::Value> = result.activities
                    .into_iter()
                    .map(|activity| activity.into())
                    .collect();
                
                let response = json!({
                    "code": 0,
                    "msg": "获取活动成功",
                    "data": {
                        "activities": frontend_activities,
                        "has_more": result.has_more,
                        "next_cursor": result.next_cursor,
                        "total_count": result.total_count
                    }
                });
                
                ok_response(response)
            }
            Err(e) => {
                error!("获取活动失败: {:?}", e);
                
                // 直接使用统一的错误处理
                
                map_domain_error_to_http_response(e)
            }
        }
    }
    
    /// 获取演示活动数据
    pub async fn get_demo_activities(
        axum::extract::State(controller): axum::extract::State<ActivityController<U, G, M>>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
    ) -> Response {
        info!("用户 {} 请求演示活动数据", current_user.user_id);
        
        match controller.activity_service.generate_demo_activities().await {
            Ok(result) => {
                info!("生成演示活动成功: {}个活动", result.activities.len());
                
                // 转换为前端需要的格式
                let frontend_activities: Vec<serde_json::Value> = result.activities
                    .into_iter()
                    .map(|activity| activity.into())
                    .collect();
                
                let response = json!({
                    "code": 0,
                    "msg": "获取演示活动成功",
                    "data": {
                        "activities": frontend_activities,
                        "has_more": result.has_more,
                        "next_cursor": result.next_cursor,
                        "total_count": result.total_count
                    }
                });
                
                ok_response(response)
            }
            Err(e) => {
                error!("生成演示活动失败: {:?}", e);
                {
                    let error_response = json!({
                        "code": 5000,
                        "msg": "生成演示活动失败",
                        "data": null
                    });
                    (axum::http::StatusCode::OK, Json(error_response)).into_response()
                }
            }
        }
    }
}
