use axum::{
    body::Body,
    http::{Request, StatusCode},
};
use hotspot_backend::{
    dto::{request::*, response::*},
    state::AppState,
};
use serde_json::json;
use sqlx::PgPool;
use tower::ServiceExt;

/// 集成测试：用户模块
///
/// 测试场景：
/// 1. 按昵称搜索用户
/// 2. 获取用户资料
/// 3. 更新用户资料
/// 4. 发送好友申请
/// 5. 接受好友申请
/// 6. 关注用户
/// 7. 拉黑用户

#[sqlx::test]
async fn test_search_user_by_name(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    let (app, _token, _user_id) =
        setup_authenticated_test(pool, "search_test@example.com", "搜索测试用户").await?;

    // 搜索用户
    let search_req = SearchUserByNameRequest {
        name: "搜索".to_string(),
        limit: Some(20),
        cursor: None,
    };

    let request = Request::builder()
        .method("POST")
        .uri("/api/v1/public/user/search_by_name")
        .header("content-type", "application/json")
        .body(Body::from(serde_json::to_string(&search_req)?))
        .unwrap();

    let response = app.oneshot(request).await?;

    assert_eq!(response.status(), StatusCode::OK);

    let body = hyper::body::to_bytes(response.into_body()).await?;
    let api_response: ApiResponse<UserSearchData> = serde_json::from_slice(&body)?;

    assert_eq!(api_response.code, 0);
    assert!(!api_response.data.items.is_empty());

    Ok(())
}

#[sqlx::test]
async fn test_get_user_profile(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    let (app, token, user_id) =
        setup_authenticated_test(pool, "profile_test@example.com", "资料测试用户").await?;

    // 获取自己的资料
    let request = Request::builder()
        .method("POST")
        .uri("/api/v1/protected/user/open_the_door")
        .header("content-type", "application/json")
        .header("authorization", format!("Bearer {}", token))
        .body(Body::from(serde_json::to_string(&GetUserProfileRequest {
            id: None,
        })?))
        .unwrap();

    let response = app.oneshot(request).await?;

    assert_eq!(response.status(), StatusCode::OK);

    let body = hyper::body::to_bytes(response.into_body()).await?;
    let api_response: ApiResponse<UserProfileDetailData> = serde_json::from_slice(&body)?;

    assert_eq!(api_response.code, 0);
    assert_eq!(api_response.data.user_id, user_id.to_string());
    assert_eq!(api_response.data.nickname, "资料测试用户");

    Ok(())
}

#[sqlx::test]
async fn test_update_user_profile(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    let (app, token, _user_id) =
        setup_authenticated_test(pool, "update_test@example.com", "更新测试用户").await?;

    // 更新资料
    let update_req = UpdateProfileRequest {
        nickname: Some("新昵称".to_string()),
        avatar_url: Some("https://example.com/avatar.jpg".to_string()),
        bio: Some("这是我的个人简介".to_string()),
    };

    let request = Request::builder()
        .method("POST")
        .uri("/api/v1/protected/user/update_profile")
        .header("content-type", "application/json")
        .header("authorization", format!("Bearer {}", token))
        .body(Body::from(serde_json::to_string(&update_req)?))
        .unwrap();

    let response = app.oneshot(request).await?;

    assert_eq!(response.status(), StatusCode::OK);

    let body = hyper::body::to_bytes(response.into_body()).await?;
    let api_response: ApiResponse<()> = serde_json::from_slice(&body)?;

    assert_eq!(api_response.code, 0);

    Ok(())
}

#[sqlx::test]
async fn test_friend_request_flow(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    // 创建两个用户
    let (app1, token1, user_id1) =
        setup_authenticated_test(pool.clone(), "user1@example.com", "用户1").await?;

    let (app2, token2, user_id2) =
        setup_authenticated_test(pool.clone(), "user2@example.com", "用户2").await?;

    // 用户1发送好友申请给用户2
    let friend_req = SendFriendRequestRequest {
        to_user_id: user_id2.to_string(),
        message: Some("你好，我想加你为好友".to_string()),
    };

    let request = Request::builder()
        .method("POST")
        .uri("/api/v1/protected/user/send_friend_request")
        .header("content-type", "application/json")
        .header("authorization", format!("Bearer {}", token1))
        .body(Body::from(serde_json::to_string(&friend_req)?))
        .unwrap();

    let response = app1.oneshot(request).await?;
    assert_eq!(response.status(), StatusCode::OK);

    // 用户2查看好友申请列表
    let get_requests = GetFriendRequestsRequest {
        status: Some("received".to_string()),
        limit: Some(20),
        cursor: None,
    };

    let request = Request::builder()
        .method("POST")
        .uri("/api/v1/protected/user/get_friend_requests")
        .header("content-type", "application/json")
        .header("authorization", format!("Bearer {}", token2))
        .body(Body::from(serde_json::to_string(&get_requests)?))
        .unwrap();

    let response = app2.clone().oneshot(request).await?;
    assert_eq!(response.status(), StatusCode::OK);

    let body = hyper::body::to_bytes(response.into_body()).await?;
    let api_response: ApiResponse<FriendRequestListData> = serde_json::from_slice(&body)?;

    assert_eq!(api_response.code, 0);
    assert!(!api_response.data.items.is_empty());

    let request_id = &api_response.data.items[0].id;

    // 用户2接受好友申请
    let accept_req = AcceptFriendRequestRequest {
        id: request_id.clone(),
    };

    let request = Request::builder()
        .method("POST")
        .uri("/api/v1/protected/user/accept_friend_request")
        .header("content-type", "application/json")
        .header("authorization", format!("Bearer {}", token2))
        .body(Body::from(serde_json::to_string(&accept_req)?))
        .unwrap();

    let response = app2.oneshot(request).await?;
    assert_eq!(response.status(), StatusCode::OK);

    Ok(())
}

// 辅助函数：设置已认证的测试环境
async fn setup_authenticated_test(
    pool: PgPool,
    login_id: &str,
    nickname: &str,
) -> Result<(axum::Router, String, i64), Box<dyn std::error::Error>> {
    let app_state = setup_test_state(pool).await?;
    let app = setup_test_app(app_state);

    // 注册用户
    let register_req = RegisterRequest {
        login_id: login_id.to_string(),
        nickname: nickname.to_string(),
        password_hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            .to_string(),
    };

    let request = Request::builder()
        .method("POST")
        .uri("/api/v1/public/auth/register")
        .header("content-type", "application/json")
        .body(Body::from(serde_json::to_string(&register_req)?))
        .unwrap();

    let response = app.clone().oneshot(request).await?;

    let body = hyper::body::to_bytes(response.into_body()).await?;
    let api_response: ApiResponse<AuthData> = serde_json::from_slice(&body)?;

    let token = api_response.data.access_token;
    let user_id = api_response.data.user_id.parse::<i64>()?;

    Ok((app, token, user_id))
}

async fn setup_test_state(pool: PgPool) -> Result<AppState, Box<dyn std::error::Error>> {
    use hotspot_backend::config::Config;
    use redis::aio::ConnectionManager as RedisConnectionManager;
    use std::sync::Arc;

    let redis_client = redis::Client::open("redis://127.0.0.1:6379")?;
    let redis_conn = RedisConnectionManager::new(redis_client).await?;

    let config = Config {
        database_url: "".to_string(),
        redis_url: "redis://127.0.0.1:6379".to_string(),
        jwt_secret: "test_secret_key_for_integration_tests".to_string(),
        server_port: 8080,
        access_token_ttl: 3600,
        refresh_token_ttl: 2592000,
    };

    Ok(AppState {
        db: pool,
        redis: redis_conn,
        config: Arc::new(config),
        ws_event_publisher: todo!("实现WebSocket事件发布器"),
    })
}

fn setup_test_app(state: AppState) -> axum::Router {
    use hotspot_backend::routes;

    routes::create_router(state)
}


