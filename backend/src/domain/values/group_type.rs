/// 群组类型值对象
/// 
/// 定义群组的类型分类
use serde::{Deserialize, Serialize};
use std::fmt::{self, Display};
use sqlx::{Type};

/// 群组类型枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "group_type", rename_all = "lowercase")]
pub enum GroupType {
    /// 公开群组
    Public,
    /// 私有群组
    Private,
    /// 临时群组
    Temporary,
}

impl GroupType {
    /// 获取所有群组类型
    pub fn all() -> Vec<Self> {
        vec![Self::Public, Self::Private, Self::Temporary]
    }
    
    /// 是否为公开群组
    pub fn is_public(&self) -> bool {
        matches!(self, Self::Public)
    }
    
    /// 是否为私有群组
    pub fn is_private(&self) -> bool {
        matches!(self, Self::Private)
    }
    
    /// 是否为临时群组
    pub fn is_temporary(&self) -> bool {
        matches!(self, Self::Temporary)
    }
}

impl Display for GroupType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let s = match self {
            Self::Public => "公开",
            Self::Private => "私有",
            Self::Temporary => "临时",
        };
        write!(f, "{}", s)
    }
}

impl Default for GroupType {
    fn default() -> Self {
        Self::Public
    }
}

/// 群组可见性枚举
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, Type)]
#[sqlx(type_name = "group_visibility", rename_all = "lowercase")]
pub enum GroupVisibility {
    /// 公开可见
    Public,
    /// 仅成员可见
    MembersOnly,
    /// 隐藏
    Hidden,
}

impl GroupVisibility {
    /// 获取所有可见性选项
    pub fn all() -> Vec<Self> {
        vec![Self::Public, Self::MembersOnly, Self::Hidden]
    }
    
    /// 是否公开可见
    pub fn is_public(&self) -> bool {
        matches!(self, Self::Public)
    }
    
    /// 是否仅成员可见
    pub fn is_members_only(&self) -> bool {
        matches!(self, Self::MembersOnly)
    }
    
    /// 是否隐藏
    pub fn is_hidden(&self) -> bool {
        matches!(self, Self::Hidden)
    }
}

impl Display for GroupVisibility {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let s = match self {
            Self::Public => "公开",
            Self::MembersOnly => "仅成员",
            Self::Hidden => "隐藏",
        };
        write!(f, "{}", s)
    }
}

impl Default for GroupVisibility {
    fn default() -> Self {
        Self::Public
    }
}


