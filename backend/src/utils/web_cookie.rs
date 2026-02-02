use axum::http::HeaderMap;

/// 极简 Cookie 解析：从请求头中提取某个 cookie 的值。
///
///
///
/// - 不做 RFC 全覆盖解析（够用即可）
/// - 只用于读取 HttpOnly refresh token（服务端自己写入）
pub fn get_cookie_value(headers: &HeaderMap, cookie_name: &str) -> Option<String> {
    let cookie_header = headers.get(axum::http::header::COOKIE)?.to_str().ok()?;

    // Cookie: a=1; b=2; hs_refresh_token=xxx
    for part in cookie_header.split(';') {
        let trimmed = part.trim();
        let (k, v) = trimmed.split_once('=')?;
        if k.trim() == cookie_name {
            return Some(v.trim().to_string());
        }
    }

    None
}

/// 生成 access_token 的 Set-Cookie 字符串（HttpOnly）
///
/// 约定：
/// - Path=/api：所有 API 请求自动携带
/// - HttpOnly：防止 JavaScript 访问
/// - SameSite=Lax：防止 CSRF 攻击
/// - Secure：仅 HTTPS 传输（生产环境）
pub fn build_access_token_set_cookie(
    access_token: &str,
    max_age_seconds: i64,
    secure: bool,
) -> String {
    let same_site = if secure { "None" } else { "Lax" };

    let mut cookie = format!(
        "hs_access_token={}; Max-Age={}; Path=/api; HttpOnly; SameSite={}",
        access_token, max_age_seconds, same_site
    );

    if secure {
        cookie.push_str("; Secure");
    }

    cookie
}

/// 生成 refresh_token 的 Set-Cookie 字符串（HttpOnly）
///
/// 约定：
/// - Path=/api/v1/auth/refresh_token：仅刷新 token API 自动携带
/// - HttpOnly：防止 JavaScript 访问
/// - SameSite=Lax：防止 CSRF 攻击
/// - Secure：仅 HTTPS 传输（生产环境）
pub fn build_refresh_token_set_cookie(
    refresh_token: &str,
    max_age_seconds: i64,
    secure: bool,
) -> String {
    let same_site = if secure { "None" } else { "Lax" };

    // 注意：Path 限制为 /api/v1/auth/refresh_token，仅刷新 API 可用
    let mut cookie = format!(
        "hs_refresh_token={}; Max-Age={}; Path=/api/v1/auth/refresh_token; HttpOnly; SameSite={}",
        refresh_token, max_age_seconds, same_site
    );

    if secure {
        cookie.push_str("; Secure");
    }

    cookie
}

/// 生成 CSRF Token 的 Set-Cookie 字符串（非 HttpOnly，供前端读取后放入 Header）
///
/// 约定：
/// - Path 使用 "/"：保持与 refresh_token 一致
/// - SameSite 策略与 refresh_token 对齐
pub fn build_csrf_token_set_cookie(csrf_token: &str, max_age_seconds: i64, secure: bool) -> String {
    let same_site = if secure { "None" } else { "Lax" };

    let mut cookie = format!(
        "hs_csrf_token={}; Max-Age={}; Path=/; SameSite={}",
        csrf_token, max_age_seconds, same_site
    );

    if secure {
        cookie.push_str("; Secure");
    }

    cookie
}

/// 清除 access_token cookie（登出/强制失效）
pub fn build_access_token_clear_cookie(secure: bool) -> String {
    let same_site = if secure { "None" } else { "Lax" };

    let mut cookie = format!(
        "hs_access_token=; Max-Age=0; Path=/api; HttpOnly; SameSite={}",
        same_site
    );

    if secure {
        cookie.push_str("; Secure");
    }

    cookie
}

/// 清除 refresh_token cookie（登出/强制失效）
pub fn build_refresh_token_clear_cookie(secure: bool) -> String {
    let same_site = if secure { "None" } else { "Lax" };

    let mut cookie = format!(
        "hs_refresh_token=; Max-Age=0; Path=/api/v1/auth/refresh_token; HttpOnly; SameSite={}",
        same_site
    );

    if secure {
        cookie.push_str("; Secure");
    }

    cookie
}

/// 清除 CSRF Token cookie
pub fn build_csrf_token_clear_cookie(secure: bool) -> String {
    let same_site = if secure { "None" } else { "Lax" };

    let mut cookie = format!("hs_csrf_token=; Max-Age=0; Path=/; SameSite={}", same_site);

    if secure {
        cookie.push_str("; Secure");
    }

    cookie
}
