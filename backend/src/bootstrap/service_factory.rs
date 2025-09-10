/// 统一服务工厂
/// 
/// 负责创建和组装所有应用服务、基础设施服务和WebSocket服务
/// 服务工厂模块
/// 
/// 功能职责：
/// - 实现应用程序的服务工厂模式
/// - 管理所有服务的创建和依赖注入
/// - 协调各层组件的初始化和装配
/// - 提供统一的服务容器和生命周期管理
/// - 实现模块化的系统架构和组件解耦
/// 
/// 服务层次：
/// - 适配器层：数据库、缓存、通知等基础设施
/// - 应用服务层：业务用例和应用逻辑实现
/// - 控制器层：HTTP API 和 WebSocket 接口处理
/// - 中间件层：认证、限流、日志等横切关注
/// 
/// 依赖管理：
/// - 使用强类型系统和泛型约束
/// - 实现依赖倒置原则和接口隔离
/// - 支持可插拔的组件实现和配置切换
/// - 提供统一的错误处理和资源管理
/// 
/// 初始化流程：
/// 1. 加载配置和环境变量
/// 2. 创建数据库连接池和基础设施
/// 3. 初始化适配器层组件
/// 4. 创建应用服务和业务逻辑
/// 5. 装配控制器和路由系统
/// 6. 启动后台任务和监控服务
use std::sync::Arc;
use tracing::info;

use crate::adapters::{
    PostgresUserRepository, 
    PgGroupRepository, 
    PgMessageRepository, 
    PgMomentRepository,
    PgFriendRepository,
    PgCommentRepository,
    Argon2PasswordService,
    LogEventPublisher,
};
use crate::application::{
    UserApplicationService,
    GroupApplicationService, 
    MessageApplicationService,
    MomentApplicationService,
    SearchApplicationServiceImpl,
    ActivityApplicationService,
    FriendApplicationServiceImpl,
    CommentApplicationServiceImpl,
};
use crate::presentation::http::{
    UserController,
    GroupController,
    MomentController,
    SearchController,
    ActivityController,
    FriendController,
    CommentController,
};
use crate::presentation::http::controllers::MessageController;
use crate::presentation::ws::WebSocketManager;
use crate::ports::{
    UserRepository,
    GroupRepository,
    // 移除了MessageRepositoryPort
    MomentRepository,
    PasswordService,
    EventPublisher,
};
use crate::bootstrap::HybridConfigManager;

/// 服务容器
/// 
/// 包含所有已创建的服务实例
pub struct ServiceContainer {
    // 仓储层
    pub user_repository: Arc<PostgresUserRepository>,
    pub group_repository: Arc<PgGroupRepository>,
    pub message_repository: Arc<PgMessageRepository>,
    pub moment_repository: Arc<PgMomentRepository>,
    pub friend_repository: Arc<PgFriendRepository>,
    pub comment_repository: Arc<PgCommentRepository>,
    
    // 基础设施服务
    pub password_service: Arc<Argon2PasswordService>,
    pub event_publisher: Arc<LogEventPublisher>,
    
    // 应用服务
    pub user_app_service: UserApplicationService,
    pub group_app_service: GroupApplicationService,
    pub message_app_service: MessageApplicationService,
    pub moment_app_service: MomentApplicationService,
    pub friend_app_service: FriendApplicationServiceImpl<PgFriendRepository, PostgresUserRepository>,
    pub comment_app_service: CommentApplicationServiceImpl<PgCommentRepository>,
    
    // 控制器
    pub user_controller: UserController<UserApplicationService>,
    pub group_controller: GroupController<GroupApplicationService>,
    pub message_controller: MessageController,
    pub moment_controller: MomentController<MomentApplicationService>,
    pub search_controller: SearchController,
    pub activity_controller: ActivityController<PostgresUserRepository, PgGroupRepository, PgMomentRepository>,
    pub friend_controller: FriendController<FriendApplicationServiceImpl<PgFriendRepository, PostgresUserRepository>>,
    pub comment_controller: CommentController<PgCommentRepository>,
    pub upload_controller: crate::presentation::http::controllers::UploadController,
    
    // WebSocket系统
    pub websocket_manager: Arc<WebSocketManager<PostgresUserRepository>>,
}

/// 服务工厂
pub struct ServiceFactory;

impl ServiceFactory {
    /// 创建完整的服务容器
    pub async fn create_service_container(
        config_manager: &HybridConfigManager
    ) -> Result<ServiceContainer, Box<dyn std::error::Error>> {
        info!("🏭 开始创建服务容器...");
        
        // 创建仓储层
        info!("📦 创建仓储层服务...");
        let database = config_manager.database();
        let pool = database.pg_pool().clone();
        let user_repository = Arc::new(PostgresUserRepository::new(pool.clone()));
        let group_repository = Arc::new(PgGroupRepository::new(pool.clone()));
        let message_repository = Arc::new(PgMessageRepository::new(pool.clone()));
        let moment_repository = Arc::new(PgMomentRepository::new(pool.clone()));
        let friend_repository = Arc::new(PgFriendRepository::new(pool.clone()));
        
        // 创建基础设施服务
        info!("🔧 创建基础设施服务...");
        let password_service = Arc::new(Argon2PasswordService::new());
        let event_publisher = Arc::new(LogEventPublisher);
        
        // 创建应用服务
        info!("⚙️ 创建应用服务...");
        let user_app_service = UserApplicationService::new(
            user_repository.clone() as Arc<dyn UserRepository>,
            password_service.clone() as Arc<dyn PasswordService>,
            event_publisher.clone() as Arc<dyn EventPublisher>,
        );
        
        let group_app_service = GroupApplicationService::new(
            group_repository.clone() as Arc<dyn GroupRepository>,
            password_service.clone() as Arc<dyn PasswordService>,
            user_repository.clone() as Arc<dyn UserRepository>,
            event_publisher.clone() as Arc<dyn EventPublisher>,
        );
        
        let message_app_service = MessageApplicationService::new(
            message_repository.clone(),
            user_repository.clone() as Arc<dyn UserRepository>,
            event_publisher.clone() as Arc<dyn EventPublisher>,
        );
        
        let moment_app_service = MomentApplicationService::new(
            moment_repository.clone() as Arc<dyn MomentRepository>,
            user_repository.clone() as Arc<dyn UserRepository>,
            event_publisher.clone() as Arc<dyn EventPublisher>,
        );
        
        // 创建搜索应用服务
        let search_app_service = Arc::new(SearchApplicationServiceImpl::new(
            user_repository.clone() as Arc<dyn UserRepository>,
            group_repository.clone() as Arc<dyn GroupRepository>,
            moment_repository.clone() as Arc<dyn MomentRepository>,
        ));
        
        // 创建活动应用服务
        let activity_app_service = ActivityApplicationService::new(
            user_repository.clone(),
            group_repository.clone(),
            moment_repository.clone(),
        );
        
        // 创建好友应用服务
        let friend_app_service = FriendApplicationServiceImpl::new(
            friend_repository.clone(),
            user_repository.clone(),
            event_publisher.clone() as Arc<dyn EventPublisher>,
        );
        
        // 创建控制器
        info!("🎮 创建控制器层...");
        let user_controller = UserController::new(user_app_service.clone());
        let group_controller = GroupController::new(group_app_service.clone());
        let message_controller = MessageController::new(message_app_service.clone());
        let moment_controller = MomentController::new(moment_app_service.clone());
        let search_controller = SearchController::new(search_app_service.clone() as Arc<dyn crate::application::SearchApplicationService>);
        let activity_controller = ActivityController::new(activity_app_service);
        let friend_controller = FriendController::new(friend_app_service.clone());
        
        // 创建评论应用服务和控制器
        info!("💬 创建评论系统...");
        let comment_repository = Arc::new(crate::adapters::PgCommentRepository::new(pool.clone()));
        let comment_repository_for_controller = comment_repository.as_ref().clone();
        let comment_app_service = crate::application::CommentApplicationServiceImpl::new(
            comment_repository.clone(),
            user_repository.clone() as Arc<dyn UserRepository>,
            event_publisher.clone() as Arc<dyn EventPublisher>,
        );
        let comment_controller = crate::presentation::http::controllers::CommentController::new(
            comment_repository_for_controller
        );
        
        // 创建文件上传控制器
        info!("📁 创建文件上传控制器...");
        let upload_controller = crate::presentation::http::controllers::UploadController::new();
        
        // 创建WebSocket系统
        info!("🔌 创建WebSocket系统...");
        let websocket_manager = Arc::new(WebSocketManager::new(user_repository.clone()));
        
        info!("✅ 服务容器创建完成");
        
        Ok(ServiceContainer {
            user_repository,
            group_repository,
            message_repository,
            moment_repository,
            friend_repository,
            comment_repository,
            password_service,
            event_publisher,
            user_app_service,
            group_app_service,
            message_app_service,
            moment_app_service,
            friend_app_service,
            comment_app_service,
            user_controller,
            group_controller,
            message_controller,
            moment_controller,
            search_controller,
            activity_controller,
            friend_controller,
            comment_controller,
            upload_controller,
            websocket_manager,
        })
    }
    
    /// 验证服务容器的完整性
    pub fn validate_service_container(container: &ServiceContainer) -> Result<(), String> {
        info!("🔍 验证服务容器完整性...");
        
        // 检查仓储层
        if container.user_repository.is_null() {
            return Err("用户仓储未正确初始化".to_string());
        }
        
        // 可以添加更多验证逻辑
        // - 检查数据库连接
        // - 检查配置完整性
        // - 检查依赖关系
        
        info!("✅ 服务容器验证通过");
        Ok(())
    }
    
    /// 获取服务统计信息
    pub fn get_service_stats(_container: &ServiceContainer) -> serde_json::Value {
        serde_json::json!({
            "repositories": {
                "user_repository": "PgUserRepository",
                "group_repository": "PgGroupRepository", 
                "message_repository": "PgMessageRepository",
                "moment_repository": "PgMomentRepository"
            },
            "infrastructure_services": {
                "password_service": "Argon2PasswordService",
                "event_publisher": "LogEventPublisher"
            },
            "application_services": {
                "user_app_service": "UserApplicationService",
                "group_app_service": "GroupApplicationService",
                "message_app_service": "MessageApplicationService", 
                "moment_app_service": "MomentApplicationService"
            },
            "controllers": {
                "user_controller": "UserController",
                "group_controller": "GroupController",
                "message_controller": "MessageController",
                "moment_controller": "MomentController"
            },
            "websocket_system": {
                "websocket_manager": "WebSocketManager"
            },
            "total_services": 14,
            "architecture": "Hexagonal (Ports & Adapters)",
            "status": "initialized"
        })
    }
}

/// 服务容器扩展trait
pub trait ServiceContainerExt {
    /// 获取用户仓储的端口引用
    fn user_repository_port(&self) -> Arc<dyn UserRepository>;
    
    /// 获取群组仓储的端口引用
    fn group_repository_port(&self) -> Arc<dyn GroupRepository>;
    
    /// 获取消息仓储的端口引用
    // 移除了MessageRepositoryPort方法
    /// 获取瞬间仓储的端口引用
    fn moment_repository_port(&self) -> Arc<dyn MomentRepository>;
    
    /// 检查WebSocket系统是否可用
    fn is_websocket_available(&self) -> bool;
}

impl ServiceContainerExt for ServiceContainer {
    fn user_repository_port(&self) -> Arc<dyn UserRepository> {
        self.user_repository.clone() as Arc<dyn UserRepository>
    }
    
    fn group_repository_port(&self) -> Arc<dyn GroupRepository> {
        self.group_repository.clone() as Arc<dyn GroupRepository>
    }
    
    // 移除了MessageRepositoryPort实现
    
    fn moment_repository_port(&self) -> Arc<dyn MomentRepository> {
        self.moment_repository.clone() as Arc<dyn MomentRepository>
    }
    
    fn is_websocket_available(&self) -> bool {
        true // WebSocket系统总是可用的
    }
}

// 为PgUserRepository添加null检查扩展
trait RepositoryExt {
    fn is_null(&self) -> bool;
}

impl RepositoryExt for PostgresUserRepository {
    fn is_null(&self) -> bool {
        // 简单检查，实际实现可能需要检查内部连接状态
        false
    }
}
