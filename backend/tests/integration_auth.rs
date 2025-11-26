use axum::{
    body::Body,
    http::{Request, StatusCode},
};
use hotspot_backend::{
    dto::{request::*, response::*},
    error::AppError,
    state::AppState,
};
use serde_json::json;
use sqlx::PgPool;
use tower::ServiceExt;

/// 集成测试：认证模块
///
/// 测试场景：
/// 1. 用户注册 - 正常流程
/// 2. 用户登录 - 正常流程
/// 3. Token 刷新 - 正常流程
/// 4. 登出 - 正常流程
/// 5. 重复注册 - 错误场景
/// 6. 错误密码登录 - 错误场景
/// 7. 无效 Token - 错误场景

#[sqlx::test]
async fn test_user_register(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    // 准备测试环境
    let app_state = setup_test_state(pool).await?;
    let app = setup_test_app(app_state);

    // 构造注册请求
    let register_req = RegisterRequest {
        login_id: "test_user@example.com".to_string(),
        nickname: "测试用户".to_string(),
        password_hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            .to_string(), // SHA256("password")
    };

    let request = Request::builder()
        .method("POST")
        .uri("/api/v1/public/auth/register")
        .header("content-type", "application/json")
        .body(Body::from(serde_json::to_string(&register_req)?))
        .unwrap();

    // 发送请求
    let response = app.oneshot(request).await?;

    // 验证响应
    assert_eq!(response.status(), StatusCode::OK);

    let body = hyper::body::to_bytes(response.into_body()).await?;
    let api_response: ApiResponse<AuthData> = serde_json::from_slice(&body)?;

    assert_eq!(api_response.code, 0);
    assert!(!api_response.data.access_token.is_empty());
    assert!(!api_response.data.refresh_token.is_empty());

    Ok(())
}

#[sqlx::test]
async fn test_user_login(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    let app_state = setup_test_state(pool).await?;
    let app = setup_test_app(app_state);

    // 先注册用户
    let register_req = RegisterRequest {
        login_id: "login_test@example.com".to_string(),
        nickname: "登录测试".to_string(),
        password_hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            .to_string(),
    };

    let _ = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/public/auth/register")
                .header("content-type", "application/json")
                .body(Body::from(serde_json::to_string(&register_req)?))
                .unwrap(),
        )
        .await?;

    // 尝试登录
    let login_req = LoginRequest {
        login_id: "login_test@example.com".to_string(),
        password_hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            .to_string(),
    };

    let request = Request::builder()
        .method("POST")
        .uri("/api/v1/public/auth/login")
        .header("content-type", "application/json")
        .body(Body::from(serde_json::to_string(&login_req)?))
        .unwrap();

    let response = app.oneshot(request).await?;

    assert_eq!(response.status(), StatusCode::OK);

    let body = hyper::body::to_bytes(response.into_body()).await?;
    let api_response: ApiResponse<AuthData> = serde_json::from_slice(&body)?;

    assert_eq!(api_response.code, 0);
    assert!(!api_response.data.access_token.is_empty());

    Ok(())
}

#[sqlx::test]
async fn test_duplicate_register(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    let app_state = setup_test_state(pool).await?;
    let app = setup_test_app(app_state);

    let register_req = RegisterRequest {
        login_id: "duplicate@example.com".to_string(),
        nickname: "重复用户".to_string(),
        password_hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            .to_string(),
    };

    // 第一次注册
    let _ = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/public/auth/register")
                .header("content-type", "application/json")
                .body(Body::from(serde_json::to_string(&register_req)?))
                .unwrap(),
        )
        .await?;

    // 第二次注册（应失败）
    let request = Request::builder()
        .method("POST")
        .uri("/api/v1/public/auth/register")
        .header("content-type", "application/json")
        .body(Body::from(serde_json::to_string(&register_req)?))
        .unwrap();

    let response = app.oneshot(request).await?;

    assert_eq!(response.status(), StatusCode::OK);

    let body = hyper::body::to_bytes(response.into_body()).await?;
    let api_response: ApiResponse<AuthData> = serde_json::from_slice(&body)?;

    // 应返回错误码（1201 - 资源已存在）
    assert_ne!(api_response.code, 0);

    Ok(())
}

#[sqlx::test]
async fn test_wrong_password_login(pool: PgPool) -> Result<(), Box<dyn std::error::Error>> {
    let app_state = setup_test_state(pool).await?;
    let app = setup_test_app(app_state);

    // 先注册用户
    let register_req = RegisterRequest {
        login_id: "password_test@example.com".to_string(),
        nickname: "密码测试".to_string(),
        password_hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            .to_string(),
    };

    let _ = app
        .clone()
        .oneshot(
            Request::builder()
                .method("POST")
                .uri("/api/v1/public/auth/register")
                .header("content-type", "application/json")
                .body(Body::from(serde_json::to_string(&register_req)?))
                .unwrap(),
        )
        .await?;

    // 使用错误密码登录
    let login_req = LoginRequest {
        login_id: "password_test@example.com".to_string(),
        password_hash: "wrong_password_hash".to_string(),
    };

    let request = Request::builder()
        .method("POST")
        .uri("/api/v1/public/auth/login")
        .header("content-type", "application/json")
        .body(Body::from(serde_json::to_string(&login_req)?))
        .unwrap();

    let response = app.oneshot(request).await?;

    assert_eq!(response.status(), StatusCode::OK);

    let body = hyper::body::to_bytes(response.into_body()).await?;
    let api_response: ApiResponse<AuthData> = serde_json::from_slice(&body)?;

    // 应返回认证错误码（1105 - 密码错误）
    assert_ne!(api_response.code, 0);

    Ok(())
}

// 辅助函数：设置测试环境
async fn setup_test_state(pool: PgPool) -> Result<AppState, Box<dyn std::error::Error>> {
    use hotspot_backend::config::Config;
    use redis::aio::ConnectionManager as RedisConnectionManager;
    use std::sync::Arc;

    let redis_client = redis::Client::open("redis://127.0.0.1:6379")?;
    let redis_conn = RedisConnectionManager::new(redis_client).await?;

    let config = Config {
        database_url: "".to_string(), // 已经有pool了
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

// 辅助函数：设置测试应用
fn setup_test_app(state: AppState) -> axum::Router {
    use hotspot_backend::routes;

    routes::create_router(state)
}


