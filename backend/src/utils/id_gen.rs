//! 用户ID生成器
//!
//! 用户ID分配规则：
//! - 系统用户区间：1 - 1,000,000（通过管理员接口创建）
//! - 临时用户和注册用户区间：1,000,001+（共用序列，方便临时用户转正）

use sqlx::PgPool;

pub struct IdGenerator;

impl IdGenerator {
    /// 生成用户ID（用于临时用户和注册用户）
    ///
    /// 从统一的 user_id_seq 序列获取下一个值
    ///
    /// # Arguments
    /// * `pool` - 数据库连接池
    pub async fn generate_user_id(pool: &PgPool) -> Result<i64, sqlx::Error> {
        let result: (i64,) = sqlx::query_as("SELECT nextval('user_id_seq')")
            .fetch_one(pool)
            .await?;

        Ok(result.0)
    }
}
