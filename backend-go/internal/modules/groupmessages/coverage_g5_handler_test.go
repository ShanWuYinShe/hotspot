package groupmessages

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
)

// TestHandlerSendShareAndFileErrors 覆盖 Send 的 share_code 无效（10400）
// 与附件不属于发送者（10300）的契约映射。
func TestHandlerSendShareAndFileErrors(t *testing.T) {
	// share_code 校验失败 → 10400。
	svc := NewServiceWithDeps(&stubGroupMessageRepo{}, nil, nil, gmCovShareValidator{err: errors.New("invalid share")})
	handler := NewHandler(svc)
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","type":"share","share_code":"sc_bad"}`))
	req.Header.Set("Idempotency-Key", "g5share00000000000000000000000001")
	rr := httptest.NewRecorder()
	handler.Send(rr, req)
	gmAssertCode(t, rr, httpx.CodeConflict)

	// 附件不存在/不属于发送者 → 10300。
	handler = NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	req = gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","type":"image","file_id":"file_not_owned"}`))
	req.Header.Set("Idempotency-Key", "g5file000000000000000000000000001")
	rr = httptest.NewRecorder()
	handler.Send(rr, req)
	gmAssertCode(t, rr, httpx.CodeNotFound)
}

// TestHandlerUpdateRejectsOverlongContent 覆盖编辑路径的 5000 字上限
// （防「先发短消息再编辑绕过上限」）。
func TestHandlerUpdateRejectsOverlongContent(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))
	body := fmt.Sprintf(`{"message_id":"m1","content":"%s"}`, strings.Repeat("好", messageContentMaxRunes+1))
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.Update(rr, req)
	gmAssertCode(t, rr, httpx.CodeInvalidRequest)
}

// TestHandlerMarkReadScopeValidation 覆盖 mark_read 的 scope 必传/互斥校验。
func TestHandlerMarkReadScopeValidation(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubGroupMessageRepo{}))

	for _, tc := range []struct {
		name string
		body string
	}{
		{"scope missing", `{}`},
		{"scope=all with up_to", `{"scope":"all","up_to_message_id":"m1"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := gmCtx("u1")
			req.Body = io.NopCloser(strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			handler.MarkRead(rr, req)
			gmAssertCode(t, rr, httpx.CodeInvalidRequest)
		})
	}
}

// TestHandlerMarkReadErrorMapping 覆盖 group scope 下非成员（10200）与其他
// 仓储错误（10600）的分派；后者不能被静默吞成成功。
func TestHandlerMarkReadErrorMapping(t *testing.T) {
	t.Run("not group member", func(t *testing.T) {
		repo := &stubGroupMessageRepo{isMemberErr: ErrNotGroupMember}
		handler := NewHandler(NewServiceWithRepo(repo))
		req := gmCtx("u1")
		req.Body = io.NopCloser(strings.NewReader(`{"scope":"group","group_id":"g1"}`))
		rr := httptest.NewRecorder()
		handler.MarkRead(rr, req)
		gmAssertCode(t, rr, httpx.CodeForbidden)
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &stubGroupMessageRepo{markReadFn: func(_ context.Context, _ string, _ MarkGroupMessagesReadRequest) error {
			return errors.New("boom")
		}}
		handler := NewHandler(NewServiceWithRepo(repo))
		req := gmCtx("u1")
		req.Body = io.NopCloser(strings.NewReader(`{"scope":"group","group_id":"g1"}`))
		rr := httptest.NewRecorder()
		handler.MarkRead(rr, req)
		gmAssertCode(t, rr, httpx.CodeInternalUnavailable)
	})
}

// TestHandlerHistoryAcceptsValidCursor 覆盖分页游标校验的合法分支：
// 合法 cursor 必须放行到 service（此前只测过非法 cursor）。
func TestHandlerHistoryAcceptsValidCursor(t *testing.T) {
	repo := &stubGroupMessageRepo{histResult: PaginatedGroupMessages{Items: []GroupMessageItem{{MessageID: "m1"}}}}
	handler := NewHandler(NewServiceWithRepo(repo))
	cursor := encodeGroupMessageCursor(time.Now().UTC().Format(time.RFC3339Nano), "m1")
	if cursor == "" {
		t.Fatal("failed to encode cursor fixture")
	}
	body := fmt.Sprintf(`{"group_id":"g1","pagination":{"limit":10,"cursor":%q}}`, cursor)
	req := gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(body))
	rr := httptest.NewRecorder()
	handler.History(rr, req)
	gmAssertCode(t, rr, 0)

	// 只带 limit、不带 cursor 同样合法（首屏请求）。
	req = gmCtx("u1")
	req.Body = io.NopCloser(strings.NewReader(`{"group_id":"g1","pagination":{"limit":10}}`))
	rr = httptest.NewRecorder()
	handler.History(rr, req)
	gmAssertCode(t, rr, 0)
}
