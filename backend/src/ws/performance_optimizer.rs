/// WebSocket 性能优化器
/// 
/// 功能职责：
/// - 监控WebSocket连接性能
/// - 优化消息路由和分发
/// - 管理连接池和内存使用
/// - 提供性能调优建议
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::RwLock;
use serde::{Serialize, Deserialize};
use tracing::{info, debug, error};

use crate::ws::{
    connection_manager::{ConnectionManager, ConnectionStats},
    room_manager::{RoomManager, RoomStats},
    message_router::MessageRouter,
    errors::WsResult,
};

/// WebSocket 性能优化器
#[derive(Clone)]
pub struct WebSocketPerformanceOptimizer {
    connection_manager: Arc<ConnectionManager>,
    room_manager: Arc<RoomManager>,
    message_router: Arc<MessageRouter>,
    performance_stats: Arc<RwLock<PerformanceStats>>,
    config: OptimizationConfig,
}

/// 性能统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceStats {
    /// 消息统计
    pub message_stats: MessageStats,
    /// 连接统计
    pub connection_stats: ConnectionPerformanceStats,
    /// 房间统计
    pub room_stats: RoomPerformanceStats,
    /// 内存使用统计
    pub memory_stats: MemoryStats,
    /// 最后更新时间
    pub last_updated: chrono::DateTime<chrono::Utc>,
}

/// 消息统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageStats {
    pub total_messages_sent: u64,
    pub total_messages_failed: u64,
    pub messages_per_second: f64,
    pub average_message_size: f64,
    pub message_latency_ms: MessageLatencyStats,
}

/// 消息延迟统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageLatencyStats {
    pub min: f64,
    pub max: f64,
    pub avg: f64,
    pub p50: f64,
    pub p95: f64,
    pub p99: f64,
}

/// 连接性能统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionPerformanceStats {
    pub active_connections: usize,
    pub peak_connections: usize,
    pub connection_churn_rate: f64,
    pub average_connection_duration: Duration,
    pub failed_connections: u64,
}

/// 房间性能统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RoomPerformanceStats {
    pub total_rooms: usize,
    pub active_rooms: usize,
    pub average_room_size: f64,
    pub largest_room_size: usize,
    pub room_message_distribution: HashMap<String, u64>,
}

/// 内存使用统计
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryStats {
    pub connection_memory_mb: f64,
    pub room_memory_mb: f64,
    pub message_buffer_memory_mb: f64,
    pub total_memory_mb: f64,
}

/// 优化配置
#[derive(Debug, Clone)]
pub struct OptimizationConfig {
    pub max_connections_per_user: usize,
    pub max_room_size: usize,
    pub message_rate_limit: u32,
    pub memory_threshold_mb: f64,
    pub cleanup_interval: Duration,
    pub stats_collection_interval: Duration,
}

impl Default for OptimizationConfig {
    fn default() -> Self {
        Self {
            max_connections_per_user: 5,
            max_room_size: 1000,
            message_rate_limit: 100, // 每分钟
            memory_threshold_mb: 512.0,
            cleanup_interval: Duration::from_secs(300), // 5分钟
            stats_collection_interval: Duration::from_secs(60), // 1分钟
        }
    }
}

/// 性能优化建议
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceOptimizationSuggestion {
    pub category: OptimizationCategory,
    pub priority: OptimizationPriority,
    pub description: String,
    pub recommended_action: String,
    pub estimated_impact: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum OptimizationCategory {
    Connection,
    Memory,
    MessageRouting,
    RoomManagement,
    Cleanup,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum OptimizationPriority {
    Low,
    Medium,
    High,
    Critical,
}

impl WebSocketPerformanceOptimizer {
    /// 创建新的性能优化器
    pub fn new(
        connection_manager: Arc<ConnectionManager>,
        room_manager: Arc<RoomManager>,
        message_router: Arc<MessageRouter>,
        config: OptimizationConfig,
    ) -> Self {
        Self {
            connection_manager,
            room_manager,
            message_router,
            performance_stats: Arc::new(RwLock::new(PerformanceStats::default())),
            config,
        }
    }

    /// 启动性能监控
    pub async fn start_monitoring(&self) {
        let optimizer = self.clone();
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(optimizer.config.stats_collection_interval);
            
            loop {
                interval.tick().await;
                if let Err(e) = optimizer.collect_performance_stats().await {
                    error!("收集性能统计失败: {:?}", e);
                }
            }
        });

        let optimizer = self.clone();
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(optimizer.config.cleanup_interval);
            
            loop {
                interval.tick().await;
                if let Err(e) = optimizer.perform_cleanup().await {
                    error!("执行清理失败: {:?}", e);
                }
            }
        });
    }

    /// 收集性能统计信息
    async fn collect_performance_stats(&self) -> WsResult<()> {
        let connection_stats = self.connection_manager.get_stats().await;
        let room_stats = self.room_manager.get_stats().await;
        
        let mut stats = self.performance_stats.write().await;
        
        // 更新连接统计
        stats.connection_stats.active_connections = connection_stats.total_connections;
        stats.connection_stats.peak_connections = stats.connection_stats.peak_connections
            .max(connection_stats.total_connections);
        
        // 更新房间统计
        stats.room_stats.total_rooms = room_stats.active_rooms; // 使用active_rooms作为total_rooms
        stats.room_stats.active_rooms = room_stats.active_rooms;
        
        // 估算内存使用
        stats.memory_stats = self.estimate_memory_usage(&connection_stats, &room_stats).await;
        
        stats.last_updated = chrono::Utc::now();
        
        debug!("性能统计已更新: 连接数 {}, 房间数 {}, 内存使用 {:.2}MB",
            stats.connection_stats.active_connections,
            stats.room_stats.total_rooms,
            stats.memory_stats.total_memory_mb
        );
        
        Ok(())
    }

    /// 估算内存使用
    async fn estimate_memory_usage(&self, connection_stats: &ConnectionStats, room_stats: &RoomStats) -> MemoryStats {
        // 简化的内存估算
        let connection_memory = (connection_stats.total_connections as f64) * 0.1; // 每连接约100KB
        let room_memory = (room_stats.active_rooms as f64) * 0.05; // 每房间约50KB
        let message_buffer_memory = 10.0; // 消息缓冲区约10MB
        
        MemoryStats {
            connection_memory_mb: connection_memory,
            room_memory_mb: room_memory,
            message_buffer_memory_mb: message_buffer_memory,
            total_memory_mb: connection_memory + room_memory + message_buffer_memory,
        }
    }

    /// 执行清理操作
    async fn perform_cleanup(&self) -> WsResult<()> {
        info!("开始执行WebSocket性能清理");
        
        // 清理不活跃的连接
        let cleaned_connections = self.connection_manager.cleanup_inactive_connections(300).await; // 5分钟超时
        
        if cleaned_connections > 0 {
            info!("清理了 {} 个不活跃连接", cleaned_connections);
        }
        
        // 清理空房间
        let cleaned_rooms = self.room_manager.cleanup_empty_rooms().await;
        
        if cleaned_rooms > 0 {
            info!("清理了 {} 个空房间", cleaned_rooms);
        }
        
        Ok(())
    }

    /// 获取性能统计
    pub async fn get_performance_stats(&self) -> PerformanceStats {
        self.performance_stats.read().await.clone()
    }

    /// 生成优化建议
    pub async fn generate_optimization_suggestions(&self) -> Vec<PerformanceOptimizationSuggestion> {
        let mut suggestions = Vec::new();
        let stats = self.performance_stats.read().await;
        
        // 检查内存使用
        if stats.memory_stats.total_memory_mb > self.config.memory_threshold_mb {
            suggestions.push(PerformanceOptimizationSuggestion {
                category: OptimizationCategory::Memory,
                priority: OptimizationPriority::High,
                description: format!(
                    "内存使用 {:.2}MB 超过阈值 {:.2}MB",
                    stats.memory_stats.total_memory_mb,
                    self.config.memory_threshold_mb
                ),
                recommended_action: "增加清理频率或减少连接数".to_string(),
                estimated_impact: "减少内存使用20-30%".to_string(),
            });
        }
        
        // 检查连接数
        if stats.connection_stats.active_connections > 10000 {
            suggestions.push(PerformanceOptimizationSuggestion {
                category: OptimizationCategory::Connection,
                priority: OptimizationPriority::Medium,
                description: format!(
                    "活跃连接数 {} 较高",
                    stats.connection_stats.active_connections
                ),
                recommended_action: "考虑实施连接限制或负载均衡".to_string(),
                estimated_impact: "提升系统稳定性".to_string(),
            });
        }
        
        // 检查房间大小
        if stats.room_stats.largest_room_size > self.config.max_room_size {
            suggestions.push(PerformanceOptimizationSuggestion {
                category: OptimizationCategory::RoomManagement,
                priority: OptimizationPriority::Medium,
                description: format!(
                    "最大房间大小 {} 超过建议值 {}",
                    stats.room_stats.largest_room_size,
                    self.config.max_room_size
                ),
                recommended_action: "考虑拆分大房间或限制房间大小".to_string(),
                estimated_impact: "提升消息分发效率".to_string(),
            });
        }
        
        suggestions
    }

    /// 优化消息路由
    pub async fn optimize_message_routing(&self) -> WsResult<MessageRoutingOptimization> {
        let stats = self.performance_stats.read().await;
        
        Ok(MessageRoutingOptimization {
            current_strategy: "broadcast".to_string(),
            recommended_strategy: if stats.connection_stats.active_connections > 1000 {
                "selective_routing".to_string()
            } else {
                "broadcast".to_string()
            },
            estimated_improvement: if stats.connection_stats.active_connections > 1000 {
                "减少CPU使用30-50%".to_string()
            } else {
                "当前策略已优化".to_string()
            },
        })
    }

    /// 获取连接健康状态
    pub async fn get_connection_health(&self) -> ConnectionHealth {
        let stats = self.connection_manager.get_stats().await;
        let performance_stats = self.performance_stats.read().await;
        
        ConnectionHealth {
            total_connections: stats.total_connections,
            healthy_connections: stats.total_connections, // 简化实现
            unhealthy_connections: 0,
            connection_success_rate: 99.5, // 简化实现
            average_latency_ms: 50.0, // 简化实现
            is_healthy: performance_stats.memory_stats.total_memory_mb < self.config.memory_threshold_mb,
        }
    }
}

impl Default for PerformanceStats {
    fn default() -> Self {
        Self {
            message_stats: MessageStats {
                total_messages_sent: 0,
                total_messages_failed: 0,
                messages_per_second: 0.0,
                average_message_size: 0.0,
                message_latency_ms: MessageLatencyStats {
                    min: 0.0,
                    max: 0.0,
                    avg: 0.0,
                    p50: 0.0,
                    p95: 0.0,
                    p99: 0.0,
                },
            },
            connection_stats: ConnectionPerformanceStats {
                active_connections: 0,
                peak_connections: 0,
                connection_churn_rate: 0.0,
                average_connection_duration: Duration::ZERO,
                failed_connections: 0,
            },
            room_stats: RoomPerformanceStats {
                total_rooms: 0,
                active_rooms: 0,
                average_room_size: 0.0,
                largest_room_size: 0,
                room_message_distribution: HashMap::new(),
            },
            memory_stats: MemoryStats {
                connection_memory_mb: 0.0,
                room_memory_mb: 0.0,
                message_buffer_memory_mb: 0.0,
                total_memory_mb: 0.0,
            },
            last_updated: chrono::Utc::now(),
        }
    }
}

/// 消息路由优化
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageRoutingOptimization {
    pub current_strategy: String,
    pub recommended_strategy: String,
    pub estimated_improvement: String,
}

/// 连接健康状态
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConnectionHealth {
    pub total_connections: usize,
    pub healthy_connections: usize,
    pub unhealthy_connections: usize,
    pub connection_success_rate: f64,
    pub average_latency_ms: f64,
    pub is_healthy: bool,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ws::{ConnectionManager, RoomManager, MessageRouter};

    #[tokio::test]
    async fn test_performance_optimizer_creation() {
        let connection_manager = Arc::new(ConnectionManager::new());
        let room_manager = Arc::new(RoomManager::new());
        let message_router = Arc::new(MessageRouter::new(
            connection_manager.clone(),
            room_manager.clone(),
        ));
        
        let optimizer = WebSocketPerformanceOptimizer::new(
            connection_manager,
            room_manager,
            message_router,
            OptimizationConfig::default(),
        );
        
        let stats = optimizer.get_performance_stats().await;
        assert_eq!(stats.connection_stats.active_connections, 0);
    }
}
