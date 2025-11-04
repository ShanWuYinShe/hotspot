use axum::{
    extract::{Request, State},
    middleware::Next,
    response::Response,
};

use crate::{error::AppError, middleware::auth::extract_user_id, state::AppState};

/// 管理员权限验证中间件
/// 必须在 auth_middleware 之后使用
pub async fn admin_auth_middleware(
    State(state): State<AppState>,
    req: Request,
    next: Next,
) -> Result<Response, AppError> {
    // 1. 提取用户ID（由 auth_middleware 注入）
    let user_id = extract_user_id(&req)?;

    // 2. 查询用户是否为管理员
    let is_admin = sqlx::query_scalar::<_, bool>(
        r#"
        SELECT is_admin
        FROM users
        WHERE user_id = $1
        "#,
    )
    .bind(user_id)
    .fetch_optional(&state.db)
    .await?
    .unwrap_or(false);

    if !is_admin {
        return Err(AppError::Forbidden("需要管理员权限".to_string()));
    }

    // 3. 继续处理请求
    Ok(next.run(req).await)
}











