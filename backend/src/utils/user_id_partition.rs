//! 用户ID分区系统
//! 
//! 64位用户ID分层架构设计:
//! - 总长度: 64位 (符合数据库bigint类型)
//! - 高8位: 用户类型标识符 (0-255) - 与后端文档规范保持一致
//! - 低56位: 用户序列号 (0-72057594037927935)
//! 
//! 用户类型标识符分配（与domain/entities/user.rs保持一致）:
//! - 1: 临时用户
//! - 2: 普通用户  
//! - 3: 系统用户（官方账号等）
//! - 4-255: 预留给未来用户类型

use std::fmt;
use serde::{Deserialize, Serialize};

/// 用户类型枚举 - 与domain/entities/user.rs保持一致
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum UserType {
    /// 临时用户
    Temporary = 1,
    /// 普通注册用户
    Regular = 2,
    /// 系统用户（官方账号等）
    System = 3,
}

impl UserType {
    /// 从标识符获取用户类型
    pub fn from_identifier(identifier: u8) -> Option<Self> {
        match identifier {
            1 => Some(UserType::Temporary),
            2 => Some(UserType::Regular),
            3 => Some(UserType::System),
            _ => None,
        }
    }

    /// 获取用户类型的标识符
    pub fn identifier(self) -> u8 {
        self as u8
    }

    /// 获取用户类型的名称
    pub fn name(self) -> &'static str {
        match self {
            UserType::Temporary => "临时用户",
            UserType::Regular => "普通用户",
            UserType::System => "系统用户",
        }
    }

    /// 判断是否为特权用户类型
    pub fn is_privileged(self) -> bool {
        matches!(self, UserType::System)
    }

    /// 判断是否为临时用户
    pub fn is_temporary(self) -> bool {
        matches!(self, UserType::Temporary)
    }
}

impl fmt::Display for UserType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.name())
    }
}

/// 用户ID分区工具
pub struct UserIdPartition;

impl UserIdPartition {
    #[allow(dead_code)]
    /// 64位ID的位段配置 - 与后端文档规范保持一致
    const TYPE_BITS: u8 = 8;          // 类型标识符位数
    #[allow(dead_code)]
    const SEQUENCE_BITS: u8 = 56;     // 序列号位数
    const TYPE_SHIFT: u8 = 56;        // 类型标识符左移位数
    const SEQUENCE_MASK: i64 = 0x00FF_FFFF_FFFF_FFFF; // 序列号掩码 (56位)
    const TYPE_MASK: i64 = 0xFF;      // 类型标识符掩码 (8位)

    /// 生成新的用户ID
    /// 
    /// # 参数
    /// - `user_type`: 用户类型
    /// - `sequence`: 序列号（56位，0-72057594037927935）
    /// 
    /// # 返回
    /// 64位用户ID
    pub fn generate_user_id(user_type: UserType, sequence: i64) -> Result<i64, String> {
        if !(0..=Self::SEQUENCE_MASK).contains(&sequence) {
            return Err(format!(
                "序列号超出范围: {} (有效范围: 0-{})", 
                sequence, 
                Self::SEQUENCE_MASK
            ));
        }

        let type_id = user_type.identifier() as i64;
        let user_id = (type_id << Self::TYPE_SHIFT) | sequence;
        
        Ok(user_id)
    }

    /// 解析用户ID获取用户类型和序列号
    /// 
    /// # 参数
    /// - `user_id`: 64位用户ID
    /// 
    /// # 返回
    /// 用户类型和序列号
    pub fn parse_user_id(user_id: i64) -> Result<(UserType, i64), String> {
        if user_id <= 0 {
            return Err(format!("无效的用户ID: {}", user_id));
        }

        let type_id = ((user_id >> Self::TYPE_SHIFT) & Self::TYPE_MASK) as u8;
        let sequence = user_id & Self::SEQUENCE_MASK;

        let user_type = UserType::from_identifier(type_id)
            .ok_or_else(|| format!("未知的用户类型标识符: {}", type_id))?;

        Ok((user_type, sequence))
    }

    /// 获取用户类型（不解析序列号）
    pub fn get_user_type(user_id: i64) -> Result<UserType, String> {
        let (user_type, _) = Self::parse_user_id(user_id)?;
        Ok(user_type)
    }

    /// 获取序列号（不解析用户类型）
    pub fn get_sequence(user_id: i64) -> Result<i64, String> {
        let (_, sequence) = Self::parse_user_id(user_id)?;
        Ok(sequence)
    }

    /// 检查用户ID是否属于指定类型
    pub fn is_user_type(user_id: i64, user_type: UserType) -> bool {
        Self::get_user_type(user_id)
            .map(|t| t == user_type)
            .unwrap_or(false)
    }

    /// 获取用户类型的ID范围
    /// 
    /// # 参数
    /// - `user_type`: 用户类型
    /// 
    /// # 返回
    /// (最小ID, 最大ID)
    pub fn get_type_range(user_type: UserType) -> (i64, i64) {
        let type_id = user_type.identifier() as i64;
        let min_id = type_id << Self::TYPE_SHIFT;
        let max_id = min_id | Self::SEQUENCE_MASK;
        (min_id, max_id)
    }

    /// 检查用户ID是否有效
    pub fn is_valid_user_id(user_id: i64) -> bool {
        Self::parse_user_id(user_id).is_ok()
    }

    /// 格式化用户ID为可读字符串
    pub fn format_user_id(user_id: i64) -> String {
        match Self::parse_user_id(user_id) {
            Ok((user_type, sequence)) => {
                format!("{}#{:014X}", user_type.name(), sequence)
            }
            Err(_) => format!("无效ID#{:016X}", user_id)
        }
    }

    /// 获取用户类型的下一个可用序列号
    pub fn next_sequence_for_type(user_type: UserType, current_max: i64) -> Result<i64, String> {
        let (min_id, max_id) = Self::get_type_range(user_type);
        
        // 如果当前最大值小于该类型的最小ID，则从1开始
        let next_sequence = if current_max < min_id {
            1
        } else {
            // 提取当前最大ID的序列号，然后加1
            let current_sequence = Self::get_sequence(current_max)?;
            current_sequence + 1
        };

        // 检查是否超出范围
        let next_id = Self::generate_user_id(user_type, next_sequence)?;
        if next_id > max_id {
            return Err(format!("用户类型 {} 的ID容量已满", user_type.name()));
        }

        Ok(next_sequence)
    }

    /// 生成系统用户ID（用于系统消息、通知等）
    pub fn generate_system_user_id() -> i64 {
        // 使用系统用户类型，序列号为0
        Self::generate_user_id(UserType::System, 0).expect("系统用户ID生成失败")
    }

    /// 生成系统管理员用户ID（用于管理功能）
    #[allow(dead_code)]
    pub fn generate_admin_user_id(sequence: i64) -> Result<i64, String> {
        // 管理员也是系统用户类型
        Self::generate_user_id(UserType::System, sequence)
    }
}


