//! 用户ID生成器
//!
//! 用户ID分配规则：
//! - 系统用户区间：1 - 1,000,000（通过管理员接口创建）
//! - 临时用户和注册用户区间：1,000,001+（共用序列，方便临时用户转正）

use sqlx::PgPool;

#[allow(unused_imports)]
use crate::constants::{SYSTEM_USER_ID_MAX, REGULAR_USER_ID_START};

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

    /// 判断是否为系统用户ID
    ///
    /// 系统用户ID范围：1 - 1,000,000
    #[allow(dead_code)]
    pub fn is_system_user_id(user_id: i64) -> bool {
        user_id >= 1 && user_id <= SYSTEM_USER_ID_MAX
    }

    /// 判断是否为普通用户ID（临时用户或注册用户）
    ///
    /// 普通用户ID范围：1,000,001+
    #[allow(dead_code)]
    pub fn is_regular_user_id(user_id: i64) -> bool {
        user_id > SYSTEM_USER_ID_MAX
    }
}
