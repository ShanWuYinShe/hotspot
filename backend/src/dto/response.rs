//! Response DTO 定义
//!
//! 所有响应类型严格遵循 API 规范 (spec/*.mdc)
//! 使用强类型枚举，无向后兼容代码

use serde::{Deserialize, Serialize};
use serde_json::Value;

// 导入所有需要的枚举类型
use crate::types::enums::{
    AnnouncementSeverity, ApplicationStatus, Gender, GroupJoinMethod, GroupRole, MapItemType,
    MessageType, NotificationType, PrivacyLevel, PrivateChatPermission, ReportReason, ReportStatus,
    ReportTargetType, ShareType, SystemHealthStatus, TagTrend, UserAccountStatus, UserPresenceStatus,
    UserType,
};

// ============================================================
// ID → string 序列化辅助
// ============================================================

/// 将 i64 序列化为 JSON string（spec 要求所有 ID 为 string 类型）
pub mod id_as_string {
    use serde::{self, Serializer};

    pub fn serialize<S>(id: &i64, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(&id.to_string())
    }
}

/// 将 Option<i64> 序列化为 JSON string 或 null
pub mod option_id_as_string {
    use serde::{self, Serializer};

    pub fn serialize<S>(id: &Option<i64>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match id {
            Some(id) => serializer.serialize_str(&id.to_string()),
            None => serializer.serialize_none(),
        }
    }
}

/// 将 Vec<i64> 序列化为 JSON string 数组
pub mod vec_id_as_string {
    use serde::{self, Serializer};

    pub fn serialize<S>(ids: &[i64], serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        use serde::ser::SerializeSeq;
        let mut seq = serializer.serialize_seq(Some(ids.len()))?;
        for id in ids {
            seq.serialize_element(&id.to_string())?;
        }
        seq.end()
    }
}

/// 将 Option<Vec<i64>> 序列化为 JSON string 数组或 null
pub mod option_vec_id_as_string {
    use serde::{self, Serializer};

    pub fn serialize<S>(ids: &Option<Vec<i64>>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match ids {
            Some(ids) => {
                use serde::ser::SerializeSeq;
                let mut seq = serializer.serialize_seq(Some(ids.len()))?;
                for id in ids {
                    seq.serialize_element(&id.to_string())?;
                }
                seq.end()
            }
            None => serializer.serialize_none(),
        }
    }
}

// ============================================================
// 统一 API 响应格式
// ============================================================

/// 统一 API 响应格式 - API 文档: ApiResponse<T>
#[derive(Debug, Serialize)]
pub struct ApiResponse<T> {
    pub code: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<T>,
    pub request_id: String,
}

impl<T> ApiResponse<T> {
    pub fn success(data: T) -> Self {
        Self {
            code: 0,
            message: None,
            data: Some(data),
            request_id: String::new(),
        }
    }

    pub fn error(code: i32, message: String) -> Self {
        Self {
            code,
            message: Some(message),
            data: None,
            request_id: String::new(),
        }
    }
}

impl ApiResponse<()> {
    pub fn success_empty() -> Self {
        Self {
            code: 0,
            message: None,
            data: None,
            request_id: String::new(),
        }
    }
}

// ============================================================
// 通用基础类型 (API 文档定义)
// ============================================================

/// 基础坐标点 (经纬度) - API 文档: GeoPoint
#[derive(Debug, Serialize, Deserialize, Clone, Copy)]
pub struct GeoPoint {
    pub latitude: f64,
    pub longitude: f64,
}

/// 相对位置 - API 文档: RelativeLocation
/// 用途：所有 response 响应数据中的位置字段统一使用此类型
/// - 搜索/列表/Feed 场景：distance 和 bearing_degrees 有值（计算相对距离）
/// - 详情/Admin/用户自己的内容：distance 和 bearing_degrees 为 None（不需要距离）
/// 注意：如果 moment 没有位置信息，整个 location 字段应为 Option<RelativeLocation>
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct RelativeLocation {
    pub point: GeoPoint,
    pub location_name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub distance: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bearing_degrees: Option<f64>,
}

/// 消息限制 - API 文档: MessageLimit
#[derive(Debug, Serialize)]
pub struct MessageLimit {
    pub remaining: i32,
    pub total: i32,
    pub is_unlocked: bool,
}

/// 分页数据容器 - API 文档: PaginatedData<T>
/// next_cursor 为 None 时表示没有更多数据
#[derive(Debug, Serialize)]
pub struct PaginatedData<T> {
    pub items: Vec<T>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_cursor: Option<String>,
}

// ============================================================
// 用户相关类型
// ============================================================

/// 用户基础信息 - API 文档: UserInfo
/// 注意：不含 user_type（普通 API 不暴露管理员身份）
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserInfo {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
}

/// 用户关系信息 - API 文档: Relationship
/// 设计说明：不暴露 is_blocked_by（对方是否拉黑了我），被拉黑的用户不应感知到自己被拉黑
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Relationship {
    pub is_following: bool,   // 我是否关注了对方
    pub is_followed_by: bool, // 对方是否关注了我
    pub is_blocked: bool,     // 我是否拉黑了对方
}

/// 兼容别名（供迁移过渡使用）
pub type UserRelation = Relationship;

/// 社交上下文（展示与当前用户的共同点）- API 文档: UserSearchItem.social_context
#[derive(Debug, Serialize, Clone)]
pub struct SocialContext {
    pub mutual_friends_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mutual_friends_preview: Option<Vec<UserInfo>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub common_groups_count: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub common_tags: Option<Vec<String>>,
}

/// 用户统一数据类型 - API 文档: UserItem
/// 用于列表、搜索、详情、Feed 等所有场景
/// 根据是否有认证 token，返回不同字段：
/// - 基础字段（始终返回）: user, bio, tags, presence_status, last_online, location
/// - @auth-only 字段（仅登录态返回）: stats, relation, social_context
#[derive(Debug, Serialize, Clone)]
pub struct UserItem {
    pub user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    // 隐私受控字段（隐私限制时不返回，不暴露原因）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub presence_status: Option<UserPresenceStatus>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_online: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
    // @auth-only 字段（登录态返回）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub stats: Option<UserStats>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub relation: Option<Relationship>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub social_context: Option<SocialContext>,
}

/// 兼容别名（供迁移过渡使用）
pub type UserSearchItem = UserItem;

/// 用户统计数据 - API 文档: UserStats
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct UserStats {
    pub moments_count: i32,
    pub friends_count: i32,
    pub followers_count: i32,
    pub following_count: i32,
    pub groups_count: i32,
}

/// 兼容别名（供迁移过渡使用）
pub type UserCardStats = UserStats;
pub type UserStatsData = UserStats;

/// 当前用户资料 - API 文档: MyProfile
#[derive(Debug, Serialize)]
pub struct MyProfile {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub gender: Option<Gender>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub email: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub phone: Option<String>,
    pub location: RelativeLocation,
    pub is_online: bool,
    pub last_online: String,
    pub created_at: String,
    pub stats: UserStatsData,
}

/// 用户设置数据 - API 文档: UserSettings
#[derive(Debug, Serialize, Deserialize)]
pub struct UserSettingsData {
    // 隐私设置
    pub privacy_level: PrivacyLevel,
    pub location_visible: bool,
    pub online_visible: bool,
    pub last_online_visible: bool,
    pub friends_visible: bool,
    pub following_visible: bool,
    pub followers_visible: bool,
    pub groups_visible: bool,
    pub moments_visible: bool,
    pub liked_moments_visible: bool,
    pub stats_visible: bool,
    // 社交设置
    pub allow_follow: bool,
    pub allow_private_chat: PrivateChatPermission,
    // 搜索设置
    pub searchable: bool,
    pub phone_searchable: bool,
    pub nickname_searchable: bool,
    pub location_searchable: bool,
}

/// 好友列表项 - API 文档: FriendItem
#[derive(Debug, Serialize)]
pub struct FriendItem {
    pub user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>, // 隐私受控：对方隐私设置时不返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub remark: Option<String>,
    pub followed_at: String,
    pub followed_back_at: String,
}

/// 关注列表项 - API 文档: FollowingItem
#[derive(Debug, Serialize)]
pub struct FollowingItem {
    pub user: UserInfo,
    pub followed_at: String,
    pub is_mutual: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>, // 隐私受控：对方隐私设置时不返回
}

/// 粉丝列表项 - API 文档: FollowerItem
#[derive(Debug, Serialize)]
pub struct FollowerItem {
    pub user: UserInfo,
    pub followed_at: String,
    pub is_following_back: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>, // 隐私受控：对方隐私设置时不返回
}

/// 拉黑用户列表项 - API 文档: BlockedUserSearchItem
#[derive(Debug, Serialize)]
pub struct BlockedUserSearchItem {
    pub user: UserInfo,
    pub blocked_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
}

// ============================================================
// 认证相关类型
// ============================================================

/// 认证令牌数据 - API 文档: AuthTokenPairWeb/AuthTokenPairApp
#[derive(Debug, Serialize, Clone)]
pub struct AuthData {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    pub access_token: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub refresh_token: Option<String>,
    pub access_token_expires_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub refresh_token_expires_at: Option<String>,
}

/// 认证状态检查数据 - API 文档: CheckStatusData
#[derive(Debug, Serialize)]
pub struct CheckStatusData {
    pub is_valid: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[serde(serialize_with = "option_id_as_string::serialize")]
    pub user_id: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub nickname: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<String>,
}

/// 会话信息项 - API 文档: SessionItem
///
/// 安全说明：
/// - display_id: 用于展示的短标识（如 "ses_abc123"），不可用于任何操作
/// - revoke_token: 一次性撤销令牌，使用后立即失效
/// - 真实的 session_id 不对外暴露，防止批量撤销攻击
#[derive(Debug, Serialize)]
pub struct SessionItem {
    /// 展示用的会话标识（如 "ses_abc123"）
    pub display_id: String,
    /// 一次性撤销令牌，用于调用 revoke 接口
    pub revoke_token: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub device_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub device_name: Option<String>,
    /// IP 地址（已脱敏，如 "192.168.xxx.xxx"）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ip_location: Option<String>,
    pub created_at: String,
    pub last_active_at: String,
    pub is_current: bool,
}

// ============================================================
// 瞬间相关类型
// ============================================================

/// 瞬间图片 - API 文档: MomentImage
#[derive(Debug, Serialize, Clone)]
pub struct MomentImage {
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content_type: Option<String>,
}

/// 瞬间统一数据类型 - API 文档: MomentItem
/// 用于列表、搜索、详情、Feed 等所有场景
#[derive(Debug, Serialize, Clone)]
pub struct MomentItem {
    pub moment_id: String,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub images: Option<Vec<MomentImage>>,
    pub tags: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[serde(serialize_with = "option_vec_id_as_string::serialize")]
    pub mentions: Option<Vec<i64>>,
    pub created_at: String,
    pub likes_count: i32,
    pub comments_count: i32,
    pub shares_count: i32,
    pub user: UserInfo,
    pub location: RelativeLocation,
    // @owner-only（仅资源拥有者可见）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub privacy_level: Option<PrivacyLevel>,
    // @auth-only 字段（登录态返回）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_state: Option<MyMomentState>,
    // Feed 场景返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recommendation: Option<RecommendationContext>,
}

/// 兼容别名
pub type MomentSearchItem = MomentItem;

/// 我的瞬间互动状态 - API 文档: MyMomentState
#[derive(Debug, Serialize, Clone)]
pub struct MyMomentState {
    pub is_liked: bool,
    pub is_favorited: bool,
}

/// 用户瞬间列表项 - 使用 MomentItem 统一结构
pub type UserMomentListItem = MomentItem;

/// 瞬间点赞项 - API 文档: MomentLikeItem
#[derive(Debug, Serialize)]
pub struct MomentLikeItem {
    pub user: UserInfo,
    pub liked_at: String,
}

// ============================================================
// 评论相关类型
// ============================================================

/// 评论列表项 - API 文档: CommentListItem
#[derive(Debug, Serialize)]
pub struct CommentListItem {
    pub comment_id: String,
    pub user: UserInfo,
    pub content: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub parent_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[serde(serialize_with = "option_vec_id_as_string::serialize")]
    pub mentions: Option<Vec<i64>>,
    pub created_at: String,
    pub likes_count: i32,
    pub replies_count: i32,
}

/// 评论点赞项 - API 文档: CommentLikeItem
#[derive(Debug, Serialize)]
pub struct CommentLikeItem {
    pub user: UserInfo,
    pub liked_at: String,
}

// ============================================================
// 群组相关类型
// ============================================================

/// 群组基础信息 - API 文档: GroupInfo
#[derive(Debug, Serialize, Clone)]
pub struct GroupInfo {
    pub group_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub members_count: i32,
}

/// 群组统一数据类型 - API 文档: GroupItem
/// 用于列表、搜索、详情、Feed 等所有场景
#[derive(Debug, Serialize)]
pub struct GroupItem {
    pub group: GroupInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    pub has_password: bool,
    pub location: RelativeLocation,
    // 列表/详情页返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_activity_at: Option<String>,
    // 详情页返回，搜索/列表不返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub created_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub owner_id: Option<String>,
    // === @auth-only ===
    #[serde(skip_serializing_if = "Option::is_none")]
    pub my_state: Option<GroupMyStateData>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub friends_in_group: Option<i32>,
    // === @group-admin-only ===
    #[serde(skip_serializing_if = "Option::is_none")]
    pub join_method: Option<GroupJoinMethod>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_searchable: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_members: Option<i32>,
}

/// 兼容别名
pub type GroupSearchItem = GroupItem;
pub type GroupDetailData = GroupItem;

/// 我在群内的状态 - API 文档: Membership
#[derive(Debug, Serialize)]
pub struct GroupMyStateData {
    pub role: GroupRole,
    pub is_joined: bool,
    pub unread_count: i32,
    pub is_muted: bool,
    pub is_pinned: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub joined_at: Option<String>,
}

/// 群组设置 - API 文档: GroupSettings
#[derive(Debug, Serialize, Deserialize)]
pub struct GroupSettings {
    pub join_method: GroupJoinMethod,
    pub is_searchable: bool,
    pub max_members: i32,
    pub location: crate::dto::request::Location,
}

/// 已加入的群组列表项 - 使用 GroupItem 统一结构
/// MyGroupItem 现在是 GroupItem 的别名（my_state 包含 role 和 unread_count）
pub type MyGroupItem = GroupItem;

/// 群公告项 - API 文档: GroupAnnouncementItem
#[derive(Debug, Serialize)]
pub struct GroupAnnouncementItem {
    pub announcement_id: String,
    pub group_id: String,
    pub title: String,
    pub content: String,
    pub created_by: UserInfo,
    pub created_at: String,
    pub is_pinned: bool,
}

// ============================================================
// 群组成员相关类型
// ============================================================

/// 群组成员项 - API 文档: GroupMemberItem
#[derive(Debug, Serialize)]
pub struct GroupMemberItem {
    pub user: UserInfo,
    pub role: GroupRole,
    pub joined_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub muted_until: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_online: Option<bool>, // 隐私受控：成员隐私设置时不返回
}

/// 被封禁成员项 - API 文档: BannedMemberItem
#[derive(Debug, Serialize, Clone)]
pub struct BannedMemberItem {
    pub user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
    pub banned_at: String,
}

/// 邀请成员返回数据
#[derive(Debug, Serialize)]
pub struct InviteMemberData {
    pub invite_id: String,
    pub expires_at: String,
}

/// 群组申请项 - API 文档: GroupApplicationItem
#[derive(Debug, Serialize)]
pub struct GroupApplicationItem {
    pub application_id: String,
    pub applicant: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    pub status: ApplicationStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_by: Option<UserInfo>,
}

/// 群组邀请项 - API 文档: GroupInviteItem
#[derive(Debug, Serialize)]
pub struct GroupInviteItem {
    pub invite_id: String,
    pub group: GroupInfo,
    pub inviter: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    pub status: ApplicationStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<String>,
}

// ============================================================
// 私聊相关类型
// ============================================================

/// 私聊最后一条消息 - API 文档: PrivateChatLastMessage
#[derive(Debug, Serialize)]
pub struct PrivateChatLastMessage {
    pub message_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(rename = "type")]
    pub message_type: MessageType,
    pub sender: UserInfo,
    pub sent_at: String,
}

/// 私聊列表项 - API 文档: PrivateChatListItem
#[derive(Debug, Serialize)]
pub struct PrivateChatListItem {
    pub other_user: UserInfo,
    pub unread_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_message: Option<PrivateChatLastMessage>,
    pub last_activity_at: String,
    pub is_muted: bool,
    pub is_pinned: bool,
    pub is_online: bool,
}

/// 最近消息 - 使用 PaginatedData<PrivateMessageItem>
pub type RecentMessages = PaginatedData<PrivateMessageItem>;

/// 私聊详情数据 - API 文档: PrivateChatDetailData
#[derive(Debug, Serialize)]
pub struct PrivateChatDetailData {
    pub other_user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message_limit: Option<MessageLimit>,
    pub unread_count: i32,
    pub is_muted: bool,
    pub is_pinned: bool,
    pub last_activity_at: String,
    pub created_at: String,
    pub can_message: bool,
    pub recent_messages: RecentMessages,
}

// ============================================================
// 消息相关类型
// ============================================================

/// 消息回复引用 - API 文档: MessageReplyTo
#[derive(Debug, Serialize, Clone)]
pub struct MessageReplyTo {
    pub message_id: String,
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    pub content: String,
}

/// 消息媒体信息 - API 文档: MessageMedia
#[derive(Debug, Serialize, Clone, Deserialize)]
pub struct MessageMedia {
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_size: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mime_type: Option<String>,
}

/// 发送私聊消息返回数据
#[derive(Debug, Serialize)]
pub struct SendPrivateMessageData {
    pub message_id: String,
}

/// 私聊消息项 - API 文档: PrivateMessageItem
#[derive(Debug, Serialize)]
pub struct PrivateMessageItem {
    pub message_id: String,
    #[serde(rename = "type")]
    pub message_type: MessageType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<MessageReplyTo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media: Option<MessageMedia>,
    pub sent_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub edited_at: Option<String>,
    pub is_recalled: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_at: Option<String>,
}

/// 发送群聊消息返回数据
#[derive(Debug, Serialize)]
pub struct SendGroupMessageData {
    pub message_id: String,
}

/// 群聊消息项 - API 文档: GroupMessageItem
#[derive(Debug, Serialize)]
pub struct GroupMessageItem {
    pub message_id: String,
    #[serde(rename = "type")]
    pub message_type: MessageType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    pub sender: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reply_to: Option<MessageReplyTo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    #[serde(serialize_with = "option_vec_id_as_string::serialize")]
    pub mentions: Option<Vec<i64>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media: Option<MessageMedia>,
    pub sent_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub edited_at: Option<String>,
    pub is_recalled: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_by: Option<UserInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recalled_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub read_count: Option<i32>,
}

// ============================================================
// 举报相关类型
// ============================================================

/// 创建举报返回数据
#[derive(Debug, Serialize)]
pub struct CreateReportData {
    pub report_id: String,
}

/// 举报内容快照 - API 文档: ReportTargetSnapshot
#[derive(Debug, Serialize)]
pub struct ReportTargetSnapshot {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub media_urls: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub owner_id: Option<String>,
}

/// 举报项 - API 文档: ReportItem
#[derive(Debug, Serialize)]
pub struct ReportItem {
    pub report_id: String,
    pub target_type: ReportTargetType,
    pub target_id: String, // 统一为 string（user_id 也转为字符串格式）
    pub reason: ReportReason,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    pub status: ReportStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_snapshot: Option<ReportTargetSnapshot>,
}

// ============================================================
// 通知相关类型
// ============================================================

/// 瞬间摘要快照 - API 文档: MomentSummary
/// 用于通知负载中的 moment_summary 字段
#[derive(Debug, Serialize, Clone)]
pub struct MomentSummary {
    pub moment_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cover_url: Option<String>,
    pub text_preview: String,
}

/// 通知负载 - 按 type 区分的联合类型
/// 序列化为 JSON 时使用 serde(tag = "type", rename_all = "snake_case")
#[derive(Debug, Serialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum NotificationPayload {
    NewFollower {
        actor: UserInfo,
        #[serde(skip_serializing_if = "Option::is_none")]
        source: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        source_moment_id: Option<String>,
    },
    MutualFollow {
        actor: UserInfo,
    },
    MomentLiked {
        actor: UserInfo,
        moment_id: String,
        moment_summary: MomentSummary,
    },
    MomentCommented {
        actor: UserInfo,
        moment_id: String,
        moment_summary: MomentSummary,
        comment_id: String,
        comment_preview: String,
    },
    CommentLiked {
        actor: UserInfo,
        comment_id: String,
        comment_preview: String,
        moment_id: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        moment_summary: Option<MomentSummary>,
    },
    CommentReply {
        actor: UserInfo,
        comment_id: String,
        comment_preview: String,
        moment_id: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        moment_summary: Option<MomentSummary>,
    },
    Mentioned {
        actor: UserInfo,
        target_type: String,
        moment_id: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        moment_summary: Option<MomentSummary>,
        #[serde(skip_serializing_if = "Option::is_none")]
        comment_id: Option<String>,
        #[serde(skip_serializing_if = "Option::is_none")]
        comment_preview: Option<String>,
    },
    GroupInviteReceived {
        actor: UserInfo,
        group_id: String,
        group_name: String,
        invite_id: String,
    },
    GroupApplicationAccepted {
        group_id: String,
        group_name: String,
        application_id: String,
    },
    SystemAnnouncement {
        announcement_id: String,
        severity: String,
    },
}

/// 通知项 - API 文档: NotificationItem
#[derive(Debug, Serialize)]
pub struct NotificationItem {
    pub notification_id: String,
    #[serde(rename = "type")]
    pub notification_type: NotificationType,
    pub is_read: bool,
    pub created_at: String,
    pub payload: NotificationPayload,
}

/// 未读通知计数
#[derive(Debug, Serialize)]
pub struct UnreadCountResult {
    pub total: i64,
    pub by_type: UnreadCountByType,
}

#[derive(Debug, Serialize)]
pub struct UnreadCountByType {
    pub social: i64,
    pub interaction: i64,
    pub group: i64,
    pub system: i64,
}

/// 通知订阅设置 - API 文档: SubscriptionsData
#[derive(Debug, Serialize, Deserialize)]
pub struct SubscriptionsData {
    pub new_follower_enabled: bool,
    pub mutual_follow_enabled: bool,
    pub follow_request_enabled: bool, // 收到关注请求（需审批时）
    pub follow_request_handled_enabled: bool, // 关注请求被处理
    pub group_invite_enabled: bool,
    pub group_invite_response_enabled: bool,
    pub group_application_enabled: bool,
    pub group_application_response_enabled: bool,
    pub group_announcement_enabled: bool,
    pub message_mention_enabled: bool,
    pub moment_like_enabled: bool,
    pub moment_comment_enabled: bool,
    pub comment_reply_enabled: bool,
    pub system_announcement_enabled: bool,
}

// ============================================================
// 系统健康检查
// ============================================================

/// 健康检查数据 - API 文档: HealthCheckData
#[derive(Debug, Serialize)]
pub struct HealthCheckData {
    pub status: SystemHealthStatus,
    pub version: String,
    pub timestamp: String,
}

// ============================================================
// 文件上传相关类型
// ============================================================

/// 上传媒体数据
#[derive(Debug, Serialize)]
pub struct UploadMediaData {
    pub file_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_url: Option<String>,
    pub file_size: i64,
    pub mime_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub height: Option<i32>,
}

/// 预签名上传数据 - API 文档: PresignUploadData
#[derive(Debug, Serialize)]
pub struct PresignUploadData {
    pub upload_id: String,
    pub upload_url: String,
    pub headers: std::collections::HashMap<String, String>,
    pub expires_at: String,
}

// ============================================================
// 信息流相关类型
// ============================================================

/// Feed 中的群组/用户/瞬间使用统一的 GroupItem/UserItem/MomentItem
pub type FeedGroupItem = GroupItem;
pub type FeedUserItem = UserItem;
pub type FeedMomentItem = MomentItem;

/// 推荐上下文（推荐理由）- API 文档: RecommendationContext
#[derive(Debug, Serialize, Clone)]
pub struct RecommendationContext {
    #[serde(rename = "type")]
    pub recommendation_type: crate::types::enums::RecommendationType,
}

/// 热门标签项 - API 文档: TrendingTagItem
#[derive(Debug, Serialize)]
pub struct TrendingTagItem {
    pub name: String,
    pub usage_count: i32,
    pub trend: TagTrend,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hot_moment_preview: Option<FeedMomentItem>,
}

/// 信息流热门数据 - API 文档: FeedTrendingData
#[derive(Debug, Serialize)]
pub struct FeedTrendingData {
    pub trending_tags: Vec<TrendingTagItem>,
    pub hot_moments: Vec<FeedMomentItem>,
}

// ============================================================
// 标签相关类型
// ============================================================

/// 标签项 - API 文档: TagItem
#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct TagItem {
    pub name: String,
    pub usage_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trend: Option<TagTrend>,
}

/// 标签建议数据 - API 文档: TagSuggestData
#[derive(Debug, Serialize)]
pub struct TagSuggestData {
    pub items: Vec<TagSuggestItem>,
}

/// 标签建议项 - API 文档: TagSuggestItem
#[derive(Debug, Serialize)]
pub struct TagSuggestItem {
    pub name: String,
    pub usage_count: i32,
}

/// 综合搜索建议数据 - API 文档: SearchSuggestions
#[derive(Debug, Serialize)]
pub struct SearchSuggestions {
    pub items: Vec<SearchSuggestItem>,
}

/// 综合搜索建议项 - API 规范: @api_spec_search.mdc SearchSuggestItem
#[derive(Debug, Serialize)]
#[serde(tag = "type", content = "payload", rename_all = "snake_case")]
pub enum SearchSuggestItem {
    Tag(TagSuggestItem),
    User(UserSuggestItem),
    Group(GroupSuggestItem),
    Moment(MomentSuggestItem),
}

/// 用户建议项 - API 规范: @api_spec_search.mdc UserSuggestItem
/// 用途：显示用户基本信息，点击后跳转到用户主页
#[derive(Debug, Serialize)]
pub struct UserSuggestItem {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    /// 个性签名，用于副标题展示
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    /// 距离（米），仅当请求包含 location 时返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub distance: Option<f64>,
}

/// 群组建议项 - API 规范: @api_spec_search.mdc GroupSuggestItem
/// 用途：显示群组基本信息，点击后跳转到群组页
#[derive(Debug, Serialize)]
pub struct GroupSuggestItem {
    pub group_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    /// 群组简介，用于副标题展示（优先于成员数）
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    /// 成员数量，当无 description 时显示"X 位成员"
    pub members_count: i32,
    /// 距离（米），仅当请求包含 location 时返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub distance: Option<f64>,
}

/// 瞬间建议项 - API 规范: @api_spec_search.mdc MomentSuggestItem
/// 用途：显示瞬间预览信息，点击后跳转到瞬间详情页
#[derive(Debug, Serialize)]
pub struct MomentSuggestItem {
    pub moment_id: String,
    /// 瞬间文字内容（后端建议限制长度，如前100字符）
    pub content: String,
    /// 作者昵称，用于显示"来自 XXX"
    pub author_nickname: String,
    /// 作者头像，用于显示作者头像而非瞬间图片
    #[serde(skip_serializing_if = "Option::is_none")]
    pub author_avatar_url: Option<String>,
    /// 距离（米），仅当请求包含 location 时返回
    #[serde(skip_serializing_if = "Option::is_none")]
    pub distance: Option<f64>,
}

// ============================================================
// 系统配置相关类型
// ============================================================

/// 上传配置 - API 文档: UploadConfig
#[derive(Debug, Serialize, Deserialize)]
pub struct UploadConfig {
    pub image_max_size: i64,
    pub video_max_size: i64,
    pub audio_max_size: i64,
    pub file_max_size: i64,
    pub allowed_image_types: Vec<String>,
    pub allowed_video_types: Vec<String>,
}

/// 默认位置 - API 文档: DefaultLocation (对应 spec 中的 Location 类型)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DefaultLocation {
    pub point: GeoPoint,
    pub location_name: String,
}

/// 内容默认值 - API 文档: ContentDefaults
#[derive(Debug, Serialize, Deserialize)]
pub struct ContentDefaults {
    pub default_location: DefaultLocation,
    pub default_group_avatar: String,
    pub default_user_avatar: String,
    pub default_group_settings: GroupSettings,
}

/// 功能开关 - API 文档: FeatureSwitches
#[derive(Debug, Serialize, Deserialize)]
pub struct FeatureSwitches {
    pub enable_public_registration: bool,
    pub enable_guest_login: bool,
    pub enable_moments: bool,
    pub enable_nearby_people: bool,
}

/// 联系信息 - API 文档: ContactInfo
#[derive(Debug, Serialize, Deserialize)]
pub struct ContactInfo {
    pub support_email: String,
    pub privacy_policy_url: String,
    pub terms_of_service_url: String,
    pub official_website: String,
}

/// 应用限制 - API 文档: AppLimits
#[derive(Debug, Serialize, Deserialize)]
pub struct AppLimits {
    pub max_group_members: i32,
    pub max_moment_tags: i32,
    pub max_moment_images: i32,
    pub moment_description_max_length: i32,
    pub comment_max_length: i32,
    pub search_radius_max: i32,
}

/// 频率限制配置 - API 文档: RateLimitConfig
#[derive(Debug, Serialize, Deserialize)]
pub struct RateLimitConfig {
    // IP级限速（单IP限制，考虑多用户共享同一IP的场景）
    pub ip_per_hour: u32,
    // 用户级限流
    pub user_per_5minutes: u32,
    pub user_per_hour: u32,
    // 认证相关限速
    pub auth_send_code_per_minute: u32,
    pub auth_verify_code_per_10minutes: u32,
    pub auth_register_per_10minutes: u32,
    pub auth_login_per_10minutes: u32,
    pub auth_guest_per_10minutes: u32,
    pub auth_refresh_per_10minutes: u32,
    pub auth_reset_request_per_minute: u32,
    pub auth_reset_verify_per_10minutes: u32,
    pub auth_reset_per_10minutes: u32,
    // 业务相关限速
    pub moments_create_per_minute: u32,
    pub comments_create_per_minute: u32,
    pub reports_create_per_hour: u32,
}

/// 全局设置 - API 文档: GlobalSettings
#[derive(Debug, Serialize, Deserialize)]
pub struct GlobalSettings {
    pub upload: UploadConfig,
    pub defaults: ContentDefaults,
    pub features: FeatureSwitches,
    pub contact: ContactInfo,
    pub limits: AppLimits,
    pub rate_limits: RateLimitConfig,
}

/// 管理员全局设置数据 - API 文档: AdminGlobalSettingsData
#[derive(Debug, Serialize)]
pub struct AdminGlobalSettingsData {
    #[serde(flatten)]
    pub settings: GlobalSettings,
    pub updated_at: String,
    #[serde(serialize_with = "id_as_string::serialize")]
    pub updated_by_user_id: i64,
}

/// 更新上传策略响应
#[derive(Debug, Serialize)]
pub struct UpdateUploadConfigData {
    pub upload: UploadConfig,
    pub updated_at: String,
}

/// 更新内容默认值响应
#[derive(Debug, Serialize)]
pub struct UpdateDefaultsConfigData {
    pub defaults: ContentDefaults,
    pub updated_at: String,
}

/// 更新功能开关响应
#[derive(Debug, Serialize)]
pub struct UpdateFeaturesConfigData {
    pub features: FeatureSwitches,
    pub updated_at: String,
}

/// 更新联系信息响应
#[derive(Debug, Serialize)]
pub struct UpdateContactConfigData {
    pub contact: ContactInfo,
    pub updated_at: String,
}

/// 更新系统限制响应
#[derive(Debug, Serialize)]
pub struct UpdateLimitsConfigData {
    pub limits: AppLimits,
    pub updated_at: String,
}

/// 更新频率限制响应
#[derive(Debug, Serialize)]
pub struct UpdateRateLimitsConfigData {
    pub rate_limits: RateLimitConfig,
    pub updated_at: String,
}

// ============================================================
// 管理员 RBAC 相关类型
// ============================================================

/// 管理员权限项 - API 文档: AdminPermissionItem
#[derive(Debug, Serialize)]
pub struct AdminPermissionItem {
    pub key: String,
    pub description: String,
}

/// 管理员权限数据
#[derive(Debug, Serialize)]
pub struct AdminPermissionsData {
    pub items: Vec<AdminPermissionItem>,
}

/// 管理员角色项 - API 文档: AdminRoleItem
#[derive(Debug, Serialize)]
pub struct AdminRoleItem {
    pub role_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    pub permissions: Vec<String>,
}

/// 管理员角色数据
#[derive(Debug, Serialize)]
pub struct AdminRolesData {
    pub items: Vec<AdminRoleItem>,
}

/// 管理员角色基础信息 - API 文档: AdminRoleBasic
#[derive(Debug, Serialize)]
pub struct AdminRoleBasic {
    pub role_id: String,
    pub name: String,
}

/// 管理员项 - API 文档: AdminAdminItem
#[derive(Debug, Serialize)]
pub struct AdminAdminItem {
    pub user: UserInfo,
    pub permissions: Vec<String>,
    pub roles: Vec<AdminRoleBasic>,
}

/// 管理员列表数据
#[derive(Debug, Serialize)]
pub struct AdminAdminsData {
    pub items: Vec<AdminAdminItem>,
}

/// 当前管理员数据 - API 文档: AdminMeData
#[derive(Debug, Serialize)]
pub struct AdminMeData {
    pub permissions: Vec<String>,
    pub roles: Vec<AdminRoleBasic>,
}

// ============================================================
// 管理员用户管理相关类型
// ============================================================

/// 管理员用户项 - API 文档: AdminUserSearchItem
#[derive(Debug, Serialize)]
pub struct AdminUserSearchItem {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: UserType,
    pub status: UserAccountStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_active_at: Option<String>,
    pub login_id: String,
    pub is_online: bool,
}

/// 管理员用户详情 - API 文档: AdminUserDetail
#[derive(Debug, Serialize)]
pub struct AdminUserDetail {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: UserType,
    pub status: UserAccountStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_active_at: Option<String>,
    pub login_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub email: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub phone: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
    pub is_online: bool,
    pub stats: UserCardStats,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ban_history: Option<Vec<AdminBanRecord>>,
}

/// 管理员封禁记录 - API 文档: AdminBanRecord
#[derive(Debug, Serialize, Clone)]
pub struct AdminBanRecord {
    pub banned_at: String,
    pub banned_by: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration_days: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unbanned_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unbanned_by: Option<UserInfo>,
}

// ============================================================
// 管理员群组管理相关类型
// ============================================================

/// 管理员群组项 - API 文档: AdminGroupSearchItem
#[derive(Debug, Serialize)]
pub struct AdminGroupSearchItem {
    pub group_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub members_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
    pub owner: UserInfo,
    pub created_at: String,
}

/// 管理员群组详情 - API 文档: AdminGroupDetail
#[derive(Debug, Serialize)]
pub struct AdminGroupDetail {
    pub group_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub members_count: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    pub max_members: i32,
    pub owner: UserInfo,
    pub created_at: String,
    pub settings: AdminGroupSettingsSummary,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub admins: Option<Vec<UserInfo>>,
    pub stats: AdminGroupStats,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub dismissal_history: Option<Vec<AdminDismissalRecord>>,
}

#[derive(Debug, Serialize)]
pub struct AdminGroupSettingsSummary {
    pub join_method: GroupJoinMethod,
    pub is_searchable: bool,
    pub has_password: bool,
}

#[derive(Debug, Serialize)]
pub struct AdminGroupStats {
    pub messages_count: i32,
    pub active_members_today: i32,
}

#[derive(Debug, Serialize, Clone)]
pub struct AdminDismissalRecord {
    pub dismissed_at: String,
    pub dismissed_by: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
}

// ============================================================
// 管理员审计日志相关类型
// ============================================================

/// 审计日志项 - API 文档: AuditLogItem
#[derive(Debug, Serialize, Clone)]
pub struct AuditLogItem {
    pub log_id: String,
    pub action: String,
    pub operator: UserInfo,
    pub target_type: String,
    pub target_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_agent: Option<String>,
    pub created_at: String,
}

/// 审计日志详情 - API 文档: AuditLogDetail
#[derive(Debug, Serialize)]
pub struct AuditLogDetail {
    pub log_id: String,
    pub action: String,
    pub operator: UserInfo,
    pub target_type: String,
    pub target_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_agent: Option<String>,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub before_state: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub after_state: Option<Value>,
}

// ============================================================
// 管理员系统公告相关类型
// ============================================================

/// 创建公告返回数据
#[derive(Debug, Serialize)]
pub struct CreateAnnouncementData {
    pub announcement_id: String,
    pub created_at: String,
}

/// 系统公告项 - API 文档: SystemAnnouncementItem
#[derive(Debug, Serialize)]
pub struct SystemAnnouncementItem {
    pub announcement_id: String,
    pub title: String,
    pub content: String,
    pub severity: AnnouncementSeverity,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub action_url: Option<String>,
    pub created_at: String,
}

/// 公开系统公告列表响应（非分页，返回条数由服务端 ResponseDefaults 控制）
#[derive(Debug, Serialize)]
pub struct SystemAnnouncementListData {
    pub items: Vec<SystemAnnouncementItem>,
}

// ============================================================
// 地图渲染相关类型
// ============================================================

/// 地图点 - API 文档: GeoPoint
#[derive(Debug, Serialize, Clone)]
pub struct MapPoint {
    pub latitude: f64,
    pub longitude: f64,
}

/// 地图聚合数据 - API 文档: MapClusterData
#[derive(Debug, Serialize)]
pub struct MapClusterData {
    pub count: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub label: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub preview_urls: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub boundary: Option<Vec<MapPoint>>,
}

/// 地图实体数据 - API 文档: MapEntityData
#[derive(Debug, Serialize)]
pub struct MapEntityData {
    pub title: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub subtitle: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cover_url: Option<String>,
    pub payload_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_highlighted: Option<bool>,
}

/// 地图渲染项 - API 文档: MapRenderItem
#[derive(Debug, Serialize)]
pub struct MapRenderItem {
    pub id: String,
    #[serde(rename = "type")]
    pub item_type: MapItemType,
    pub location: MapPoint,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub cluster: Option<MapClusterData>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub entity: Option<MapEntityData>,
}

/// 地图渲染数据 - API 文档: MapRenderData
#[derive(Debug, Serialize)]
pub struct MapRenderData {
    pub items: Vec<MapRenderItem>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub toast_message: Option<String>,
}

// ============================================================
// 位置分享相关响应
// ============================================================

// ==================== 分享模块响应 ====================

/// 分享信息 - API 文档: Share
#[derive(Debug, Serialize, Clone)]
pub struct Share {
    pub share_id: String,
    pub share_code: String,
    pub share_type: ShareType,
    pub creator: UserInfo,
    pub payload: serde_json::Value,
    pub created_at: String,
}
