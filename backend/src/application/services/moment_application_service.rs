/// 瞬间应用服务
use async_trait::async_trait;
use std::sync::Arc;
use crate::application::commands::{CommandHandler, CreateMomentCommand, DeleteMomentCommand, LikeMomentCommand, UnlikeMomentCommand};
use crate::application::queries::{QueryHandler, PaginatedResult, SearchMomentsByTagsQuery, GetMomentByIdQuery, SearchMomentsByLocationQuery, GetUserMomentsQuery};
use crate::domain::{DomainResult, Moment};
use crate::ports::MomentRepositoryPort;

#[derive(Clone)]
pub struct MomentApplicationService {
    moment_repository: Arc<dyn MomentRepositoryPort>,
}

impl MomentApplicationService {
    pub fn new(moment_repository: Arc<dyn MomentRepositoryPort>) -> Self {
        Self { moment_repository }
    }
}

#[async_trait]
impl CommandHandler<CreateMomentCommand> for MomentApplicationService {
    type Result = Moment;
    async fn handle_command(&self, command: CreateMomentCommand) -> DomainResult<Self::Result> {
        // 验证命令
        command.validate()?;
        
        let moment = Moment::new(
            command.user_id, 
            Some(command.content), 
            command.latitude.unwrap_or(35.16), 
            command.longitude.unwrap_or(112.68), 
            command.address.unwrap_or_else(|| "未知位置".to_string())
        );
        
        // 注意：tags和image_urls字段暂时不在Moment实体中存储
        // 在未来版本中可以添加专门的标签和图片表来支持这些功能
        
        self.moment_repository.create_moment(&moment).await
    }
}

#[async_trait]
impl CommandHandler<DeleteMomentCommand> for MomentApplicationService {
    type Result = ();
    async fn handle_command(&self, command: DeleteMomentCommand) -> DomainResult<Self::Result> {
        // 权限校验：仅作者可删除
        if let Some(moment) = self.moment_repository.find_by_id(command.moment_id).await? {
            if moment.user_id != command.requester_id {
                return Err(crate::domain::DomainError::Moment(crate::domain::MomentDomainError::NotMomentAuthor));
            }
        } else {
            return Err(crate::domain::DomainError::Moment(crate::domain::MomentDomainError::MomentNotFound { moment_id: command.moment_id }));
        }
        self.moment_repository.delete_moment(command.moment_id).await
    }
}

#[async_trait]
impl CommandHandler<LikeMomentCommand> for MomentApplicationService {
    type Result = ();
    async fn handle_command(&self, command: LikeMomentCommand) -> DomainResult<Self::Result> {
        // 检查moment是否存在
        if self.moment_repository.find_by_id(command.moment_id).await?.is_none() {
            return Err(crate::domain::DomainError::Moment(crate::domain::MomentDomainError::MomentNotFound { 
                moment_id: command.moment_id 
            }));
        }
        
        self.moment_repository.like_moment(command.moment_id, command.user_id).await?;
        Ok(())
    }
}

#[async_trait]
impl CommandHandler<UnlikeMomentCommand> for MomentApplicationService {
    type Result = ();
    async fn handle_command(&self, command: UnlikeMomentCommand) -> DomainResult<Self::Result> {
        // 检查moment是否存在
        if self.moment_repository.find_by_id(command.moment_id).await?.is_none() {
            return Err(crate::domain::DomainError::Moment(crate::domain::MomentDomainError::MomentNotFound { 
                moment_id: command.moment_id 
            }));
        }
        
        self.moment_repository.unlike_moment(command.moment_id, command.user_id).await?;
        Ok(())
    }
}

#[async_trait]
impl QueryHandler<SearchMomentsByTagsQuery> for MomentApplicationService {
    type Result = PaginatedResult<Moment>;
    async fn handle_query(&self, query: SearchMomentsByTagsQuery) -> DomainResult<Self::Result> {
        query.validate()?;
        
        let moments = self.moment_repository.search_by_tags(
            query.tags.clone(),
            query.pagination.cursor.unwrap_or(0),
            query.pagination.limit.unwrap_or(20),
        ).await?;
        
        let has_more = moments.len() as i32 >= query.pagination.limit.unwrap_or(20);
        let next_cursor = if has_more && !moments.is_empty() {
            Some(moments.last().unwrap().id)
        } else {
            None
        };
        
        Ok(PaginatedResult {
            items: moments,
            has_more,
            next_cursor,
        })
    }
}

#[async_trait]
impl QueryHandler<GetMomentByIdQuery> for MomentApplicationService {
    type Result = Option<Moment>;
    async fn handle_query(&self, query: GetMomentByIdQuery) -> DomainResult<Self::Result> {
        self.moment_repository.find_by_id(query.moment_id).await
    }
}

#[async_trait]
impl QueryHandler<SearchMomentsByLocationQuery> for MomentApplicationService {
    type Result = PaginatedResult<Moment>;
    async fn handle_query(&self, query: SearchMomentsByLocationQuery) -> DomainResult<Self::Result> {
        query.validate()?;
        
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        
        let rows = self.moment_repository.find_nearby_moments(
            query.latitude, 
            query.longitude, 
            query.radius_km, 
            cursor, 
            limit + 1
        ).await?;
        
        let has_more = rows.len() > limit as usize;
        let mut items = rows;
        if has_more { items.pop(); }
        let next_cursor = if has_more && !items.is_empty() { 
            Some(items.last().unwrap().id) 
        } else { 
            None 
        };
        
        Ok(PaginatedResult::new(items, has_more, next_cursor))
    }
}

#[async_trait]
impl QueryHandler<GetUserMomentsQuery> for MomentApplicationService {
    type Result = PaginatedResult<Moment>;
    async fn handle_query(&self, query: GetUserMomentsQuery) -> DomainResult<Self::Result> {
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        
        let rows = self.moment_repository.find_user_moments(
            query.user_id, 
            cursor, 
            limit + 1
        ).await?;
        
        let has_more = rows.len() > limit as usize;
        let mut items = rows;
        if has_more { items.pop(); }
        let next_cursor = if has_more && !items.is_empty() { 
            Some(items.last().unwrap().id) 
        } else { 
            None 
        };
        
        Ok(PaginatedResult::new(items, has_more, next_cursor))
    }
}