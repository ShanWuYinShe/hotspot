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

// EventTypeSubscriptionRevoked 仅用于跨实例控制通道（redis bridge revoke:/
// groupdrop:），不下发客户端：spec 未定义订阅撤销事件，客户端经既有
// GroupMemberRemoved / GroupMemberLeft / GroupDismissed 感知成员资格变更。
const EventTypeSubscriptionRevoked = "SubscriptionRevoked"

// SubscriptionRevokedPayload 是撤销控制事件的载荷（仅实例间流转）。
type SubscriptionRevokedPayload struct {
	GroupID string `json:"group_id"`
}

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
	EventTypeContentMentioned = "ContentMentioned"
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

// 多端同步（self-sync）事件不做：已读状态/设置/关系等具体状态不做跨设备实时同步
// （见 api_spec_index.md「预留」）。未来如需通用的多端状态实时同步，将作为独立
// WS 协议版本重新设计，届时再定义事件形状。

// Event 是 WS 协议的服务端出站事件。ServerSeq 在 resume 续传路径上由 Hub 分配
// （全局单调递增，见 resume_store.go），客户端可按 seq 去重（spec L173）；
// 非 resume 路径（握手/心跳/presence 等）事件 omitempty 不下发该字段。
//
// ServerSeq 的 JSON 序列化用 `,string`（int64 → JSON 字符串）：spec 契约定义
// server_seq 为 string——JS/ArkTS 的 number 是 IEEE754 double，无法无损表示
// int64（安全整数上限 2^53），跨端统一用字符串透传/去重，客户端不做数值运算
// （spec L330「客户端仅透传与去重，不做数值运算」）。后端内部仍用 int64 保持
// 数值单调语义与运算正确性。
type Event struct {
	Type      string `json:"type"`
	Payload   any    `json:"payload"`
	ServerSeq int64  `json:"server_seq,omitempty,string"`
}

type ConnectionEstablishedPayload struct {
	ConnectionID string `json:"connection_id"`
	UserID       string `json:"user_id"`
	ServerTime   string `json:"server_time"`
	// ResumeToken 用于断线重连增量恢复（spec L155-156）：客户端在重连 Auth 中
	// 原样带回。Redis 不可用时为空（omitempty），客户端回退全量重放（spec L161）。
	ResumeToken string `json:"resume_token,omitempty"`
	// ServerSeq 是当前该 user 事件流的最新序列号（spec L133：CE 含 server_seq，
	// 即使为 0 也下发，故不加 omitempty）；客户端据此判断 resume 起点。
	// `,string`：与 Event.ServerSeq 同因——int64 跨端统一序列化为字符串（spec L330）。
	ServerSeq int64 `json:"server_seq,string"`
	Config    struct {
		PingInterval     int `json:"ping_interval"`
		PingTimeout      int `json:"ping_timeout"`
		MaxSubscriptions struct {
			User  int `json:"user"`
			Group int `json:"group"`
		} `json:"max_subscriptions"`
	} `json:"config"`
}

// ErrorCode 是 WebSocket 业务阶段错误码的类型化枚举
// （权威定义：spec/api_spec_errors.md「WebSocket 连接后业务阶段错误码」）。
type ErrorCode int

const (
	CodeWSProtocol   ErrorCode = 20000 // 连接/协议错误（鉴权失败/心跳超时/消息格式错误）
	CodeWSRateLimit  ErrorCode = 20300 // 限流
	CodeWSPermission ErrorCode = 20400 // 权限错误（无权限/操作受限/群组已解散）
	// CodeWSAccountBanned 账号被封禁（握手鉴权阶段拒绝，spec 10201 同源语义：
	// 客户端须清除本地会话并展示封禁页，不可自动重连）。
	CodeWSAccountBanned ErrorCode = 20401
)

type ErrorPayload struct {
	Code       ErrorCode         `json:"code"`
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

// SubscribeAckPayload 是 SetSubscriptions 的确认事件（spec api_spec_websocket.md
// SubscribeAckPayload）：accepted 按 type 分组回传已接受项；只要 failed_ids 非空
// code 必须为 20100。
type SubscribeAckPayload struct {
	ClientRequestID string               `json:"client_request_id"`
	Operation       string               `json:"operation"`
	Accepted        []SubscriptionItem   `json:"accepted"`
	Code            int                  `json:"code"`
	Message         string               `json:"message,omitempty"`
	FailedIDs       []FailedSubscription `json:"failed_ids,omitempty"`
}

// FailedSubscription 对应 spec FailedSubscriptionId（含 type：批量请求需明确
// 失败项的订阅类型，客户端按 type+subscription_id 回滚本地订阅）。
type FailedSubscription struct {
	Type           string `json:"type"`
	SubscriptionID string `json:"subscription_id"`
	Reason         string `json:"reason"`
}

type SubscriptionsSnapshotPayload struct {
	ClientRequestID string   `json:"client_request_id,omitempty"`
	UserIDs         []string `json:"user_ids,omitempty"`
	GroupIDs        []string `json:"group_ids,omitempty"`
}

type AuthPayload struct {
	ClientType string `json:"client_type,omitempty"`
	// AccessToken 由 App 端携带（Web 端用 Cookie）；DeviceID 支撑单设备单会话
	// 模型（spec AuthPayload：App 端必填）。
	AccessToken string `json:"access_token,omitempty"`
	DeviceID    string `json:"device_id,omitempty"`
	// ResumeToken 由客户端在重连时携带（spec L155），服务端据此尝试增量续传。
	ResumeToken string `json:"resume_token,omitempty"`
}

// ResumeState 是 resume_token 在服务端缓存的可恢复状态（spec L156, L159）：
// 连接断开后，服务端在 TTL 内保留该用户的最后序列号与订阅快照，重连时据此回放
// 增量事件并恢复订阅。存储于跨实例共享的 Redis——单实例内存不可用（spec L159）。
type ResumeState struct {
	UserID  string
	LastSeq int64
	Subs    SubscriptionsSnapshotPayload
}

type PingPayload struct{}

// SubscriptionItem 是 SetSubscriptions/SubscribeAck 的订阅项
// （spec api_spec_websocket.md SubscriptionItem：type + ids 分组）。
type SubscriptionItem struct {
	Type string   `json:"type"`
	IDs  []string `json:"ids"`
}

// SetSubscriptionsPayload 是客户端订阅管理事件（spec api_spec_websocket.md
// SetSubscriptionsPayload）：批量结构，单请求可同时操作 user 和 group 两类订阅。
type SetSubscriptionsPayload struct {
	ClientRequestID string             `json:"client_request_id"`
	Operation       string             `json:"operation"`
	Items           []SubscriptionItem `json:"items"`
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
