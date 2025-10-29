// TODO: 实现会话Repository
// 包括：
// - 创建会话
// - 查询会话列表
// - 撤销会话
// - 更新最后活跃时间

use sqlx::PgPool;
use crate::error::Result;

pub struct SessionRepository {
    pool: PgPool,
}

impl SessionRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
    
    // TODO: 实现会话相关的数据访问方法
}

