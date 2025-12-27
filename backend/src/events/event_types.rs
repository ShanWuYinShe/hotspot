use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// 应用事件类型
/// 严格对应 API 文档中定义的 WebSocket 事件
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", content = "data")]
pub enum AppEvent {
    // ========== 用户相关事件 ==========
    UserRegistered {
        user_id: i64,
    },
    UserProfileUpdated {
        user_id: i64,
    },
    /// 用户位置更新
    UserLocationUpdated {
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: String,
    },

    // ========== 关注关系事件（好友 = 互相关注）==========
    NewFollower {
        follower_id: i64,
        user_id: i64,
        is_mutual: bool,
    },
    MutualFollow {
        user_id: i64,
        mutual_user_id: i64,
    },
    Unfollowed {
        follower_id: i64,
        user_id: i64,
        was_mutual: bool,
    },

    // ========== 群组管理事件 ==========
    GroupCreated {
        group_id: Uuid,
        creator_id: i64,
    },
    GroupMemberAdded {
        group_id: Uuid,
        user_id: i64,
        added_by: i64,
    },
    GroupMemberRemoved {
        group_id: Uuid,
        user_id: i64,
        removed_by: i64,
    },
    /// 群组资料更新
    GroupProfileUpdated {
        group_id: Uuid,
        name: Option<String>,
        description: Option<String>,
        avatar_url: Option<String>,
    },
    /// 成员角色变更
    GroupMemberRoleChanged {
        group_id: Uuid,
        user_id: i64,
        old_role: String,
        new_role: String,
    },
    /// 成员被禁言
    GroupMemberMuted {
        group_id: Uuid,
        user_id: i64,
        muted_until: String,
    },
    /// 成员取消禁言
    GroupMemberUnmuted {
        group_id: Uuid,
        user_id: i64,
    },
    /// 群组解散
    GroupDismissed {
        group_id: Uuid,
        dismissed_by: i64,
    },
    /// 群组邀请发送
    GroupInviteSent {
        invite_id: Uuid,
        group_id: Uuid,
        group_name: String,
        from_user_id: i64,
        to_user_id: i64,
        message: Option<String>,
    },
    /// 群组邀请响应
    GroupInviteResponse {
        invite_id: Uuid,
        group_id: Uuid,
        inviter_id: i64,
        invitee_id: i64,
        status: String, // "accepted" | "rejected"
    },
    /// 收到群组申请（通知管理员）
    GroupApplicationReceived {
        application_id: Uuid,
        group_id: Uuid,
        applicant_id: i64,
        message: Option<String>,
    },
    /// 群组申请响应（通知申请者）
    GroupApplicationResponse {
        application_id: Uuid,
        group_id: Uuid,
        group_name: String,
        applicant_id: i64,
        status: String, // "accepted" | "rejected"
    },

    // ========== 群聊消息事件 ==========
    GroupMessageSent {
        message_id: Uuid,
        group_chat_id: Uuid,
        sender_id: i64,
        message_type: String,
        content: Option<String>,
        reply_to: Option<Uuid>,
        mentions: Option<Vec<i64>>,
        media_url: Option<String>,
        thumbnail_url: Option<String>,
        file_name: Option<String>,
        file_size: Option<i64>,
        duration: Option<i32>,
        width: Option<i32>,
        height: Option<i32>,
    },
    GroupMessageEdited {
        message_id: Uuid,
        group_chat_id: Uuid,
        content: String,
        edited_at: String,
    },
    GroupMessageRecalled {
        message_id: Uuid,
        group_chat_id: Uuid,
        recalled_by: i64,
        recalled_at: String,
    },
    GroupMessageDeleted {
        message_id: Uuid,
        group_chat_id: Uuid,
        deleted_by: i64,
        for_everyone: bool,
    },
    GroupMessageRead {
        group_chat_id: Uuid,
        reader_id: i64,
        up_to: Uuid,
        read_at: String,
    },
    GroupMessageReaction {
        message_id: Uuid,
        group_chat_id: Uuid,
        user_id: i64,
        reaction_type: String,
        action: String, // "add" | "remove"
    },
    GroupUserTyping {
        group_chat_id: Uuid,
        user_id: i64,
        is_typing: bool,
    },

    // ========== 私聊消息事件 ==========
    PrivateMessageSent {
        message_id: Uuid,
        private_chat_id: Uuid,
        sender_id: i64,
        recipient_id: i64,
        message_type: String,
        content: Option<String>,
        reply_to: Option<Uuid>,
        media_url: Option<String>,
        thumbnail_url: Option<String>,
        file_name: Option<String>,
        file_size: Option<i64>,
        duration: Option<i32>,
        width: Option<i32>,
        height: Option<i32>,
    },
    PrivateMessageEdited {
        message_id: Uuid,
        private_chat_id: Uuid,
        sender_id: i64,
        recipient_id: i64,
        content: String,
        edited_at: String,
    },
    PrivateMessageRecalled {
        message_id: Uuid,
        private_chat_id: Uuid,
        sender_id: i64,
        recipient_id: i64,
        recalled_by: i64,
        recalled_at: String,
    },
    PrivateMessageDeleted {
        message_id: Uuid,
        private_chat_id: Uuid,
        deleted_by: i64,
    },
    PrivateMessageRead {
        private_chat_id: Uuid,
        sender_id: i64, // 被读消息的发送者
        reader_id: i64,
        up_to: Uuid,
        read_at: String,
    },
    PrivateMessageReaction {
        message_id: Uuid,
        private_chat_id: Uuid,
        other_user_id: i64, // 私聊对方
        user_id: i64,
        reaction_type: String,
        action: String, // "add" | "remove"
    },
    PrivateUserTyping {
        private_chat_id: Uuid,
        other_user_id: i64, // 私聊对方
        user_id: i64,
        is_typing: bool,
    },

    // ========== 瞬间相关事件 ==========
    MomentCreated {
        moment_id: Uuid,
        user_id: i64,
    },
    MomentLiked {
        moment_id: Uuid,
        user_id: i64,
        liker_id: i64,
    },
    MomentCommented {
        moment_id: Uuid,
        comment_id: Uuid,
        user_id: i64, // 瞬间作者
        commenter_id: i64,
        content: String,
    },

    // ========== 评论相关事件 ==========
    CommentAdded {
        comment_id: Uuid,
        target_type: String,
        target_id: String,
        user_id: i64,
    },
    CommentLiked {
        comment_id: Uuid,
        user_id: i64,
        liker_id: i64,
    },
    /// 评论被回复
    CommentReplied {
        moment_id: Uuid,
        comment_id: Uuid, // 原评论ID
        reply_id: Uuid,   // 回复评论ID
        user_id: i64,     // 原评论作者
        replier_id: i64,  // 回复者
        content: String,
    },

    // ========== 系统事件 ==========
    SystemNotification {
        user_id: i64,
        title: String,
        content: String,
    },
}

impl AppEvent {
    /// 获取事件类型名称
    pub fn event_type(&self) -> &str {
        match self {
            // 用户相关
            AppEvent::UserRegistered { .. } => "user_registered",
            AppEvent::UserProfileUpdated { .. } => "user_profile_updated",
            AppEvent::UserLocationUpdated { .. } => "user_location_updated",
            // 关注关系
            AppEvent::NewFollower { .. } => "new_follower",
            AppEvent::MutualFollow { .. } => "mutual_follow",
            AppEvent::Unfollowed { .. } => "unfollowed",
            // 群组管理
            AppEvent::GroupCreated { .. } => "group_created",
            AppEvent::GroupMemberAdded { .. } => "group_member_added",
            AppEvent::GroupMemberRemoved { .. } => "group_member_removed",
            AppEvent::GroupProfileUpdated { .. } => "group_profile_updated",
            AppEvent::GroupMemberRoleChanged { .. } => "group_member_role_changed",
            AppEvent::GroupMemberMuted { .. } => "group_member_muted",
            AppEvent::GroupMemberUnmuted { .. } => "group_member_unmuted",
            AppEvent::GroupDismissed { .. } => "group_dismissed",
            AppEvent::GroupInviteSent { .. } => "group_invite_sent",
            AppEvent::GroupInviteResponse { .. } => "group_invite_response",
            AppEvent::GroupApplicationReceived { .. } => "group_application_received",
            AppEvent::GroupApplicationResponse { .. } => "group_application_response",
            // 群聊消息
            AppEvent::GroupMessageSent { .. } => "group_message_sent",
            AppEvent::GroupMessageEdited { .. } => "group_message_edited",
            AppEvent::GroupMessageRecalled { .. } => "group_message_recalled",
            AppEvent::GroupMessageDeleted { .. } => "group_message_deleted",
            AppEvent::GroupMessageRead { .. } => "group_message_read",
            AppEvent::GroupMessageReaction { .. } => "group_message_reaction",
            AppEvent::GroupUserTyping { .. } => "group_user_typing",
            // 私聊消息
            AppEvent::PrivateMessageSent { .. } => "private_message_sent",
            AppEvent::PrivateMessageEdited { .. } => "private_message_edited",
            AppEvent::PrivateMessageRecalled { .. } => "private_message_recalled",
            AppEvent::PrivateMessageDeleted { .. } => "private_message_deleted",
            AppEvent::PrivateMessageRead { .. } => "private_message_read",
            AppEvent::PrivateMessageReaction { .. } => "private_message_reaction",
            AppEvent::PrivateUserTyping { .. } => "private_user_typing",
            // 瞬间相关
            AppEvent::MomentCreated { .. } => "moment_created",
            AppEvent::MomentLiked { .. } => "moment_liked",
            AppEvent::MomentCommented { .. } => "moment_commented",
            // 评论相关
            AppEvent::CommentAdded { .. } => "comment_added",
            AppEvent::CommentLiked { .. } => "comment_liked",
            AppEvent::CommentReplied { .. } => "comment_replied",
            // 系统事件
            AppEvent::SystemNotification { .. } => "system_notification",
        }
    }
}
