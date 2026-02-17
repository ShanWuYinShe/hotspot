//! 信息流相关响应类型

use crate::types::enums::RecommendationType;
use serde::Serialize;

use super::moment::MomentItem;

/// 推荐上下文（推荐理由）- API 文档: RecommendationContext
#[derive(Debug, Serialize, Clone)]
pub struct RecommendationContext {
    #[serde(rename = "type")]
    pub recommendation_type: RecommendationType,
}

/// 热门标签项 - API 文档: TrendingTagItem
#[derive(Debug, Serialize)]
pub struct TrendingTagItem {
    pub name: String,
    pub usage_count: i32,
    pub trend: crate::types::enums::TagTrend,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hot_moment_preview: Option<MomentItem>,
}

/// 信息流热门数据 - API 文档: FeedTrendingData
#[derive(Debug, Serialize)]
pub struct FeedTrendingData {
    pub trending_tags: Vec<TrendingTagItem>,
    pub hot_moments: Vec<MomentItem>,
}
