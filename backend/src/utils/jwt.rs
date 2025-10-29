use chrono::{Duration, Utc};
use jsonwebtoken::{decode, encode, DecodingKey, EncodingKey, Header, Validation};
use serde::{Deserialize, Serialize};

use crate::error::{AppError, Result};

#[derive(Debug, Serialize, Deserialize)]
pub struct Claims {
    pub sub: String,        // Subject (用户ID)
    pub exp: i64,           // Expiration (过期时间)
    pub iat: i64,           // Issued At (签发时间)
    pub user_type: u8,      // 用户类型
}

pub struct JwtUtil;

impl JwtUtil {
    /// 生成 JWT Token
    pub fn generate_token(user_id: i64, expires_in_seconds: i64, secret: &str) -> Result<String> {
        let now = Utc::now();
        let exp = now + Duration::seconds(expires_in_seconds);
        
        // 提取用户类型（高8位）
        let user_type = ((user_id >> 56) & 0xFF) as u8;
        
        let claims = Claims {
            sub: user_id.to_string(),
            exp: exp.timestamp(),
            iat: now.timestamp(),
            user_type,
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_generate_and_verify_token() {
        let secret = "test_secret";
        let user_id = (0x02_i64 << 56) | 12345; // 普通用户
        
        let token = JwtUtil::generate_token(user_id, 3600, secret).unwrap();
        let claims = JwtUtil::verify_token(&token, secret).unwrap();
        
        assert_eq!(claims.sub, user_id.to_string());
        assert_eq!(claims.user_type, 0x02);
    }
}

