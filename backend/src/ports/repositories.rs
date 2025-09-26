/// 仓储接口定义 - 数据访问抽象
/// 
/// 仓储特征：
/// - 聚合根的持久化
/// - 查询接口
/// - 事务支持
/// - 强类型泛型设计

use async_trait::async_trait;
use crate::domain::{
    DomainResult, Group, UserId, GroupId, MessageId, Moment,
    UserProfile, UserAuth, UserDetail, UserSearchResult, Comment,
    GroupProfile, GroupAuth, GroupDetail, GroupSearchResult,
};
use crate::domain::entities::{
    FriendProfile, FriendRequest, FriendshipStatus, Friendship,
    /* Block, BlockSettings, BlockStats, BlockType, BlockReason, */
};
use crate::domain::entities::{
    user::UserProfile as EntityUserProfile,
    comment::Comment as EntityComment,
};
use crate::application::{
    PaginationQuery, LocationQuery, PaginatedResult,
    UserView, GroupView,
};

// === 用户仓储接口 ===

/// 用户仓储接口
#[async_trait]
pub trait UserRepository: Send + Sync {
    /// 获取用户基础资料（轻量级）
    async fn find_user_profile(&self, user_id: i64) -> DomainResult<Option<UserProfile>>;
    
    /// 获取用户认证信息
    async fn find_user_auth(&self, login_id: &str) -> DomainResult<Option<UserAuth>>;
    
    /// 按昵称搜索用户资料
    async fn search_user_profiles_by_name(&self, keyword: &str, cursor: i64, limit: i32) -> DomainResult<Vec<UserSearchResult>>;
    
    /// 查找附近用户（带距离信息）
    async fn find_nearby_users_with_distance(&self, latitude: f64, longitude: f64, radius_meters: f64, cursor: i64, limit: i32) -> DomainResult<Vec<UserSearchResult>>;
    
    /// 获取用户详细信息
    async fn find_user_detail(&self, user_id: i64) -> DomainResult<Option<UserDetail>>;
    
    /// 创建用户
    async fn create_user(&self, login_id: Option<&str>, nickname: &str, password_hash: Option<&str>, email: Option<&str>, expires_at: Option<chrono::DateTime<chrono::Utc>>) -> DomainResult<UserProfile>;
    
    /// 更新用户资料
    async fn update_user_profile(&self, user_id: i64, nickname: Option<&str>, avatar_url: Option<&str>) -> DomainResult<UserProfile>;
    
    /// 设置用户密码
    async fn set_user_password(&self, user_id: i64, password_hash: &str) -> DomainResult<()>;
    
    /// 升级临时用户为正式用户
    async fn upgrade_temp_user(&self, user_id: i64, login_id: &str, password_hash: &str) -> DomainResult<UserProfile>;
    
    /// 删除用户
    async fn delete_user(&self, user_id: i64) -> DomainResult<()>;
    
    /// 更新用户位置
    async fn update_user_location(&self, user_id: i64, latitude: f64, longitude: f64, location_name: &str) -> DomainResult<()>;
    
    /// 查找用户位置
    async fn find_user_location(&self, user_id: i64) -> DomainResult<Option<(f64, f64, String)>>;
    
    /// 获取用户总数
    async fn get_total_user_count(&self) -> DomainResult<i64>;
    
    /// 获取指定ID范围内的用户数量
    async fn get_user_count_by_id_range(&self, start_id: i64, end_id: i64) -> DomainResult<i64>;
    
    /// 获取指定ID范围内的最大用户ID
    async fn get_max_user_id_by_range(&self, start_id: i64, end_id: i64) -> DomainResult<Option<i64>>;
}

/// 用户统计信息
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct UserStats {
    pub total_groups: u32,
    pub total_messages: u32,
    pub total_moments: u32,
    pub total_likes_received: u32,
}

// === 群组仓储接口 ===

/// 群组仓储接口
#[async_trait]
pub trait GroupRepository: Send + Sync {
    // === 命令操作（写） ===
    
    /// 保存群组（创建或更新）
    async fn save(&self, group: &Group) -> DomainResult<()>;
    
    /// 根据ID删除群组
    async fn delete_by_id(&self, group_id: GroupId) -> DomainResult<bool>;
    
    // === 查询操作（读） ===
    
    /// 根据ID查找群组
    async fn find_by_id(&self, group_id: GroupId, viewer_user_id: Option<UserId>) -> DomainResult<Option<Group>>;
    
    /// 检查用户是否为群组成员
    async fn is_member(&self, group_id: &GroupId, user_id: &UserId) -> DomainResult<bool>;
    
    /// 根据用户ID查找群组列表
    async fn find_by_user_id(&self, user_id: &UserId) -> DomainResult<Vec<Group>>;
    
    /// 根据用户ID查找群组列表（包含最后消息信息）
    async fn find_by_user_id_with_last_message(
        &self,
        user_id: &UserId,
        cursor: Option<&str>,
        limit: i32,
        include_archived: bool,
    ) -> DomainResult<crate::application::handlers::PaginatedResult<crate::application::services::message_application_service::GroupWithLastMessage>>;
    
    /// 按名称搜索群组（分页）
    async fn search_by_name(
        &self,
        query: &str,
        pagination: &PaginationQuery,
    ) -> DomainResult<PaginatedResult<GroupView>>;
    
    /// 按位置搜索附近群组（分页）
    async fn search_nearby(
        &self,
        location: &LocationQuery,
        pagination: &PaginationQuery,
    ) -> DomainResult<PaginatedResult<GroupView>>;
    
    /// 获取群组成员列表（分页）
    async fn get_members(
        &self,
        group_id: GroupId,
        pagination: &PaginationQuery,
    ) -> DomainResult<PaginatedResult<UserView>>;
    
    /// 获取用户加入的群组（分页）
    async fn get_user_groups(
        &self,
        user_id: UserId,
        pagination: &PaginationQuery,
    ) -> DomainResult<PaginatedResult<GroupView>>;
    
    /// 获取用户拥有的群组（分页）
    async fn get_user_owned_groups(
        &self,
        user_id: UserId,
        pagination: &PaginationQuery,
    ) -> DomainResult<PaginatedResult<GroupView>>;
    
    // 重复定义已移除，使用上面的 is_member 方法
    
    /// 获取群组统计信息
    async fn get_group_stats(&self, group_id: GroupId) -> DomainResult<GroupStats>;
    
    /// 获取群组基础资料（轻量级）
    async fn find_group_profile(&self, group_id: &str) -> DomainResult<Option<GroupProfile>>;
    
    /// 获取群组认证信息
    async fn find_group_auth(&self, group_id: &str) -> DomainResult<Option<GroupAuth>>;
    
    /// 获取群组详细信息
    async fn find_group_detail(&self, group_id: &str) -> DomainResult<Option<GroupDetail>>;
    
    /// 按名称搜索群组
    async fn search_groups_by_name(&self, keyword: &str, cursor: i64, limit: i32) -> DomainResult<Vec<GroupSearchResult>>;
    
    /// 按位置搜索附近群组
    async fn find_nearby_groups_with_distance(&self, latitude: f64, longitude: f64, radius_meters: f64, cursor: i64, limit: i32) -> DomainResult<Vec<GroupSearchResult>>;
    
    /// 查找用户加入的群组
    async fn find_user_groups(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<GroupProfile>>;
    
    /// 查找用户拥有的群组
    async fn find_owned_groups(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<GroupProfile>>;
    
    /// 创建群组
    async fn create_group(
        &self,
        name: &str,
        description: Option<&str>,
        owner_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: &str,
        is_public: bool,
        password_hash: Option<&str>,
    ) -> DomainResult<GroupProfile>;
    
    /// 更新群组
    async fn update_group(&self, group_id: &str, name: Option<&str>, description: Option<&str>) -> DomainResult<GroupProfile>;
    
    /// 删除群组
    async fn delete_group(&self, group_id: &str) -> DomainResult<()>;
    
    /// 添加群组成员
    async fn add_group_member(&self, group_id: &str, user_id: i64, role: &str) -> DomainResult<()>;
    
    /// 移除群组成员
    async fn remove_group_member(&self, group_id: &str, user_id: i64) -> DomainResult<()>;
    
    /// 检查是否为群组成员
    async fn is_group_member(&self, group_id: &str, user_id: i64) -> DomainResult<bool>;
    
    /// 获取用户在群组中的角色
    async fn get_user_role_in_group(&self, group_id: &str, user_id: i64) -> DomainResult<Option<String>>;
    
    /// 转移群组所有权
    async fn transfer_group_ownership(&self, group_id: &str, new_owner_id: i64) -> DomainResult<()>;
    
    /// 获取群组成员列表
    async fn get_group_members(&self, group_id: &str, cursor: i64, limit: i32) -> DomainResult<Vec<EntityUserProfile>>;
}

/// 群组统计信息
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct GroupStats {
    pub member_count: u32,
    pub message_count: u32,
    pub active_members_24h: u32,
    pub created_at: String,
}

// === 消息仓储接口 ===

/// 消息仓储接口
#[async_trait]
pub trait MessageRepository: Send + Sync {
    // === 命令操作（写） ===
    
    /// 创建消息（统一接口）
    async fn create(&self, message: crate::domain::entities::message::Message) -> DomainResult<crate::domain::entities::message::Message>;
    
    /// 更新消息
    async fn update(&self, message: crate::domain::entities::message::Message) -> DomainResult<crate::domain::entities::message::Message>;
    
    /// 删除消息
    async fn delete_message(&self, message_id: MessageId) -> DomainResult<bool>;
    
    /// 根据ID查找消息
    async fn find_by_id(&self, message_id: &MessageId) -> DomainResult<Option<crate::domain::entities::message::Message>>;
    
    /// 根据群组ID分页查找消息
    async fn find_by_group_id_paginated(
        &self,
        group_id: &GroupId,
        cursor: Option<&str>,
        limit: i32,
        include_system_messages: bool,
    ) -> DomainResult<crate::application::handlers::PaginatedResult<crate::domain::entities::message::Message>>;
    
    /// 标记群组消息已读到指定消息
    async fn mark_read_until_message(
        &self,
        group_id: &GroupId,
        user_id: &UserId,
        message_id: &MessageId,
    ) -> DomainResult<()>;
    
    /// 标记群组所有消息已读
    async fn mark_all_read(
        &self,
        group_id: &GroupId,
        user_id: &UserId,
    ) -> DomainResult<()>;
    
    /// 获取群组未读消息数
    async fn get_unread_count(
        &self,
        group_id: &GroupId,
        user_id: &UserId,
    ) -> DomainResult<i32>;
    
    // === 查询操作（读） ===
    
    // 旧的查询方法已移除，使用统一的消息架构方法
}

// 兼容性消息类型已移除，请使用统一的 Message 实体
// 根据需求文档的统一消息架构，所有消息都通过 group_id 管理

// === 瞬间仓储接口 ===

/// 瞬间仓储接口
#[async_trait]
pub trait MomentRepository: Send + Sync {
    /// 根据ID查找瞬间
    async fn find_by_id(&self, moment_id: &str) -> DomainResult<Option<Moment>>;
    
    /// 创建瞬间
    async fn create_moment(&self, user_id: i64, description: &str, latitude: f64, longitude: f64, location_name: &str, tags: &[String]) -> DomainResult<Moment>;
    
    /// 更新瞬间
    async fn update_moment(&self, moment_id: &str, user_id: i64, description: Option<&str>, tags: Option<&[String]>) -> DomainResult<Moment>;
    
    /// 删除瞬间
    async fn delete_moment(&self, moment_id: &str, user_id: i64) -> DomainResult<()>;
    
    /// 查找用户瞬间
    async fn find_user_moments(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Moment>>;
    
    /// 查找附近瞬间
    async fn find_nearby_moments(&self, latitude: f64, longitude: f64, radius_km: f64, cursor: i64, limit: i32) -> DomainResult<Vec<Moment>>;
    
    /// 点赞瞬间
    async fn like_moment(&self, moment_id: &str, user_id: i64) -> DomainResult<()>;
    
    /// 取消点赞瞬间
    async fn unlike_moment(&self, moment_id: &str, user_id: i64) -> DomainResult<()>;
    
    /// 按标签搜索瞬间
    async fn search_by_tags(&self, tags: &[String], cursor: i64, limit: i32) -> DomainResult<Vec<Moment>>;
    
    /// 检查瞬间是否被用户点赞
    async fn is_moment_liked(&self, moment_id: &str, user_id: i64) -> DomainResult<bool>;
    
    /// 获取瞬间点赞数
    async fn get_moment_likes_count(&self, moment_id: &str) -> DomainResult<i64>;
}


// === 仓储工厂接口 ===

/// 仓储工厂 - 创建和管理仓储实例
pub trait RepositoryFactory: Send + Sync + Clone {
    type UserRepo: UserRepository;
    type GroupRepo: GroupRepository;
    type MessageRepo: MessageRepository;
    type MomentRepo: MomentRepository;
    
    /// 创建用户仓储
    fn user_repository(&self) -> Self::UserRepo;
    
    /// 创建群组仓储
    fn group_repository(&self) -> Self::GroupRepo;
    
    /// 创建消息仓储
    fn message_repository(&self) -> Self::MessageRepo;
    
    /// 创建瞬间仓储
    fn moment_repository(&self) -> Self::MomentRepo;
}

// === 其他仓储接口 ===

/// 评论仓储接口
#[async_trait]
pub trait CommentRepository: Send + Sync {
    /// 创建评论
    async fn create_comment(&self, moment_id: &str, user_id: i64, content: &str, parent_comment_id: Option<&str>) -> DomainResult<EntityComment>;
    
    /// 根据ID查找评论
    async fn find_comment_by_id(&self, comment_id: &str) -> DomainResult<Option<EntityComment>>;
    
    /// 查找瞬间的评论
    async fn find_comments_by_moment(&self, moment_id: &str, cursor: i64, limit: i32) -> DomainResult<Vec<Comment>>;
    
    /// 查找用户的评论
    async fn find_comments_by_user(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Comment>>;
    
    /// 删除评论
    async fn delete_comment(&self, comment_id: &str, user_id: i64) -> DomainResult<()>;
    
    /// 更新评论
    async fn update_comment(&self, comment_id: &str, user_id: i64, content: &str) -> DomainResult<EntityComment>;
    
    /// 查找评论的回复
    async fn find_comment_replies(&self, parent_comment_id: &str, cursor: i64, limit: i32) -> DomainResult<Vec<Comment>>;
    
    /// 获取瞬间评论数
    async fn get_moment_comments_count(&self, moment_id: &str) -> DomainResult<i64>;
}

// 好友仓储接口
#[async_trait]
pub trait FriendRepository: Send + Sync {
    /// 保存好友关系
    async fn save(&self, friendship: &Friendship) -> DomainResult<()>;
    
    /// 根据用户ID查找好友列表
    async fn find_friends_by_user_id(&self, user_id: i64) -> DomainResult<Vec<Friendship>>;
    
    /// 发送好友请求
    async fn send_friend_request(&self, requester_id: i64, target_id: i64, message: Option<&str>) -> DomainResult<FriendRequest>;
    
    /// 接受好友请求
    async fn accept_friend_request(&self, request_id: &str, target_id: i64) -> DomainResult<Friendship>;
    
    /// 拒绝好友请求
    async fn reject_friend_request(&self, request_id: &str, target_id: i64) -> DomainResult<()>;
    
    /// 移除好友关系
    async fn remove_friendship(&self, user_id: i64, friend_id: i64) -> DomainResult<()>;
    
    /// 获取好友列表
    async fn get_friends(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendProfile>>;
    
    /// 关注用户（返回关系ID）
    async fn follow(&self, follower_id: i64, target_id: i64) -> DomainResult<String>;
    
    /// 取消关注
    async fn unfollow(&self, follower_id: i64, target_id: i64) -> DomainResult<()>;
    
    /// 拉黑用户（返回关系ID）
    async fn block_user(&self, blocker_id: i64, blocked_id: i64, reason: Option<&str>) -> DomainResult<String>;
    
    /// 获取拉黑列表
    async fn get_blocked_users(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendProfile>>;
    
    /// 获取收到的好友请求
    async fn get_received_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>>;
    
    /// 获取发送的好友请求
    async fn get_sent_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendRequest>>;
    
    /// 检查好友关系
    async fn check_friendship(&self, user_id: i64, friend_id: i64) -> DomainResult<FriendshipStatus>;
    
    /// 获取待处理请求数量
    async fn get_pending_requests_count(&self, user_id: i64) -> DomainResult<i64>;
}

// 屏蔽仓储接口
// 暂时禁用，等待重新设计
/*
#[async_trait]
pub trait BlockRepository: Send + Sync + Clone {
    /// 保存屏蔽记录
    async fn save(&self, block: &Block) -> DomainResult<()>;
    
    /// 根据用户ID查找屏蔽列表
    async fn find_blocks_by_user_id(&self, user_id: UserId) -> DomainResult<Vec<Block>>;
    
    /// 屏蔽用户
    async fn block_user(&self, blocker_id: i64, blocked_id: i64, block_type: BlockType, reason: BlockReason) -> DomainResult<Block>;
    
    /// 取消屏蔽用户
    async fn unblock_user(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<()>;
    
    /// 检查是否被屏蔽
    async fn is_blocked(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<bool>;
    
    /// 获取屏蔽详情
    async fn get_block_details(&self, blocker_id: i64, blocked_id: i64) -> DomainResult<Option<Block>>;
    
    /// 获取被屏蔽的用户列表
    async fn get_blocked_users(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Block>>;
    
    /// 获取屏蔽我的用户列表
    async fn get_blocking_users(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<Block>>;
    
    /// 获取屏蔽设置
    async fn get_block_settings(&self, user_id: i64) -> DomainResult<Option<BlockSettings>>;
    
    /// 更新屏蔽设置
    async fn update_block_settings(&self, user_id: i64, settings: &BlockSettings) -> DomainResult<()>;
    
    /// 获取屏蔽统计
    async fn get_block_stats(&self, user_id: i64) -> DomainResult<BlockStats>;
    
    /// 批量检查屏蔽状态
    async fn batch_check_blocked(&self, blocker_id: i64, user_ids: &[i64]) -> DomainResult<Vec<(i64, bool)>>;
}
*/

// 举报仓储接口
#[async_trait]
pub trait ReportRepository: Send + Sync {
    /// 创建举报记录
    async fn create_report(
        &self,
        reporter_id: i64,
        target_type: crate::adapters::adapters_pg::report_repository::ReportTargetType,
        target_id: String,
        reason: crate::adapters::adapters_pg::report_repository::ReportReason,
        description: Option<String>,
    ) -> DomainResult<crate::adapters::adapters_pg::report_repository::Report>;
    
    /// 获取用户的举报记录
    async fn get_user_reports(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<crate::adapters::adapters_pg::report_repository::Report>>;
    
    /// 获取举报列表（管理员）
    async fn get_reports(
        &self,
        status: Option<crate::adapters::adapters_pg::report_repository::ReportStatus>,
        target_type: Option<crate::adapters::adapters_pg::report_repository::ReportTargetType>,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<crate::adapters::adapters_pg::report_repository::Report>>;
    
    /// 处理举报
    async fn handle_report(
        &self,
        report_id: String,
        handler_id: i64,
        action: String,
        note: Option<String>,
    ) -> DomainResult<crate::adapters::adapters_pg::report_repository::Report>;
    
    /// 根据ID查找举报
    async fn find_by_id(&self, report_id: String) -> DomainResult<Option<crate::adapters::adapters_pg::report_repository::Report>>;
    
    /// 检查重复举报
    async fn check_duplicate_report(
        &self,
        reporter_id: i64,
        target_type: crate::adapters::adapters_pg::report_repository::ReportTargetType,
        target_id: String,
    ) -> DomainResult<bool>;
}

// 通知仓储接口
#[async_trait]
pub trait NotificationRepository: Send + Sync {
    /// 创建通知
    async fn create_notification(&self, notification: crate::adapters::adapters_pg::notification_repository::Notification) -> DomainResult<crate::adapters::adapters_pg::notification_repository::Notification>;
    
    /// 批量创建通知
    async fn create_notifications(&self, notifications: Vec<crate::adapters::adapters_pg::notification_repository::Notification>) -> DomainResult<Vec<crate::adapters::adapters_pg::notification_repository::Notification>>;
    
    /// 获取用户通知列表
    async fn get_user_notifications(
        &self,
        user_id: i64,
        status: Option<crate::adapters::adapters_pg::notification_repository::NotificationStatus>,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<crate::adapters::adapters_pg::notification_repository::Notification>>;
    
    /// 标记通知为已读
    async fn mark_notification_read(&self, notification_id: String, user_id: i64) -> DomainResult<()>;
    
    /// 批量标记通知为已读
    async fn mark_notifications_read(&self, notification_ids: Vec<String>, user_id: i64) -> DomainResult<()>;
    
    /// 标记所有通知为已读
    async fn mark_all_notifications_read(&self, user_id: i64) -> DomainResult<()>;
    
    /// 获取未读通知数量
    async fn get_unread_count(&self, user_id: i64) -> DomainResult<i64>;
    
    /// 删除通知
    async fn delete_notification(&self, notification_id: String, user_id: i64) -> DomainResult<()>;
    
    /// 清理过期通知
    async fn cleanup_expired_notifications(&self) -> DomainResult<i64>;
    
    /// 获取通知设置
    async fn get_notification_settings(&self, user_id: i64) -> DomainResult<Option<crate::adapters::adapters_pg::notification_repository::NotificationSettings>>;
    
    /// 更新通知设置
    async fn update_notification_settings(&self, settings: crate::adapters::adapters_pg::notification_repository::NotificationSettings) -> DomainResult<crate::adapters::adapters_pg::notification_repository::NotificationSettings>;
    
    /// 根据ID查找通知
    async fn find_by_id(&self, notification_id: String) -> DomainResult<Option<crate::adapters::adapters_pg::notification_repository::Notification>>;
}

// 注意：所有类型定义已从 crate::domain 导入，无需重复定义
