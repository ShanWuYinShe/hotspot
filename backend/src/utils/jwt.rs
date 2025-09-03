use chrono::{Duration, Utc};
use jsonwebtoken::{DecodingKey, EncodingKey, Header, Validation, decode, encode};
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
pub struct TokenClaims {
    pub sub: i64,        // 用户ID
    pub iat: usize,      // 颁发时间
    pub exp: usize,      // 过期时间
    pub temp: bool,      // 是否为临时用户
}

#[derive(Debug, Serialize, Deserialize)]
pub struct RefreshTokenClaims {
    pub sub: i64,        // 用户ID
    pub iat: usize,      // 颁发时间
    pub exp: usize,      // 过期时间
    pub temp: bool,      // 是否为临时用户
    pub token_type: String, // 令牌类型，固定为 "refresh"
}

pub fn generate_jwt_token(
    user_id: i64,
    secret: &str,
    expires_in: i64,
    is_temp: bool,
) -> Result<(String, chrono::DateTime<Utc>), jsonwebtoken::errors::Error> {
    let now = Utc::now();
    let expires_at = now + Duration::minutes(expires_in);
    let exp = expires_at.timestamp() as usize;

    let claims = TokenClaims {
        sub: user_id,
        iat: now.timestamp() as usize,
        exp,
        temp: is_temp,
    };

    let token = encode(
        &Header::default(),
        &claims,
        &EncodingKey::from_secret(secret.as_bytes()),
    )?;

    Ok((token, expires_at))
}

pub fn verify_jwt_token(
    token: &str,
    secret: &str,
) -> Result<TokenClaims, jsonwebtoken::errors::Error> {
    let decoded = decode::<TokenClaims>(
        token,
        &DecodingKey::from_secret(secret.as_bytes()),
        &Validation::default(),
    )?;

    Ok(decoded.claims)
}

/// 生成refresh token
pub fn generate_refresh_token(
    user_id: i64,
    secret: &str,
    expires_in: i64, // 以分钟为单位
    is_temp: bool,
) -> Result<(String, chrono::DateTime<Utc>), jsonwebtoken::errors::Error> {
    let now = Utc::now();
    let expires_at = now + Duration::minutes(expires_in);
    let exp = expires_at.timestamp() as usize;

    let claims = RefreshTokenClaims {
        sub: user_id,
        iat: now.timestamp() as usize,
        exp,
        temp: is_temp,
        token_type: "refresh".to_string(),
    };

    let token = encode(
        &Header::default(),
        &claims,
        &EncodingKey::from_secret(secret.as_bytes()),
    )?;

    Ok((token, expires_at))
}

/// 验证refresh token
pub fn verify_refresh_token(
    token: &str,
    secret: &str,
) -> Result<RefreshTokenClaims, jsonwebtoken::errors::Error> {
    let decoded = decode::<RefreshTokenClaims>(
        token,
        &DecodingKey::from_secret(secret.as_bytes()),
        &Validation::default(),
    )?;

    // 验证token类型
    if decoded.claims.token_type != "refresh" {
        return Err(jsonwebtoken::errors::Error::from(
            jsonwebtoken::errors::ErrorKind::InvalidToken
        ));
    }

    Ok(decoded.claims)
}

/// 生成access token和refresh token对
pub fn generate_token_pair(
    user_id: i64,
    secret: &str,
    is_temp: bool,
) -> Result<(String, String, chrono::DateTime<Utc>, chrono::DateTime<Utc>), jsonwebtoken::errors::Error> {
    // Access token: 1小时有效期
    let (access_token, access_expires_at) = generate_jwt_token(user_id, secret, 60, is_temp)?;
    
    // Refresh token: 7天有效期
    let (refresh_token, refresh_expires_at) = generate_refresh_token(user_id, secret, 7 * 24 * 60, is_temp)?;
    
    Ok((access_token, refresh_token, access_expires_at, refresh_expires_at))
}
