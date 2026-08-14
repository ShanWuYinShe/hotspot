package groupmessages

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

// TestHandlerSendNonMemberForbidden verifies spec api_spec_chat.md:
// group_messages/send 非群成员 → 10200 + data.forbidden_reason=not_group_member（不再是 10400）。
func TestHandlerSendNonMemberForbidden(t *testing.T) {
	repo := &stubGroupMessageRepo{isMemberErr: ErrNotGroupMember}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","type":"text","content":"hi"}`))
	req.Header.Set("Idempotency-Key", "sp_nonmember_send_0000000000000000")
	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	assertGMForbidden(t, rr, "not_group_member")
}

// TestHandlerHistoryNonMemberForbidden verifies spec api_spec_chat.md:
// group_messages/list 非群成员 → 10200 + data.forbidden_reason=not_group_member。
func TestHandlerHistoryNonMemberForbidden(t *testing.T) {
	repo := &stubGroupMessageRepo{isMemberErr: ErrNotGroupMember}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1"}`))
	rr := httptest.NewRecorder()
	handler.History(rr, req)
	assertGMForbidden(t, rr, "not_group_member")
}

// TestHandlerRecallNonAuthorForbidden verifies spec api_spec_chat.md:
// group_messages/recall 仅作者；非作者 → 10200 + data.forbidden_reason=not_resource_owner。
func TestHandlerRecallNonAuthorForbidden(t *testing.T) {
	repo := &stubGroupMessageRepo{recallErr: ErrPermissionDenied}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"message_id":"m1"}`))
	rr := httptest.NewRecorder()
	handler.Recall(rr, req)
	assertGMForbidden(t, rr, "not_resource_owner")
}

// TestHandlerUpdateNonAuthorForbidden verifies spec api_spec_chat.md:
// group_messages/update 仅作者；非作者 → 10200 + data.forbidden_reason=not_resource_owner。
func TestHandlerUpdateNonAuthorForbidden(t *testing.T) {
	repo := &stubGroupMessageRepo{updateErr: ErrPermissionDenied}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"message_id":"m1","content":"edited"}`))
	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	assertGMForbidden(t, rr, "not_resource_owner")
}

// TestHandlerSendMutedConflict verifies mute 仍为 10400 + data.mute.muted_until。
func TestHandlerSendMutedConflict(t *testing.T) {
	repo := &stubGroupMessageRepo{
		sendErr: &MutedError{MutedUntil: "2026-01-01T00:00:00Z"},
	}
	handler := NewHandler(NewServiceWithRepo(repo))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","type":"text","content":"hi"}`))
	req.Header.Set("Idempotency-Key", "sp_muted_send_000000000000000000")
	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	var resp struct {
		Code httpx.Code `json:"code"` 
		Data struct {
			Mute struct {
				MutedUntil string `json:"muted_until"`
			} `json:"mute"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, rr.Body.String())
	}
	if resp.Code != httpx.CodeConflict {
		t.Fatalf("expected code httpx.CodeConflict(10400) for mute, got %d", resp.Code)
	}
	if resp.Data.Mute.MutedUntil != "2026-01-01T00:00:00Z" {
		t.Fatalf("expected MutedUntil in mute data, got %#v", resp.Data)
	}
}

// TestServiceRecallMessageAsAdmin verifies the exported admin recall used by
// groups/admin/messages/recall（spec api_spec_group.md）：做群归属校验、不走作者/时间窗
// 校验、已撤回返回 ErrMessageRecalled、无时间窗。
func TestServiceRecallMessageAsAdmin(t *testing.T) {
	repo := &specAdminRepo{owner: MessageOwnerMeta{GroupID: "g1", SenderUserID: "u1"}}
	svc := NewServiceWithRepo(repo)

	if err := svc.RecallMessageAsAdmin(context.Background(), "admin1", "g1", "m1"); err != nil {
		t.Fatalf("admin recall should succeed for member message: %v", err)
	}
	if repo.recalledMsgID != "m1" {
		t.Fatalf("expected recall of m1, got %q", repo.recalledMsgID)
	}

	// 消息不属于该群 → ErrMessageNotFound
	wrongGroup := &specAdminRepo{owner: MessageOwnerMeta{GroupID: "g9", SenderUserID: "u1"}}
	if err := NewServiceWithRepo(wrongGroup).RecallMessageAsAdmin(context.Background(), "admin1", "g1", "m1"); !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("expected ErrMessageNotFound for group mismatch, got %v", err)
	}

	// 已撤回 → ErrMessageRecalled
	recalled := &specAdminRepo{owner: MessageOwnerMeta{GroupID: "g1", SenderUserID: "u1"}, recalled: true}
	if err := NewServiceWithRepo(recalled).RecallMessageAsAdmin(context.Background(), "admin1", "g1", "m1"); !errors.Is(err, ErrMessageRecalled) {
		t.Fatalf("expected ErrMessageRecalled, got %v", err)
	}
}

// specAdminRepo 兼容之前 handler 测试所用 Repository 接口的桩实现，
// 仅服务于 RecallMessageAsAdmin 单测。
type specAdminRepo struct {
	owner        MessageOwnerMeta
	recalled     bool
	recalledMsgID string
}

func (r *specAdminRepo) Send(context.Context, SendGroupMessageRequest) (MessageSent, error) { return MessageSent{}, nil }
func (r *specAdminRepo) History(context.Context, string, *PaginationParams) (PaginatedGroupMessages, error) { return PaginatedGroupMessages{}, nil }
func (r *specAdminRepo) Update(context.Context, UpdateGroupMessageRequest) (MessageEventMeta, error) { return MessageEventMeta{}, nil }
func (r *specAdminRepo) Recall(context.Context, RecallGroupMessageRequest) (MessageEventMeta, error) { return MessageEventMeta{}, nil }
func (r *specAdminRepo) RecallAsAdmin(context.Context, string) (MessageEventMeta, error) {
	if r.recalled {
		return MessageEventMeta{}, ErrMessageRecalled
	}
	r.recalledMsgID = "m1"
	return MessageEventMeta{GroupID: "g1", MessageID: "m1", SentAt: "2026-01-01T00:00:00Z"}, nil
}
func (r *specAdminRepo) MarkAsRead(context.Context, string, MarkGroupMessagesReadRequest) error { return nil }
func (r *specAdminRepo) MarkAllRead(context.Context, string) error { return nil }
func (r *specAdminRepo) IsMember(context.Context, string, string) (bool, error) { return true, nil }
func (r *specAdminRepo) GetMessageForReply(context.Context, string) (map[string]any, bool, error) { return nil, false, nil }
func (r *specAdminRepo) GetMediaForMessage(context.Context, string) (map[string]any, bool, error) { return nil, false, nil }
func (r *specAdminRepo) MemberMuteStatus(context.Context, string, string) (bool, string, bool, error) { return false, "", false, nil }
func (r *specAdminRepo) MessageExists(context.Context, string) (bool, string, error) { return false, "", nil }
func (r *specAdminRepo) MessageGroupID(context.Context, string) (string, bool, error) { return "", false, nil }
func (r *specAdminRepo) GetGroupName(context.Context, string) (string, bool, error) { return "", false, nil }
func (r *specAdminRepo) LookupMessageMeta(context.Context, string) (MessageOwnerMeta, bool, error) {
	return r.owner, true, nil
}

func assertGMForbidden(t *testing.T, rr *httptest.ResponseRecorder, reason string) {
	t.Helper()
	body := rr.Body.String()
	var resp struct {
		Code httpx.Code `json:"code"` 
		Data struct {
			ForbiddenReason string `json:"forbidden_reason"`
		} `json:"data"`
	}
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, body)
	}
	if resp.Code != httpx.CodeForbidden {
		t.Fatalf("expected code httpx.CodeForbidden(10200), got %d (body=%s)", resp.Code, body)
	}
	if resp.Data.ForbiddenReason != reason {
		t.Fatalf("expected forbidden_reason=%q, got %q", reason, resp.Data.ForbiddenReason)
	}
}
