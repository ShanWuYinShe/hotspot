use crate::domain::{DomainResult, Group, Moment, User};
/// 仓储端口定义
///
/// 定义所有数据访问相关的抽象接口
use async_trait::async_trait;

/// 用户仓储端口
#[async_trait]
pub trait UserRepositoryPort: Send + Sync {
    /// 根据ID查找用户
    async fn find_by_id(&self, user_id: i64) -> DomainResult<Option<User>>;

    /// 根据登录ID查找用户
    async fn find_by_login_id(&self, login_id: &str) -> DomainResult<Option<User>>;

    /// 创建用户
    async fn create_user(&self, user: &User) -> DomainResult<User>;

    /// 更新用户
    async fn update_user(&self, user: &User) -> DomainResult<User>;

    /// 删除用户
    async fn delete_user(&self, user_id: i64) -> DomainResult<()>;

    /// 搜索附近用户
    async fn find_nearby_users(
        &self,
        latitude: f64,
        longitude: f64,
        radius_km: f64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<User>>;

    /// 按昵称搜索用户
    async fn search_users_by_nickname(
        &self,
        keyword: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<User>>;

    /// 统计所有用户总数
    async fn get_total_user_count(&self) -> DomainResult<i64>;

    /// 统计指定ID区间内的用户数量（闭区间）
    async fn get_user_count_by_id_range(&self, start_id: i64, end_id: i64) -> DomainResult<i64>;

    /// 获取指定ID区间内的最大用户ID（若不存在则返回None）
    async fn get_max_user_id_by_range(
        &self,
        start_id: i64,
        end_id: i64,
    ) -> DomainResult<Option<i64>>;

    /// 查找用户位置信息
    async fn find_user_location(&self, user_id: i64) -> DomainResult<Option<crate::domain::UserLocation>>;

    /// 创建或更新用户位置信息
    async fn upsert_user_location(&self, user_location: &crate::domain::UserLocation) -> DomainResult<crate::domain::UserLocation>;

    /// 搜索附近用户（包含距离信息）
    async fn find_nearby_users_with_distance(
        &self,
        latitude: f64,
        longitude: f64,
        radius_km: f64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<crate::application::NearbyUser>>;
}

/// 群组仓储端口

pub struct GroupMemberInfo {
    pub member_id: i64,
    pub user_id: i64,
    pub nickname: String,
    pub role: String,
    pub last_active_at: chrono::DateTime<chrono::Utc>,
    pub joined_at: chrono::DateTime<chrono::Utc>,
}

#[async_trait]
pub trait GroupRepositoryPort: Send + Sync {
    /// 根据ID查找群组
    async fn find_by_id(&self, group_id: i64) -> DomainResult<Option<Group>>;

    /// 创建群组
    async fn create_group(&self, group: &Group) -> DomainResult<Group>;

    /// 更新群组
    async fn update_group(&self, group: &Group) -> DomainResult<Group>;

    /// 删除群组
    async fn delete_group(&self, group_id: i64) -> DomainResult<()>;

    /// 搜索附近群组
    async fn find_nearby_groups(
        &self,
        latitude: f64,
        longitude: f64,
        radius_km: f64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Group>>;

    /// 按名称搜索群组
    async fn search_groups_by_name(
        &self,
        keyword: &str,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Group>>;

    // -------- 成员管理（新增） --------

    /// 成员加入群组（可选密码）
    async fn join_group(
        &self,
        group_id: i64,
        user_id: i64,
        password: Option<&str>,
    ) -> DomainResult<(bool, String)>;

    /// 成员离开群组
    async fn leave_group(&self, group_id: i64, user_id: i64) -> DomainResult<bool>;

    /// 踢出成员（需要管理员/群主）
    async fn kick_member(
        &self,
        group_id: i64,
        target_user_id: i64,
        admin_id: i64,
    ) -> DomainResult<bool>;

    /// 转移群主
    async fn transfer_ownership(
        &self,
        group_id: i64,
        new_owner_id: i64,
        current_owner_id: i64,
    ) -> DomainResult<bool>;

    /// 获取群组成员列表（分页）
    async fn get_group_members(
        &self,
        group_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<(Vec<GroupMemberInfo>, bool, Option<i64>)>;

    /// 检查群组是否存在
    async fn check_group_exists(&self, group_id: i64) -> DomainResult<bool>;

    /// 检查用户是否在群组中
    async fn check_user_in_group(&self, group_id: i64, user_id: i64) -> DomainResult<bool>;

    /// 检查是否为群主
    async fn check_user_is_group_owner(&self, group_id: i64, user_id: i64) -> DomainResult<bool>;

    /// 检查是否为管理员
    async fn check_user_is_group_admin(&self, group_id: i64, user_id: i64) -> DomainResult<bool>;

    /// 获取用户加入的群组
    async fn find_user_groups(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Group>>;

    /// 获取用户拥有的群组
    async fn find_owned_groups(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Group>>;
}

/// 端口层会话摘要DTO（保留给新架构使用）
pub struct ConversationSummaryPort {
    pub conversation_id: String,
    pub user1_id: i64,
    pub user2_id: i64,
    pub peer_id: i64,
    pub last_message_preview: Option<String>,
    pub last_message_at: chrono::DateTime<chrono::Utc>,
    pub created_at: chrono::DateTime<chrono::Utc>,
}

/// 瞬间仓储端口
#[async_trait]
pub trait MomentRepositoryPort: Send + Sync {
    /// 根据ID查找瞬间
    async fn find_by_id(&self, moment_id: i64) -> DomainResult<Option<Moment>>;

    /// 创建瞬间
    async fn create_moment(&self, moment: &Moment) -> DomainResult<Moment>;

    /// 更新瞬间
    async fn update_moment(&self, moment: &Moment) -> DomainResult<Moment>;

    /// 删除瞬间
    async fn delete_moment(&self, moment_id: i64) -> DomainResult<()>;

    /// 获取用户瞬间列表
    async fn find_user_moments(
        &self,
        user_id: i64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Moment>>;

    /// 搜索附近瞬间
    async fn find_nearby_moments(
        &self,
        latitude: f64,
        longitude: f64,
        radius_km: f64,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Moment>>;

    /// 点赞瞬间
    async fn like_moment(&self, moment_id: i64, user_id: i64) -> DomainResult<()>;

    /// 取消点赞瞬间
    async fn unlike_moment(&self, moment_id: i64, user_id: i64) -> DomainResult<()>;

    /// 按标签搜索瞬间
    async fn search_by_tags(
        &self,
        tags: Vec<String>,
        cursor: i64,
        limit: i32,
    ) -> DomainResult<Vec<Moment>>;
}

/// 好友关系状态
#[derive(Debug, Clone, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum FriendshipStatus {
    Pending,
    Accepted,
    Rejected,
    Blocked,
}

/// 好友关系信息
#[derive(Debug, Clone)]
pub struct FriendshipInfo {
    pub friendship_id: String,
    pub requester_id: i64,
    pub addressee_id: i64,
    pub status: FriendshipStatus,
    pub created_at: chrono::DateTime<chrono::Utc>,
    pub updated_at: chrono::DateTime<chrono::Utc>,
}

/// 好友仓储端口
#[async_trait]
pub trait FriendRepositoryPort: Send + Sync {
    /// 发送好友请求
    async fn send_friend_request(&self, requester_id: i64, addressee_id: i64) -> DomainResult<FriendshipInfo>;

    /// 接受好友请求
    async fn accept_friend_request(&self, requester_id: i64, addressee_id: i64) -> DomainResult<()>;

    /// 拒绝好友请求
    async fn reject_friend_request(&self, requester_id: i64, addressee_id: i64) -> DomainResult<()>;

    /// 删除好友关系
    async fn remove_friendship(&self, user_id: i64, friend_id: i64) -> DomainResult<()>;

    /// 获取用户的好友列表
    async fn get_friends(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<i64>>;

    /// 获取用户收到的好友请求
    async fn get_received_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendshipInfo>>;

    /// 获取用户发送的好友请求
    async fn get_sent_friend_requests(&self, user_id: i64, cursor: i64, limit: i32) -> DomainResult<Vec<FriendshipInfo>>;

    /// 检查两个用户是否是好友
    async fn are_friends(&self, user_id1: i64, user_id2: i64) -> DomainResult<bool>;

    /// 获取待处理的好友请求数量
    async fn get_pending_requests_count(&self, user_id: i64) -> DomainResult<i64>;
}
