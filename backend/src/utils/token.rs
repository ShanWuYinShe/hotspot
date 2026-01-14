use base64::engine::general_purpose::URL_SAFE_NO_PAD;
use base64::Engine as _;
use rand::RngCore;
use sha2::{Digest, Sha256};

/// Token 工具：
/// - refresh_token 生成：opaque 随机串（不可自解释）
/// - token 存储：仅存 hash（DB/日志中不落原文）
pub struct TokenUtil;

impl TokenUtil {
    /// 生成 refresh_token（URL-safe，无 padding）
    ///
    ///
    /// - 生成 32 bytes 随机数（256-bit）
    /// - 进行 base64url 编码，得到可安全放在 HTTP header/body/cookie 的字符串
    pub fn generate_refresh_token() -> String {
        let mut bytes = [0u8; 32];
        rand::rng().fill_bytes(&mut bytes);
        URL_SAFE_NO_PAD.encode(bytes)
    }

    /// 对 token 做不可逆哈希（用于 DB 存储与查找）
    ///
    /// 推荐做法：加 pepper（服务端 secret）后再 sha256，以降低数据库泄露后的离线攻击价值。
    pub fn hash_token(token: &str, pepper: &str) -> String {
        let mut hasher = Sha256::new();
        hasher.update(pepper.as_bytes());
        hasher.update(b":");
        hasher.update(token.as_bytes());
        hex::encode(hasher.finalize())
    }

    /// 生成数字验证码（6位数字）
    ///
    /// 用于短信/邮件验证码
    pub fn generate_numeric_code(length: usize) -> String {
        use rand::Rng;
        let mut rng = rand::rng();
        (0..length)
            .map(|_| rng.random_range(0..10).to_string())
            .collect()
    }

    /// 生成通用 token（用于 verify_token等）
    ///
    /// 生成 URL-safe 的随机 token
    pub fn generate_token() -> String {
        let mut bytes = [0u8; 24]; // 24 bytes = 192 bits
        rand::rng().fill_bytes(&mut bytes);
        URL_SAFE_NO_PAD.encode(bytes)
    }
}
