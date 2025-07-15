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
