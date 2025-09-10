/// 内容值对象
/// 
/// 提供类型安全的内容验证和处理
use serde::{Deserialize, Serialize};
use std::fmt::{self, Display};

use crate::domain::DomainError;

/// 文本内容值对象
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct TextContent {
    value: String,
    max_length: usize,
}

impl TextContent {
    /// 创建文本内容
    pub fn new(value: String, max_length: usize) -> Result<Self, DomainError> {
        let trimmed = value.trim().to_string();
        
        if trimmed.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "文本内容不能为空".to_string(),
            });
        }
        
        if trimmed.len() > max_length {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("文本内容长度不能超过{}个字符", max_length),
            });
        }
        
        Ok(TextContent {
            value: trimmed,
            max_length,
        })
    }

    /// 获取原始值
    pub fn value(&self) -> &str {
        &self.value
    }

    /// 获取长度
    pub fn len(&self) -> usize {
        self.value.len()
    }

    /// 检查是否为空
    pub fn is_empty(&self) -> bool {
        self.value.is_empty()
    }

    /// 获取预览（截断显示）
    pub fn preview(&self, max_preview_length: usize) -> String {
        if self.value.len() <= max_preview_length {
            self.value.clone()
        } else {
            format!("{}...", &self.value[..max_preview_length])
        }
    }

    /// 检查是否包含关键词
    pub fn contains_keyword(&self, keyword: &str) -> bool {
        self.value.to_lowercase().contains(&keyword.to_lowercase())
    }

    /// 检查是否包含任何关键词
    pub fn contains_any_keywords(&self, keywords: &[String]) -> bool {
        let content_lower = self.value.to_lowercase();
        keywords.iter().any(|keyword| {
            content_lower.contains(&keyword.to_lowercase())
        })
    }
}

impl Display for TextContent {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.value)
    }
}

impl AsRef<str> for TextContent {
    fn as_ref(&self) -> &str {
        &self.value
    }
}

/// 消息内容值对象
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct MessageContent(TextContent);

impl MessageContent {
    /// 消息内容最大长度
    pub const MAX_LENGTH: usize = 2000;

    /// 创建消息内容
    pub fn new(value: String) -> Result<Self, DomainError> {
        Ok(MessageContent(TextContent::new(value, Self::MAX_LENGTH)?))
    }

    /// 获取内容
    pub fn content(&self) -> &str {
        self.0.value()
    }

    /// 获取预览
    pub fn preview(&self) -> String {
        self.0.preview(50)
    }

    /// 检查是否为系统消息格式
    pub fn is_system_message(&self) -> bool {
        self.0.value().starts_with("[系统]") || self.0.value().starts_with("[System]")
    }
}

impl Display for MessageContent {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

/// 群组名称值对象
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct GroupName(TextContent);

impl GroupName {
    /// 群组名称最大长度
    pub const MAX_LENGTH: usize = 50;

    /// 创建群组名称
    pub fn new(value: String) -> Result<Self, DomainError> {
        Ok(GroupName(TextContent::new(value, Self::MAX_LENGTH)?))
    }

    /// 获取名称
    pub fn name(&self) -> &str {
        self.0.value()
    }
}

impl Display for GroupName {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

/// 用户昵称值对象
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Nickname(TextContent);

impl Nickname {
    /// 昵称最大长度
    pub const MAX_LENGTH: usize = 32;
    /// 昵称最小长度
    pub const MIN_LENGTH: usize = 2;

    /// 创建昵称
    pub fn new(value: String) -> Result<Self, DomainError> {
        let content = TextContent::new(value, Self::MAX_LENGTH)?;
        
        if content.len() < Self::MIN_LENGTH {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("昵称长度不能少于{}个字符", Self::MIN_LENGTH),
            });
        }
        
        Ok(Nickname(content))
    }

    /// 获取昵称
    pub fn nickname(&self) -> &str {
        self.0.value()
    }

    /// 检查是否为有效昵称格式
    pub fn is_valid_format(&self) -> bool {
        // 简单的昵称格式检查：不能全是数字或特殊字符
        let nickname = self.0.value();
        !nickname.chars().all(|c| c.is_numeric() || c.is_ascii_punctuation())
    }
}

impl Display for Nickname {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

/// 描述内容值对象
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Description(TextContent);

impl Description {
    /// 描述最大长度
    pub const MAX_LENGTH: usize = 500;

    /// 创建描述
    pub fn new(value: String) -> Result<Self, DomainError> {
        Ok(Description(TextContent::new(value, Self::MAX_LENGTH)?))
    }

    /// 获取描述
    pub fn description(&self) -> &str {
        self.0.value()
    }

    /// 获取摘要（更短的预览）
    pub fn summary(&self) -> String {
        self.0.preview(100)
    }
}

impl Display for Description {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

/// 标签值对象
#[derive(Debug, Clone, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct Tag(String);

impl Tag {
    /// 标签最大长度
    pub const MAX_LENGTH: usize = 20;
    /// 标签最小长度
    pub const MIN_LENGTH: usize = 1;

    /// 创建标签
    pub fn new(value: String) -> Result<Self, DomainError> {
        let normalized = Self::normalize_tag(&value);
        
        if normalized.is_empty() {
            return Err(DomainError::BusinessRuleViolation {
                rule: "标签不能为空".to_string(),
            });
        }
        
        if normalized.len() < Self::MIN_LENGTH {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("标签长度不能少于{}个字符", Self::MIN_LENGTH),
            });
        }
        
        if normalized.len() > Self::MAX_LENGTH {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("标签长度不能超过{}个字符", Self::MAX_LENGTH),
            });
        }
        
        Ok(Tag(normalized))
    }

    /// 获取标签值
    pub fn value(&self) -> &str {
        &self.0
    }

    /// 标准化标签（去除空格、转小写等）
    fn normalize_tag(tag: &str) -> String {
        tag.trim()
            .to_lowercase()
            .replace(' ', "")
            .chars()
            .filter(|c| c.is_alphanumeric() || *c == '_' || *c == '-')
            .collect()
    }

    /// 检查标签是否匹配搜索词
    pub fn matches(&self, search_term: &str) -> bool {
        let normalized_search = Self::normalize_tag(search_term);
        self.0.contains(&normalized_search)
    }
}

impl Display for Tag {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "#{}", self.0)
    }
}

/// 标签集合
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct TagSet {
    tags: Vec<Tag>,
}

impl TagSet {
    /// 最大标签数量
    pub const MAX_TAGS: usize = 10;

    /// 创建标签集合
    pub fn new() -> Self {
        Self { tags: Vec::new() }
    }

    /// 从字符串列表创建
    pub fn from_strings(tag_strings: Vec<String>) -> Result<Self, DomainError> {
        if tag_strings.len() > Self::MAX_TAGS {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("标签数量不能超过{}", Self::MAX_TAGS),
            });
        }

        let mut tags = Vec::new();
        for tag_string in tag_strings {
            let tag = Tag::new(tag_string)?;
            if !tags.contains(&tag) {
                tags.push(tag);
            }
        }

        Ok(Self { tags })
    }

    /// 添加标签
    pub fn add_tag(&mut self, tag: Tag) -> Result<(), DomainError> {
        if self.tags.len() >= Self::MAX_TAGS {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("标签数量不能超过{}", Self::MAX_TAGS),
            });
        }

        if !self.tags.contains(&tag) {
            self.tags.push(tag);
        }

        Ok(())
    }

    /// 移除标签
    pub fn remove_tag(&mut self, tag: &Tag) -> bool {
        if let Some(pos) = self.tags.iter().position(|t| t == tag) {
            self.tags.remove(pos);
            true
        } else {
            false
        }
    }

    /// 获取所有标签
    pub fn tags(&self) -> &[Tag] {
        &self.tags
    }

    /// 获取标签字符串列表
    pub fn tag_strings(&self) -> Vec<String> {
        self.tags.iter().map(|tag| tag.value().to_string()).collect()
    }

    /// 检查是否包含标签
    pub fn contains(&self, tag: &Tag) -> bool {
        self.tags.contains(tag)
    }

    /// 检查是否包含任何指定标签
    pub fn contains_any(&self, tags: &[Tag]) -> bool {
        tags.iter().any(|tag| self.contains(tag))
    }

    /// 获取标签数量
    pub fn len(&self) -> usize {
        self.tags.len()
    }

    /// 检查是否为空
    pub fn is_empty(&self) -> bool {
        self.tags.is_empty()
    }
}

impl Default for TagSet {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_text_content() {
        // 有效内容
        let content = TextContent::new("Hello World".to_string(), 100).unwrap();
        assert_eq!(content.value(), "Hello World");
        assert_eq!(content.len(), 11);

        // 空内容
        assert!(TextContent::new("".to_string(), 100).is_err());
        assert!(TextContent::new("   ".to_string(), 100).is_err());

        // 超长内容
        assert!(TextContent::new("a".repeat(101), 100).is_err());
    }

    #[test]
    fn test_message_content() {
        let content = MessageContent::new("Hello World".to_string()).unwrap();
        assert_eq!(content.content(), "Hello World");
        assert_eq!(content.preview(), "Hello World");

        // 系统消息检测
        let system_msg = MessageContent::new("[系统] 用户加入群组".to_string()).unwrap();
        assert!(system_msg.is_system_message());
    }

    #[test]
    fn test_nickname() {
        // 有效昵称
        let nickname = Nickname::new("张三".to_string()).unwrap();
        assert_eq!(nickname.nickname(), "张三");

        // 太短的昵称
        assert!(Nickname::new("a".to_string()).is_err());

        // 太长的昵称
        assert!(Nickname::new("a".repeat(50), ).is_err());
    }

    #[test]
    fn test_tag() {
        // 有效标签
        let tag = Tag::new("Rust".to_string()).unwrap();
        assert_eq!(tag.value(), "rust");

        // 标签标准化
        let tag = Tag::new(" Hello World! ".to_string()).unwrap();
        assert_eq!(tag.value(), "helloworld");

        // 标签匹配
        assert!(tag.matches("hello"));
        assert!(tag.matches("WORLD"));
    }

    #[test]
    fn test_tag_set() {
        let mut tag_set = TagSet::new();
        
        // 添加标签
        let tag1 = Tag::new("rust".to_string()).unwrap();
        let tag2 = Tag::new("programming".to_string()).unwrap();
        
        tag_set.add_tag(tag1.clone()).unwrap();
        tag_set.add_tag(tag2.clone()).unwrap();
        
        assert_eq!(tag_set.len(), 2);
        assert!(tag_set.contains(&tag1));
        assert!(tag_set.contains(&tag2));

        // 重复添加
        tag_set.add_tag(tag1.clone()).unwrap();
        assert_eq!(tag_set.len(), 2); // 不应该增加

        // 移除标签
        assert!(tag_set.remove_tag(&tag1));
        assert_eq!(tag_set.len(), 1);
        assert!(!tag_set.contains(&tag1));
    }
}
