//! ID生成器
//!
//! 提供分享码等ID相关生成功能

use rand::Rng;

/// 生成6位分享码
pub fn generate_share_code() -> String {
    const CHARSET: &[u8] = b"ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
    let mut rng = rand::rng();

    (0..6)
        .map(|_| {
            let idx = rng.random_range(0..CHARSET.len());
            CHARSET[idx] as char
        })
        .collect()
}
