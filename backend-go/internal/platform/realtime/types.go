package realtime

// 连接管理
const (
	EventTypeConnectionEstablished = "ConnectionEstablished"
	EventTypePong                  = "Pong"
	EventTypeError                 = "Error"
	EventTypeSubscribeAck          = "SubscribeAck"
	EventTypeSubscriptionsSnapshot = "SubscriptionsSnapshot"
)

// 用户状态
const (
	EventTypeUserOnline  = "UserOnline"
	EventTypeUserOffline = "UserOffline"
)

// 社交
const (
	EventTypeNewFollower  = "NewFollower"
	EventTypeMutualFollow = "MutualFollow"
)

// 瞬间互动
const (
	EventTypeMomentLiked      = "MomentLiked"
	EventTypeMomentCommented  = "MomentCommented"
	EventTypeCommentLiked     = "CommentLiked"
	EventTypeCommentReply     = "CommentReply"
	EventTypeMessageMentioned = "MessageMentioned"
)

// 私聊
const (
	EventTypePrivateMessageNew      = "PrivateMessageNew"
	EventTypePrivateMessageEdited   = "PrivateMessageEdited"
	EventTypePrivateMessageRecalled = "PrivateMessageRecalled"
	EventTypePrivateMessageRead     = "PrivateMessageRead"
	EventTypePrivateUserTyping      = "PrivateUserTyping"
)

// 群聊
const (
	EventTypeGroupMessageNew      = "GroupMessageNew"
	EventTypeGroupMessageEdited   = "GroupMessageEdited"
	EventTypeGroupMessageRecalled = "GroupMessageRecalled"
	EventTypeGroupUserTyping      = "GroupUserTyping"
)

// 群组管理
const (
	EventTypeGroupMemberJoined        = "GroupMemberJoined"
	EventTypeGroupMemberLeft          = "GroupMemberLeft"
	EventTypeGroupMemberRemoved       = "GroupMemberRemoved"
	EventTypeGroupMemberMuted         = "GroupMemberMuted"
	EventTypeGroupMemberUnmuted       = "GroupMemberUnmuted"
	EventTypeGroupInviteReceived      = "GroupInviteReceived"
	EventTypeGroupInviteResponse      = "GroupInviteResponse"
	EventTypeGroupApplicationReceived = "GroupApplicationReceived"
	EventTypeGroupApplicationResponse = "GroupApplicationResponse"
	EventTypeGroupAnnouncementCreated = "GroupAnnouncementCreated"
	EventTypeGroupUpdated             = "GroupUpdated"
	EventTypeGroupDismissed           = "GroupDismissed"
)

// 用户
const EventTypeUserProfileUpdated = "UserProfileUpdated"

// 系统
const (
	EventTypeSystemAnnouncement = "SystemAnnouncement"
	EventTypeAccountBanned      = "AccountBanned"
)

// 多端同步（Deferred）
const (
	EventTypeSelfReadSync         = "SelfReadSync"
	EventTypeSelfChatSettingsSync = "SelfChatSettingsSync"
	EventTypeSelfSettingsUpdated  = "SelfSettingsUpdated"
	EventTypeSelfRelationSync     = "SelfRelationSync"
)

type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type ConnectionEstablishedPayload struct {
	ConnectionID string `json:"connection_id"`
	UserID       string `json:"user_id"`
	ServerTime   string `json:"server_time"`
	Config       struct {
		PingInterval     int `json:"ping_interval"`
		PingTimeout      int `json:"ping_timeout"`
		MaxSubscriptions struct {
			User  int `json:"user"`
			Group int `json:"group"`
		} `json:"max_subscriptions"`
	} `json:"config"`
}

type ErrorPayload struct {
	Code       int               `json:"code"`
	Message    string            `json:"message"`
	Severity   string            `json:"severity"`
	Retryable  bool              `json:"retryable"`
	RetryAfter int               `json:"retry_after,omitempty"`
	Context    map[string]string `json:"context,omitempty"`
}

type SubscribeAckPayload struct {
	ClientRequestID string               `json:"client_request_id"`
	Operation       string               `json:"operation"`
	Type            string               `json:"type"`
	IDs             []string             `json:"ids"`
	Code            int                  `json:"code"`
	Message         string               `json:"message,omitempty"`
	FailedIDs       []FailedSubscription `json:"failed_ids,omitempty"`
}

type FailedSubscription struct {
	SubscriptionID string `json:"subscription_id"`
	Reason         string `json:"reason"`
}

type SubscriptionsSnapshotPayload struct {
	ClientRequestID string   `json:"client_request_id,omitempty"`
	UserIDs         []string `json:"user_ids,omitempty"`
	GroupIDs        []string `json:"group_ids,omitempty"`
}

type AuthPayload struct {
	ClientType  string `json:"client_type,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
}

type PingPayload struct{}

type SetSubscriptionsPayload struct {
	ClientRequestID string   `json:"client_request_id"`
	Operation       string   `json:"operation"`
	Type            string   `json:"type"`
	IDs             []string `json:"ids"`
}

type GetSubscriptionsPayload struct {
	ClientRequestID string `json:"client_request_id,omitempty"`
}

type UserOnlinePayload struct {
	UserID   string `json:"user_id"`
	OnlineAt string `json:"online_at"`
}

type UserOfflinePayload struct {
	UserID    string `json:"user_id"`
	OfflineAt string `json:"offline_at"`
}

type PrivateTypingPayload struct {
	TargetUserID string `json:"target_user_id"`
	IsTyping     bool   `json:"is_typing"`
}

type GroupTypingPayload struct {
	GroupID  string `json:"group_id"`
	IsTyping bool   `json:"is_typing"`
}
