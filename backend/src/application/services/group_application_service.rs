use crate::application::commands::{
    CommandHandler, CreateGroupCommand, DeleteGroupCommand, GroupCommandResult, JoinGroupCommand,
    JoinGroupResult, KickMemberCommand, KickMemberResult, LeaveGroupCommand, LeaveGroupResult,
    TransferOwnershipCommand, TransferOwnershipResult, UpdateGroupInfoCommand,
};
use crate::application::queries::{
    FindNearbyGroupsQuery, GroupMemberInfo, GroupMembersQuery, GroupSearchResult, NearbyGroup,
    PaginatedResult, QueryHandler, SearchGroupsByNameQuery, GetGroupByIdQuery, GetUserGroupsQuery, GetOwnedGroupsQuery,
};
use crate::domain::{DomainError, DomainResult, Group, GroupDomainError};
use crate::ports::{GroupMemberInfo as PortGroupMemberInfo, GroupRepositoryPort, PasswordServicePort};
/// 群组应用服务（骨架）
use async_trait::async_trait;
use std::sync::Arc;

#[derive(Clone)]
pub struct GroupApplicationService {
    group_repository: Arc<dyn GroupRepositoryPort>,
    password_service: Arc<dyn PasswordServicePort>,
}

impl GroupApplicationService {
    pub fn new(group_repository: Arc<dyn GroupRepositoryPort>, password_service: Arc<dyn PasswordServicePort>) -> Self {
        Self { group_repository, password_service }
    }
}

#[async_trait]
impl CommandHandler<CreateGroupCommand> for GroupApplicationService {
    type Result = GroupCommandResult;
    async fn handle_command(&self, command: CreateGroupCommand) -> DomainResult<Self::Result> {
        command.validate()?;
        
        let group_id = uuid::Uuid::new_v4().to_string(); // 生成UUID作为group_id
        let password_hash = if let Some(password) = command.password {
            Some(self.password_service.hash_password(&password).await?)
        } else {
            None
        };
        
        let group = Group::new(
            group_id,
            command.name,
            command.description,
            command.owner_id,
            command.latitude,
            command.longitude,
            command.location_name,
            password_hash,
        );
        let saved = self.group_repository.create_group(&group).await?;
        Ok(GroupCommandResult::success_with_message(
            saved.id,
            "群组创建成功".to_string(),
        ))
    }
}

#[async_trait]
impl CommandHandler<UpdateGroupInfoCommand> for GroupApplicationService {
    type Result = GroupCommandResult;
    async fn handle_command(&self, command: UpdateGroupInfoCommand) -> DomainResult<Self::Result> {
        command.validate()?;
        let mut group = self
            .group_repository
            .find_by_id(command.group_id)
            .await?
            .ok_or_else(|| DomainError::BusinessRuleViolation {
                rule: "Group not found".to_string(),
            })?;
        group.update_info(command.name, command.description);
        let saved = self.group_repository.update_group(&group).await?;
        Ok(GroupCommandResult::success_with_message(
            saved.id,
            "群组更新成功".to_string(),
        ))
    }
}

#[async_trait]
impl CommandHandler<DeleteGroupCommand> for GroupApplicationService {
    type Result = GroupCommandResult;
    async fn handle_command(&self, command: DeleteGroupCommand) -> DomainResult<Self::Result> {
        command.validate()?;
        self.group_repository.delete_group(command.group_id).await?;
        Ok(GroupCommandResult::success_with_message(
            command.group_id,
            "群组已删除".to_string(),
        ))
    }
}

#[async_trait]
impl QueryHandler<FindNearbyGroupsQuery> for GroupApplicationService {
    type Result = PaginatedResult<NearbyGroup>;
    async fn handle_query(&self, query: FindNearbyGroupsQuery) -> DomainResult<Self::Result> {
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        let rows = self
            .group_repository
            .find_nearby_groups(
                query.latitude,
                query.longitude,
                query
                    .radius_km
                    .unwrap_or(crate::domain::constants::DEFAULT_NEARBY_RADIUS_KM),
                cursor,
                limit + 1,
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
                let distance = crate::domain::values::location::Coordinates::new(
                    query.latitude,
                    query.longitude,
                )
                .and_then(|q| {
                    crate::domain::values::location::Coordinates::new(g.latitude, g.longitude)
                        .map(|c| q.distance_to_km(&c))
                })
                .unwrap_or(0.0);
                NearbyGroup {
                    group_id: g.id,
                    name: g.name,
                    latitude: g.latitude,
                    longitude: g.longitude,
                    distance_km: distance,
                    created_at: g.created_at,
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
    async fn handle_query(&self, query: SearchGroupsByNameQuery) -> DomainResult<Self::Result> {
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        let rows = self
            .group_repository
            .search_groups_by_name(&query.keyword, cursor, limit + 1)
            .await?;
        let has_more = rows.len() > limit as usize;
        let mut items = rows;
        if has_more {
            items.pop();
        }
        let result = items
            .into_iter()
            .map(|g| GroupSearchResult {
                group_id: g.id,
                name: g.name,
                has_location: true,
                created_at: g.created_at,
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
    async fn handle_command(&self, command: JoinGroupCommand) -> DomainResult<Self::Result> {
        command.validate()?;
        if !self
            .group_repository
            .check_group_exists(command.group_id)
            .await?
        {
            return Err(GroupDomainError::GroupNotFound {
                group_id: command.group_id,
            }
            .into());
        }
        if self
            .group_repository
            .check_user_in_group(command.group_id, command.user_id)
            .await?
        {
            return Err(GroupDomainError::UserAlreadyMember {
                user_id: command.user_id,
                group_id: command.group_id,
            }
            .into());
        }
        let (_ok, role) = self
            .group_repository
            .join_group(
                command.group_id,
                command.user_id,
                command.password.as_deref(),
            )
            .await?;
        Ok(JoinGroupResult {
            group_id: command.group_id,
            user_id: command.user_id,
            role,
        })
    }
}

#[async_trait]
impl CommandHandler<LeaveGroupCommand> for GroupApplicationService {
    type Result = LeaveGroupResult;
    async fn handle_command(&self, command: LeaveGroupCommand) -> DomainResult<Self::Result> {
        command.validate()?;
        if !self
            .group_repository
            .check_group_exists(command.group_id)
            .await?
        {
            return Err(GroupDomainError::GroupNotFound {
                group_id: command.group_id,
            }
            .into());
        }
        if !self
            .group_repository
            .check_user_in_group(command.group_id, command.user_id)
            .await?
        {
            return Err(GroupDomainError::UserNotMember {
                user_id: command.user_id,
                group_id: command.group_id,
            }
            .into());
        }
        let ok = self
            .group_repository
            .leave_group(command.group_id, command.user_id)
            .await?;
        Ok(LeaveGroupResult {
            group_id: command.group_id,
            user_id: command.user_id,
            success: ok,
        })
    }
}

#[async_trait]
impl CommandHandler<KickMemberCommand> for GroupApplicationService {
    type Result = KickMemberResult;
    async fn handle_command(&self, command: KickMemberCommand) -> DomainResult<Self::Result> {
        command.validate()?;
        if !self
            .group_repository
            .check_group_exists(command.group_id)
            .await?
        {
            return Err(GroupDomainError::GroupNotFound {
                group_id: command.group_id,
            }
            .into());
        }
        // 需要管理员权限
        if !self
            .group_repository
            .check_user_is_group_admin(command.group_id, command.admin_id)
            .await?
        {
            return Err(GroupDomainError::InsufficientPermission {
                required_role: "admin".to_string(),
            }
            .into());
        }
        // 目标必须是成员
        if !self
            .group_repository
            .check_user_in_group(command.group_id, command.target_user_id)
            .await?
        {
            return Err(GroupDomainError::UserNotMember {
                user_id: command.target_user_id,
                group_id: command.group_id,
            }
            .into());
        }
        // 不能移除群主
        if self
            .group_repository
            .check_user_is_group_owner(command.group_id, command.target_user_id)
            .await?
        {
            return Err(GroupDomainError::CannotRemoveOwner.into());
        }
        let ok = self
            .group_repository
            .kick_member(command.group_id, command.target_user_id, command.admin_id)
            .await?;
        Ok(KickMemberResult {
            group_id: command.group_id,
            user_id: command.target_user_id,
            success: ok,
        })
    }
}

#[async_trait]
impl CommandHandler<TransferOwnershipCommand> for GroupApplicationService {
    type Result = TransferOwnershipResult;
    async fn handle_command(
        &self,
        command: TransferOwnershipCommand,
    ) -> DomainResult<Self::Result> {
        command.validate()?;
        if !self
            .group_repository
            .check_group_exists(command.group_id)
            .await?
        {
            return Err(GroupDomainError::GroupNotFound {
                group_id: command.group_id,
            }
            .into());
        }
        // 只有现任群主可以转让
        if !self
            .group_repository
            .check_user_is_group_owner(command.group_id, command.current_owner_id)
            .await?
        {
            return Err(GroupDomainError::InsufficientPermission {
                required_role: "owner".to_string(),
            }
            .into());
        }
        // 新群主必须是成员
        if !self
            .group_repository
            .check_user_in_group(command.group_id, command.new_owner_id)
            .await?
        {
            return Err(GroupDomainError::UserNotMember {
                user_id: command.new_owner_id,
                group_id: command.group_id,
            }
            .into());
        }
        let ok = self
            .group_repository
            .transfer_ownership(
                command.group_id,
                command.new_owner_id,
                command.current_owner_id,
            )
            .await?;
        Ok(TransferOwnershipResult {
            group_id: command.group_id,
            new_owner_id: command.new_owner_id,
            success: ok,
        })
    }
}

#[async_trait]
impl QueryHandler<GroupMembersQuery> for GroupApplicationService {
    type Result = PaginatedResult<GroupMemberInfo>;
    async fn handle_query(&self, query: GroupMembersQuery) -> DomainResult<Self::Result> {
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        let (items, has_more, next_cursor): (Vec<PortGroupMemberInfo>, bool, Option<i64>) = self
            .group_repository
            .get_group_members(query.group_id, cursor, limit)
            .await?;
        let mapped = items
            .into_iter()
            .map(|m| GroupMemberInfo {
                member_id: m.member_id,
                user_id: m.user_id,
                nickname: m.nickname,
                role: m.role,
                last_active_at: m.last_active_at,
                joined_at: m.joined_at,
            })
            .collect::<Vec<_>>();
        Ok(PaginatedResult::new(mapped, has_more, next_cursor))
    }
}

#[async_trait]
impl QueryHandler<GetGroupByIdQuery> for GroupApplicationService {
    type Result = Option<Group>;
    
    async fn handle_query(&self, query: GetGroupByIdQuery) -> DomainResult<Self::Result> {
        self.group_repository.find_by_id(query.group_id).await
    }
}

#[async_trait]
impl QueryHandler<GetUserGroupsQuery> for GroupApplicationService {
    type Result = PaginatedResult<Group>;
    
    async fn handle_query(&self, query: GetUserGroupsQuery) -> DomainResult<Self::Result> {
        let _limit = query.pagination.get_limit();
        let _cursor = query.pagination.get_cursor();
        
        let groups = self.group_repository.find_user_groups(
            query.user_id,
            query.pagination.cursor.unwrap_or(0),
            query.pagination.limit.unwrap_or(20),
        ).await?;
        
        let has_more = groups.len() as i32 >= query.pagination.limit.unwrap_or(20);
        let next_cursor = if has_more && !groups.is_empty() {
            Some(groups.last().unwrap().id)
        } else {
            None
        };
        
        Ok(PaginatedResult {
            items: groups,
            has_more,
            next_cursor,
        })
    }
}

#[async_trait]
impl QueryHandler<GetOwnedGroupsQuery> for GroupApplicationService {
    type Result = PaginatedResult<Group>;
    
    async fn handle_query(&self, query: GetOwnedGroupsQuery) -> DomainResult<Self::Result> {
        let _limit = query.pagination.get_limit();
        let _cursor = query.pagination.get_cursor();
        
        let groups = self.group_repository.find_owned_groups(
            query.owner_id,
            query.pagination.cursor.unwrap_or(0),
            query.pagination.limit.unwrap_or(20),
        ).await?;
        
        let has_more = groups.len() as i32 >= query.pagination.limit.unwrap_or(20);
        let next_cursor = if has_more && !groups.is_empty() {
            Some(groups.last().unwrap().id)
        } else {
            None
        };
        
        Ok(PaginatedResult {
            items: groups,
            has_more,
            next_cursor,
        })
    }
}
