/// 评论应用服务
/// 
/// 功能职责：
/// - 实现评论相关的业务逻辑
/// - 协调评论仓储和其他服务
/// - 处理评论的权限验证和业务规则
/// - 提供评论统计和分析功能
use async_trait::async_trait;
use std::sync::Arc;

use crate::{
    domain::{DomainResult, Comment},
    ports::repository_ports::CommentRepository,
};

/// 评论应用服务接口
#[async_trait]
pub trait CommentApplicationService: Send + Sync {
    /// 创建评论
    async fn create_comment(
        &self,
        moment_id: &str,
        user_id: i64,
        content: &str,
        parent_comment_id: Option<&str>,
    ) -> DomainResult<Comment>;

    /// 删除评论
    async fn delete_comment(&self, comment_id: &str, user_id: i64) -> DomainResult<()>;

    /// 更新评论
    async fn update_comment(
        &self,
        comment_id: &str,
        user_id: i64,
        content: &str,
    ) -> DomainResult<Comment>;

    /// 获取生活瞬间的评论列表
    async fn get_moment_comments(
        &self,
        moment_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Comment>>;

    /// 获取用户评论历史
    async fn get_user_comments(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Comment>>;

    /// 获取评论回复
    async fn get_comment_replies(
        &self,
        parent_comment_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Comment>>;

    /// 获取生活瞬间评论数
    async fn get_moment_comments_count(&self, moment_id: &str) -> DomainResult<i64>;
}

/// 评论应用服务实现
#[derive(Clone)]
pub struct CommentApplicationServiceImpl<R>
where
    R: CommentRepository,
{
    base: super::BaseApplicationService,
    comment_repository: Arc<R>,
}

impl<R> CommentApplicationServiceImpl<R>
where
    R: CommentRepository,
{
    /// 创建新的评论应用服务
    pub fn new(
        comment_repository: Arc<R>,
        user_repository: Arc<dyn crate::ports::UserRepository>,
        event_publisher: Arc<dyn crate::ports::EventPublisher>,
    ) -> Self {
        Self { 
            base: super::BaseApplicationService::new("CommentApplicationService")
                .with_user_repository(user_repository)
                .with_event_publisher(event_publisher),
            comment_repository,
        }
    }

    /// 验证评论内容（使用基类方法）
    fn validate_content(content: &str) -> DomainResult<()> {
        super::ValidationRules::validate_text_content(content, 1000, "评论内容")
    }
}

#[async_trait]
impl<R> CommentApplicationService for CommentApplicationServiceImpl<R>
where
    R: CommentRepository,
{
    /// 创建评论
    async fn create_comment(
        &self,
        moment_id: &str,
        user_id: i64,
        content: &str,
        parent_comment_id: Option<&str>,
    ) -> DomainResult<Comment> {
        // 使用基类验证用户存在
        self.base.validate_user_exists(user_id).await?;
        
        // 验证评论内容
        Self::validate_content(content)?;
        
        // 记录操作日志
        self.base.log_operation("create_comment", Some(user_id), Some(&format!("moment: {}", moment_id)));

        // 创建评论
        self.comment_repository
            .create_comment(moment_id, user_id, content.trim(), parent_comment_id)
            .await
    }

    /// 删除评论
    async fn delete_comment(&self, comment_id: &str, user_id: i64) -> DomainResult<()> {
        self.comment_repository
            .delete_comment(comment_id, user_id)
            .await
    }

    /// 更新评论
    async fn update_comment(
        &self,
        comment_id: &str,
        user_id: i64,
        content: &str,
    ) -> DomainResult<Comment> {
        // 验证评论内容
        Self::validate_content(content)?;

        // 更新评论
        self.comment_repository
            .update_comment(comment_id, user_id, content.trim())
            .await
    }

    /// 获取生活瞬间的评论列表
    async fn get_moment_comments(
        &self,
        moment_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Comment>> {
        // 限制分页大小
        let limit = limit.min(100).max(1);
        
        self.comment_repository
            .find_comments_by_moment(moment_id, cursor, limit)
            .await
    }

    /// 获取用户评论历史
    async fn get_user_comments(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Comment>> {
        // 限制分页大小
        let limit = limit.min(100).max(1);
        
        self.comment_repository
            .find_comments_by_user(user_id, cursor, limit)
            .await
    }

    /// 获取评论回复
    async fn get_comment_replies(
        &self,
        parent_comment_id: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Comment>> {
        // 限制分页大小
        let limit = limit.min(100).max(1);
        
        self.comment_repository
            .find_comment_replies(parent_comment_id, cursor, limit)
            .await
    }

    /// 获取生活瞬间评论数
    async fn get_moment_comments_count(&self, moment_id: &str) -> DomainResult<i64> {
        self.comment_repository
            .get_moment_comments_count(moment_id)
            .await
    }
}

