/// 用户应用服务
/// 
/// 协调用户相关的用例，连接领域层和基础设施层

use crate::domain::{User, UserEvent, DomainResult, DomainError, UserDomainError};
use crate::application::commands::{
    CommandHandler, RegisterUserCommand, CreateTempUserCommand, UpgradeUserCommand,
    UpdateUserLocationCommand, UpdateUserProfileCommand, SetUserPasswordCommand,
    DeleteUserAccountCommand,
    UserCommandResult, UserRegistrationResult, LocationUpdateResult,
};
use crate::application::queries::{
    QueryHandler, GetUserByIdQuery, FindNearbyUsersQuery,
    GetUserProfileQuery, CheckPasswordStatusQuery, SearchUsersByNicknameQuery,
    UserProfile, NearbyUser, PasswordStatus, PaginatedResult, UserSearchResult,
};
use async_trait::async_trait;
use std::sync::Arc;

// 使用新的端口接口
use crate::ports::{UserRepositoryPort, PasswordServicePort, EventPublisherPort};

/// 用户应用服务接口
#[async_trait]
pub trait UserApplicationServicePort: Send + Sync {
    /// 验证用户密码
    async fn verify_user_password(&self, login_id: &str, password: &str) -> DomainResult<User>;
}

/// 用户应用服务
#[derive(Clone)]
pub struct UserApplicationService {
    user_repository: Arc<dyn UserRepositoryPort>,
    password_service: Arc<dyn PasswordServicePort>,
    event_publisher: Arc<dyn EventPublisherPort>,
}

impl UserApplicationService {
    /// 创建新的用户应用服务
    pub fn new(
        user_repository: Arc<dyn UserRepositoryPort>,
        password_service: Arc<dyn PasswordServicePort>,
        event_publisher: Arc<dyn EventPublisherPort>,
    ) -> Self {
        Self {
            user_repository,
            password_service,
            event_publisher,
        }
    }
    
    /// 验证用户密码
    pub async fn verify_user_password(&self, login_id: &str, password: &str) -> DomainResult<User> {
        // 查找用户
        let user = self.user_repository.find_by_login_id(login_id).await?
            .ok_or(DomainError::User(UserDomainError::UserNotFound { user_id: 0 }))?;
        
        // 检查密码
        if let Some(ref password_hash) = user.password_hash {
            let is_valid = self.password_service.verify_password(password, password_hash).await?;
            if is_valid {
                Ok(user)
            } else {
                Err(DomainError::User(UserDomainError::PasswordVerificationFailed))
            }
        } else {
            Err(DomainError::User(UserDomainError::PasswordVerificationFailed))
        }
    }
    
    /// 检查登录ID是否已存在
    async fn check_login_id_exists(&self, login_id: &str) -> DomainResult<bool> {
        match self.user_repository.find_by_login_id(login_id).await {
            Ok(Some(_)) => Ok(true),
            Ok(None) => Ok(false),
            Err(e) => Err(e),
        }
    }
    
    /// 发布事件并忽略错误
    async fn publish_event_ignore_error(&self, event: UserEvent) {
        if let Err(e) = self.event_publisher.publish_user_event(event).await {
            // 记录错误但不影响主流程
            log::error!("Failed to publish user event: {}", e);
        }
    }
}

#[async_trait]
impl UserApplicationServicePort for UserApplicationService {
    async fn verify_user_password(&self, login_id: &str, password: &str) -> DomainResult<User> {
        self.verify_user_password(login_id, password).await
    }
}

// 实现命令处理器
#[async_trait]
impl CommandHandler<RegisterUserCommand> for UserApplicationService {
    type Result = UserRegistrationResult;
    
    async fn handle_command(&self, command: RegisterUserCommand) -> DomainResult<Self::Result> {
        // 验证命令
        command.validate()?;
        
        // 检查登录ID是否已存在
        if self.check_login_id_exists(&command.login_id).await? {
            return Err(DomainError::User(UserDomainError::LoginIdAlreadyExists {
                login_id: command.login_id,
            }));
        }
        
        // 加密密码
        let password_hash = self.password_service.hash_password(&command.password).await?;
        
        // 创建用户实体
        let nickname = command.nickname.unwrap_or_else(|| "用户".to_string());
        let user_id = chrono::Utc::now().timestamp_millis(); // 生成基于时间戳的用户ID
        let user = User::new_regular(user_id, command.login_id.clone(), nickname.clone(), password_hash);
        
        // 保存用户
        let saved_user = self.user_repository.create_user(&user).await?;
        
        // 创建事件
        let event = crate::domain::events::EventFactory::user_registered(
            saved_user.id,
            command.login_id,
            nickname,
        );
        
        // 发布事件（异步，不阻塞主流程）
        let event_clone = event.clone();
        let event_publisher = self.event_publisher.clone();
        tokio::spawn(async move {
            if let Err(e) = event_publisher.publish_user_event(event_clone).await {
                log::error!("Failed to publish user registered event: {}", e);
            }
        });
        
        Ok(UserRegistrationResult {
            user: saved_user,
            events: vec![event],
        })
    }
}

#[async_trait]
impl CommandHandler<CreateTempUserCommand> for UserApplicationService {
    type Result = UserRegistrationResult;
    
    async fn handle_command(&self, command: CreateTempUserCommand) -> DomainResult<Self::Result> {
        // 验证命令
        command.validate()?;
        
        // 创建临时用户实体
        let nickname = command.nickname.unwrap_or_else(|| "临时用户".to_string());
        let user_id = chrono::Utc::now().timestamp_millis(); // 生成基于时间戳的用户ID
        let expires_at = chrono::Utc::now() + chrono::Duration::days(7); // 临时用户7天后过期
        let user = User::new_temporary(user_id, nickname.clone(), expires_at);
        
        // 保存用户
        let saved_user = self.user_repository.create_user(&user).await?;
        
        // 创建事件
        let event = crate::domain::events::EventFactory::temp_user_created(
            saved_user.id,
            nickname,
        );
        
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
    
    async fn handle_command(&self, command: UpgradeUserCommand) -> DomainResult<Self::Result> {
        // 验证命令
        command.validate()?;
        
        // 查找用户
        let mut user = self.user_repository.find_by_id(command.user_id).await?
            .ok_or(DomainError::User(UserDomainError::UserNotFound { user_id: command.user_id }))?;
        
        // 检查是否为临时用户
        if !user.is_temporary() {
            return Err(DomainError::User(UserDomainError::InvalidUserTypeConversion {
                reason: "只有临时用户可以升级".to_string(),
            }));
        }
        
        // 检查登录ID是否已存在
        if self.check_login_id_exists(&command.login_id).await? {
            return Err(DomainError::User(UserDomainError::LoginIdAlreadyExists {
                login_id: command.login_id,
            }));
        }
        
        // 加密密码
        let password_hash = self.password_service.hash_password(&command.password).await?;
        
        // 升级用户
        user.upgrade_to_regular(command.login_id.clone(), password_hash)?;
        
        // 保存用户
        let saved_user = self.user_repository.update_user(&user).await?;
        
        // 创建事件
        let event = crate::domain::events::UserEvent::UserUpgraded {
            event_id: crate::domain::events::EventFactory::generate_event_id(),
            user_id: saved_user.id,
            login_id: command.login_id,
            occurred_at: chrono::Utc::now(),
        };
        
        // 发布事件
        self.publish_event_ignore_error(event.clone()).await;
        
        Ok(UserCommandResult::success_with_message(
            saved_user.id,
            "账户升级成功".to_string(),
            vec![event],
        ))
    }
}

#[async_trait]
impl CommandHandler<UpdateUserLocationCommand> for UserApplicationService {
    type Result = LocationUpdateResult;
    
    async fn handle_command(&self, command: UpdateUserLocationCommand) -> DomainResult<Self::Result> {
        // 验证命令
        command.validate()?;
        
        // 查找用户
        let user = self.user_repository.find_by_id(command.user_id).await?
            .ok_or(DomainError::User(UserDomainError::UserNotFound { user_id: command.user_id }))?;
        
        // 更新用户位置信息
        let location_name = command.address.clone().unwrap_or_else(|| "未知位置".to_string());
        let user_location = crate::domain::UserLocation {
            id: 0, // 这个字段在插入时会被忽略
            user_id: command.user_id,
            latitude: command.latitude,
            longitude: command.longitude,
            location_name,
            updated_at: chrono::Utc::now(),
        };
        let _user_location = self.user_repository
            .upsert_user_location(&user_location)
            .await?;
        
        let saved_user = user;
        
        // 创建事件
        let event = crate::domain::events::UserEvent::UserLocationUpdated {
            event_id: crate::domain::events::EventFactory::generate_event_id(),
            user_id: saved_user.id,
            latitude: command.latitude,
            longitude: command.longitude,
            address: command.address.clone(),
            occurred_at: chrono::Utc::now(),
        };
        
        // 发布事件
        self.publish_event_ignore_error(event).await;
        
        Ok(LocationUpdateResult {
            user_id: saved_user.id,
            latitude: command.latitude,
            longitude: command.longitude,
            address: command.address,
            updated_at: saved_user.updated_at,
        })
    }
}

#[async_trait]
impl CommandHandler<UpdateUserProfileCommand> for UserApplicationService {
    type Result = UserCommandResult;
    
    async fn handle_command(&self, command: UpdateUserProfileCommand) -> DomainResult<Self::Result> {
        command.validate()?;
        let mut user = self.user_repository.find_by_id(command.user_id).await?
            .ok_or(DomainError::User(UserDomainError::UserNotFound { user_id: command.user_id }))?;
        
        if let Some(nickname) = command.nickname {
            user.update_nickname(nickname)?;
        }
        
        let saved_user = self.user_repository.update_user(&user).await?;
        let event = crate::domain::events::UserEvent::UserProfileUpdated {
            event_id: crate::domain::events::EventFactory::generate_event_id(),
            user_id: saved_user.id,
            occurred_at: chrono::Utc::now(),
        };
        self.publish_event_ignore_error(event.clone()).await;
        
        Ok(UserCommandResult::success_with_message(saved_user.id, "资料更新成功".to_string(), vec![event]))
    }
}

#[async_trait]
impl CommandHandler<SetUserPasswordCommand> for UserApplicationService {
    type Result = UserCommandResult;
    
    async fn handle_command(&self, command: SetUserPasswordCommand) -> DomainResult<Self::Result> {
        // 查找用户
        let mut user = self.user_repository.find_by_id(command.user_id).await?
            .ok_or(DomainError::User(UserDomainError::UserNotFound { user_id: command.user_id }))?;
        
        // 加密新密码
        let password_hash = self.password_service.hash_password(&command.password).await?;
        
        // 设置密码
        user.set_password(password_hash)?;
        
        // 保存用户
        let saved_user = self.user_repository.update_user(&user).await?;
        
        // 创建事件
        let event = crate::domain::events::UserEvent::UserPasswordSet {
            event_id: crate::domain::events::EventFactory::generate_event_id(),
            user_id: saved_user.id,
            occurred_at: chrono::Utc::now(),
        };
        
        // 发布事件
        self.publish_event_ignore_error(event.clone()).await;
        
        Ok(UserCommandResult::success_with_message(
            saved_user.id,
            "密码设置成功".to_string(),
            vec![event],
        ))
    }
}

#[async_trait]
impl CommandHandler<DeleteUserAccountCommand> for UserApplicationService {
    type Result = UserCommandResult;
    
    async fn handle_command(&self, command: DeleteUserAccountCommand) -> DomainResult<Self::Result> {
        // 验证权限：只能删除自己的账户
        if command.user_id != command.requester_id {
            return Err(DomainError::User(UserDomainError::InsufficientPermissions {
                required: "删除账户权限".to_string(),
            }));
        }
        
        // 查找用户
        let user = self.user_repository.find_by_id(command.user_id).await?
            .ok_or(DomainError::User(UserDomainError::UserNotFound { user_id: command.user_id }))?;
        
        // 删除用户（软删除）
        self.user_repository.delete_user(command.user_id).await?;
        
        // 创建事件
        let event = crate::domain::events::UserEvent::UserDeleted {
            event_id: crate::domain::events::EventFactory::generate_event_id(),
            user_id: user.id,
            occurred_at: chrono::Utc::now(),
        };
        
        // 发布事件
        self.publish_event_ignore_error(event.clone()).await;
        
        Ok(UserCommandResult::success_with_message(
            user.id,
            "账户删除成功".to_string(),
            vec![event],
        ))
    }
}

// 实现查询处理器
#[async_trait]
impl QueryHandler<GetUserByIdQuery> for UserApplicationService {
    type Result = Option<User>;
    
    async fn handle_query(&self, query: GetUserByIdQuery) -> DomainResult<Self::Result> {
        self.user_repository.find_by_id(query.user_id).await
    }
}

#[async_trait]
impl QueryHandler<GetUserProfileQuery> for UserApplicationService {
    type Result = UserProfile;
    
    async fn handle_query(&self, query: GetUserProfileQuery) -> DomainResult<Self::Result> {
        let user = self.user_repository.find_by_id(query.user_id).await?
            .ok_or(DomainError::User(UserDomainError::UserNotFound { user_id: query.user_id }))?;
        
        // 获取用户位置信息
        let location = self.user_repository.find_user_location(query.user_id).await?;
        
        let mut profile = UserProfile::from(user);
        if let Some(user_location) = location {
            profile.location = Some(crate::application::UserLocation {
                latitude: user_location.latitude,
                longitude: user_location.longitude,
                address: Some(user_location.location_name),
            });
        }
        
        Ok(profile)
    }
}

#[async_trait]
impl QueryHandler<FindNearbyUsersQuery> for UserApplicationService {
    type Result = PaginatedResult<NearbyUser>;
    
    async fn handle_query(&self, query: FindNearbyUsersQuery) -> DomainResult<Self::Result> {
        // 验证查询
        query.validate().map_err(|errors| DomainError::User(UserDomainError::ValidationFailed { errors }))?;
        
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        let radius = query.get_radius_km();
        
        // 查询时多取一条判断是否有更多数据
        let nearby_users = self.user_repository.find_nearby_users_with_distance(
            query.latitude,
            query.longitude,
            radius,
            cursor,
            limit + 1,
        ).await?;
        
        let has_more = nearby_users.len() > limit as usize;
        let mut result_users = nearby_users;
        if has_more {
            result_users.pop(); // 移除多取的一条
        }
        
        let next_cursor = if has_more && !result_users.is_empty() {
            Some(result_users.last().unwrap().user_id)
        } else {
            None
        };
        
        Ok(PaginatedResult::new(result_users, has_more, next_cursor))
    }
}

#[async_trait]
impl QueryHandler<SearchUsersByNicknameQuery> for UserApplicationService {
    type Result = PaginatedResult<UserSearchResult>;
    
    async fn handle_query(&self, query: SearchUsersByNicknameQuery) -> DomainResult<Self::Result> {
        query.validate().map_err(|errors| DomainError::User(UserDomainError::ValidationFailed { errors }))?;
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        
        let users = self.user_repository
            .search_users_by_nickname(&query.keyword, cursor, limit + 1)
            .await?;
        
        let has_more = users.len() > limit as usize;
        let mut result_users = users;
        if has_more { result_users.pop(); }
        
        // 为每个用户查询位置信息
        let mut items: Vec<UserSearchResult> = Vec::new();
        for user in result_users {
            let has_location = self.user_repository.find_user_location(user.id).await?.is_some();
            items.push(UserSearchResult {
                user_id: user.id,
                nickname: user.nickname.clone(),
                user_type: user.user_type.as_str().to_string(),
                has_location,
                created_at: user.created_at,
            });
        }
        let next_cursor = if has_more && !items.is_empty() { Some(items.last().unwrap().user_id) } else { None };
        Ok(PaginatedResult::new(items, has_more, next_cursor))
    }
}

#[async_trait]
impl QueryHandler<CheckPasswordStatusQuery> for UserApplicationService {
    type Result = PasswordStatus;
    
    async fn handle_query(&self, query: CheckPasswordStatusQuery) -> DomainResult<Self::Result> {
        let user = self.user_repository.find_by_id(query.user_id).await?
            .ok_or(DomainError::User(UserDomainError::UserNotFound { user_id: query.user_id }))?;
        
        Ok(PasswordStatus::from(user))
    }
}
