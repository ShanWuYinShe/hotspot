use chrono::{Duration, Utc};
use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};

use crate::error::Result;

#[derive(Debug, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String, // Subject (用户ID)
    pub sid: String, // Session ID (会话ID)
    pub exp: i64,    // Expiration (过期时间)
    pub iat: i64,    // Issued At (签发时间)
}

pub struct JwtUtil;

impl JwtUtil {
    /// 生成 JWT Token
    pub fn generate_token(
        user_id: i64,
        session_id: uuid::Uuid,
        expires_in_seconds: i64,
        secret: &str,
    ) -> Result<String> {
        let now = Utc::now();
        let exp = now + Duration::seconds(expires_in_seconds);

        let claims = Claims {
            sub: user_id.to_string(),
            sid: session_id.to_string(),
            exp: exp.timestamp(),
            iat: now.timestamp(),
        };

        let token = encode(
            &Header::default(),
            &claims,
            &EncodingKey::from_secret(secret.as_bytes()),
        )?;

        Ok(token)
    }

    /// 验证 JWT Token
    pub fn verify_token(token: &str, secret: &str) -> Result<Claims> {
        let token_data = decode::<Claims>(
            token,
            &DecodingKey::from_secret(secret.as_bytes()),
            &Validation::default(),
        )?;

        Ok(token_data.claims)
    }
}
