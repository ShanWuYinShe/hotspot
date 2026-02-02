//! 会话令牌工具模块
//!
//! 提供安全的会话管理令牌生成和验证功能。
//!
//! ## 设计理念
//!
//! - **display_id**: 用户可见的会话标识（如 "ses_abc123"），便于识别但无法用于操作
//! - **revoke_token**: 一次性撤销令牌，用于安全地撤销会话，防止批量攻击
//!
//! ## 安全特性
//!
//! 1. **一次性使用**: revoke_token 使用后立即失效
//! 2. **时间限制**: revoke_token 有效期仅 5 分钟
//! 3. **HMAC 签名**: 防止伪造和篡改
//! 4. **不暴露真实 ID**: 真实 session_id 不对外暴露

use chrono::Utc;
use hmac::{Hmac, Mac};
use sha2::Sha256;
use uuid::Uuid;

type HmacSha256 = Hmac<Sha256>;

/// 会话令牌生成器
#[derive(Clone)]
pub struct SessionTokenGenerator {
    secret: Vec<u8>,
}

impl SessionTokenGenerator {
    /// 创建新的令牌生成器
    pub fn new(secret: &str) -> Self {
        Self {
            secret: secret.as_bytes().to_vec(),
        }
    }

    /// 生成展示用的会话ID
    ///
    /// 格式：`ses_{前8位UUID}`
    ///
    /// 例如：`ses_a1b2c3d4`
    pub fn generate_display_id(&self, session_id: Uuid) -> String {
        format!("ses_{}", &session_id.to_string()[..8])
    }

    /// 生成一次性撤销令牌
    ///
    /// ## 令牌结构
    ///
    /// ```text
    /// revoke_token = base64(HMAC-SHA256(secret, payload))
    ///
    /// payload = session_id:user_id:salt:timestamp
    /// ```
    ///
    /// ## 安全特性
    ///
    /// - 使用 HMAC-SHA256 签名，防止伪造
    /// - 包含时间戳，支持过期检查
    /// - 包含随机盐，每次生成都不同
    /// - Base64 编码，URL 安全
    pub fn generate_revoke_token(&self, session_id: Uuid, user_id: i64) -> String {
        let salt = Uuid::new_v4();
        let timestamp = Utc::now().timestamp();

        // 构造待签名的数据
        let payload = format!("{}:{}:{}:{}", session_id, user_id, salt, timestamp);

        // HMAC-SHA256 签名
        // 注意：HMAC 可以接受任意大小的密钥，因此这里理论上不会失败
        let mut mac = HmacSha256::new_from_slice(&self.secret)
            .expect("HMAC accepts keys of any size; this should never fail");
        mac.update(payload.as_bytes());
        let result = mac.finalize();
        let signature = result.into_bytes();

        // Base64 URL-safe 编码
        use base64::engine::Engine;
        base64::engine::general_purpose::URL_SAFE_NO_PAD.encode(signature)
    }

    /// 验证撤销令牌的格式
    ///
    /// 注意：此方法只验证格式，不验证是否已使用或是否有效。
    /// 真实的验证需要查询 Redis。
    pub fn verify_revoke_token_format(&self, token: &str) -> bool {
        // 检查非空
        if token.is_empty() {
            return false;
        }

        // 检查 Base64 格式和长度
        use base64::engine::Engine;
        match base64::engine::general_purpose::URL_SAFE_NO_PAD.decode(token) {
            Ok(decoded) => decoded.len() >= 32, // HMAC-SHA256 = 32 bytes
            Err(_) => false,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_generate_display_id() {
        let generator = SessionTokenGenerator::new("test_secret");
        let session_id = Uuid::parse_str("550e8400-e29b-41d4-a716-446655440000")
            .expect("Valid UUID should parse successfully");
        let display_id = generator.generate_display_id(session_id);

        assert_eq!(display_id, "ses_550e8400");
        assert!(display_id.starts_with("ses_"));
        assert_eq!(display_id.len(), 12); // "ses_" + 8 chars
    }

    #[test]
    fn test_generate_revoke_token() {
        let generator = SessionTokenGenerator::new("test_secret");
        let session_id = Uuid::new_v4();
        let user_id = 123i64;

        let token1 = generator.generate_revoke_token(session_id, user_id);
        let token2 = generator.generate_revoke_token(session_id, user_id);

        // 每次生成的 token 都应该不同（因为包含随机盐）
        assert_ne!(token1, token2);

        // Token 应该是有效的 Base64
        assert!(generator.verify_revoke_token_format(&token1));
        assert!(generator.verify_revoke_token_format(&token2));
    }

    #[test]
    fn test_verify_revoke_token_format() {
        let generator = SessionTokenGenerator::new("test_secret");

        // 有效的 token
        let valid_token = generator.generate_revoke_token(Uuid::new_v4(), 123);
        assert!(generator.verify_revoke_token_format(&valid_token));

        // 无效的 token
        assert!(!generator.verify_revoke_token_format("invalid!!!"));
        assert!(!generator.verify_revoke_token_format(""));
    }

    #[test]
    fn test_token_uniqueness() {
        let generator = SessionTokenGenerator::new("test_secret");
        let session_id = Uuid::new_v4();
        let user_id = 456i64;

        // 生成 100 个 token，应该都不重复
        let mut tokens = std::collections::HashSet::new();
        for _ in 0..100 {
            let token = generator.generate_revoke_token(session_id, user_id);
            tokens.insert(token);
        }

        assert_eq!(tokens.len(), 100);
    }
}
