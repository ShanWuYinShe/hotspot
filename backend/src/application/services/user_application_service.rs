/// 用户应用服务
/// 
/// 协调用户相关的用例，连接领域层和基础设施层
use crate::domain::{UserProfile, UserAuth, UserEvent, DomainError, DomainResult};
use crate::application::{ApplicationResult, ApplicationError};
use crate::application::commands::{
    RegisterUserCommand, CreateTempUserCommand, UpgradeUserCommand,
    UpdateUserLocationCommand, UpdateUserProfileCommand, SetUserPasswordCommand,
    DeleteUserAccountCommand,
    UserCommandResult, UserRegistrationResult, LocationUpdateResult,
};
use crate::application::handlers::CommandHandler;
use crate::application::queries::{
    GetUserByIdQuery, SearchUsersByLocationQuery,
    SearchUsersByIdQuery, CheckPasswordStatusQuery, SearchUsersByNameQuery,
    NearbyUser, PasswordStatus, UserSearchResult,
};
use crate::application::handlers::{QueryHandler, PaginatedResult};
use async_trait::async_trait;
/// 用户应用服务
/// 
/// 功能职责：
/// - 实现用户相关的业务用例和应用逻辑
/// - 协调用户领域实体和仓储层的交互
/// - 管理用户认证、注册、资料管理等流程
/// - 处理用户位置更新和附近搜索逻辑
/// - 提供用户数据的分页查询和统计功能
/// 
/// 主要用例：
/// - 用户注册和临时用户创建
/// - 用户登录和身份验证
/// - 用户资料更新和账户升级
/// - 用户位置管理和附近用户搜索
/// - 用户设置管理和隐私控制
/// 
/// 事务管理：
/// - 管理用户数据一致性和事务边界
/// - 协调用户相关的异步任务和通知
/// - 处理用户操作的错误和异常情况
use std::sync::Arc;

// 使用新的端口接口
use crate::ports::{UserRepository, PasswordService, EventPublisher};

/// 用户应用服务接口
#[async_trait]
pub trait UserApplicationServicePort: Send + Sync {
    /// 验证用户密码
    async fn verify_user_password(&self, login_id: &str, password: &str) -> DomainResult<UserAuth>;
}

/// 用户应用服务
#[derive(Clone)]
pub struct UserApplicationService {
    base: super::BaseApplicationService,
    user_repository: Arc<dyn UserRepository>,
    password_service: Arc<dyn PasswordService>,
}

impl UserApplicationService {
    /// 创建新的用户应用服务
    pub fn new(
        user_repository: Arc<dyn UserRepository>,
        password_service: Arc<dyn PasswordService>,
        event_publisher: Arc<dyn EventPublisher>,
    ) -> Self {
        let base = super::BaseApplicationService::new("UserApplicationService")
            .with_user_repository(user_repository.clone())
            .with_event_publisher(event_publisher);
            
        Self {
            base,
            user_repository,
            password_service,
        }
    }
    
    /// 验证用户密码
    pub async fn verify_user_password(&self, login_id: &str, password: &str) -> DomainResult<UserAuth> {
        // 查找用户认证信息
        let user_auth = self.user_repository.find_user_auth(login_id).await?
            .ok_or(DomainError::UserNotFound)?;
        
        // 检查密码
        if let Some(ref password_hash) = user_auth.password_hash {
            let is_valid = self.password_service.verify_password(password, password_hash).await?;
            if is_valid {
                Ok(user_auth)
            } else {
                Err(DomainError::InvalidCredentials)
            }
        } else {
            Err(DomainError::InvalidCredentials)
        }
    }
    
    /// 检查登录ID是否已存在
    async fn check_login_id_exists(&self, login_id: &str) -> DomainResult<bool> {
        match self.user_repository.find_user_auth(login_id).await {
            Ok(Some(_)) => Ok(true),
            Ok(None) => Ok(false),
            Err(e) => Err(e),
        }
    }
    
    /// 发布事件并忽略错误
    async fn publish_event_ignore_error(&self, event: UserEvent) {
        if let Err(e) = self.base.publish_user_event(event).await {
            self.base.log_error("publish_user_event", &e, None);
        }
    }
}

#[async_trait]
impl UserApplicationServicePort for UserApplicationService {
    async fn verify_user_password(&self, login_id: &str, password: &str) -> DomainResult<UserAuth> {
        self.verify_user_password(login_id, password).await
    }
}

// 实现命令处理器
#[async_trait]
impl CommandHandler<RegisterUserCommand> for UserApplicationService {
    type Result = UserRegistrationResult;
    
    async fn handle(&self, command: RegisterUserCommand) -> ApplicationResult<Self::Result> {
        // 验证命令
        command.validate()?;
        
        // 检查登录ID是否已存在
        if self.check_login_id_exists(&command.login_id).await? {
            return Err(ApplicationError::Domain(DomainError::LoginIdAlreadyExists));
        }
        
        // 加密密码
        let password_hash = self.password_service.hash_password(&command.password).await?;
        
        // 创建用户
        let nickname = command.nickname.unwrap_or_else(|| "用户".to_string());
        
        // 保存用户
        let saved_user = self.user_repository.create_user(
            Some(&command.login_id),
            &nickname,
            Some(&password_hash),
            None, // email
            None, // expires_at (正式用户不过期)
        ).await?;
        
        // 创建事件
        let event = crate::domain::events::DomainEvent::UserRegistered {
            user_id: crate::domain::UserId::new(saved_user.user_id).unwrap(),
            login_id: command.login_id,
            nickname,
            registered_at: chrono::Utc::now(),
        };
        
        // 发布事件（异步，不阻塞主流程）
        self.publish_event_ignore_error(event.clone()).await;
        
        Ok(UserRegistrationResult {
            user: saved_user,
            events: vec![event],
        })
    }
}

#[async_trait]
impl CommandHandler<CreateTempUserCommand> for UserApplicationService {
    type Result = UserRegistrationResult;
    
    async fn handle(&self, command: CreateTempUserCommand) -> ApplicationResult<Self::Result> {
        // 验证命令
        command.validate()?;
        
        // 创建临时用户
        let nickname = command.nickname.unwrap_or_else(|| "临时用户".to_string());
        let expires_at = chrono::Utc::now() + chrono::Duration::days(7); // 临时用户7天后过期
        
        // 保存用户
        let saved_user = self.user_repository.create_user(
            None, // login_id (临时用户没有登录ID)
            &nickname,
            None, // password_hash (临时用户没有密码)
            None, // email
            Some(expires_at), // expires_at (临时用户7天后过期)
        ).await?;
        
        // 创建事件
        let event = crate::domain::events::DomainEvent::TempUserCreated {
            user_id: crate::domain::UserId::new(saved_user.user_id).unwrap(),
            nickname,
            created_at: chrono::Utc::now(),
        };
        
        // 发布事件
        self.publish_event_ignore_error(event.clone()).await;
        
        Ok(UserRegistrationResult {
            user: saved_user,
            events: vec![event],
        })
    }
}

#[async_trait]
impl CommandHandler<UpgradeUserCommand> for UserApplicationService {
    type Result = UserCommandResult;
    
    async fn handle(&self, command: UpgradeUserCommand) -> ApplicationResult<Self::Result> {
        // 验证命令
        command.validate()?;
        
        // 记录操作日志
        self.base.log_operation("upgrade_user", Some(command.user_id), Some(&format!("login_id: {}", command.login_id)));
        
        // 验证用户存在
        self.base.validate_user_exists(command.user_id).await?;
        let user = self.user_repository.find_user_profile(command.user_id).await?
            .ok_or(DomainError::UserNotFound)?;
        
        // 检查是否为临时用户
        if !user.is_temporary() {
            return Err(ApplicationError::Domain(DomainError::InvalidUserTypeConversion));
        }
        
        // 检查登录ID是否已存在
        if self.check_login_id_exists(&command.login_id).await? {
            return Err(ApplicationError::Domain(DomainError::LoginIdAlreadyExists));
        }
        
        // 加密密码
        let password_hash = self.password_service.hash_password(&command.password).await?;
        
        // 升级用户 - 直接调用仓库方法升级临时用户
        let saved_user = self.user_repository.upgrade_temp_user(
            user.user_id,
            &command.login_id,
            &password_hash,
        ).await?;
        
        // 创建事件
        let event = crate::domain::events::DomainEvent::UserUpgraded {
            user_id: crate::domain::UserId::new(saved_user.user_id).unwrap(),
            upgraded_at: chrono::Utc::now(),
        };
        
        // 发布事件
        self.publish_event_ignore_error(event.clone()).await;
        
        Ok(UserCommandResult::success_with_message(
            saved_user.user_id,
            "账户升级成功".to_string(),
            vec![event],
        ))
    }
}

#[async_trait]
impl CommandHandler<UpdateUserLocationCommand> for UserApplicationService {
    type Result = LocationUpdateResult;
    
    async fn handle(&self, command: UpdateUserLocationCommand) -> ApplicationResult<Self::Result> {
        // 验证命令
        command.validate()?;
        
        // 使用基类验证坐标
        super::ValidationRules::validate_coordinates(command.latitude, command.longitude)?;
        
        // 记录操作日志
        self.base.log_operation("update_location", Some(command.user_id), Some(&format!("lat: {}, lon: {}", command.latitude, command.longitude)));
        
        // 查找用户
        let user = self.user_repository.find_user_profile(command.user_id).await?
            .ok_or(DomainError::UserNotFound)?;
        
        // 更新用户位置信息
        let location_name = command.address.clone().unwrap_or_else(|| "未知位置".to_string());
        self.user_repository
            .update_user_location(
                command.user_id,
                command.latitude,
                command.longitude,
                &location_name,
            )
            .await?;
        
        let saved_user = user;
        
        // 创建事件
        let new_location = crate::domain::entities::UserLocation {
            coordinates: crate::domain::values::Coordinates::new(command.latitude, command.longitude)
                .unwrap_or_else(|_| crate::domain::values::Coordinates::new(0.0, 0.0).unwrap()),
            location_name: command.address.clone().unwrap_or_default(),
            updated_at: chrono::Utc::now(),
        };
        let event = crate::domain::events::DomainEvent::UserLocationUpdated {
            user_id: crate::domain::UserId::new(saved_user.user_id).unwrap(),
            old_location: None, // TODO: 需要从数据库获取旧位置
            new_location,
            updated_at: chrono::Utc::now(),
        };
        
        // 发布事件
        self.publish_event_ignore_error(event).await;
        
        Ok(LocationUpdateResult {
            user_id: saved_user.user_id,
            latitude: command.latitude,
            longitude: command.longitude,
            address: command.address,
            updated_at: saved_user.last_active_at,
        })
    }
}

#[async_trait]
impl CommandHandler<UpdateUserProfileCommand> for UserApplicationService {
    type Result = UserCommandResult;
    
    async fn handle(&self, command: UpdateUserProfileCommand) -> ApplicationResult<Self::Result> {
        command.validate()?;
        let _user = self.user_repository.find_user_profile(command.user_id).await?
            .ok_or(DomainError::UserNotFound)?;
        
        let saved_user = self.user_repository.update_user_profile(
            command.user_id,
            command.nickname.as_deref(),
            None, // avatar_url 暂时不支持更新
        ).await?;
        let event = crate::domain::events::DomainEvent::UserProfileUpdated {
            user_id: crate::domain::UserId::new(saved_user.user_id).unwrap(),
            updated_at: chrono::Utc::now(),
        };
        self.publish_event_ignore_error(event.clone()).await;
        
        Ok(UserCommandResult::success_with_message(saved_user.user_id, "资料更新成功".to_string(), vec![event]))
    }
}

#[async_trait]
impl CommandHandler<SetUserPasswordCommand> for UserApplicationService {
    type Result = UserCommandResult;
    
    async fn handle(&self, command: SetUserPasswordCommand) -> ApplicationResult<Self::Result> {
        // 查找用户
        let user = self.user_repository.find_user_profile(command.user_id).await?
            .ok_or(DomainError::UserNotFound)?;
        
        // 加密新密码
        let password_hash = self.password_service.hash_password(&command.password).await?;
        
        // 设置密码
        self.user_repository.set_user_password(user.user_id, &password_hash).await?;
        
        // 获取更新后的用户信息
        let saved_user = self.user_repository.find_user_profile(user.user_id).await?
            .ok_or(DomainError::UserNotFound)?;
        
        // 创建事件
        let event = crate::domain::events::DomainEvent::UserPasswordSet {
            user_id: crate::domain::UserId::new(saved_user.user_id).unwrap(),
            set_at: chrono::Utc::now(),
        };
        
        // 发布事件
        self.publish_event_ignore_error(event.clone()).await;
        
        Ok(UserCommandResult::success_with_message(
            saved_user.user_id,
            "密码设置成功".to_string(),
            vec![event],
        ))
    }
}

#[async_trait]
impl CommandHandler<DeleteUserAccountCommand> for UserApplicationService {
    type Result = UserCommandResult;
    
    async fn handle(&self, command: DeleteUserAccountCommand) -> ApplicationResult<Self::Result> {
        // 验证权限：只能删除自己的账户
        if command.user_id != command.requester_id {
            return Err(ApplicationError::Domain(DomainError::InsufficientPermission {
                required_role: "删除账户权限".to_string(),
            }));
        }
        
        // 查找用户
        let user = self.user_repository.find_user_profile(command.user_id).await?
            .ok_or(DomainError::UserNotFound)?;
        
        // 删除用户（软删除）
        self.user_repository.delete_user(command.user_id).await?;
        
        // 创建事件
        let event = crate::domain::events::DomainEvent::UserDeleted {
            user_id: crate::domain::UserId::new(user.user_id).unwrap(),
            deleted_at: chrono::Utc::now(),
        };
        
        // 发布事件
        self.publish_event_ignore_error(event.clone()).await;
        
        Ok(UserCommandResult::success_with_message(
            user.user_id,
            "账户删除成功".to_string(),
            vec![event],
        ))
    }
}

// 实现查询处理器
#[async_trait]
impl QueryHandler<GetUserByIdQuery> for UserApplicationService {
    type Result = Option<UserProfile>;
    
    async fn handle(&self, query: GetUserByIdQuery) -> ApplicationResult<Self::Result> {
        self.user_repository.find_user_profile(query.user_id).await.map_err(ApplicationError::Domain)
    }
}

#[async_trait]
impl QueryHandler<SearchUsersByIdQuery> for UserApplicationService {
    type Result = crate::application::queries::user_queries::UserProfileQuery;
    
    async fn handle(&self, query: SearchUsersByIdQuery) -> ApplicationResult<Self::Result> {
        // 🆕 使用新的轻量级查询方法
        let user_profile = self.user_repository.find_user_profile(query.user_id).await?
            .ok_or(DomainError::UserNotFoundWithId { user_id: query.user_id })?;
        
        // 获取用户位置信息
        let location = self.user_repository.find_user_location(query.user_id).await?;
        
        // 🚀 性能优化：直接从UserProfile转换，避免创建完整User实体
        let mut profile = crate::application::queries::user_queries::UserProfileQuery::from_domain_profile(user_profile);
        if let Some((latitude, longitude, location_name)) = location {
            profile.location = Some(crate::application::queries::user_queries::UserLocation {
                latitude,
                longitude,
                address: Some(location_name),
            });
        }
        
        Ok(profile)
    }
}

#[async_trait]
impl QueryHandler<SearchUsersByLocationQuery> for UserApplicationService {
    type Result = PaginatedResult<NearbyUser>;
    
    async fn handle(&self, query: SearchUsersByLocationQuery) -> ApplicationResult<Self::Result> {
        // 验证查询
        query.validate().map_err(|errors| DomainError::ValidationFailed { errors })?;
        
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        let radius = query.get_radius_km();
        
        // 查询时多取一条判断是否有更多数据
        let nearby_users = self.user_repository.find_nearby_users_with_distance(
            query.latitude,
            query.longitude,
            radius,
            cursor,
            (limit + 1) as i32,
        ).await?;
        
        let has_more = nearby_users.len() > limit as usize;
        let mut result_users = nearby_users;
        if has_more {
            result_users.pop(); // 移除多取的一条
        }
        
        // 转换为 NearbyUser 类型
        let nearby_user_results: Vec<crate::application::queries::user_queries::NearbyUser> = result_users
            .into_iter()
            .map(|user| crate::application::queries::user_queries::NearbyUser {
                user_id: user.user_id,
                nickname: user.nickname,
                location: crate::application::queries::user_queries::UserLocation {
                    latitude: 0.0, // TODO: 需要从实际位置数据获取
                    longitude: 0.0, // TODO: 需要从实际位置数据获取
                    address: Some("未知位置".to_string()), // TODO: 需要从实际位置数据获取
                },
                distance_km: 0.0, // TODO: 需要从实际距离数据获取
                created_at: chrono::Utc::now(), // UserSearchResult 没有 created_at 字段，使用当前时间
            })
            .collect();
        
        let next_cursor = if has_more && !nearby_user_results.is_empty() {
            Some(nearby_user_results.last().unwrap().user_id)
        } else {
            None
        };
        
        Ok(PaginatedResult::new(nearby_user_results, has_more, next_cursor))
    }
}

#[async_trait]
impl QueryHandler<SearchUsersByNameQuery> for UserApplicationService {
    type Result = PaginatedResult<UserSearchResult>;
    
    async fn handle(&self, query: SearchUsersByNameQuery) -> ApplicationResult<Self::Result> {
        query.validate().map_err(|errors| DomainError::ValidationFailed { errors })?;
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        
        // 🆕 使用新的轻量级查询方法
        let user_profiles = self.user_repository
            .search_user_profiles_by_name(&query.keyword, cursor, (limit + 1) as i32)
            .await?;
        
        let has_more = user_profiles.len() > limit as usize;
        let mut result_profiles = user_profiles;
        if has_more { result_profiles.pop(); }
        
        // 🚀 性能优化：直接从UserProfile转换，无需额外查询位置信息
        let mut items: Vec<UserSearchResult> = Vec::new();
        for profile in result_profiles {
            // 检查是否有位置信息（可选优化：批量查询或缓存）
            let has_location = self.user_repository.find_user_location(profile.user_id).await?.is_some();
            items.push(UserSearchResult {
                user_id: profile.user_id,
                nickname: profile.nickname,
                user_type: profile.user_type.as_str().to_string(),
                has_location,
                created_at: profile.last_active_at, // 使用last_active_at代替created_at，更有业务意义
            });
        }
        let next_cursor = if has_more && !items.is_empty() { Some(items.last().unwrap().user_id) } else { None };
        Ok(PaginatedResult::new(items, has_more, next_cursor))
    }
}

#[async_trait]
impl QueryHandler<CheckPasswordStatusQuery> for UserApplicationService {
    type Result = PasswordStatus;
    
    async fn handle(&self, query: CheckPasswordStatusQuery) -> ApplicationResult<Self::Result> {
        let user = self.user_repository.find_user_profile(query.user_id).await?
            .ok_or(DomainError::UserNotFoundWithId { user_id: query.user_id })?;
        
        Ok(PasswordStatus::from(user))
    }
}

// 实现增强应用服务trait
#[async_trait]
impl super::EnhancedApplicationService for UserApplicationService {
    fn service_name(&self) -> &'static str {
        self.base.get_service_name()
    }
    
    async fn health_check(&self) -> DomainResult<bool> {
        // 检查用户仓储是否可用
        match self.user_repository.find_user_profile(1).await {
            Ok(_) => Ok(true),
            Err(_) => Ok(false),
        }
    }
    
    async fn get_service_stats(&self) -> DomainResult<serde_json::Value> {
        Ok(serde_json::json!({
            "service_name": self.service_name(),
            "status": "healthy",
            "features": ["user_registration", "user_authentication", "user_profile_management", "location_services"]
        }))
    }
}
