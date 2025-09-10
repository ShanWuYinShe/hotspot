/// 查询优化管理控制器
/// 
/// 提供数据库查询优化分析和建议的HTTP接口
use axum::{Json, response::Response, extract::State};
use serde::{Deserialize, Serialize};
use crate::adapters::adapters_pg::query_optimizer::{PgQueryOptimizer, OptimizationSuggestion};
use crate::presentation::http::controllers::{Controller, ok_response, error_response};
use crate::presentation::http::auth::CurrentUser;
use crate::domain::DomainResult;

#[derive(Clone)]
pub struct OptimizationController {
    query_optimizer: PgQueryOptimizer,
}

impl OptimizationController {
    pub fn new(query_optimizer: PgQueryOptimizer) -> Self {
        Self { query_optimizer }
    }
    
    /// 获取地理查询优化建议
    pub async fn get_geo_optimization_suggestions(
        State(controller): State<Self>,
        axum::extract::Extension(_current_user): axum::extract::Extension<CurrentUser>,
    ) -> Response {
        match controller.query_optimizer.optimize_geo_query_performance().await {
            Ok(suggestions) => {
                let response = serde_json::json!({
                    "code": 0,
                    "msg": "获取优化建议成功",
                    "data": {
                        "suggestions": suggestions,
                        "total_count": suggestions.len()
                    }
                });
                ok_response(response)
            }
            Err(e) => error_response(5000, &format!("获取优化建议失败: {}", e)),
        }
    }
    
    /// 检查空间索引状态
    pub async fn check_spatial_indexes(
        State(controller): State<Self>,
        axum::extract::Extension(_current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<CheckSpatialIndexRequest>,
    ) -> Response {
        match controller.query_optimizer.check_spatial_index_usage(&request.table_name).await {
            Ok(index_info) => {
                let response = serde_json::json!({
                    "code": 0,
                    "msg": "检查空间索引成功",
                    "data": index_info
                });
                ok_response(response)
            }
            Err(e) => error_response(5000, &format!("检查空间索引失败: {}", e)),
        }
    }
    
    /// 分析查询计划
    pub async fn analyze_query_plan(
        State(controller): State<Self>,
        axum::extract::Extension(_current_user): axum::extract::Extension<CurrentUser>,
        Json(request): Json<AnalyzeQueryRequest>,
    ) -> Response {
        match controller.query_optimizer.analyze_query_plan(&request.sql, &[]).await {
            Ok(plan) => {
                let response = serde_json::json!({
                    "code": 0,
                    "msg": "分析查询计划成功",
                    "data": plan
                });
                ok_response(response)
            }
            Err(e) => error_response(5000, &format!("分析查询计划失败: {}", e)),
        }
    }
}

/// 检查空间索引请求
#[derive(Debug, Deserialize)]
pub struct CheckSpatialIndexRequest {
    pub table_name: String,
}

/// 分析查询请求
#[derive(Debug, Deserialize)]
pub struct AnalyzeQueryRequest {
    pub sql: String,
}

impl Controller for OptimizationController {}
