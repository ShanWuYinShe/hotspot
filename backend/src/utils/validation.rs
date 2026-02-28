use std::collections::HashSet;

use uuid::Uuid;

use crate::constants::DEFAULT_PAGE_LIMIT;
use crate::error::{AppError, Result};

pub struct Validator;

impl Validator {
    /// 通用消息内容约束（spec: content(message) 1-5000）
    pub fn validate_message_content(content: &str) -> Result<()> {
        Self::validate_not_empty(content, "消息内容")?;
        Self::validate_length(content, "消息内容", 1, 5000)?;
        Ok(())
    }

    /// 群组名称约束（spec: 1-50）
    pub fn validate_group_name(name: &str) -> Result<()> {
        Self::validate_not_empty(name, "群组名称")?;
        Self::validate_length(name, "群组名称", 1, 50)?;
        Ok(())
    }

    /// 群组描述约束（spec: 0-500）
    pub fn validate_group_description(description: &str) -> Result<()> {
        Self::validate_length(description, "群组描述", 0, 500)?;
        Ok(())
    }

    /// 群组密码约束（spec: 4-32）
    pub fn validate_group_password(password: &str) -> Result<()> {
        if password.trim().is_empty() {
            return Err(AppError::BadRequest("群组密码不能为空".to_string()));
        }
        Self::validate_length(password, "群组密码", 4, 32)?;
        Ok(())
    }

    /// 群组最大成员数约束（spec: 2-2000）
    pub fn validate_group_max_members(max_members: i32) -> Result<()> {
        if !(2..=2000).contains(&max_members) {
            return Err(AppError::OutOfRange(
                "max_members 必须在 2 到 2000 之间".to_string(),
            ));
        }
        Ok(())
    }

    /// 邀请/申请附言约束（spec: 0-200）
    pub fn validate_invite_or_apply_message(message: &str) -> Result<()> {
        Self::validate_length(message, "附言", 0, 200)?;
        Ok(())
    }

    /// 群公告标题约束（spec: 1-100）
    pub fn validate_announcement_title(title: &str) -> Result<()> {
        Self::validate_not_empty(title, "公告标题")?;
        Self::validate_length(title, "公告标题", 1, 100)?;
        Ok(())
    }

    /// 群公告内容约束（spec: 1-5000）
    pub fn validate_announcement_content(content: &str) -> Result<()> {
        Self::validate_not_empty(content, "公告内容")?;
        Self::validate_length(content, "公告内容", 1, 5000)?;
        Ok(())
    }

    /// 成员禁言时长约束（spec: 1-43200 分钟）
    pub fn validate_mute_duration_minutes(duration_minutes: i32) -> Result<()> {
        if !(1..=43200).contains(&duration_minutes) {
            return Err(AppError::OutOfRange(
                "duration_minutes 必须在 1 到 43200 之间".to_string(),
            ));
        }
        Ok(())
    }

    /// 通用关键字约束（spec: 1-100）
    pub fn validate_keyword(keyword: &str, field_name: &str) -> Result<()> {
        Self::validate_not_empty(keyword, field_name)?;
        Self::validate_length(keyword, field_name, 1, 100)?;
        Ok(())
    }

    /// 校验登录标识（手机号/邮箱/用户名）
    pub fn validate_login_id(login_id: &str) -> Result<()> {
        Self::validate_not_empty(login_id, "login_id")?;
        Self::validate_length(login_id, "login_id", 3, 64)?;
        Ok(())
    }

    /// 校验昵称
    pub fn validate_nickname(nickname: &str) -> Result<()> {
        Self::validate_not_empty(nickname, "昵称")?;
        Self::validate_length(nickname, "昵称", 1, 30)?;
        Ok(())
    }

    /// 校验密码复杂度
    /// 约束：8-128 字符，且至少包含 1 个字母与 1 个数字
    pub fn validate_password(password: &str, field_name: &str) -> Result<()> {
        Self::validate_not_empty(password, field_name)?;
        Self::validate_length(password, field_name, 8, 128)?;
        let has_letter = password.chars().any(|c| c.is_alphabetic());
        let has_digit = password.chars().any(|c| c.is_ascii_digit());
        if !has_letter || !has_digit {
            return Err(AppError::BadRequest(format!(
                "{field_name} 必须同时包含字母和数字"
            )));
        }
        Ok(())
    }

    /// 验证分页参数
    /// - None -> 默认值 20
    /// - 超出 1..=max_limit 范围 -> 返回错误
    pub fn validate_limit(limit: Option<i32>, max_limit: i32) -> Result<i32> {
        let limit = limit.unwrap_or(DEFAULT_PAGE_LIMIT as i32);
        if limit < 1 {
            return Err(AppError::OutOfRange(format!(
                "limit 必须在 1 到 {} 之间",
                max_limit
            )));
        }

        Ok(limit.min(max_limit))
    }

    /// 验证可选搜索半径
    /// - None -> 默认值 5000
    /// - 超出 100..=max_radius 范围 -> 返回错误
    pub fn validate_radius(radius: Option<i32>, max_radius: i32) -> Result<i32> {
        let radius = radius.unwrap_or(5000);
        if !(100..=max_radius).contains(&radius) {
            return Err(AppError::OutOfRange(format!(
                "radius 必须在 100 到 {} 之间",
                max_radius
            )));
        }
        Ok(radius)
    }

    /// 验证必填搜索半径 (f64)
    pub fn validate_radius_required(radius: f64, max_radius: f64) -> Result<f64> {
        if !(100.0..=max_radius).contains(&radius) {
            return Err(AppError::OutOfRange(format!(
                "radius 必须在 100 到 {} 之间",
                max_radius
            )));
        }
        Ok(radius)
    }

    /// 验证非空字符串
    pub fn validate_not_empty(value: &str, field_name: &str) -> Result<()> {
        if value.trim().is_empty() {
            return Err(AppError::BadRequest(format!("{} 不能为空", field_name)));
        }
        Ok(())
    }

    /// 验证字符串长度
    pub fn validate_length(value: &str, field_name: &str, min: usize, max: usize) -> Result<()> {
        let len = value.chars().count();
        if len < min || len > max {
            return Err(AppError::BadRequest(format!(
                "{} 长度必须在 {} 到 {} 之间",
                field_name, min, max
            )));
        }
        Ok(())
    }

    /// 验证地理坐标合法性
    pub fn validate_coordinates(latitude: f64, longitude: f64) -> Result<()> {
        if !(-90.0..=90.0).contains(&latitude) {
            return Err(AppError::BadRequest(
                "纬度必须在 -90 到 90 之间".to_string(),
            ));
        }
        if !(-180.0..=180.0).contains(&longitude) {
            return Err(AppError::BadRequest(
                "经度必须在 -180 到 180 之间".to_string(),
            ));
        }
        Ok(())
    }

    /// 解析用户 ID (String -> i64)
    pub fn parse_user_id(id: &str) -> Result<i64> {
        id.parse::<i64>()
            .map_err(|_| AppError::InvalidFormat("无效的用户 ID".to_string()))
    }

    /// 解析群组 ID (String -> i64)
    pub fn parse_group_id(id: &str) -> Result<i64> {
        id.parse::<i64>()
            .map_err(|_| AppError::InvalidFormat("无效的群组 ID".to_string()))
    }

    /// 解析角色 ID (String -> Uuid)
    pub fn parse_role_id(id: &str) -> Result<Uuid> {
        Uuid::parse_str(id).map_err(|_| AppError::InvalidFormat("无效的角色 ID".to_string()))
    }

    /// 解析 UUID 字符串（通用方法，field_name 用于错误消息）
    pub fn parse_uuid(id: &str, field_name: &str) -> Result<Uuid> {
        Uuid::parse_str(id).map_err(|_| AppError::InvalidFormat(format!("无效的{field_name}")))
    }

    /// 解析 i64 ID 字符串（通用方法，field_name 用于错误消息）
    pub fn parse_id(id: &str, field_name: &str) -> Result<i64> {
        id.parse::<i64>()
            .map_err(|_| AppError::InvalidFormat(format!("无效的{field_name}")))
    }

    /// 解析 @ 提及的用户 ID 列表（String -> i64）
    pub fn parse_mentions(mentions: Option<&[String]>) -> Result<Option<Vec<i64>>> {
        let Some(ids) = mentions else {
            return Ok(None);
        };
        let mut parsed = Vec::with_capacity(ids.len());
        let mut seen = HashSet::with_capacity(ids.len());
        for id in ids {
            let parsed_id = Self::parse_user_id(id)?;
            if seen.insert(parsed_id) {
                parsed.push(parsed_id);
            }
        }
        if parsed.len() > 20 {
            return Err(AppError::OutOfRange("mentions 最多 20 个".to_string()));
        }
        Ok(Some(parsed))
    }
}
