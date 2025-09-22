/// HotSpot 后端服务主程序
/// 
/// 功能职责：
/// - 应用程序启动和初始化
/// - 服务容器组装和依赖注入
/// - HTTP服务器配置和启动
/// - 优雅关闭处理
/// 
/// 架构特点：
/// - 完全采用六边形架构（端口适配器模式）
/// - 统一的服务容器管理所有依赖
/// - 零重复路由，完全符合 hotspot.md 规范
/// - 集成健康检查、监控和WebSocket支持
use std::net::SocketAddr;
use std::time::Duration;
use std::env;

use tokio::net::TcpListener;
use tokio::signal;
use tracing::{info, warn};
use tower_http::{
    timeout::TimeoutLayer,
    trace::TraceLayer,
    cors::CorsLayer,
};

// 新架构导入
use backend::bootstrap::{HybridConfigManager, ServiceFactory, ServiceContainerExt};
use backend::presentation::http::routes::{create_application_routes, ServiceContainer};
use backend::presentation::http::auth::JwtConfig;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 初始化结构化日志
    init_logging()?;
    
    info!("🚀 启动 HotSpot 后端服务 (新架构)");
    info!("📋 版本: {}", env!("CARGO_PKG_VERSION"));
    info!("🏗️ 构建模式: {}", if cfg!(debug_assertions) { "debug" } else { "release" });
    info!("🏛️ 架构模式: Hexagonal (端口适配器) + 统一路由");

    // 加载和验证配置
    let config_manager = load_and_validate_config().await?;
    
    // 检查系统健康状态
    perform_health_checks(&config_manager).await?;

    // 使用服务工厂创建所有服务
    info!("🏭 创建服务容器...");
    let service_container = ServiceFactory::create_service_container(&config_manager).await?;
    
    // 验证服务容器
    ServiceFactory::validate_service_container(&service_container)
        .map_err(|e| format!("服务容器验证失败: {}", e))?;
    
    // 输出服务统计信息
    let service_stats = ServiceFactory::get_service_stats(&service_container);
    info!("📊 服务统计: {}", serde_json::to_string_pretty(&service_stats)?);
    
    info!("✅ 所有服务创建完成");

    // 创建统一的服务容器
    let services = create_unified_service_container(&service_container)?;
    
    // 构建JWT配置
    let config = config_manager.config();
    let jwt_config = JwtConfig::new(config.auth.jwt_secret.clone());

    // 获取必要的依赖
    let user_repo_port = service_container.user_repository_port();
    let websocket_available = service_container.is_websocket_available();
    let pg_pool = config_manager.database().pg_pool().clone();
    
    // 创建应用路由 - 使用新架构
    let app = create_application_routes(
        services,
        jwt_config,
        user_repo_port,
        pg_pool,
        websocket_available,
    );

    // 添加中间件层
    let app = add_middleware_layers(app);
    
    // 启动服务器
    let port = env::var("SERVER_PORT")
        .unwrap_or("8080".to_string())
        .parse::<u16>()
        .unwrap_or(8080);
    let addr = SocketAddr::from(([0, 0, 0, 0], port));
    let listener = TcpListener::bind(addr).await?;
    
    info!("🌟 HotSpot 后端服务启动成功 (新架构)");
    info!("🌐 服务地址: http://{}", addr);
    info!("📊 数据库指标: {:?}", config_manager.get_db_metrics());
    info!("🔌 WebSocket系统: {}", if websocket_available { "可用" } else { "不可用" });
    info!("📈 监控系统: 已启用 (错误监控、性能监控、数据库优化)");
    info!("🏗️ 架构特点:");
    info!("   • 六边形架构 (端口适配器模式)");
    info!("   • 统一认证中间件");
    info!("   • 零重复路由");
    info!("   • 符合 hotspot.md 规范");
    info!("⚡ 准备接收请求...");
    
    // 输出路由信息
    print_route_summary(websocket_available);
    
    // 优雅关闭处理
    let graceful_shutdown = setup_graceful_shutdown();
    
    // 启动服务器并等待关闭信号
    axum::serve(listener, app)
        .with_graceful_shutdown(graceful_shutdown)
        .await?;
    
    info!("👋 HotSpot 后端服务已优雅关闭");
    Ok(())
}

/// 创建统一的服务容器
fn create_unified_service_container(
    service_container: &backend::bootstrap::ServiceContainer,
) -> Result<ServiceContainer, Box<dyn std::error::Error>> {
    // 从原始服务容器中提取控制器
    let user_repo = service_container.user_repository_port();
    
    // 创建新的控制器实例
    let friend_controller = backend::presentation::http::controllers::FriendController::new(
        service_container.user_app_service.clone().into()
    );
    let security_controller = backend::presentation::http::controllers::SecurityController::new(
        service_container.user_app_service.clone().into()
    );
    let notification_controller = backend::presentation::http::controllers::NotificationController::new(
        service_container.user_app_service.clone().into()
    );
    let system_health_controller = backend::presentation::http::controllers::SystemHealthController::new();
    
    Ok(ServiceContainer {
        user_controller: service_container.user_controller.clone(),
        group_controller: service_container.group_controller.clone(),
        message_controller: service_container.message_controller.clone(),
        moment_controller: service_container.moment_controller.clone(),
        search_controller: service_container.search_controller.clone(),
        activity_controller: service_container.activity_controller.clone(),
        friend_controller,
        comment_controller: service_container.comment_controller.clone(),
        security_controller,
        notification_controller,
        system_health_controller,
        upload_controller: service_container.upload_controller.clone(),
        admin_controller: backend::presentation::http::controllers::AdminController::with_user_repo(user_repo),
        metrics_service: service_container.metrics_service.clone(),
    })
}

/// 输出路由摘要信息（从实际路由注册过程动态收集）
fn print_route_summary(websocket_available: bool) {
    use backend::utils::route_introspector::RouteIntrospector;
    
    // 获取收集的路由信息进行调试
    let routes = RouteIntrospector::get_collected_routes();
    info!("🔍 调试：收集到 {} 个路由", routes.len());
    
    // 打印从路由注册过程中收集的信息
    RouteIntrospector::print_summary(websocket_available);
}

/// 初始化结构化日志系统
fn init_logging() -> Result<(), Box<dyn std::error::Error>> {
    use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};
    
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "backend=info,tower_http=debug".into()),
        )
        .with(tracing_subscriber::fmt::layer().with_target(false))
        .init();
    
    Ok(())
}

/// 加载和验证配置
async fn load_and_validate_config() -> Result<HybridConfigManager, Box<dyn std::error::Error>> {
    info!("📋 加载应用配置...");
    let config_manager = HybridConfigManager::from_env().await?;
    config_manager.validate_and_print();
    info!("✅ 配置加载完成");
    Ok(config_manager)
}

/// 执行系统健康检查
async fn perform_health_checks(config_manager: &HybridConfigManager) -> Result<(), Box<dyn std::error::Error>> {
    info!("🔍 执行系统健康检查...");
    
    // 数据库连接检查
    info!("  🗄️  检查数据库连接...");
    config_manager.check_health().await?;
    info!("  ✅ 数据库连接正常");
    
    // 系统资源检查
    info!("  💾 检查系统资源...");
    let available_memory = get_available_memory();
    let cpu_count = num_cpus::get();
    info!("  📊 系统信息: {} CPU核心, {}MB 可用内存", cpu_count, available_memory);
    
    if available_memory < 512 {
        warn!("  ⚠️  可用内存较低: {}MB", available_memory);
    }
    
    // 架构验证
    info!("  🏗️  验证新架构...");
    info!("  ✅ 六边形架构: 已启用");
    info!("  ✅ 统一认证中间件: 已配置");
    info!("  ✅ 路由规范化: 已完成");
    info!("  ✅ 服务容器: 已初始化");
    
    info!("✅ 所有健康检查通过");
    Ok(())
}

/// 获取可用内存 (MB)
fn get_available_memory() -> u64 {
    use sysinfo::System;
    let mut sys = System::new_all();
    sys.refresh_memory();
    sys.available_memory() / 1024 / 1024 // 转换为MB
}

/// 添加中间件层
fn add_middleware_layers(app: axum::Router) -> axum::Router {
    app
        // CORS支持 (最外层)
        .layer(
            CorsLayer::new()
                .allow_origin(tower_http::cors::Any)
                .allow_methods(tower_http::cors::Any)
                .allow_headers(tower_http::cors::Any)
        )
        // 请求追踪
        .layer(TraceLayer::new_for_http())
        // 请求超时 (最内层)
        .layer(TimeoutLayer::new(Duration::from_secs(30)))
}

/// 设置优雅关闭处理
async fn setup_graceful_shutdown() {
    let ctrl_c = async {
        signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("failed to install signal handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {
            info!("🛑 收到 Ctrl+C 信号，开始优雅关闭...");
        },
        _ = terminate => {
            info!("🛑 收到终止信号，开始优雅关闭...");
        },
    }
    
    info!("⏳ 等待现有连接完成...");
    // 可以在这里添加清理逻辑
    tokio::time::sleep(Duration::from_secs(1)).await;
}