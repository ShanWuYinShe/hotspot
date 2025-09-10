use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use axum::{Extension, Json, extract::State};
use serde::{Deserialize, Serialize};
use crate::utils::JsSafeI64;

use crate::presentation::http::auth::CurrentUser;
use crate::presentation::http::controllers::ok_response;
use crate::utils::{UserIdPartition, UserType};

/// 管理端控制器：用户ID分区相关接口
/// 提供：验证、格式化、范围查询、批量生成
#[derive(Clone, Default)]
pub struct AdminController {
    pub(crate) user_repo: Option<std::sync::Arc<dyn crate::ports::UserRepository>>,
}

impl AdminController {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn with_user_repo(user_repo: std::sync::Arc<dyn crate::ports::UserRepository>) -> Self {
        Self {
            user_repo: Some(user_repo),
        }
    }

    fn is_admin_user(user_id: i64) -> bool {
        match UserIdPartition::get_user_type(user_id) {
            Ok(t) => t.is_privileged(),
            Err(_) => false,
        }
    }

    fn map_user_type_key(t: UserType) -> &'static str {
        match t {
            UserType::Regular => "regular",
            UserType::Temporary => "temporary",
            UserType::Admin => "admin",
            UserType::System => "system",
        }
    }

    fn bad_request(msg: &str) -> Response {
        let body = serde_json::json!({
            "code": 1,
            "msg": msg,
            "data": null
        });
        (StatusCode::OK, Json(body)).into_response()
    }

    fn forbidden(msg: &str) -> Response {
        let body = serde_json::json!({
            "code": 1,
            "msg": msg,
            "data": null
        });
        (StatusCode::OK, Json(body)).into_response()
    }
}

/// 验证用户ID请求
#[derive(Debug, Clone, Deserialize)]
pub struct ValidateUserIdRequest {
    pub user_id: JsSafeI64,
}

/// 验证用户ID响应
#[derive(Debug, Clone, Serialize)]
pub struct ValidateUserIdResponse {
    pub is_valid: bool,
    pub user_type_key: Option<String>,
    pub user_type_name: Option<String>,
    pub sequence: Option<i64>,
    pub formatted: Option<String>,
    pub error: Option<String>,
}

/// 批量生成用户ID请求
#[derive(Debug, Clone, Deserialize)]
pub struct GenerateUserIdsRequest {
    /// 用户类型："regular" | "temporary" | "admin" | "system"
    pub user_type: String,
    /// 生成数量，最多100
    pub count: usize,
    /// 起始序列号（默认1）
    pub start_sequence: Option<i64>,
}

/// 生成的用户ID信息
#[derive(Debug, Clone, Serialize)]
pub struct GeneratedUserId {
    pub user_id: JsSafeI64,
    pub sequence: JsSafeI64,
    pub formatted: String,
}

/// 批量生成用户ID响应
#[derive(Debug, Clone, Serialize)]
pub struct GenerateUserIdsResponse {
    pub user_type_key: String,
    pub generated_ids: Vec<GeneratedUserId>,
    pub success_count: usize,
    pub error_count: usize,
}

/// 格式化用户ID请求
#[derive(Debug, Clone, Deserialize)]
pub struct FormatUserIdRequest {
    pub user_id: JsSafeI64,
}

/// 格式化用户ID响应
#[derive(Debug, Clone, Serialize)]
pub struct FormatUserIdResponse {
    pub user_id: JsSafeI64,
    pub formatted: String,
    pub user_type_key: String,
    pub user_type_name: String,
}

/// 用户类型范围信息
#[derive(Debug, Clone, Serialize)]
pub struct UserTypeRangeInfo {
    pub user_type_key: String,
    pub user_type_name: String,
    pub identifier: u16,
    pub min_id: JsSafeI64,
    pub max_id: JsSafeI64,
    pub capacity: JsSafeI64,
}

/// 分区统计信息（不访问数据库，仅根据ID分区定义计算）
#[derive(Debug, Clone, Serialize)]
pub struct UserIdPartitionStats {
    pub total_partitions: usize,
    pub active_partitions: usize,
    pub total_users: JsSafeI64,
    pub regular_users: JsSafeI64,
    pub temporary_users: JsSafeI64,
    pub admin_users: JsSafeI64,
    pub system_users: JsSafeI64,
    pub ranges: Vec<UserTypeRangeInfo>,
}

/// 用户ID信息请求
#[derive(Debug, Clone, Deserialize)]
pub struct UserIdInfoRequest {
    pub user_id: JsSafeI64,
}

/// 用户ID信息响应
#[derive(Debug, Clone, Serialize)]
pub struct UserIdInfoResponse {
    pub user_id: JsSafeI64,
    pub user_type_key: String,
    pub user_type_name: String,
    pub identifier: u16,
    pub sequence: JsSafeI64,
    pub formatted: String,
}

/// 下一个可用序列号响应
#[derive(Debug, Clone, Serialize)]
pub struct NextSequenceResponse {
    pub user_type_key: String,
    pub user_type_name: String,
    pub next_sequence: JsSafeI64,
    pub next_user_id: JsSafeI64,
    pub formatted: String,
    pub capacity_remaining: JsSafeI64,
}

/// 批量预分配请求（基于DB中当前最大ID推导起始序列）
#[derive(Debug, Clone, Deserialize)]
pub struct PreviewAllocateRequest {
    /// "regular" | "temporary" | "admin" | "system"
    pub user_type: String,
    /// 需要预览的数量，最多100
    pub count: usize,
}

/// 批量预分配响应
#[derive(Debug, Clone, Serialize)]
pub struct PreviewAllocateResponse {
    pub user_type_key: String,
    pub user_type_name: String,
    pub start_sequence: JsSafeI64,
    pub preview_ids: Vec<GeneratedUserId>,
    pub capacity_remaining: JsSafeI64,
}

#[async_trait::async_trait]
impl crate::presentation::http::controllers::Controller for AdminController {
    fn name(&self) -> &'static str {
        "AdminController"
    }
}

impl AdminController {
    /// 验证用户ID有效性与类型
    pub async fn validate_user_id(
        State(_controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(req): Json<ValidateUserIdRequest>,
    ) -> Response {
        // 管理权限校验（Admin/System）
        if !Self::is_admin_user(current_user.user_id) {
            return Self::forbidden("权限不足：需要管理员权限");
        }

        match UserIdPartition::parse_user_id(req.user_id.value()) {
            Ok((t, seq)) => ok_response(ValidateUserIdResponse {
                is_valid: true,
                user_type_key: Some(Self::map_user_type_key(t).to_string()),
                user_type_name: Some(t.name().to_string()),
                sequence: Some(seq),
                formatted: Some(UserIdPartition::format_user_id(req.user_id.value())),
                error: None,
            }),
            Err(e) => ok_response(ValidateUserIdResponse {
                is_valid: false,
                user_type_key: None,
                user_type_name: None,
                sequence: None,
                formatted: None,
                error: Some(e),
            }),
        }
    }

    /// 批量生成用户ID（测试/工具用途）
    pub async fn generate_user_ids(
        State(_controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(req): Json<GenerateUserIdsRequest>,
    ) -> Response {
        if !Self::is_admin_user(current_user.user_id) {
            return Self::forbidden("权限不足：需要管理员权限");
        }

        if req.count == 0 || req.count > 100 {
            return Self::bad_request("生成数量必须在 1..=100 之间");
        }

        let user_type = match req.user_type.as_str() {
            "regular" => UserType::Regular,
            "temporary" => UserType::Temporary,
            "admin" => UserType::Admin,
            "system" => UserType::System,
            _ => return Self::bad_request("无效的用户类型（regular|temporary|admin|system）"),
        };

        let start_seq = req.start_sequence.unwrap_or(1);
        let mut generated = Vec::with_capacity(req.count);
        let mut success = 0usize;
        let mut errors = 0usize;

        for i in 0..req.count {
            let seq = start_seq + i as i64;
            match UserIdPartition::generate_user_id(user_type, seq) {
                Ok(uid) => {
                    generated.push(GeneratedUserId {
                        user_id: JsSafeI64::new(uid),
                        sequence: JsSafeI64::new(seq),
                        formatted: UserIdPartition::format_user_id(uid),
                    });
                    success += 1;
                }
                Err(_) => {
                    errors += 1;
                }
            }
        }

        ok_response(GenerateUserIdsResponse {
            user_type_key: req.user_type,
            generated_ids: generated,
            success_count: success,
            error_count: errors,
        })
    }

    /// 格式化用户ID显示
    pub async fn format_user_id(
        State(_controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(req): Json<FormatUserIdRequest>,
    ) -> Response {
        if !Self::is_admin_user(current_user.user_id) {
            return Self::forbidden("权限不足：需要管理员权限");
        }

        match UserIdPartition::parse_user_id(req.user_id.value()) {
            Ok((t, _)) => ok_response(FormatUserIdResponse {
                user_id: req.user_id,
                formatted: UserIdPartition::format_user_id(req.user_id.value()),
                user_type_key: Self::map_user_type_key(t).to_string(),
                user_type_name: t.name().to_string(),
            }),
            Err(e) => Self::bad_request(&format!("无效的用户ID: {}", e)),
        }
    }

    /// 获取所有用户类型的范围信息
    pub async fn get_user_type_ranges(
        State(_controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(_): Json<serde_json::Value>, // 对齐其它接口风格，接受空体
    ) -> Response {
        if !Self::is_admin_user(current_user.user_id) {
            return Self::forbidden("权限不足：需要管理员权限");
        }

        let types = [
            UserType::Regular,
            UserType::Temporary,
            UserType::Admin,
            UserType::System,
        ];
        let mut ranges = Vec::with_capacity(types.len());

        for t in types {
            let (min_id, max_id) = UserIdPartition::get_type_range(t);
            // 容量 = 54位序列空间，或根据范围直接计算（包含端点）
            let capacity = max_id - min_id + 1;

            ranges.push(UserTypeRangeInfo {
                user_type_key: Self::map_user_type_key(t).to_string(),
                user_type_name: t.name().to_string(),
                identifier: t.identifier(),
                min_id: JsSafeI64::new(min_id),
                max_id: JsSafeI64::new(max_id),
                capacity: JsSafeI64::new(capacity),
            });
        }

        ok_response(ranges)
    }

    /// 获取用户ID分区统计信息（实时统计 + 分区定义；若统计失败回退为0）
    pub async fn get_partition_stats(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(_): Json<serde_json::Value>,
    ) -> Response {
        if !Self::is_admin_user(current_user.user_id) {
            return Self::forbidden("权限不足：需要管理员权限");
        }

        let types = [
            UserType::Regular,
            UserType::Temporary,
            UserType::Admin,
            UserType::System,
        ];
        let mut ranges = Vec::with_capacity(types.len());

        for t in types {
            let (min_id, max_id) = UserIdPartition::get_type_range(t);
            let capacity = max_id - min_id + 1;
            ranges.push(UserTypeRangeInfo {
                user_type_key: Self::map_user_type_key(t).to_string(),
                user_type_name: t.name().to_string(),
                identifier: t.identifier(),
                min_id: JsSafeI64::new(min_id),
                max_id: JsSafeI64::new(max_id),
                capacity: JsSafeI64::new(capacity),
            });
        }

        // 统计数据（默认0，失败回退）
        let mut total_users: i64 = 0;
        let mut regular_users: i64 = 0;
        let mut temporary_users: i64 = 0;
        let mut admin_users: i64 = 0;
        let mut system_users: i64 = 0;

        if let Some(repo) = &controller.user_repo {
            if let Ok(v) = repo.get_total_user_count().await {
                total_users = v;
            }
            // 按类型统计
            let type_list = [
                UserType::Regular,
                UserType::Temporary,
                UserType::Admin,
                UserType::System,
            ];
            for t in type_list {
                let (min_id, max_id) = UserIdPartition::get_type_range(t);
                if let Ok(v) = repo.get_user_count_by_id_range(min_id, max_id).await {
                    match t {
                        UserType::Regular => regular_users = v,
                        UserType::Temporary => temporary_users = v,
                        UserType::Admin => admin_users = v,
                        UserType::System => system_users = v,
                    }
                }
            }
        }

        let stats = UserIdPartitionStats {
            total_partitions: ranges.len(),
            active_partitions: ranges.len(),
            total_users: JsSafeI64::new(total_users),
            regular_users: JsSafeI64::new(regular_users),
            temporary_users: JsSafeI64::new(temporary_users),
            admin_users: JsSafeI64::new(admin_users),
            system_users: JsSafeI64::new(system_users),
            ranges,
        };

        ok_response(stats)
    }



    /// 查询单个用户ID的类型与序列等信息
    pub async fn get_user_id_info(
        State(_controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(req): Json<UserIdInfoRequest>,
    ) -> Response {
        if !Self::is_admin_user(current_user.user_id) {
            return Self::forbidden("权限不足：需要管理员权限");
        }

        match UserIdPartition::parse_user_id(req.user_id.value()) {
            Ok((t, seq)) => {
                let resp = UserIdInfoResponse {
                    user_id: req.user_id,
                    user_type_key: Self::map_user_type_key(t).to_string(),
                    user_type_name: t.name().to_string(),
                    identifier: t.identifier(),
                    sequence: JsSafeI64::new(seq),
                    formatted: UserIdPartition::format_user_id(req.user_id.value()),
                };
                ok_response(resp)
            }
            Err(e) => Self::bad_request(&format!("无效的用户ID: {}", e)),
        }
    }

    /// 计算下一个序列与容量（含DB查询）
    async fn compute_next_from_db(
        repo: &Option<std::sync::Arc<dyn crate::ports::UserRepository>>,
        t: UserType,
    ) -> (i64, i64, i64) {
        let (min_id, max_id) = UserIdPartition::get_type_range(t);
        let mut current_max_id = min_id - 1;

        if let Some(repo) = repo {
            if let Ok(opt_max) = repo.get_max_user_id_by_range(min_id, max_id).await {
                if let Some(v) = opt_max {
                    current_max_id = v;
                }
            }
        }
        (min_id, max_id, current_max_id)
    }

    /// 统一执行：根据类型计算下一个序列并返回响应
    async fn next_sequence_core(controller: &Self, t: UserType) -> Response {
        let (_min_id, max_id, current_max_id) =
            Self::compute_next_from_db(&controller.user_repo, t).await;
        let next_seq = match UserIdPartition::next_sequence_for_type(t, current_max_id) {
            Ok(s) => s,
            Err(e) => return Self::bad_request(&format!("计算下一个序列失败: {}", e)),
        };
        let next_user_id = match UserIdPartition::generate_user_id(t, next_seq) {
            Ok(id) => id,
            Err(e) => return Self::bad_request(&format!("生成下一个用户ID失败: {}", e)),
        };
        let capacity_remaining = max_id - current_max_id;

        ok_response(NextSequenceResponse {
            user_type_key: Self::map_user_type_key(t).to_string(),
            user_type_name: t.name().to_string(),
            next_sequence: JsSafeI64::new(next_seq),
            next_user_id: JsSafeI64::new(next_user_id),
            formatted: UserIdPartition::format_user_id(next_user_id),
            capacity_remaining: JsSafeI64::new(capacity_remaining),
        })
    }

    /// 下一个序列（regular）
    pub async fn next_regular_sequence(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(_): Json<serde_json::Value>,
    ) -> Response {
        if !Self::is_admin_user(current_user.user_id) {
            return Self::forbidden("权限不足：需要管理员权限");
        }
        Self::next_sequence_core(&controller, UserType::Regular).await
    }

    /// 下一个序列（temporary）
    pub async fn next_temporary_sequence(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(_): Json<serde_json::Value>,
    ) -> Response {
        if !Self::is_admin_user(current_user.user_id) {
            return Self::forbidden("权限不足：需要管理员权限");
        }
        Self::next_sequence_core(&controller, UserType::Temporary).await
    }

    /// 下一个序列（admin）
    pub async fn next_admin_sequence(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(_): Json<serde_json::Value>,
    ) -> Response {
        if !Self::is_admin_user(current_user.user_id) {
            return Self::forbidden("权限不足：需要管理员权限");
        }
        Self::next_sequence_core(&controller, UserType::Admin).await
    }

    /// 下一个序列（system）
    pub async fn next_system_sequence(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(_): Json<serde_json::Value>,
    ) -> Response {
        if !Self::is_admin_user(current_user.user_id) {
            return Self::forbidden("权限不足：需要管理员权限");
        }
        Self::next_sequence_core(&controller, UserType::System).await
    }

    /// 基于DB当前最大ID的批量预分配预览（不落库，仅计算）
    pub async fn preview_allocate_user_ids(
        State(controller): State<Self>,
        Extension(current_user): Extension<CurrentUser>,
        Json(req): Json<PreviewAllocateRequest>,
    ) -> Response {
        if !Self::is_admin_user(current_user.user_id) {
            return Self::forbidden("权限不足：需要管理员权限");
        }

        if req.count == 0 || req.count > 100 {
            return Self::bad_request("预览数量必须在 1..=100 之间");
        }

        let t = match req.user_type.as_str() {
            "regular" => UserType::Regular,
            "temporary" => UserType::Temporary,
            "admin" => UserType::Admin,
            "system" => UserType::System,
            _ => return Self::bad_request("无效的用户类型（regular|temporary|admin|system）"),
        };

        let (_min_id, max_id, current_max_id) =
            Self::compute_next_from_db(&controller.user_repo, t).await;

        let start_sequence = match UserIdPartition::next_sequence_for_type(t, current_max_id) {
            Ok(s) => s,
            Err(e) => return Self::bad_request(&format!("计算起始序列失败: {}", e)),
        };

        // 可用容量（从当前最大ID之后到max_id，包含端点）
        let capacity_remaining = max_id - current_max_id;
        let max_preview = std::cmp::min(req.count as i64, capacity_remaining) as usize;

        let mut preview_ids = Vec::with_capacity(max_preview);
        for i in 0..max_preview {
            let seq = start_sequence + i as i64;
            match UserIdPartition::generate_user_id(t, seq) {
                Ok(uid) => {
                    preview_ids.push(GeneratedUserId {
                        user_id: JsSafeI64::new(uid),
                        sequence: JsSafeI64::new(seq),
                        formatted: UserIdPartition::format_user_id(uid),
                    });
                }
                Err(e) => {
                    return Self::bad_request(&format!("生成预览ID失败: {}", e));
                }
            }
        }

        ok_response(PreviewAllocateResponse {
            user_type_key: Self::map_user_type_key(t).to_string(),
            user_type_name: t.name().to_string(),
            start_sequence: JsSafeI64::new(start_sequence),
            preview_ids,
            capacity_remaining: JsSafeI64::new(capacity_remaining),
        })
    }
}
