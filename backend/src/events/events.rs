use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// 应用事件类型
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", content = "data")]
pub enum AppEvent {
    // 用户相关事件
    UserRegistered { user_id: i64 },
    UserProfileUpdated { user_id: i64 },
    
    // 好友相关事件
    FriendRequestSent { from_user_id: i64, to_user_id: i64 },
    FriendRequestAccepted { from_user_id: i64, to_user_id: i64 },
    FriendRequestRejected { from_user_id: i64, to_user_id: i64 },
    FriendRemoved { user_id: i64, friend_id: i64 },
    
    // 群组相关事件
    GroupCreated { group_id: Uuid, creator_id: i64 },
    GroupMemberAdded { group_id: Uuid, user_id: i64, added_by: i64 },
    GroupMemberRemoved { group_id: Uuid, user_id: i64, removed_by: i64 },
    GroupInviteSent { group_id: Uuid, from_user_id: i64, to_user_id: i64 },
    
    // 消息相关事件
    MessageSent { 
        message_id: Uuid, 
        group_chat_id: Uuid,
        sender_id: i64, 
        content: String,
        message_type: String,
    },
    MessageEdited { 
        message_id: Uuid,
        group_chat_id: Uuid,
        content: String,
        edited_at: String,
    },
    MessageRecalled { 
        message_id: Uuid,
        group_chat_id: Uuid,
        recalled_by: i64,
        recalled_at: String,
    },
    MessageDeleted {
        message_id: Uuid,
        group_chat_id: Uuid,
        deleted_by: i64,
        for_everyone: bool,
    },
    MessageRead { 
        group_chat_id: Uuid,
        reader_id: i64,
        up_to: Uuid,
        read_at: String,
    },
    MessageReaction {
        message_id: Uuid,
        group_chat_id: Uuid,
        user_id: i64,
        reaction_type: String,
        action: String,
    },
    UserTyping {
        group_chat_id: Uuid,
        user_id: i64,
        is_typing: bool,
    },
    
    // 瞬间相关事件
    MomentCreated { moment_id: Uuid, user_id: i64 },
    MomentLiked { moment_id: Uuid, user_id: i64, liker_id: i64 },
    MomentCommented { moment_id: Uuid, comment_id: Uuid, user_id: i64, commenter_id: i64 },
    
    // 评论相关事件
    CommentAdded { comment_id: Uuid, target_type: String, target_id: String, user_id: i64 },
    CommentLiked { comment_id: Uuid, user_id: i64, liker_id: i64 },
    
    // 系统事件
    SystemNotification { user_id: i64, title: String, content: String },
}

impl AppEvent {
    /// 获取事件涉及的用户ID列表
    pub fn get_affected_users(&self) -> Vec<i64> {
        match self {
            AppEvent::UserRegistered { user_id } => vec![*user_id],
            AppEvent::UserProfileUpdated { user_id } => vec![*user_id],
            AppEvent::FriendRequestSent { from_user_id, to_user_id } => vec![*from_user_id, *to_user_id],
            AppEvent::FriendRequestAccepted { from_user_id, to_user_id } => vec![*from_user_id, *to_user_id],
            AppEvent::FriendRequestRejected { from_user_id, to_user_id } => vec![*from_user_id, *to_user_id],
            AppEvent::FriendRemoved { user_id, friend_id } => vec![*user_id, *friend_id],
            AppEvent::GroupMemberAdded { user_id, .. } => vec![*user_id],
            AppEvent::GroupMemberRemoved { user_id, .. } => vec![*user_id],
            AppEvent::GroupInviteSent { to_user_id, .. } => vec![*to_user_id],
            AppEvent::MessageSent { sender_id, .. } => vec![*sender_id],
            AppEvent::MessageEdited { .. } => vec![],
            AppEvent::MessageRecalled { recalled_by, .. } => vec![*recalled_by],
            AppEvent::MessageDeleted { deleted_by, .. } => vec![*deleted_by],
            AppEvent::MessageRead { reader_id, .. } => vec![*reader_id],
            AppEvent::MessageReaction { user_id, .. } => vec![*user_id],
            AppEvent::UserTyping { user_id, .. } => vec![*user_id],
            AppEvent::MomentCreated { user_id, .. } => vec![*user_id],
            AppEvent::MomentLiked { user_id, .. } => vec![*user_id],
            AppEvent::MomentCommented { user_id, .. } => vec![*user_id],
            AppEvent::CommentAdded { user_id, .. } => vec![*user_id],
            AppEvent::CommentLiked { user_id, .. } => vec![*user_id],
            AppEvent::SystemNotification { user_id, .. } => vec![*user_id],
            _ => vec![],
        }
    }

    /// 获取事件类型名称
    pub fn event_type(&self) -> &str {
        match self {
            AppEvent::UserRegistered { .. } => "user_registered",
            AppEvent::UserProfileUpdated { .. } => "user_profile_updated",
            AppEvent::FriendRequestSent { .. } => "friend_request_sent",
            AppEvent::FriendRequestAccepted { .. } => "friend_request_accepted",
            AppEvent::FriendRequestRejected { .. } => "friend_request_rejected",
            AppEvent::FriendRemoved { .. } => "friend_removed",
            AppEvent::GroupCreated { .. } => "group_created",
            AppEvent::GroupMemberAdded { .. } => "group_member_added",
            AppEvent::GroupMemberRemoved { .. } => "group_member_removed",
            AppEvent::GroupInviteSent { .. } => "group_invite_sent",
            AppEvent::MessageSent { .. } => "message_sent",
            AppEvent::MessageEdited { .. } => "message_edited",
            AppEvent::MessageRecalled { .. } => "message_recalled",
            AppEvent::MessageDeleted { .. } => "message_deleted",
            AppEvent::MessageRead { .. } => "message_read",
            AppEvent::MessageReaction { .. } => "message_reaction",
            AppEvent::UserTyping { .. } => "user_typing",
            AppEvent::MomentCreated { .. } => "moment_created",
            AppEvent::MomentLiked { .. } => "moment_liked",
            AppEvent::MomentCommented { .. } => "moment_commented",
            AppEvent::CommentAdded { .. } => "comment_added",
            AppEvent::CommentLiked { .. } => "comment_liked",
            AppEvent::SystemNotification { .. } => "system_notification",
        }
    }
}

