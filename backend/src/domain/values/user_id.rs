/// 用户ID值对象
/// 
/// 提供类型安全的用户ID，防止原始类型的误用
use serde::{Deserialize, Serialize};
use std::fmt::{self, Display};
use std::ops::Deref;

use crate::domain::DomainError;

/// 用户ID值对象
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct UserId(i64);

impl UserId {
    /// 创建新的用户ID
    pub fn new(id: i64) -> Result<Self, DomainError> {
        if id <= 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "用户ID必须为正数".to_string(),
            });
        }
        Ok(UserId(id))
    }

    /// 创建用户ID（不验证，用于已知有效的ID）
    pub fn from_trusted(id: i64) -> Self {
        UserId(id)
    }

    /// 获取原始值
    pub fn value(&self) -> i64 {
        self.0
    }

    /// 检查是否为临时用户ID
    pub fn is_temporary(&self) -> bool {
        // 临时用户ID通常在特定范围内，这里假设大于某个值
        self.0 > 1_000_000_000
    }

    /// 检查是否为系统用户ID
    pub fn is_system(&self) -> bool {
        // 系统用户ID通常为负数或特殊值
        self.0 < 0
    }
}

impl Display for UserId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl Deref for UserId {
    type Target = i64;

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl From<UserId> for i64 {
    fn from(user_id: UserId) -> Self {
        user_id.0
    }
}

impl TryFrom<i64> for UserId {
    type Error = DomainError;

    fn try_from(id: i64) -> Result<Self, Self::Error> {
        UserId::new(id)
    }
}

/// 用户ID集合
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserIdSet {
    ids: Vec<UserId>,
}

impl UserIdSet {
    /// 创建新的用户ID集合
    pub fn new() -> Self {
        Self { ids: Vec::new() }
    }

    /// 从ID列表创建
    pub fn from_ids(ids: Vec<i64>) -> Result<Self, DomainError> {
        let user_ids: Result<Vec<_>, _> = ids.into_iter().map(UserId::new).collect();
        Ok(Self { ids: user_ids? })
    }

    /// 添加用户ID
    pub fn add(&mut self, user_id: UserId) {
        if !self.ids.contains(&user_id) {
            self.ids.push(user_id);
        }
    }

    /// 移除用户ID
    pub fn remove(&mut self, user_id: &UserId) -> bool {
        if let Some(pos) = self.ids.iter().position(|id| id == user_id) {
            self.ids.remove(pos);
            true
        } else {
            false
        }
    }

    /// 检查是否包含用户ID
    pub fn contains(&self, user_id: &UserId) -> bool {
        self.ids.contains(user_id)
    }

    /// 获取所有用户ID
    pub fn ids(&self) -> &[UserId] {
        &self.ids
    }

    /// 获取原始ID列表
    pub fn raw_ids(&self) -> Vec<i64> {
        self.ids.iter().map(|id| id.value()).collect()
    }

    /// 获取数量
    pub fn len(&self) -> usize {
        self.ids.len()
    }

    /// 检查是否为空
    pub fn is_empty(&self) -> bool {
        self.ids.is_empty()
    }

    /// 清空
    pub fn clear(&mut self) {
        self.ids.clear();
    }
}

impl Default for UserIdSet {
    fn default() -> Self {
        Self::new()
    }
}

impl IntoIterator for UserIdSet {
    type Item = UserId;
    type IntoIter = std::vec::IntoIter<UserId>;

    fn into_iter(self) -> Self::IntoIter {
        self.ids.into_iter()
    }
}

impl<'a> IntoIterator for &'a UserIdSet {
    type Item = &'a UserId;
    type IntoIter = std::slice::Iter<'a, UserId>;

    fn into_iter(self) -> Self::IntoIter {
        self.ids.iter()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_user_id_creation() {
        // 有效用户ID
        let user_id = UserId::new(123).unwrap();
        assert_eq!(user_id.value(), 123);

        // 无效用户ID
        assert!(UserId::new(0).is_err());
        assert!(UserId::new(-1).is_err());
    }

    #[test]
    fn test_user_id_properties() {
        let normal_id = UserId::from_trusted(123);
        let temp_id = UserId::from_trusted(1_000_000_001);
        let system_id = UserId::from_trusted(-1);

        assert!(!normal_id.is_temporary());
        assert!(!normal_id.is_system());

        assert!(temp_id.is_temporary());
        assert!(!temp_id.is_system());

        assert!(!system_id.is_temporary());
        assert!(system_id.is_system());
    }

    #[test]
    fn test_user_id_set() {
        let mut set = UserIdSet::new();
        let user1 = UserId::from_trusted(1);
        let user2 = UserId::from_trusted(2);

        // 添加用户
        set.add(user1);
        set.add(user2);
        assert_eq!(set.len(), 2);
        assert!(set.contains(&user1));

        // 重复添加
        set.add(user1);
        assert_eq!(set.len(), 2);

        // 移除用户
        assert!(set.remove(&user1));
        assert_eq!(set.len(), 1);
        assert!(!set.contains(&user1));

        // 获取原始ID列表
        let raw_ids = set.raw_ids();
        assert_eq!(raw_ids, vec![2]);
    }
}
