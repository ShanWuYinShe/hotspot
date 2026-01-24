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

/// 生成 refresh_token 的 Set-Cookie 字符串（HttpOnly）
///
/// 约定：
/// - Path 使用 "/"：避免在反向代理/网关加前缀（如 /backend/api）时导致 Cookie 不随请求携带
/// - CSRF 保护采用 Double Submit Token（Cookie + Header）
/// - 如果 `secure=true`，则 SameSite=None；否则 SameSite=Lax（兼容本地 http）
pub fn build_refresh_token_set_cookie(
    refresh_token: &str,
    max_age_seconds: i64,
    secure: bool,
) -> String {
    let same_site = if secure { "None" } else { "Lax" };

    // 注意：这里不设置 Domain，让其跟随当前 Host（更安全）
    // 同时设置 Max-Age，避免仅靠 Expires 的时区差异
    let mut cookie = format!(
        "hs_refresh_token={}; Max-Age={}; Path=/; HttpOnly; SameSite={}",
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
pub fn build_csrf_token_set_cookie(
    csrf_token: &str,
    max_age_seconds: i64,
    secure: bool,
) -> String {
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

/// 清除 refresh_token cookie（登出/强制失效）
pub fn build_refresh_token_clear_cookie(secure: bool) -> String {
    let same_site = if secure { "None" } else { "Lax" };

    let mut cookie = format!(
        "hs_refresh_token=; Max-Age=0; Path=/; HttpOnly; SameSite={}",
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
