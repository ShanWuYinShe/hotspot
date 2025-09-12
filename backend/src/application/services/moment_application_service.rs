/// 瞬间应用服务
use async_trait::async_trait;
/// 生活瞬间应用服务
/// 
/// 功能职责：
/// - 实现生活瞬间相关的业务用例和应用逻辑
/// - 协调瞬间领域实体和仓储层的交互
/// - 管理瞬间的创建、编辑、删除和查询
/// - 处理基于地理位置的瞬间搜索和推荐
/// - 管理瞬间的标签系统和分类管理
/// 
/// 主要用例：
/// - 瞬间的创建和发布流程
/// - 瞬间内容的编辑和更新
/// - 瞬间的删除和权限控制
/// - 附近瞬间的搜索和发现
/// - 按标签搜索和分类浏览
/// - 瞬间的点赞和取消点赞
/// 
/// 交互功能：
/// - 管理瞬间的点赞和评论统计
/// - 处理瞬间的分享和转发功能
/// - 协调瞬间相关的通知和推送
use std::sync::Arc;
use crate::application::commands::{CreateMomentCommand, DeleteMomentCommand, LikeMomentCommand, UnlikeMomentCommand};
use crate::application::handlers::CommandHandler;
use crate::application::queries::{SearchMomentsByTagsQuery, SearchMomentsByIdQuery, SearchMomentsByLocationQuery, SearchMomentsByUserIdQuery};
use crate::application::handlers::{QueryHandler, PaginatedResult};
use crate::domain::{Moment};
use crate::application::{ApplicationResult, ApplicationError};
use crate::ports::MomentRepository;

#[derive(Clone)]
pub struct MomentApplicationService {
    base: super::BaseApplicationService,
    moment_repository: Arc<dyn MomentRepository>,
}

impl MomentApplicationService {
    pub fn new(
        moment_repository: Arc<dyn MomentRepository>,
        user_repository: Arc<dyn crate::ports::UserRepository>,
        event_publisher: Arc<dyn crate::ports::EventPublisher>,
    ) -> Self {
        Self { 
            base: super::BaseApplicationService::new("MomentApplicationService")
                .with_user_repository(user_repository)
                .with_event_publisher(event_publisher),
            moment_repository,
        }
    }
}

#[async_trait]
impl CommandHandler<CreateMomentCommand> for MomentApplicationService {
    type Result = Moment;
    async fn handle(&self, command: CreateMomentCommand) -> ApplicationResult<Self::Result> {
        // 验证命令
        command.validate()?;
        
        // 使用基类验证用户存在
        self.base.validate_user_exists(command.user_id).await?;
        
        // 验证内容
        super::ValidationRules::validate_text_content(&command.content, 500, "瞬间内容")?;
        
        // 记录操作日志
        self.base.log_operation("create_moment", Some(command.user_id), Some("create new moment"));
        
        // 注意：tags和image_urls字段暂时不在Moment实体中存储
        // 在未来版本中可以添加专门的标签和图片表来支持这些功能
        
        let user_id = command.user_id;
        let content = command.content.clone();
        let latitude = command.latitude.unwrap_or(0.0);
        let longitude = command.longitude.unwrap_or(0.0);
        let address = command.address.clone().unwrap_or_default();
        let tags = command.tags.clone().unwrap_or_default();
        
        self.moment_repository.create_moment(
            user_id,
            &content,
            latitude,
            longitude,
            &address,
            &tags,
        ).await.map_err(ApplicationError::Domain)
    }
}

#[async_trait]
impl CommandHandler<DeleteMomentCommand> for MomentApplicationService {
    type Result = ();
    async fn handle(&self, command: DeleteMomentCommand) -> ApplicationResult<Self::Result> {
        // 权限校验：仅作者可删除
        if let Some(moment) = self.moment_repository.find_by_id(&command.moment_id.to_string()).await? {
            if moment.user_id != command.requester_id {
                return Err(ApplicationError::Domain(crate::domain::DomainError::NotMomentAuthor));
            }
        } else {
            return Err(ApplicationError::Domain(crate::domain::DomainError::MomentNotFound));
        }
        self.moment_repository.delete_moment(&command.moment_id.to_string(), command.requester_id).await.map_err(ApplicationError::Domain)
    }
}

#[async_trait]
impl CommandHandler<LikeMomentCommand> for MomentApplicationService {
    type Result = ();
    async fn handle(&self, command: LikeMomentCommand) -> ApplicationResult<Self::Result> {
        // 检查moment是否存在
        if self.moment_repository.find_by_id(&command.moment_id.to_string()).await?.is_none() {
            return Err(ApplicationError::Domain(crate::domain::DomainError::MomentNotFound));
        }
        
        self.moment_repository.like_moment(&command.moment_id.to_string(), command.user_id).await?;
        Ok(())
    }
}

#[async_trait]
impl CommandHandler<UnlikeMomentCommand> for MomentApplicationService {
    type Result = ();
    async fn handle(&self, command: UnlikeMomentCommand) -> ApplicationResult<Self::Result> {
        // 检查moment是否存在
        if self.moment_repository.find_by_id(&command.moment_id.to_string()).await?.is_none() {
            return Err(ApplicationError::Domain(crate::domain::DomainError::MomentNotFound));
        }
        
        self.moment_repository.unlike_moment(&command.moment_id.to_string(), command.user_id).await?;
        Ok(())
    }
}

#[async_trait]
impl QueryHandler<SearchMomentsByTagsQuery> for MomentApplicationService {
    type Result = PaginatedResult<Moment>;
    async fn handle(&self, query: SearchMomentsByTagsQuery) -> ApplicationResult<Self::Result> {
        query.validate()?;
        
        let moments = self.moment_repository.search_by_tags(
            &query.tags,
            query.pagination.cursor.unwrap_or(0),
            query.pagination.limit.unwrap_or(20) as i32,
        ).await?;
        
        let has_more = moments.len() as i32 >= query.pagination.limit.unwrap_or(20) as i32;
        let next_cursor = if has_more && !moments.is_empty() {
            Some(moments.last().unwrap().id)
        } else {
            None
        };
        
        Ok(PaginatedResult {
            items: moments,
            has_more,
            next_cursor: next_cursor.map(|c| c.to_string()),
        })
    }
}

#[async_trait]
impl QueryHandler<SearchMomentsByIdQuery> for MomentApplicationService {
    type Result = Option<Moment>;
    async fn handle(&self, query: SearchMomentsByIdQuery) -> ApplicationResult<Self::Result> {
        self.moment_repository.find_by_id(&query.moment_id.to_string()).await.map_err(ApplicationError::Domain)
    }
}

#[async_trait]
impl QueryHandler<SearchMomentsByLocationQuery> for MomentApplicationService {
    type Result = PaginatedResult<Moment>;
    async fn handle(&self, query: SearchMomentsByLocationQuery) -> ApplicationResult<Self::Result> {
        query.validate()?;
        
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        
        let rows = self.moment_repository.find_nearby_moments(
            query.latitude, 
            query.longitude, 
            query.radius_km, 
            cursor, 
            (limit + 1) as i32
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
impl QueryHandler<SearchMomentsByUserIdQuery> for MomentApplicationService {
    type Result = PaginatedResult<Moment>;
    async fn handle(&self, query: SearchMomentsByUserIdQuery) -> ApplicationResult<Self::Result> {
        let limit = query.pagination.get_limit();
        let cursor = query.pagination.get_cursor();
        
        let rows = self.moment_repository.find_user_moments(
            query.user_id, 
            cursor, 
            (limit + 1) as i32
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