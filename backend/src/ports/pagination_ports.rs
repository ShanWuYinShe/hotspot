/// 通用分页查询端口
/// 
/// 提供统一的分页查询接口，减少Repository层代码重复
use async_trait::async_trait;
use crate::domain::DomainResult;

/// 分页参数
#[derive(Debug, Clone)]
pub struct PaginationParams {
    pub cursor: i64,
    pub limit: i32,
}

impl PaginationParams {
    pub fn new(cursor: i64, limit: i32) -> Self {
        Self { cursor, limit }
    }
    
    pub fn first_page(limit: i32) -> Self {
        Self { cursor: 0, limit }
    }
}

/// 分页结果
#[derive(Debug, Clone)]
pub struct PaginatedResult<T> {
    pub items: Vec<T>,
    pub has_more: bool,
    pub next_cursor: Option<i64>,
}

impl<T> PaginatedResult<T> {
    pub fn new(items: Vec<T>, has_more: bool, next_cursor: Option<i64>) -> Self {
        Self {
            items,
            has_more,
            next_cursor,
        }
    }
    
    /// 从查询结果创建分页结果（自动处理has_more逻辑）
    pub fn from_query_result(mut items: Vec<T>, requested_limit: i32, get_id: impl Fn(&T) -> i64) -> Self {
        let has_more = items.len() > requested_limit as usize;
        
        if has_more {
            items.pop(); // 移除多查询的一条记录
        }
        
        let next_cursor = if has_more && !items.is_empty() {
            Some(get_id(items.last().unwrap()))
        } else {
            None
        };
        
        Self::new(items, has_more, next_cursor)
    }
}

/// 通用分页查询trait
#[async_trait]
pub trait PaginatedQuery<T> {
    /// 执行分页查询
    async fn paginated_query(
        &self,
        params: PaginationParams,
    ) -> DomainResult<PaginatedResult<T>>;
}

/// 地理位置分页查询参数
#[derive(Debug, Clone)]
pub struct GeoPaginationParams {
    pub latitude: f64,
    pub longitude: f64,
    pub radius_meters: f64,
    pub cursor: i64,
    pub limit: i32,
}

impl GeoPaginationParams {
    pub fn new(latitude: f64, longitude: f64, radius_meters: f64, cursor: i64, limit: i32) -> Self {
        Self {
            latitude,
            longitude,
            radius_meters,
            cursor,
            limit,
        }
    }
}

/// 地理位置分页查询trait
#[async_trait]
pub trait GeoPaginatedQuery<T> {
    /// 执行地理位置分页查询
    async fn geo_paginated_query(
        &self,
        params: GeoPaginationParams,
    ) -> DomainResult<PaginatedResult<T>>;
}

/// 搜索分页查询参数
#[derive(Debug, Clone)]
pub struct SearchPaginationParams {
    pub query: String,
    pub cursor: i64,
    pub limit: i32,
}

impl SearchPaginationParams {
    pub fn new(query: String, cursor: i64, limit: i32) -> Self {
        Self { query, cursor, limit }
    }
}

/// 搜索分页查询trait
#[async_trait]
pub trait SearchPaginatedQuery<T> {
    /// 执行搜索分页查询
    async fn search_paginated_query(
        &self,
        params: SearchPaginationParams,
    ) -> DomainResult<PaginatedResult<T>>;
}
