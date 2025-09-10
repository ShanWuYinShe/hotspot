use super::{Controller, map_domain_error_to_http_response, map_application_error_to_http_response, ok_response, paginated_response};
use crate::application::{CommandHandler, QueryHandler};
use crate::presentation::http::auth::CurrentUser;
/// 生活瞬间HTTP控制器
/// 
/// 实现hotspot.md中定义的8个生活瞬间API端点
use axum::{Json, response::Response, extract::State};
use serde::{Deserialize, Serialize};
use crate::utils::JsSafeI64;

// MomentApplicationService 通过泛型参数使用，不需要直接导入
use crate::application::commands::{CreateMomentCommand, DeleteMomentCommand, LikeMomentCommand, UnlikeMomentCommand};
use crate::application::queries::{SearchMomentsByTagsQuery, SearchMomentsByIdQuery, SearchMomentsByLocationQuery, SearchMomentsByUserIdQuery};

#[derive(Clone)]
pub struct MomentController<MomentAppService> {
    moment_service: MomentAppService,
}

impl<MomentAppService> MomentController<MomentAppService>
where
    MomentAppService: CommandHandler<CreateMomentCommand, Result = crate::domain::Moment>
        + CommandHandler<DeleteMomentCommand, Result = ()>
        + QueryHandler<
            SearchMomentsByUserIdQuery,
            Result = crate::application::PaginatedResult<crate::domain::Moment>,
        > + QueryHandler<
            SearchMomentsByLocationQuery,
            Result = crate::application::PaginatedResult<crate::domain::Moment>,
        > + QueryHandler<
            SearchMomentsByTagsQuery,
            Result = crate::application::PaginatedResult<crate::domain::Moment>,
        > + QueryHandler<
            SearchMomentsByIdQuery,
            Result = Option<crate::domain::Moment>,
        > + CommandHandler<LikeMomentCommand, Result = ()>
        + CommandHandler<UnlikeMomentCommand, Result = ()> + Clone
        + Send
        + Sync
        + 'static,
{
    pub fn new(moment_service: MomentAppService) -> Self {
        Self { moment_service }
    }

    pub async fn create(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<CreateMomentRequest>,
    ) -> Response {
        let cmd = CreateMomentCommand {
            user_id: current_user.user_id,
            content: req.description.unwrap_or_default(),
            latitude: Some(req.latitude),
            longitude: Some(req.longitude),
            address: req.location_name.clone(), // 使用请求中的位置名称
            tags: req.tags,
            image_urls: req.image_urls,
        };
        match CommandHandler::<CreateMomentCommand>::handle(&controller.moment_service, cmd).await {
            Ok(m) => ok_response(m),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    /// 获取生活瞬间历史
    pub async fn history(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<ListUserMomentsRequest>,
    ) -> Response {
        let user_id = req.user_id.and_then(|id| id.parse().ok()).unwrap_or(current_user.user_id);
        let q = SearchMomentsByUserIdQuery {
            user_id,
            requester_id: current_user.user_id,
            pagination: crate::application::queries::PaginationQuery {
                cursor: req.cursor.and_then(|c| c.parse().ok()),
                limit: req.limit.map(|l| l as usize),
            },
        };
        match QueryHandler::<SearchMomentsByUserIdQuery>::handle(&controller.moment_service, q).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    /// 按位置搜索生活瞬间
    pub async fn search_by_location(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<MomentSearchByLocationRequest>,
    ) -> Response {
        let q = SearchMomentsByLocationQuery {
            latitude: req.latitude,
            longitude: req.longitude,
            radius_km: req.radius / 1000.0, // 转换米为公里
            requester_id: current_user.user_id,
            pagination: crate::application::queries::PaginationQuery {
                cursor: req.cursor.and_then(|c| c.parse().ok()),
                limit: req.limit.map(|l| l as usize),
            },
        };
        match QueryHandler::<SearchMomentsByLocationQuery>::handle(&controller.moment_service, q).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    pub async fn delete(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(req): Json<DeleteMomentRequest>,
    ) -> Response {
        // 简单权限校验：瞬间作者才能删除
        let cmd = DeleteMomentCommand {
            moment_id: req.moment_id.value(),
            requester_id: current_user.user_id,
        };
        match CommandHandler::<DeleteMomentCommand>::handle(&controller.moment_service, cmd).await {
            Ok(_) => ok_response(serde_json::json!({"deleted": true})),
            Err(e) => map_application_error_to_http_response(e),
        }
    }

    /// 按标签搜索生活瞬间
    pub async fn search_by_tags(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<SearchByTagsRequest>,
    ) -> Response {
        let query = SearchMomentsByTagsQuery {
            tags: request.tags,
            requester_id: current_user.user_id,
            pagination: crate::application::queries::PaginationQuery {
                cursor: request.cursor.and_then(|c| c.parse().ok()),
                limit: request.limit,
            },
        };
        
        match QueryHandler::<SearchMomentsByTagsQuery>::handle(&controller.moment_service, query).await {
            Ok(result) => paginated_response(result.items, result.has_more, result.next_cursor),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }

    /// 按ID搜索生活瞬间
    pub async fn search_by_id(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<MomentSearchByIdRequest>,
    ) -> Response {
        let query = SearchMomentsByIdQuery {
            moment_id: request.moment_id.value(),
            requester_id: current_user.user_id,
        };
        
        match QueryHandler::<SearchMomentsByIdQuery>::handle(&controller.moment_service, query).await {
            Ok(Some(moment)) => ok_response(moment),
            Ok(None) => map_domain_error_to_http_response(
                crate::domain::DomainError::MomentNotFound
            ),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }

    /// 点赞生活瞬间
    pub async fn like_moment(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<LikeMomentRequest>,
    ) -> Response {
        let command = LikeMomentCommand {
            moment_id: request.moment_id.value(),
            user_id: current_user.user_id,
        };
        
        match CommandHandler::<LikeMomentCommand>::handle(&controller.moment_service, command).await {
            Ok(_) => ok_response(serde_json::json!({
                "moment_id": request.moment_id.value(),
                "liked": true
            })),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }

    /// 取消点赞生活瞬间
    pub async fn unlike_moment(
        State(controller): State<Self>,
        axum::extract::Extension(current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<UnlikeMomentRequest>,
    ) -> Response {
        let command = UnlikeMomentCommand {
            moment_id: request.moment_id.value(),
            user_id: current_user.user_id,
        };
        
        match CommandHandler::<UnlikeMomentCommand>::handle(&controller.moment_service, command).await {
            Ok(_) => ok_response(serde_json::json!({
                "moment_id": request.moment_id.value(),
                "liked": false
            })),
            Err(error) => map_domain_error_to_http_response(error),
        }
    }
}

#[async_trait::async_trait]
impl<T> Controller for MomentController<T>
where
    T: Send + Sync,
{
    fn name(&self) -> &'static str {
        "MomentController"
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateMomentRequest {
    pub longitude: f64, // 必填
    pub latitude: f64, // 必填
    pub description: Option<String>, // 可选，≤500字符
    pub location_name: Option<String>, // 可选，位置名称
    pub tags: Option<Vec<String>>, // 可选，单个标签1-20字符，总数≤10
    pub image_urls: Option<Vec<String>>, // 可选，总数≤9
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListUserMomentsRequest {
    pub user_id: Option<String>,
    pub cursor: Option<String>,
    pub limit: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MomentSearchByLocationRequest {
    pub longitude: f64,
    pub latitude: f64,
    pub radius: f64, // 以米为单位
    pub cursor: Option<String>,
    pub limit: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteMomentRequest {
    pub moment_id: JsSafeI64, // 注意：hotspot.md要求使用UUID，但当前实现使用i64
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchByTagsRequest {
    pub tags: Vec<String>, // 至少1个，最多10个
    pub cursor: Option<String>,
    pub limit: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MomentSearchByIdRequest {
    pub moment_id: JsSafeI64, // 注意：hotspot.md要求使用UUID，但当前实现使用i64
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LikeMomentRequest {
    pub moment_id: JsSafeI64, // 注意：hotspot.md要求使用UUID，但当前实现使用i64
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnlikeMomentRequest {
    pub moment_id: JsSafeI64, // 注意：hotspot.md要求使用UUID，但当前实现使用i64
}
