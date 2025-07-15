use chrono::Utc;
use rand::Rng;

/// 生成唯一的用户ID
/// 
/// 使用时间戳 + 随机数的方式生成唯一的用户ID，
/// 确保在分布式环境下也不会产生冲突
pub fn generate_user_id() -> i64 {
    // 使用当前时间戳（秒）
    let timestamp = Utc::now().timestamp();
    
    // 生成随机数 (0-9999)
    let random_num = rand::rng().random_range(0..10000);
    
    // 组合时间戳和随机数：timestamp * 10000 + random_num
    timestamp * 10000 + random_num
}


