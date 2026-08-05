package realtime

// 连接管理
const (
	EventTypeConnectionEstablished = "ConnectionEstablished"
	EventTypePong                  = "Pong"
	EventTypeError                 = "Error"
	EventTypeSubscribeAck          = "SubscribeAck"
	EventTypeSubscriptionsSnapshot = "SubscriptionsSnapshot"
	EventTypeSessionRevoked        = "SessionRevoked"
)

// 用户状态
const (
	EventTypeUserOnline  = "UserOnline"
	EventTypeUserOffline = "UserOffline"
)

// 社交
const (
	EventTypeNewFollower     = "NewFollower"
	EventTypeMutualFollow    = "MutualFollow"
	EventTypeNearbyNewMoment = "NearbyNewMoment"
)

// 瞬间互动
const (
	EventTypeMomentCommented  = "MomentCommented"
	EventTypeCommentReply     = "CommentReply"
	EventTypeMessageMentioned = "MessageMentioned"
)

// 瞬间生命周期（编辑/软删除，spec api_spec_websocket.md）
const (
	EventTypeMomentUpdated = "MomentUpdated"
	EventTypeMomentDeleted = "MomentDeleted"
)

// 私聊
const (
	EventTypePrivateMessageNew      = "PrivateMessageNew"
	EventTypePrivateMessageEdited   = "PrivateMessageEdited"
	EventTypePrivateMessageRecalled = "PrivateMessageRecalled"
	EventTypePrivateMessageRead     = "PrivateMessageRead"
	EventTypePrivateTyping          = "PrivateTyping"
)

// 群聊
const (
	EventTypeGroupMessageNew              = "GroupMessageNew"
	EventTypeGroupMessageEdited           = "GroupMessageEdited"
	EventTypeGroupMessageRecalled         = "GroupMessageRecalled"
	EventTypeGroupMessageReadCountUpdated = "GroupMessageReadCountUpdated"
	EventTypeGroupMessageMentioned        = "GroupMessageMentioned"
	EventTypeGroupTyping                  = "GroupTyping"
)

// 群组管理
const (
	EventTypeGroupMemberJoined         = "GroupMemberJoined"
	EventTypeGroupMemberLeft           = "GroupMemberLeft"
	EventTypeGroupMemberRemoved        = "GroupMemberRemoved"
	EventTypeGroupMemberRoleChanged    = "GroupMemberRoleChanged"
	EventTypeGroupMemberMuted          = "GroupMemberMuted"
	EventTypeGroupMemberUnmuted        = "GroupMemberUnmuted"
	EventTypeGroupInviteReceived       = "GroupInviteReceived"
	EventTypeGroupInviteRevoked        = "GroupInviteRevoked"
	EventTypeGroupInviteRejected       = "GroupInviteRejected"
	EventTypeGroupInviteExpired        = "GroupInviteExpired"
	EventTypeGroupInviteResponse       = "GroupInviteResponse"
	EventTypeGroupApplicationReceived  = "GroupApplicationReceived"
	EventTypeGroupApplicationCancelled = "GroupApplicationCancelled"
	EventTypeGroupApplicationExpired   = "GroupApplicationExpired"
	EventTypeGroupApplicationRejected  = "GroupApplicationRejected"
	EventTypeGroupApplicationResponse  = "GroupApplicationResponse"
	EventTypeGroupAnnouncementCreated  = "GroupAnnouncementCreated"
	EventTypeGroupUpdated              = "GroupUpdated"
	EventTypeGroupDismissed            = "GroupDismissed"
)

// 互动(表情反应)
const (
	EventTypeReactionAdded   = "ReactionAdded"
	EventTypeReactionRemoved = "ReactionRemoved"
)

// 举报
const (
	EventTypeReportResolved  = "ReportResolved"
	EventTypeReportDismissed = "ReportDismissed"
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

// SessionRevokedPayload notifies a connection that its session was revoked
// (api_spec_websocket.md): reason is device_evicted | revoked_by_user |
// password_changed. Server closes the connection with code 4001 afterwards.
type SessionRevokedPayload struct {
	Reason    string `json:"reason"`
	RevokedAt string `json:"revoked_at"`
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

// 客户端发送版 typing payload(api_spec_websocket.md:与服务端推送版结构不同,
// 故拆分命名避免同名冲突)
type PrivateTypingSendPayload struct {
	TargetUserID string `json:"target_user_id"`
	IsTyping     bool   `json:"is_typing"`
}

type GroupTypingSendPayload struct {
	GroupID  string `json:"group_id"`
	IsTyping bool   `json:"is_typing"`
}

// 服务端推送版 typing payload(含 actor: UserInfo 嵌套)
type PrivateTypingPayload struct {
	TargetUserID string         `json:"target_user_id"`
	Actor        map[string]any `json:"actor"`
	IsTyping     bool           `json:"is_typing"`
}

type GroupTypingPayload struct {
	GroupID  string         `json:"group_id"`
	Actor    map[string]any `json:"actor"`
	IsTyping bool           `json:"is_typing"`
}
