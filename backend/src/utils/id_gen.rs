//! ID生成器
//!
//! 用户ID分配规则：
//! - 系统用户区间：1 - 1,000,000（通过管理员接口创建）
//! - 临时用户和注册用户区间：1,000,001+（共用序列，方便临时用户转正）

use rand::Rng;
use sqlx::PgPool;

pub struct IdGenerator;

impl IdGenerator {
    /// 生成用户ID（用于临时用户和注册用户）
    pub async fn generate_user_id(pool: &PgPool) -> Result<i64, sqlx::Error> {
        let result = sqlx::query_scalar!(r#"SELECT nextval('user_id_seq')"#)
            .fetch_one(pool)
            .await?;

        Ok(result.unwrap())
    }
}

/// 生成字母数字字符串
pub fn generate_alphanumeric(length: usize) -> String {
    const CHARSET: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ\
                             abcdefghijklmnopqrstuvwxyz\
                             0123456789";
    let mut rng = rand::rng();

    (0..length)
        .map(|_| {
            let idx = rng.random_range(0..CHARSET.len());
            CHARSET[idx] as char
        })
        .collect()
}

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
