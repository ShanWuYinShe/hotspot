use bcrypt::{hash, verify};

use crate::error::{ApiError, Result};

pub struct PasswordUtil;

impl PasswordUtil {
    /// 对密码进行服务端慢哈希（bcrypt）。
    ///
    /// 约定：密码在 HTTPS/TLS 中明文传输；服务端负责慢哈希存储（业内最佳实践）。
    pub fn hash_password(password: &str) -> Result<String> {
        if password.is_empty() {
            return Err(ApiError::BadRequest("密码不能为空".to_string()));
        }

        // 使用 bcrypt，work factor = 12（后续建议配置化）
        let hashed = hash(password, 12)?;
        Ok(hashed)
    }

    /// 验证密码（明文 password vs stored bcrypt hash）
    pub fn verify_password(password: &str, stored_hash: &str) -> Result<bool> {
        let is_valid = verify(password, stored_hash)?;
        Ok(is_valid)
    }
}
