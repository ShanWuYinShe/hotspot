/// 全局搜索应用服务
/// 
/// 提供统一的搜索功能，整合用户、群组、生活瞬间的搜索结果
use std::sync::Arc;
use async_trait::async_trait;
use tracing::{info, warn, error};

use crate::{
    domain::{DomainResult, DomainError},
    ports::{UserRepositoryPort, GroupRepositoryPort, MomentRepositoryPort},
    application::queries::{
        SearchByGlobalQuery, SearchResult, SearchResultItem, SearchType,
        SearchUsersByNameQuery, SearchGroupsByNameQuery, SearchMomentsByLocationQuery,
    },
};

/// 搜索应用服务
pub struct SearchApplicationService<U, G, M> 
where
    U: UserRepositoryPort,
    G: GroupRepositoryPort,
    M: MomentRepositoryPort,
{
    user_repository: Arc<U>,
    group_repository: Arc<G>,
    moment_repository: Arc<M>,
}

impl<U, G, M> SearchApplicationService<U, G, M>
where
    U: UserRepositoryPort,
    G: GroupRepositoryPort,
    M: MomentRepositoryPort,
{
    /// 创建新的搜索服务
    pub fn new(
        user_repository: Arc<U>,
        group_repository: Arc<G>,
        moment_repository: Arc<M>,
    ) -> Self {
        Self {
            user_repository,
            group_repository,
            moment_repository,
        }
    }
    
    /// 执行全局搜索
    pub async fn global_search(&self, query: SearchByGlobalQuery) -> DomainResult<SearchResult> {
        let start_time = std::time::Instant::now();
        
        // 验证查询
        query.validate()?;
        
        let clean_query = query.get_clean_query();
        info!("执行全局搜索: '{}', 类型: {:?}", clean_query, query.search_type);
        
        let mut all_results = Vec::new();
        let mut total_found = 0i64;
        
        // 根据搜索类型执行不同的搜索
        match query.search_type.as_ref().unwrap_or(&SearchType::All) {
            SearchType::All => {
                // 搜索所有类型
                let (user_results, user_count) = self.search_users(&query).await?;
                let (group_results, group_count) = self.search_groups(&query).await?;
                let (moment_results, moment_count) = self.search_moments(&query).await?;
                
                all_results.extend(user_results);
                all_results.extend(group_results);
                all_results.extend(moment_results);
                
                total_found = user_count + group_count + moment_count;
            }
            SearchType::Users => {
                let (results, count) = self.search_users(&query).await?;
                all_results = results;
                total_found = count;
            }
            SearchType::Groups => {
                let (results, count) = self.search_groups(&query).await?;
                all_results = results;
                total_found = count;
            }
            SearchType::Moments => {
                let (results, count) = self.search_moments(&query).await?;
                all_results = results;
                total_found = count;
            }
        }
        
        // 按相关性和距离排序
        all_results.sort_by(|a, b| {
            // 首先按相关性排序
            let relevance_cmp = b.relevance_score.partial_cmp(&a.relevance_score).unwrap_or(std::cmp::Ordering::Equal);
            if relevance_cmp != std::cmp::Ordering::Equal {
                return relevance_cmp;
            }
            
            // 然后按距离排序（如果有位置信息）
            match (a.distance_km, b.distance_km) {
                (Some(dist_a), Some(dist_b)) => dist_a.partial_cmp(&dist_b).unwrap_or(std::cmp::Ordering::Equal),
                (Some(_), None) => std::cmp::Ordering::Less,
                (None, Some(_)) => std::cmp::Ordering::Greater,
                (None, None) => std::cmp::Ordering::Equal,
            }
        });
        
        // 应用分页
        let limit = query.pagination.get_limit() as usize;
        let has_more = all_results.len() > limit;
        
        if has_more {
            all_results.truncate(limit);
        }
        
        let next_cursor = if has_more && !all_results.is_empty() {
            // 这里简化处理，实际应该基于最后一个结果的ID
            Some(query.pagination.get_cursor() + limit as i64)
        } else {
            None
        };
        
        let search_time_ms = start_time.elapsed().as_millis() as u64;
        
        info!("搜索完成: 找到{}个结果, 用时{}ms", all_results.len(), search_time_ms);
        
        Ok(SearchResult {
            items: all_results,
            has_more,
            next_cursor,
            total_found: Some(total_found),
            search_time_ms,
        })
    }
    
    /// 搜索用户
    async fn search_users(&self, query: &SearchByGlobalQuery) -> DomainResult<(Vec<SearchResultItem>, i64)> {
        let user_query = SearchUsersByNameQuery {
            query: query.query.clone(),
            pagination: query.pagination.clone(),
        };
        
        match self.user_repository.search_users_by_name(user_query).await {
            Ok((users, has_more, _next_cursor)) => {
                let results: Vec<SearchResultItem> = users.into_iter().map(|user| {
                    // 计算距离（如果有位置信息）
                    let distance_km = if let (Some(query_lat), Some(query_lng), Some(user_lat), Some(user_lng)) = 
                        (query.latitude, query.longitude, user.latitude, user.longitude) {
                        Some(calculate_distance(query_lat, query_lng, user_lat, user_lng))
                    } else {
                        None
                    };
                    
                    SearchResultItem::from_user(
                        user.user_id,
                        user.nickname,
                        user.avatar_url,
                        distance_km,
                        user.last_active_at.map_or(false, |last_active| {
                            chrono::Utc::now().signed_duration_since(last_active).num_minutes() < 30
                        }),
                        user.created_at.unwrap_or_else(chrono::Utc::now),
                    )
                }).collect();
                
                Ok((results, results.len() as i64))
            }
            Err(e) => {
                warn!("搜索用户失败: {:?}", e);
                Ok((Vec::new(), 0))
            }
        }
    }
    
    /// 搜索群组
    async fn search_groups(&self, query: &SearchByGlobalQuery) -> DomainResult<(Vec<SearchResultItem>, i64)> {
        let group_query = SearchGroupsByNameQuery {
            query: query.query.clone(),
            pagination: query.pagination.clone(),
        };
        
        match self.group_repository.search_groups_by_name(group_query).await {
            Ok((groups, _has_more, _next_cursor)) => {
                let results: Vec<SearchResultItem> = groups.into_iter().map(|group| {
                    // 计算距离（如果有位置信息）
                    let distance_km = if let (Some(query_lat), Some(query_lng)) = 
                        (query.latitude, query.longitude) {
                        Some(calculate_distance(query_lat, query_lng, group.latitude, group.longitude))
                    } else {
                        None
                    };
                    
                    // 判断群组是否活跃（最近有活动）
                    let is_active = group.last_active_at.map_or(false, |last_active| {
                        chrono::Utc::now().signed_duration_since(last_active).num_days() < 7
                    });
                    
                    SearchResultItem::from_group(
                        group.group_id,
                        group.name,
                        group.description,
                        group.member_count,
                        distance_km,
                        is_active,
                        group.created_at,
                    )
                }).collect();
                
                Ok((results, results.len() as i64))
            }
            Err(e) => {
                warn!("搜索群组失败: {:?}", e);
                Ok((Vec::new(), 0))
            }
        }
    }
    
    /// 搜索生活瞬间
    async fn search_moments(&self, query: &SearchByGlobalQuery) -> DomainResult<(Vec<SearchResultItem>, i64)> {
        // 如果有位置信息，按位置搜索；否则搜索所有
        if let (Some(lat), Some(lng)) = (query.latitude, query.longitude) {
            let moment_query = SearchMomentsByLocationQuery {
                latitude: lat,
                longitude: lng,
                radius_km: Some(50.0), // 默认50公里范围
                pagination: query.pagination.clone(),
            };
            
            match self.moment_repository.search_moments_by_location(moment_query).await {
                Ok((moments, _has_more, _next_cursor)) => {
                    // 过滤包含搜索关键词的瞬间
                    let clean_query = query.get_clean_query();
                    let filtered_moments: Vec<_> = moments.into_iter()
                        .filter(|moment| {
                            moment.content.as_ref().map_or(false, |content| {
                                content.to_lowercase().contains(&clean_query)
                            })
                        })
                        .collect();
                    
                    let results: Vec<SearchResultItem> = filtered_moments.into_iter().map(|moment| {
                        let distance_km = Some(calculate_distance(lat, lng, moment.latitude, moment.longitude));
                        let is_popular = moment.likes_count > 20; // 超过20个赞认为是热门
                        
                        SearchResultItem::from_moment(
                            moment.moment_id,
                            moment.content.unwrap_or_default(),
                            moment.nickname,
                            moment.likes_count,
                            distance_km,
                            is_popular,
                            moment.created_at,
                        )
                    }).collect();
                    
                    Ok((results, results.len() as i64))
                }
                Err(e) => {
                    warn!("搜索生活瞬间失败: {:?}", e);
                    Ok((Vec::new(), 0))
                }
            }
        } else {
            // 没有位置信息，暂时返回空结果
            // 实际实现中可以添加基于内容的全文搜索
            Ok((Vec::new(), 0))
        }
    }
}

/// 计算两点间的距离（公里）
fn calculate_distance(lat1: f64, lng1: f64, lat2: f64, lng2: f64) -> f64 {
    const EARTH_RADIUS_KM: f64 = 6371.0;
    
    let lat1_rad = lat1.to_radians();
    let lat2_rad = lat2.to_radians();
    let delta_lat = (lat2 - lat1).to_radians();
    let delta_lng = (lng2 - lng1).to_radians();
    
    let a = (delta_lat / 2.0).sin().powi(2) + 
            lat1_rad.cos() * lat2_rad.cos() * (delta_lng / 2.0).sin().powi(2);
    let c = 2.0 * a.sqrt().atan2((1.0 - a).sqrt());
    
    EARTH_RADIUS_KM * c
}
