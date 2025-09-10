/// 分页值对象
/// 
/// 提供类型安全的分页参数和结果处理
use serde::{Deserialize, Serialize};
use std::fmt::{self, Display};

use crate::domain::DomainError;

/// 游标值对象
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Serialize, Deserialize)]
pub struct Cursor(i64);

impl Cursor {
    /// 创建游标
    pub fn new(value: i64) -> Result<Self, DomainError> {
        if value < 0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "游标不能为负数".to_string(),
            });
        }
        Ok(Cursor(value))
    }

    /// 创建起始游标
    pub fn start() -> Self {
        Cursor(0)
    }

    /// 从可信值创建（不验证）
    pub fn from_trusted(value: i64) -> Self {
        Cursor(value)
    }

    /// 获取原始值
    pub fn value(&self) -> i64 {
        self.0
    }

    /// 检查是否为起始游标
    pub fn is_start(&self) -> bool {
        self.0 == 0
    }

    /// 获取下一个游标
    pub fn next(&self) -> Self {
        Cursor(self.0 + 1)
    }
}

impl Display for Cursor {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl From<Cursor> for i64 {
    fn from(cursor: Cursor) -> Self {
        cursor.0
    }
}

impl TryFrom<i64> for Cursor {
    type Error = DomainError;

    fn try_from(value: i64) -> Result<Self, Self::Error> {
        Cursor::new(value)
    }
}

/// 分页大小值对象
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Serialize, Deserialize)]
pub struct PageSize(i32);

impl PageSize {
    /// 最小分页大小
    pub const MIN_SIZE: i32 = 1;
    /// 最大分页大小
    pub const MAX_SIZE: i32 = 100;
    /// 默认分页大小
    pub const DEFAULT_SIZE: i32 = 20;

    /// 创建分页大小
    pub fn new(size: i32) -> Result<Self, DomainError> {
        if size < Self::MIN_SIZE {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("分页大小不能小于{}", Self::MIN_SIZE),
            });
        }
        
        if size > Self::MAX_SIZE {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("分页大小不能大于{}", Self::MAX_SIZE),
            });
        }
        
        Ok(PageSize(size))
    }

    /// 创建默认分页大小
    pub fn default() -> Self {
        PageSize(Self::DEFAULT_SIZE)
    }

    /// 获取原始值
    pub fn value(&self) -> i32 {
        self.0
    }

    /// 计算查询大小（多查询一条用于判断是否有更多）
    pub fn query_size(&self) -> i64 {
        (self.0 + 1) as i64
    }
}

impl Display for PageSize {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl From<PageSize> for i32 {
    fn from(size: PageSize) -> Self {
        size.0
    }
}

impl TryFrom<i32> for PageSize {
    type Error = DomainError;

    fn try_from(value: i32) -> Result<Self, Self::Error> {
        PageSize::new(value)
    }
}

/// 分页请求参数
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PaginationRequest {
    pub cursor: Cursor,
    pub page_size: PageSize,
}

impl PaginationRequest {
    /// 创建分页请求
    pub fn new(cursor: Cursor, page_size: PageSize) -> Self {
        Self { cursor, page_size }
    }

    /// 创建首页请求
    pub fn first_page(page_size: Option<PageSize>) -> Self {
        Self {
            cursor: Cursor::start(),
            page_size: page_size.unwrap_or_else(PageSize::default),
        }
    }

    /// 从原始值创建
    pub fn from_raw(cursor: i64, page_size: i32) -> Result<Self, DomainError> {
        Ok(Self {
            cursor: Cursor::new(cursor)?,
            page_size: PageSize::new(page_size)?,
        })
    }

    /// 获取查询参数
    pub fn query_params(&self) -> (i64, i64) {
        (self.cursor.value(), self.page_size.query_size())
    }

    /// 检查是否为首页请求
    pub fn is_first_page(&self) -> bool {
        self.cursor.is_start()
    }
}

impl Default for PaginationRequest {
    fn default() -> Self {
        Self::first_page(None)
    }
}

/// 分页元信息
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PaginationMeta {
    pub has_more: bool,
    pub next_cursor: Option<Cursor>,
    pub page_size: PageSize,
}

impl PaginationMeta {
    /// 创建分页元信息
    pub fn new(has_more: bool, next_cursor: Option<Cursor>, page_size: PageSize) -> Self {
        Self {
            has_more,
            next_cursor,
            page_size,
        }
    }

    /// 从查询结果创建
    pub fn from_query_result<T>(
        items: &[T],
        requested_size: PageSize,
        get_id: impl Fn(&T) -> i64,
    ) -> (Self, bool) {
        let has_more = items.len() > requested_size.value() as usize;
        
        let next_cursor = if has_more && !items.is_empty() {
            // 获取倒数第二个项目的ID作为下一页游标（因为最后一个是多查询的）
            let cursor_index = requested_size.value() as usize - 1;
            Some(Cursor::from_trusted(get_id(&items[cursor_index])))
        } else {
            None
        };

        let meta = Self::new(has_more, next_cursor, requested_size);
        (meta, has_more)
    }

    /// 检查是否有下一页
    pub fn has_next_page(&self) -> bool {
        self.has_more && self.next_cursor.is_some()
    }

    /// 获取下一页请求
    pub fn next_page_request(&self) -> Option<PaginationRequest> {
        self.next_cursor.map(|cursor| {
            PaginationRequest::new(cursor, self.page_size)
        })
    }
}

/// 分页响应
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PaginatedResponse<T> {
    pub items: Vec<T>,
    pub pagination: PaginationMeta,
}

impl<T> PaginatedResponse<T> {
    /// 创建分页响应
    pub fn new(items: Vec<T>, pagination: PaginationMeta) -> Self {
        Self { items, pagination }
    }

    /// 从查询结果创建
    pub fn from_query_result(
        mut items: Vec<T>,
        request: &PaginationRequest,
        get_id: impl Fn(&T) -> i64,
    ) -> Self {
        let (pagination, has_more) = PaginationMeta::from_query_result(
            &items,
            request.page_size,
            get_id,
        );

        // 如果有更多数据，移除多查询的最后一条
        if has_more {
            items.pop();
        }

        Self::new(items, pagination)
    }

    /// 获取项目数量
    pub fn len(&self) -> usize {
        self.items.len()
    }

    /// 检查是否为空
    pub fn is_empty(&self) -> bool {
        self.items.is_empty()
    }

    /// 映射项目类型
    pub fn map<U, F>(self, f: F) -> PaginatedResponse<U>
    where
        F: FnMut(T) -> U,
    {
        PaginatedResponse {
            items: self.items.into_iter().map(f).collect(),
            pagination: self.pagination,
        }
    }

    /// 过滤项目
    pub fn filter<F>(mut self, mut f: F) -> Self
    where
        F: FnMut(&T) -> bool,
    {
        self.items.retain(|item| f(item));
        self
    }
}

/// 分页构建器
pub struct PaginationBuilder {
    cursor: Option<Cursor>,
    page_size: Option<PageSize>,
}

impl PaginationBuilder {
    /// 创建构建器
    pub fn new() -> Self {
        Self {
            cursor: None,
            page_size: None,
        }
    }

    /// 设置游标
    pub fn cursor(mut self, cursor: Cursor) -> Self {
        self.cursor = Some(cursor);
        self
    }

    /// 设置游标（从原始值）
    pub fn cursor_raw(mut self, cursor: i64) -> Result<Self, DomainError> {
        self.cursor = Some(Cursor::new(cursor)?);
        Ok(self)
    }

    /// 设置分页大小
    pub fn page_size(mut self, page_size: PageSize) -> Self {
        self.page_size = Some(page_size);
        self
    }

    /// 设置分页大小（从原始值）
    pub fn page_size_raw(mut self, page_size: i32) -> Result<Self, DomainError> {
        self.page_size = Some(PageSize::new(page_size)?);
        Ok(self)
    }

    /// 构建分页请求
    pub fn build(self) -> PaginationRequest {
        PaginationRequest {
            cursor: self.cursor.unwrap_or_else(Cursor::start),
            page_size: self.page_size.unwrap_or_else(PageSize::default),
        }
    }
}

impl Default for PaginationBuilder {
    fn default() -> Self {
        Self::new()
    }
}

/// 分页辅助函数
pub struct PaginationHelper;

impl PaginationHelper {
    /// 验证分页参数
    pub fn validate_params(cursor: i64, page_size: i32) -> Result<PaginationRequest, DomainError> {
        PaginationRequest::from_raw(cursor, page_size)
    }

    /// 处理查询结果
    pub fn process_query_result<T>(
        items: Vec<T>,
        request: &PaginationRequest,
        get_id: impl Fn(&T) -> i64,
    ) -> PaginatedResponse<T> {
        PaginatedResponse::from_query_result(items, request, get_id)
    }

    /// 创建空响应
    pub fn empty_response<T>(request: &PaginationRequest) -> PaginatedResponse<T> {
        PaginatedResponse::new(
            Vec::new(),
            PaginationMeta::new(false, None, request.page_size),
        )
    }

    /// 计算总页数（估算）
    pub fn estimate_total_pages(total_items: u64, page_size: PageSize) -> u64 {
        (total_items + page_size.value() as u64 - 1) / page_size.value() as u64
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_cursor() {
        // 有效游标
        let cursor = Cursor::new(123).unwrap();
        assert_eq!(cursor.value(), 123);
        assert!(!cursor.is_start());

        // 起始游标
        let start_cursor = Cursor::start();
        assert_eq!(start_cursor.value(), 0);
        assert!(start_cursor.is_start());

        // 无效游标
        assert!(Cursor::new(-1).is_err());
    }

    #[test]
    fn test_page_size() {
        // 有效分页大小
        let page_size = PageSize::new(20).unwrap();
        assert_eq!(page_size.value(), 20);
        assert_eq!(page_size.query_size(), 21);

        // 默认分页大小
        let default_size = PageSize::default();
        assert_eq!(default_size.value(), PageSize::DEFAULT_SIZE);

        // 无效分页大小
        assert!(PageSize::new(0).is_err());
        assert!(PageSize::new(101).is_err());
    }

    #[test]
    fn test_pagination_request() {
        // 创建分页请求
        let cursor = Cursor::new(10).unwrap();
        let page_size = PageSize::new(20).unwrap();
        let request = PaginationRequest::new(cursor, page_size);

        assert_eq!(request.cursor.value(), 10);
        assert_eq!(request.page_size.value(), 20);
        assert!(!request.is_first_page());

        // 首页请求
        let first_page = PaginationRequest::first_page(None);
        assert!(first_page.is_first_page());
        assert_eq!(first_page.page_size.value(), PageSize::DEFAULT_SIZE);
    }

    #[test]
    fn test_paginated_response() {
        let items = vec![1, 2, 3, 4, 5];
        let request = PaginationRequest::from_raw(0, 4).unwrap();
        
        let response = PaginatedResponse::from_query_result(
            items,
            &request,
            |&item| item as i64,
        );

        assert_eq!(response.len(), 4);
        assert!(response.pagination.has_more);
        assert_eq!(response.pagination.next_cursor.unwrap().value(), 4);
    }

    #[test]
    fn test_pagination_builder() {
        let request = PaginationBuilder::new()
            .cursor_raw(10).unwrap()
            .page_size_raw(25).unwrap()
            .build();

        assert_eq!(request.cursor.value(), 10);
        assert_eq!(request.page_size.value(), 25);
    }

    #[test]
    fn test_pagination_helper() {
        // 验证参数
        let request = PaginationHelper::validate_params(0, 20).unwrap();
        assert!(request.is_first_page());

        // 处理查询结果
        let items = vec![1, 2, 3];
        let response = PaginationHelper::process_query_result(
            items,
            &request,
            |&item| item as i64,
        );
        assert_eq!(response.len(), 3);
        assert!(!response.pagination.has_more);

        // 空响应
        let empty = PaginationHelper::empty_response::<i32>(&request);
        assert!(empty.is_empty());
        assert!(!empty.pagination.has_more);
    }
}
