use std::sync::atomic::{AtomicU64, Ordering};

static SEQUENCE: AtomicU64 = AtomicU64::new(1);

pub struct IdGenerator;

impl IdGenerator {
    /// 生成用户ID
    /// user_type: 0x01 (临时), 0x02 (普通), 0x03 (系统)
    pub fn generate_user_id(user_type: u8) -> i64 {
        let sequence = SEQUENCE.fetch_add(1, Ordering::SeqCst);
        let type_bits = (user_type as i64) << 56;
        let sequence_bits = sequence & 0x00FFFFFFFFFFFFFF;
        type_bits | sequence_bits
    }
    
    /// 提取用户类型
    pub fn extract_user_type(user_id: i64) -> u8 {
        ((user_id >> 56) & 0xFF) as u8
    }
    
    /// 判断是否为临时用户
    pub fn is_temp_user(user_id: i64) -> bool {
        Self::extract_user_type(user_id) == 0x01
    }
    
    /// 判断是否为普通用户
    pub fn is_normal_user(user_id: i64) -> bool {
        Self::extract_user_type(user_id) == 0x02
    }
    
    /// 判断是否为系统用户
    pub fn is_system_user(user_id: i64) -> bool {
        Self::extract_user_type(user_id) == 0x03
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_generate_user_id() {
        let user_id = IdGenerator::generate_user_id(0x02);
        let user_type = IdGenerator::extract_user_type(user_id);
        assert_eq!(user_type, 0x02);
        assert!(IdGenerator::is_normal_user(user_id));
    }

    #[test]
    fn test_temp_user() {
        let user_id = IdGenerator::generate_user_id(0x01);
        assert!(IdGenerator::is_temp_user(user_id));
    }

    #[test]
    fn test_system_user() {
        let user_id = IdGenerator::generate_user_id(0x03);
        assert!(IdGenerator::is_system_user(user_id));
    }
}

