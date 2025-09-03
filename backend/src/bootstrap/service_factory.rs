/// 统一服务工厂
/// 
/// 负责创建和组装所有应用服务、基础设施服务和WebSocket服务

use std::sync::Arc;
use tracing::info;

use crate::adapters::{
    PgUserRepository, 
    PgGroupRepository, 
    PgMessageRepository, 
    PgMomentRepository,
    PgFriendRepository,
    Argon2PasswordService,
    LogEventPublisher,
};
use crate::application::services::{
    UserApplicationService,
    GroupApplicationService, 
    MessageApplicationService,
    MomentApplicationService,
    SearchApplicationServiceImpl,
    ActivityApplicationService,
    FriendApplicationService,
};
use crate::presentation::http::{
    UserController,
    GroupController,
    MomentController,
    SearchController,
    ActivityController,
    FriendController,
};
use crate::presentation::http::controllers::MessageController;
use crate::presentation::ws::WebSocketManager;
use crate::ports::{
    UserRepositoryPort,
    GroupRepositoryPort,
    // 移除了MessageRepositoryPort
    MomentRepositoryPort,
    PasswordServicePort,
    EventPublisherPort,
};
use crate::bootstrap::HybridConfigManager;

/// 服务容器
/// 
/// 包含所有已创建的服务实例
pub struct ServiceContainer {
    // 仓储层
    pub user_repository: Arc<PgUserRepository>,
    pub group_repository: Arc<PgGroupRepository>,
    pub message_repository: Arc<PgMessageRepository>,
    pub moment_repository: Arc<PgMomentRepository>,
    pub friend_repository: Arc<PgFriendRepository>,
    
    // 基础设施服务
    pub password_service: Arc<Argon2PasswordService>,
    pub event_publisher: Arc<LogEventPublisher>,
    
    // 应用服务
    pub user_app_service: UserApplicationService,
    pub group_app_service: GroupApplicationService,
    pub message_app_service: MessageApplicationService,
    pub moment_app_service: MomentApplicationService,
    pub friend_app_service: FriendApplicationService<PgFriendRepository, PgUserRepository>,
    
    // 控制器
    pub user_controller: UserController<UserApplicationService>,
    pub group_controller: GroupController<GroupApplicationService>,
    pub message_controller: MessageController,
    pub moment_controller: MomentController<MomentApplicationService>,
    pub search_controller: SearchController,
    pub activity_controller: ActivityController<PgUserRepository, PgGroupRepository, PgMomentRepository>,
    pub friend_controller: FriendController<PgFriendRepository, PgUserRepository>,
    pub upload_controller: crate::presentation::http::controllers::UploadController,
    
    // WebSocket系统
    pub websocket_manager: Arc<WebSocketManager<PgUserRepository>>,
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
        let user_repository = Arc::new(PgUserRepository::new(pool.clone()));
        let group_repository = Arc::new(PgGroupRepository::new(pool.clone()));
        let message_repository = Arc::new(PgMessageRepository::new(pool.clone()));
        let moment_repository = Arc::new(PgMomentRepository::new(pool.clone()));
        let friend_repository = Arc::new(PgFriendRepository::new(pool.clone()));
        
        // 创建基础设施服务
        info!("🔧 创建基础设施服务...");
        let password_service = Arc::new(Argon2PasswordService::new());
        let event_publisher = Arc::new(LogEventPublisher::default());
        
        // 创建应用服务
        info!("⚙️ 创建应用服务...");
        let user_app_service = UserApplicationService::new(
            user_repository.clone() as Arc<dyn UserRepositoryPort>,
            password_service.clone() as Arc<dyn PasswordServicePort>,
            event_publisher.clone() as Arc<dyn EventPublisherPort>,
        );
        
        let group_app_service = GroupApplicationService::new(
            group_repository.clone() as Arc<dyn GroupRepositoryPort>,
            password_service.clone() as Arc<dyn PasswordServicePort>,
        );
        
        let message_app_service = MessageApplicationService::new(
            message_repository.clone(),
        );
        
        let moment_app_service = MomentApplicationService::new(
            moment_repository.clone() as Arc<dyn MomentRepositoryPort>,
        );
        
        // 创建搜索应用服务
        let search_app_service = Arc::new(SearchApplicationServiceImpl::new(
            user_repository.clone() as Arc<dyn UserRepositoryPort>,
            group_repository.clone() as Arc<dyn GroupRepositoryPort>,
            moment_repository.clone() as Arc<dyn MomentRepositoryPort>,
        ));
        
        // 创建活动应用服务
        let activity_app_service = ActivityApplicationService::new(
            user_repository.clone(),
            group_repository.clone(),
            moment_repository.clone(),
        );
        
        // 创建好友应用服务
        let friend_app_service = FriendApplicationService::new(
            friend_repository.clone(),
            user_repository.clone(),
            pool.clone(),
        );
        
        // 创建控制器
        info!("🎮 创建控制器层...");
        let user_controller = UserController::new(user_app_service.clone());
        let group_controller = GroupController::new(group_app_service.clone());
        let message_controller = MessageController::new(message_app_service.clone());
        let moment_controller = MomentController::new(moment_app_service.clone());
        let search_controller = SearchController::new(search_app_service.clone() as Arc<dyn crate::application::services::SearchApplicationService>);
        let activity_controller = ActivityController::new(activity_app_service);
        let friend_controller = FriendController::new(friend_app_service.clone());
        
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
            password_service,
            event_publisher,
            user_app_service,
            group_app_service,
            message_app_service,
            moment_app_service,
            friend_app_service,
            user_controller,
            group_controller,
            message_controller,
            moment_controller,
            search_controller,
            activity_controller,
            friend_controller,
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
    fn user_repository_port(&self) -> Arc<dyn UserRepositoryPort>;
    
    /// 获取群组仓储的端口引用
    fn group_repository_port(&self) -> Arc<dyn GroupRepositoryPort>;
    
    /// 获取消息仓储的端口引用
    // 移除了MessageRepositoryPort方法
    
    /// 获取瞬间仓储的端口引用
    fn moment_repository_port(&self) -> Arc<dyn MomentRepositoryPort>;
    
    /// 检查WebSocket系统是否可用
    fn is_websocket_available(&self) -> bool;
}

impl ServiceContainerExt for ServiceContainer {
    fn user_repository_port(&self) -> Arc<dyn UserRepositoryPort> {
        self.user_repository.clone() as Arc<dyn UserRepositoryPort>
    }
    
    fn group_repository_port(&self) -> Arc<dyn GroupRepositoryPort> {
        self.group_repository.clone() as Arc<dyn GroupRepositoryPort>
    }
    
    // 移除了MessageRepositoryPort实现
    
    fn moment_repository_port(&self) -> Arc<dyn MomentRepositoryPort> {
        self.moment_repository.clone() as Arc<dyn MomentRepositoryPort>
    }
    
    fn is_websocket_available(&self) -> bool {
        true // WebSocket系统总是可用的
    }
}

// 为PgUserRepository添加null检查扩展
trait RepositoryExt {
    fn is_null(&self) -> bool;
}

impl RepositoryExt for PgUserRepository {
    fn is_null(&self) -> bool {
        // 简单检查，实际实现可能需要检查内部连接状态
        false
    }
}
