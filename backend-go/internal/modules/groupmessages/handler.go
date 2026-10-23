package groupmessages

import (
	"errors"
	"net/http"
	"unicode/utf8"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/observability"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	if err := idempotency.RequireKey(r); err != nil {
		httpx.WriteAPIError(w, r, "Send", err)
		return
	}
	var req SendGroupMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "Send", "请求体格式错误")
		return
	}
	if req.GroupID == "" || !isValidSendRequest(req) {
		httpx.WriteInvalidRequest(w, r, "Send", "参数错误")
		return
	}
	principal, _ := platformauth.PrincipalFromContext(r.Context())
	req.SenderUserID = principal.UserID
	idempotency.ReplayOrStore(w, r, "group_messages/send", func() httpx.ApiResponse[any] {
		msg, err := h.service.Send(r.Context(), req)
		if err != nil {
			// 契约(api_spec_errors.md)：不是群成员 → 10200 + forbidden_reason=not_group_member（不再是 10400）
			if errors.Is(err, ErrNotGroupMember) {
				observability.LogBusinessError(r, "Send", int(httpx.CodeForbidden), "不是群组成员")
				data := any(httpx.ForbiddenErrorData{ForbiddenReason: httpx.ReasonNotGroupMember})
				return httpx.ApiResponse[any]{Code: httpx.CodeForbidden, Message: "不是群组成员", Data: &data}
			}
			if errors.Is(err, ErrUserMuted) {
				mutedUntil := ""
				var mutedErr *MutedError
				if errors.As(err, &mutedErr) {
					mutedUntil = mutedErr.MutedUntil
				}
				// 契约(api_spec_chat.md):禁言拒绝返回 10400,data 携带 { mute: { muted_until } }
				// 结构化携带截止时间,避免客户端从 message 文本解析时间戳。
				observability.LogBusinessError(r, "Send", int(httpx.CodeConflict), "您已被禁言")
				data := map[string]any{"mute": map[string]any{"muted_until": mutedUntil}}
				var dataPtr any = data
				return httpx.ApiResponse[any]{Code: httpx.CodeConflict, Message: "您已被禁言", Data: &dataPtr}
			}
			// 契约(api_spec_chat.md → group_messages/send 10400)：share_code 无效等业务冲突
			if errors.Is(err, ErrInvalidShareCode) {
				return httpx.Business[any](r, "Send", httpx.CodeConflict, "分享链接无效或已过期")
			}
			// file_id 不存在/不属于发送者/未完成上传 → 10300（对齐 moments 图片归属口径）
			if errors.Is(err, ErrFileNotOwned) {
				return httpx.Business[any](r, "Send", httpx.CodeNotFound, "附件不存在或不属于当前用户")
			}
			observability.LogServerError(r, "Send", err)
			return httpx.ApiResponse[any]{Code: httpx.CodeInternalUnavailable, Message: "服务暂时不可用"}
		}
		var data any = msg
		return httpx.ApiResponse[any]{Code: 0, Data: &data}
	})
}
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	var req ListGroupMessagesRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "History", "请求体格式错误")
		return
	}
	if req.GroupID == "" || !isValidPagination(req.Pagination) {
		httpx.WriteInvalidRequest(w, r, "History", "参数错误")
		return
	}
	history, err := h.service.History(r.Context(), req.GroupID, req.Pagination)
	if err != nil {
		if errors.Is(err, ErrNotGroupMember) {
			// 契约(api_spec_errors.md)：不是群成员 → 10200 + forbidden_reason=not_group_member
			httpx.WriteAPIError(w, r, "History", httpx.Forbidden(httpx.ReasonNotGroupMember, "不是群组成员"))
			return
		}
		httpx.WriteAPIError(w, r, "History", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.Success(history))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateGroupMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "Update", "请求体格式错误")
		return
	}
	if req.MessageID == "" || req.Content == "" {
		httpx.WriteInvalidRequest(w, r, "Update", "参数错误")
		return
	}
	// 编辑与 send 同受契约 content(message) 1-5000 字符约束，防止先发短消息
	// 再编辑绕过上限、把 ~1MB 内容广播给全群。
	if utf8.RuneCountInString(req.Content) > messageContentMaxRunes {
		httpx.WriteInvalidRequest(w, r, "Update", "参数错误: 内容超过长度限制")
		return
	}
	principal, _ := platformauth.PrincipalFromContext(r.Context())
	req.UserID = principal.UserID
	if err := h.service.Update(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			httpx.WriteBusinessError(w, r, "Update", httpx.CodeNotFound, "消息不存在")
		case errors.Is(err, ErrPermissionDenied):
			// 契约(api_spec_chat.md → group_messages/update)：非作者编辑 → 10200 + not_resource_owner
			httpx.WriteAPIError(w, r, "Update", httpx.Forbidden(httpx.ReasonNotResourceOwner, "权限不足"))
		case errors.Is(err, ErrMessageRecalled):
			httpx.WriteBusinessError(w, r, "Update", httpx.CodeConflict, "消息已被撤回")
		case errors.Is(err, ErrMessageNotEditable):
			httpx.WriteBusinessError(w, r, "Update", httpx.CodeConflict, "消息不可编辑")
		default:
			observability.LogServerError(r, "Update", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

func (h *Handler) Recall(w http.ResponseWriter, r *http.Request) {
	var req RecallGroupMessageRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "Recall", "请求体格式错误")
		return
	}
	if req.MessageID == "" {
		httpx.WriteInvalidRequest(w, r, "Recall", "参数错误")
		return
	}
	principal, _ := platformauth.PrincipalFromContext(r.Context())
	req.UserID = principal.UserID
	if err := h.service.Recall(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrMessageNotFound):
			httpx.WriteBusinessError(w, r, "Recall", httpx.CodeNotFound, "消息不存在")
		case errors.Is(err, ErrPermissionDenied):
			// 契约(api_spec_chat.md → group_messages/recall)：非作者撤回 → 10200 + not_resource_owner
			httpx.WriteAPIError(w, r, "Recall", httpx.Forbidden(httpx.ReasonNotResourceOwner, "权限不足"))
		case errors.Is(err, ErrMessageRecalled):
			httpx.WriteBusinessError(w, r, "Recall", httpx.CodeConflict, "消息已被撤回")
		case errors.Is(err, ErrMessageRecallWindow):
			httpx.WriteBusinessError(w, r, "Recall", httpx.CodeConflict, "撤回时间窗口已过")
		default:
			observability.LogServerError(r, "Recall", err)
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
}

// MarkRead is the unified mark_read endpoint for group messages. It replaces
// the legacy /me/mark_as_read and /me/mark_all_read endpoints. scope selects
// the operation: "all" clears every unread group message for the caller;
// "group" marks messages in group_id as read (up to the optional
// up_to_message_id). scope is required (spec 参数必显式).
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req MarkGroupReadRequest
	if err := httpx.DecodeStrictJSON(r, &req); err != nil {
		httpx.WriteInvalidRequest(w, r, "MarkRead", "请求体格式错误")
		return
	}
	scope := req.Scope
	// 参数必显式（spec api_spec_base.md）：scope 影响行为范围，缺省不得静默翻转语义
	if scope == "" {
		httpx.WriteInvalidRequest(w, r, "MarkRead", "scope 必传")
		return
	}
	// 参数互斥（spec chat.md → mark_read）：scope='all' 时不得传 up_to_message_id
	if scope == "all" && req.UpToMessageID != "" {
		httpx.WriteInvalidRequest(w, r, "MarkRead", "scope=all 时不得传 up_to_message_id")
		return
	}
	switch scope {
	case "all":
		if err := h.service.MarkAllRead(r.Context()); err != nil {
			httpx.WriteAPIError(w, r, "MarkRead", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	case "group":
		if req.GroupID == "" {
			httpx.WriteInvalidRequest(w, r, "MarkRead", "参数错误")
			return
		}
		if err := h.service.MarkAsRead(r.Context(), MarkGroupMessagesReadRequest{
			GroupID:       req.GroupID,
			UpToMessageID: req.UpToMessageID,
		}); err != nil {
			switch {
			case errors.Is(err, ErrMessageNotFound):
				httpx.WriteBusinessError(w, r, "MarkRead", httpx.CodeNotFound, "消息不存在")
			case errors.Is(err, ErrNotGroupMember):
				// 契约(api_spec_errors.md)：不是群成员 → 10200 + forbidden_reason=not_group_member
				httpx.WriteAPIError(w, r, "MarkRead", httpx.Forbidden(httpx.ReasonNotGroupMember, "不是群组成员"))
			default:
				observability.LogServerError(r, "MarkRead", err)
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInternalUnavailable, "服务暂时不可用"))
			}
			return
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.SuccessEmpty())
		return
	default:
		httpx.WriteBusinessError(w, r, "MarkRead", httpx.CodeInvalidRequest, "参数错误")
	}
}

func isValidPagination(p *PaginationParams) bool {
	if p == nil {
		return true
	}
	if p.Limit < 0 || p.Limit > 100 {
		return false
	}
	if p.Cursor == "" {
		return true
	}
	_, _, err := decodeGroupMessageCursor(p.Cursor)
	return err == nil
}

// isValidSendRequest 按消息类型校验必填字段（与私聊侧 isValidSendRequest
// 一致）：text 需要 content；image/video/audio/file 需要 file_id；share
// messageContentMaxRunes 契约级上限（api_spec_base.md 字段校验约束：content(message) 1-5000 字符）。
const messageContentMaxRunes = 5000

// mentionsMaxCount 契约级上限（api_spec_base.md 字段校验约束：mentions 最多 20 个）。
const mentionsMaxCount = 20

// 需要 share_code。此前不校验，text 空 content、share 空 share_code 都能发出。
// content 长度上限对所有类型生效（审查 P2-4，对齐 Update 路径的同款防绕过）：
// 媒体/share 类型此前不校验，type=image + 合法 file_id + 超长 content 可绕过
// 5000 字契约，随消息入库并广播给全群（全局请求体 1MiB 上限只封顶了体量）。
// 仅 text 要求 content 非空；媒体类型的可选 content 同受长度约束。
func isValidSendRequest(req SendGroupMessageRequest) bool {
	if req.Type == "" {
		return false
	}
	if len(req.Mentions) > mentionsMaxCount {
		return false
	}
	if utf8.RuneCountInString(req.Content) > messageContentMaxRunes {
		return false
	}
	switch req.Type {
	case "text":
		return req.Content != ""
	case "image", "video", "audio", "file":
		return req.FileID != ""
	case "share":
		return req.ShareCode != ""
	default:
		return false
	}
}
