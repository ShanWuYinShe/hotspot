use bcrypt::{hash, verify};

use crate::error::{AppError, Result};

pub struct PasswordUtil;

impl PasswordUtil {
    /// 对客户端哈希进行 bcrypt 二次哈希
    /// client_hash: SHA256(password) from client (64字符十六进制)
    pub fn hash_password(client_hash: &str) -> Result<String> {
        // 验证客户端哈希格式（应为64字符的十六进制）
        if client_hash.len() != 64 {
            return Err(AppError::BadRequest("密码哈希格式错误".to_string()));
        }
        
        // 验证是否为有效的十六进制字符串
        if !client_hash.chars().all(|c| c.is_ascii_hexdigit()) {
            return Err(AppError::BadRequest("密码哈希格式错误".to_string()));
        }
        
        // 使用 bcrypt 进行二次哈希，work factor = 12
        let hashed = hash(client_hash, 12)?;
        Ok(hashed)
    }
    
    /// 验证密码
    /// client_hash: SHA256(password) from client
    /// stored_hash: bcrypt hash from database
    pub fn verify_password(client_hash: &str, stored_hash: &str) -> Result<bool> {
        let is_valid = verify(client_hash, stored_hash)?;
        Ok(is_valid)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_hash_and_verify_password() {
        let client_hash = "a".repeat(64); // 模拟 SHA256 输出
        let hashed = PasswordUtil::hash_password(&client_hash).unwrap();
        assert!(PasswordUtil::verify_password(&client_hash, &hashed).unwrap());
    }

    #[test]
    fn test_invalid_hash_format() {
        let invalid_hash = "short";
        assert!(PasswordUtil::hash_password(invalid_hash).is_err());
    }
}

