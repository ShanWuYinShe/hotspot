use axum::{
    Router,
    body::Body,
    http::{Request, StatusCode},
    routing::post,
};
use serde_json::{Value, json};
use tower::util::ServiceExt;

use crate::presentation::http::auth::CurrentUser;
use crate::presentation::http::controllers::AdminController;
use crate::utils::{UserIdPartition, UserType};

fn build_admin_app(controller: AdminController) -> Router {
    Router::new()
        .route("/user-id/validate", post(AdminController::validate_user_id))
        .route(
            "/user-id/generate",
            post(AdminController::generate_user_ids),
        )
        .route("/user-id/format", post(AdminController::format_user_id))
        .route(
            "/user-id/ranges",
            post(AdminController::get_user_type_ranges),
        )
        .route("/user-id/stats", post(AdminController::get_partition_stats))
        .route("/user-id/info", post(AdminController::get_user_id_info))
        .route(
            "/user-id/next/regular",
            post(AdminController::next_regular_sequence),
        )
        .route(
            "/user-id/next/temporary",
            post(AdminController::next_temporary_sequence),
        )
        .route(
            "/user-id/next/admin",
            post(AdminController::next_admin_sequence),
        )
        .route(
            "/user-id/next/system",
            post(AdminController::next_system_sequence),
        )
        .route(
            "/user-id/preview-allocate",
            post(AdminController::preview_allocate_user_ids),
        )
        .with_state(controller)
}

async fn post_with_current_user(
    app: &Router,
    path: &str,
    payload: Value,
    current_user: CurrentUser,
) -> (StatusCode, Value) {
    let body_bytes = serde_json::to_vec(&payload).unwrap();
    let mut req = Request::builder()
        .method("POST")
        .uri(path)
        .header("content-type", "application/json")
        .body(Body::from(body_bytes))
        .unwrap();
    req.extensions_mut().insert::<CurrentUser>(current_user);

    let resp = app.clone().oneshot(req).await.unwrap();
    let status = resp.status();
    let bytes = axum::body::to_bytes(resp.into_body(), 1024 * 1024)
        .await
        .unwrap();
    let val: Value = serde_json::from_slice(&bytes).unwrap();
    (status, val)
}

fn get_admin_user() -> i64 {
    // 生成一个管理员类型的用户ID，保证具备管理员权限
    UserIdPartition::generate_user_id(UserType::System, 42).expect("generate system id")
}

#[tokio::test]
async fn test_validate_user_id_success() {
    let app = build_admin_app(AdminController::new());
    let target_id = UserIdPartition::generate_user_id(UserType::Regular, 1).unwrap();
    let (status, body) = post_with_current_user(
        &app,
        "/user-id/validate",
        json!({ "user_id": target_id }),
        CurrentUser {
            user_id: get_admin_user(),
            is_temporary: false,
            nickname: None,
        },
    )
    .await;

    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["code"], 0);
    assert_eq!(body["msg"], "success");
    assert_eq!(body["data"]["is_valid"], true);
    assert_eq!(body["data"]["user_type_key"], "regular");
    assert!(body["data"]["formatted"].as_str().is_some());
    assert!(body["data"]["sequence"].as_i64().is_some());
}

#[tokio::test]
async fn test_validate_user_id_invalid() {
    let app = build_admin_app(AdminController::new());
    // 使用非法ID（解析失败）
    let (status, body) = post_with_current_user(
        &app,
        "/user-id/validate",
        json!({ "user_id": 0 }),
        CurrentUser {
            user_id: get_admin_user(),
            is_temporary: false,
            nickname: None,
        },
    )
    .await;

    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["code"], 0);
    assert_eq!(body["msg"], "success");
    assert_eq!(body["data"]["is_valid"], false);
    assert!(body["data"]["error"].as_str().is_some());
}

#[tokio::test]
async fn test_format_user_id() {
    let app = build_admin_app(AdminController::new());
    let uid = UserIdPartition::generate_user_id(UserType::Temporary, 5).unwrap();

    let (status, body) = post_with_current_user(
        &app,
        "/user-id/format",
        json!({ "user_id": uid }),
        CurrentUser {
            user_id: get_admin_user(),
            is_temporary: false,
            nickname: None,
        },
    )
    .await;

    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["code"], 0);
    assert_eq!(body["msg"], "success");
    assert_eq!(body["data"]["user_id"], uid);
    assert_eq!(body["data"]["user_type_key"], "temporary");
    assert!(body["data"]["formatted"].as_str().is_some());
}

#[tokio::test]
async fn test_generate_user_ids() {
    let app = build_admin_app(AdminController::new());

    let (status, body) = post_with_current_user(
        &app,
        "/user-id/generate",
        json!({ "user_type": "admin", "count": 3, "start_sequence": 1 }),
        CurrentUser {
            user_id: get_admin_user(),
            is_temporary: false,
            nickname: None,
        },
    )
    .await;

    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["code"], 0);
    assert_eq!(body["msg"], "success");
    assert_eq!(body["data"]["user_type_key"], "admin");
    assert_eq!(body["data"]["success_count"], 3);
    assert_eq!(body["data"]["error_count"], 0);
    assert_eq!(body["data"]["generated_ids"].as_array().unwrap().len(), 3);
}

#[tokio::test]
async fn test_get_user_type_ranges() {
    let app = build_admin_app(AdminController::new());

    let (status, body) = post_with_current_user(
        &app,
        "/user-id/ranges",
        json!({}),
        CurrentUser {
            user_id: get_admin_user(),
            is_temporary: false,
            nickname: None,
        },
    )
    .await;

    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["code"], 0);
    assert_eq!(body["msg"], "success");
    let arr = body["data"].as_array().unwrap();
    assert_eq!(arr.len(), 4);
    let keys: Vec<String> = arr
        .iter()
        .map(|v| v["user_type_key"].as_str().unwrap().to_string())
        .collect();
    assert!(keys.contains(&"regular".to_string()));
    assert!(keys.contains(&"temporary".to_string()));
    assert!(keys.contains(&"admin".to_string()));
    assert!(keys.contains(&"system".to_string()));
}

#[tokio::test]
async fn test_get_partition_stats_without_repo_fallback() {
    // 不注入 repo，统计回退为0，但依然返回 ranges 定义
    let app = build_admin_app(AdminController::new());

    let (status, body) = post_with_current_user(
        &app,
        "/user-id/stats",
        json!({}),
        CurrentUser {
            user_id: get_admin_user(),
            is_temporary: false,
            nickname: None,
        },
    )
    .await;

    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["code"], 0);
    assert_eq!(body["msg"], "success");
    assert_eq!(body["data"]["total_partitions"], 4);
    assert_eq!(body["data"]["active_partitions"], 4);
    // 回退统计应存在并为数字（大概率为0）
    assert!(body["data"]["total_users"].is_i64());
    assert!(body["data"]["regular_users"].is_i64());
    assert!(body["data"]["temporary_users"].is_i64());
    assert!(body["data"]["admin_users"].is_i64());
    assert!(body["data"]["system_users"].is_i64());
    assert_eq!(body["data"]["ranges"].as_array().unwrap().len(), 4);
}

#[tokio::test]
async fn test_next_regular_sequence() {
    let app = build_admin_app(AdminController::new());

    let (status, body) = post_with_current_user(
        &app,
        "/user-id/next/regular",
        json!({}),
        CurrentUser {
            user_id: get_admin_user(),
            is_temporary: false,
            nickname: None,
        },
    )
    .await;

    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["code"], 0);
    assert_eq!(body["msg"], "success");
    assert_eq!(body["data"]["user_type_key"], "regular");
    assert!(body["data"]["next_sequence"].is_i64());
    assert!(body["data"]["next_user_id"].is_i64());
    assert!(body["data"]["formatted"].as_str().is_some());
    assert!(body["data"]["capacity_remaining"].is_i64());
}

#[tokio::test]
async fn test_preview_allocate_admin() {
    let app = build_admin_app(AdminController::new());

    let (status, body) = post_with_current_user(
        &app,
        "/user-id/preview-allocate",
        json!({ "user_type": "admin", "count": 5 }),
        CurrentUser {
            user_id: get_admin_user(),
            is_temporary: false,
            nickname: None,
        },
    )
    .await;

    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["code"], 0);
    assert_eq!(body["msg"], "success");
    assert_eq!(body["data"]["user_type_key"], "admin");
    assert!(body["data"]["start_sequence"].is_i64());
    assert!(body["data"]["capacity_remaining"].is_i64());

    let arr = body["data"]["preview_ids"].as_array().unwrap();
    assert_eq!(arr.len(), 5);
    for item in arr {
        assert!(item["user_id"].is_i64());
        assert!(item["sequence"].is_i64());
        assert!(item["formatted"].as_str().is_some());
    }
}

#[tokio::test]
async fn test_get_user_id_info() {
    let app = build_admin_app(AdminController::new());
    let uid = UserIdPartition::generate_user_id(UserType::System, 7).unwrap();

    let (status, body) = post_with_current_user(
        &app,
        "/user-id/info",
        json!({ "user_id": uid }),
        CurrentUser {
            user_id: get_admin_user(),
            is_temporary: false,
            nickname: None,
        },
    )
    .await;

    assert_eq!(status, StatusCode::OK);
    assert_eq!(body["code"], 0);
    assert_eq!(body["msg"], "success");
    assert_eq!(body["data"]["user_id"], uid);
    assert_eq!(body["data"]["user_type_key"], "system");
    assert_eq!(body["data"]["user_type_name"], "系统用户");
    assert!(body["data"]["sequence"].is_i64());
    assert!(body["data"]["formatted"].as_str().is_some());
}
