use crate::application::commands::{
    CreateGroupCommand, DeleteGroupCommand, GroupCommandResult, JoinGroupCommand,
    JoinGroupResult, KickMemberCommand, KickMemberResult, LeaveGroupCommand, LeaveGroupResult,
    TransferOwnershipCommand, TransferOwnershipResult, UpdateGroupInfoCommand,
};
use crate::application::handlers::CommandHandler;
use crate::application::queries::{
    SearchGroupsByLocationQuery, GroupMemberInfo, GroupMembersQuery, GroupSearchResult, NearbyGroup,
    SearchGroupsByNameQuery, SearchGroupsByIdQuery, SearchGroupsByUserIdQuery, SearchGroupsByOwnerIdQuery,
};
use crate::application::handlers::{QueryHandler, PaginatedResult};
use crate::domain::{DomainError, GroupProfile, GroupDomainError};
use crate::application::{ApplicationResult, ApplicationError};
use crate::ports::{GroupRepository, PasswordService};
/// 群组应用服务（骨架）
use async_trait::async_trait;
/// 群组应用服务
/// 
/// 功能职责：
/// - 实现群组相关的业务用例和应用逻辑
/// - 协调群组领域实体和仓储层的交互
/// - 管理群组创建、成员管理、权限控制等流程
/// - 处理基于地理位置的群组搜索和推荐
/// - 提供群组数据的统计和分析功能
/// 
/// 主要用例：
/// - 群组创建和初始化设置
/// - 群组成员的加入、退出、踢出管理
/// - 群组角色和权限的分配和变更
/// - 群主转让和管理员任免
/// - 附近群组搜索和按名称查找
/// 
/// 事务管理：
/// - 管理群组数据一致性和成员关系
/// - 协调群组相关的通知和事件分发
/// - 处理群组操作的并发和冲突解决
use std::sync::Arc;

#[derive(Clone)]
pub struct GroupApplicationService {
    base: super::BaseApplicationService,
    group_repository: Arc<dyn GroupRepository>,
    password_service: Arc<dyn PasswordService>,
}

impl GroupApplicationService {
    pub fn new(
        group_repository: Arc<dyn GroupRepository>, 
        password_service: Arc<dyn PasswordService>,
        user_repository: Arc<dyn crate::ports::UserRepository>,
        event_publisher: Arc<dyn crate::ports::EventPublisher>,
    ) -> Self {
        let base = super::BaseApplicationService::new("GroupApplicationService")
            .with_user_repository(user_repository)
            .with_event_publisher(event_publisher);
            
        Self { 
            base,
            group_repository, 
            password_service 
        }
    }
}

#[async_trait]
impl CommandHandler<CreateGroupCommand> for GroupApplicationService {
    type Result = GroupCommandResult;
    async fn handle(&self, command: CreateGroupCommand) -> ApplicationResult<Self::Result> {
        command.validate()?;
        
        // 使用基类验证坐标和用户
        super::ValidationRules::validate_coordinates(command.latitude, command.longitude)?;
        self.base.validate_user_exists(command.owner_id).await?;
        
        // 记录操作日志
        self.base.log_operation("create_group", Some(command.owner_id), Some(&format!("name: {}", command.name)));
        
        let _group_id = uuid::Uuid::new_v4().to_string(); // 生成UUID作为group_id（当前未使用）
        let has_password = command.password.is_some();
        let password_hash = if let Some(password) = command.password {
            Some(self.password_service.hash_password(&password).await?)
        } else {
            None
        };
        
        let saved = self.group_repository.create_group(
            &command.name,
            command.description.as_deref(),
            command.owner_id,
            command.latitude,
            command.longitude,
            &command.location_name,
            !has_password, // 没有密码的群组为公开群组
            password_hash.as_deref(),
        ).await?;
        Ok(GroupCommandResult::success_with_message(
            saved.group_id,
            "群组创建成功".to_string(),
        ))
    }
}

#[async_trait]
impl CommandHandler<UpdateGroupInfoCommand> for GroupApplicationService {
    type Result = GroupCommandResult;
    async fn handle(&self, command: UpdateGroupInfoCommand) -> ApplicationResult<Self::Result> {
        command.validate()?;
        let _group = self
            .group_repository
            .find_group_profile(&command.group_id)
            .await?
            .ok_or_else(|| DomainError::BusinessRuleViolation {
                rule: "Group not found".to_string(),
            })?;
        let saved = self.group_repository.update_group(
            &command.group_id,
            command.name.as_deref(),
            command.description.as_deref(),
        ).await?;
        Ok(GroupCommandResult::success_with_message(
            saved.group_id,
            "群组更新成功".to_string(),
        ))
    }
}

#[async_trait]
impl CommandHandler<DeleteGroupCommand> for GroupApplicationService {
    type Result = GroupCommandResult;
    async fn handle(&self, command: DeleteGroupCommand) -> ApplicationResult<Self::Result> {
        command.validate()?;
        self.group_repository.delete_group(&command.group_id).await?;
        Ok(GroupCommandResult::success_with_message(
            command.group_id,
            "群组已删除".to_string(),
        ))
    }
}

#[async_trait]
impl QueryHandler<SearchGroupsByLocationQuery> for GroupApplicationService {
    type Result = PaginatedResult<NearbyGroup>;
    async fn handle(&self, query: SearchGroupsByLocationQuery) -> ApplicationResult<Self::Result> {
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        let rows = self
            .group_repository
            .find_nearby_groups_with_distance(
                query.latitude,
                query.longitude,
                query
                    .radius_km
                    .unwrap_or(crate::domain::constants::DEFAULT_NEARBY_RADIUS_KM),
                cursor,
                (limit + 1) as i32,
            )
            .await?;
        let has_more = rows.len() > limit as usize;
        let mut items = rows;
        if has_more {
            items.pop();
        }
        let result = items
            .into_iter()
            .map(|g| {
                // 计算距离（与用户服务一致的近似计算）
                let distance = crate::domain::values::coordinates::Coordinates::new(
                    query.latitude,
                    query.longitude,
                )
                .and_then(|q| {
                    crate::domain::values::coordinates::Coordinates::new(g.latitude, g.longitude)
                        .map(|c| q.distance_to(&c).kilometers())
                })
                .unwrap_or(0.0);
                crate::application::queries::group_queries::NearbyGroup {
                    group_id: g.group_id.parse().unwrap_or(0),
                    name: g.name,
                    latitude: g.latitude,
                    longitude: g.longitude,
                    distance_km: distance,
                    created_at: chrono::Utc::now(), // TODO: 需要从领域对象获取创建时间
                }
            })
            .collect::<Vec<_>>();
        let next_cursor = if has_more && !result.is_empty() {
            Some(result.last().unwrap().group_id)
        } else {
            None
        };
        Ok(PaginatedResult::new(result, has_more, next_cursor))
    }
}

#[async_trait]
impl QueryHandler<SearchGroupsByNameQuery> for GroupApplicationService {
    type Result = PaginatedResult<GroupSearchResult>;
    async fn handle(&self, query: SearchGroupsByNameQuery) -> ApplicationResult<Self::Result> {
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        let rows = self
            .group_repository
            .search_groups_by_name(&query.keyword, cursor, (limit + 1) as i32)
            .await?;
        let has_more = rows.len() > limit as usize;
        let mut items = rows;
        if has_more {
            items.pop();
        }
        let result = items
            .into_iter()
            .map(|g| crate::application::queries::group_queries::GroupSearchResult {
                group_id: g.group_id.parse().unwrap_or(0),
                name: g.name,
                has_location: true,
                created_at: chrono::Utc::now(), // TODO: 需要从领域对象获取创建时间
            })
            .collect::<Vec<_>>();
        let next_cursor = if has_more && !result.is_empty() {
            Some(result.last().unwrap().group_id)
        } else {
            None
        };
        Ok(PaginatedResult::new(result, has_more, next_cursor))
    }
}

#[async_trait]
impl CommandHandler<JoinGroupCommand> for GroupApplicationService {
    type Result = JoinGroupResult;
    async fn handle(&self, command: JoinGroupCommand) -> ApplicationResult<Self::Result> {
        command.validate()?;
        if self
            .group_repository
            .find_group_profile(&command.group_id)
            .await?
            .is_none()
        {
            return Err(GroupDomainError::GroupNotFound {
            }
            .into());
        }
        if self
            .group_repository
            .is_group_member(&command.group_id, command.user_id)
            .await?
        {
            return Err(GroupDomainError::UserAlreadyMemberWithId {
                user_id: command.user_id,
            }
            .into());
        }
        // 添加用户到群组，默认角色为 member
        self.group_repository
            .add_group_member(&command.group_id, command.user_id, "member")
            .await?;
        let role = "member".to_string();
        Ok(JoinGroupResult {
            group_id: 0, // TODO: 需要数字ID，暂时使用0
            user_id: command.user_id,
            role,
        })
    }
}

#[async_trait]
impl CommandHandler<LeaveGroupCommand> for GroupApplicationService {
    type Result = LeaveGroupResult;
    async fn handle(&self, command: LeaveGroupCommand) -> ApplicationResult<Self::Result> {
        command.validate()?;
        if self
            .group_repository
            .find_group_profile(&command.group_id)            .await?
            .is_none()
        {
            return Err(GroupDomainError::GroupNotFound {
            }
            .into());
        }
        if self
            .group_repository
            .is_group_member(&command.group_id, command.user_id)
            .await?
        {
            return Err(GroupDomainError::UserNotMemberWithId {
                user_id: command.user_id,
            }
            .into());
        }
        self.group_repository
            .remove_group_member(&command.group_id, command.user_id)
            .await?;
        Ok(LeaveGroupResult {
            group_id: 0, // TODO: 需要数字ID，暂时使用0
            user_id: command.user_id,
            success: true,
        })
    }
}

#[async_trait]
impl CommandHandler<KickMemberCommand> for GroupApplicationService {
    type Result = KickMemberResult;
    async fn handle(&self, command: KickMemberCommand) -> ApplicationResult<Self::Result> {
        command.validate()?;
        if self
            .group_repository
            .find_group_profile(&command.group_id)            .await?
            .is_none()
        {
            return Err(GroupDomainError::GroupNotFound {
            }
            .into());
        }
        // 需要管理员权限
        let admin_role = self
            .group_repository
            .get_user_role_in_group(&command.group_id, command.admin_id)
            .await?;
        
        if !matches!(admin_role.as_deref(), Some("admin") | Some("owner")) {
            return Err(GroupDomainError::InsufficientPermission {
                required_role: "admin".to_string(),
            }
            .into());
        }
        // 目标必须是成员
        if !self
            .group_repository
            .is_group_member(&command.group_id, command.target_user_id)
            .await?
        {
            return Err(GroupDomainError::UserNotMemberWithId {
                user_id: command.target_user_id,
            }
            .into());
        }
        // 不能移除群主
        let target_role = self
            .group_repository
            .get_user_role_in_group(&command.group_id, command.target_user_id)
            .await?;
        
        if matches!(target_role.as_deref(), Some("owner")) {
            return Err(GroupDomainError::CannotRemoveOwner.into());
        }
        self
            .group_repository
            .remove_group_member(&command.group_id, command.target_user_id)
            .await?;
        Ok(KickMemberResult {
            group_id: 0, // TODO: 需要数字ID，暂时使用0
            user_id: command.target_user_id,
            success: true,
        })
    }
}

#[async_trait]
impl CommandHandler<TransferOwnershipCommand> for GroupApplicationService {
    type Result = TransferOwnershipResult;
    async fn handle(
        &self,
        command: TransferOwnershipCommand,
    ) -> ApplicationResult<Self::Result> {
        command.validate()?;
        if self
            .group_repository
            .find_group_profile(&command.group_id)            .await?
            .is_none()
        {
            return Err(GroupDomainError::GroupNotFound {
            }
            .into());
        }
        // 只有现任群主可以转让
        let current_owner_role = self
            .group_repository
            .get_user_role_in_group(&command.group_id, command.current_owner_id)
            .await?;
        
        if !matches!(current_owner_role.as_deref(), Some("owner")) {
            return Err(GroupDomainError::InsufficientPermission {
                required_role: "owner".to_string(),
            }
            .into());
        }
        // 新群主必须是成员
        if self
            .group_repository
            .is_group_member(&command.group_id, command.new_owner_id)
            .await?
        {
            return Err(GroupDomainError::UserNotMemberWithId {
                user_id: command.new_owner_id,
            }
            .into());
        }
        self
            .group_repository
            .transfer_group_ownership(&command.group_id, command.new_owner_id)
            .await?;
        Ok(TransferOwnershipResult {
            group_id: 0, // TODO: 需要数字ID，暂时使用0
            new_owner_id: command.new_owner_id,
            success: true,
        })
    }
}

#[async_trait]
impl QueryHandler<GroupMembersQuery> for GroupApplicationService {
    type Result = PaginatedResult<GroupMemberInfo>;
    async fn handle(&self, query: GroupMembersQuery) -> ApplicationResult<Self::Result> {
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        let members = self
            .group_repository
            .get_group_members(&query.group_id.to_string(), cursor, limit as i32)
            .await?;
        let mapped = members
            .into_iter()
            .map(|m| GroupMemberInfo {
                member_id: 0, // TODO: 需要从群组成员表获取
                user_id: m.user_id,
                nickname: m.nickname,
                role: "member".to_string(), // TODO: 需要从群组成员表获取角色
                last_active_at: m.last_active_at,
                joined_at: chrono::Utc::now(), // TODO: 需要从群组成员表获取加入时间
            })
            .collect::<Vec<_>>();
        
        // TODO: 实现正确的分页逻辑
        let has_more = mapped.len() >= limit as usize;
        let next_cursor = if has_more && !mapped.is_empty() {
            Some(mapped.last().unwrap().user_id)
        } else {
            None
        };
        
        Ok(PaginatedResult::new(mapped, has_more, next_cursor))
    }
}

#[async_trait]
impl QueryHandler<SearchGroupsByIdQuery> for GroupApplicationService {
    type Result = Option<GroupProfile>;
    
    async fn handle(&self, query: SearchGroupsByIdQuery) -> ApplicationResult<Self::Result> {
        self.group_repository.find_group_profile(&query.group_id.to_string()).await.map_err(ApplicationError::Domain)
    }
}

#[async_trait]
impl QueryHandler<SearchGroupsByUserIdQuery> for GroupApplicationService {
    type Result = PaginatedResult<GroupProfile>;
    
    async fn handle(&self, query: SearchGroupsByUserIdQuery) -> ApplicationResult<Self::Result> {
        let _limit = query.pagination.get_limit();
        let _cursor = query.pagination.get_cursor();
        
        let groups = self.group_repository.find_user_groups(
            query.user_id,
            query.pagination.cursor.unwrap_or(0),
            query.pagination.limit.unwrap_or(20) as i32,
        ).await?;
        
        let has_more = groups.len() as i32 >= query.pagination.limit.unwrap_or(20) as i32;
        let next_cursor = if has_more && !groups.is_empty() {
            Some(groups.len() as i64) // TODO: 使用正确的游标逻辑
        } else {
            None
        };
        
        Ok(PaginatedResult {
            items: groups,
            has_more,
            next_cursor: next_cursor.map(|c| c.to_string()),
        })
    }
}

#[async_trait]
impl QueryHandler<SearchGroupsByOwnerIdQuery> for GroupApplicationService {
    type Result = PaginatedResult<GroupProfile>;
    
    async fn handle(&self, query: SearchGroupsByOwnerIdQuery) -> ApplicationResult<Self::Result> {
        let _limit = query.pagination.get_limit();
        let _cursor = query.pagination.get_cursor();
        
        let groups = self.group_repository.find_owned_groups(
            query.owner_id,
            query.pagination.cursor.unwrap_or(0),
            query.pagination.limit.unwrap_or(20) as i32,
        ).await?;
        
        let has_more = groups.len() as i32 >= query.pagination.limit.unwrap_or(20) as i32;
        let next_cursor = if has_more && !groups.is_empty() {
            Some(groups.len() as i64) // TODO: 使用正确的游标逻辑
        } else {
            None
        };
        
        Ok(PaginatedResult {
            items: groups,
            has_more,
            next_cursor: next_cursor.map(|c| c.to_string()),
        })
    }
}
