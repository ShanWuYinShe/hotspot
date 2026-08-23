package idempotency

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/requestmeta"
)

// defaultStore is read on every request and written once at boot, so guard it
// with an atomic pointer instead of a bare global (which would race under
// -race and on reconfigure).
var defaultStore atomic.Pointer[InMemoryStore]

func init() {
	defaultStore.Store(NewInMemoryStore())
}

// currentStore returns the active store. Reserved for future swap-in of a
// distributed store; today it's always the in-memory one.
func currentStore() Store { return defaultStore.Load() }

func SetStore(store Store) {
	if s, ok := store.(*InMemoryStore); ok && s != nil {
		defaultStore.Store(s)
	}
}

func ReplayOrStore[T any](w http.ResponseWriter, r *http.Request, scope string, producer func() httpx.ApiResponse[T]) bool {
	key := KeyFromContext(r.Context())
	if key == "" {
		payload := producer()
		httpx.WriteJSON(w, http.StatusOK, payload)
		return true
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	store := currentStore()

	// 幂等键按「请求方身份」限定作用域：登录用户绑 userID，匿名请求绑 IP。
	// 否则用户 B 用与用户 A 相同的 Idempotency-Key 调同端点会重放 A 的响应
	// （跨用户数据泄漏）。
	actor := "ip:" + requestmeta.ClientIP(r)
	if principal, ok := platformauth.PrincipalFromContext(r.Context()); ok && principal.UserID != "" {
		actor = "u:" + principal.UserID
	}
	scopedKey := actor + "::" + key

	// Reserve atomically: returns (record, true) when we claimed it (now
	// processing), or (existing, false) when a record already exists.
	existing, claimed := store.Reserve(scope, scopedKey, expiresAt)
	if !claimed {
		if existing.Status == StatusCompleted {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(existing.Response)
			return true
		}
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeIdempotencyInFlight, "请求正在处理中，请稍后重试"))
		return true
	}

	// producer panic 时（由外层 Recoverer 接住）删除 processing 标记再重抛：
	// 否则该 Idempotency-Key 停留在 processing，24h 内重试全部得到
	// CodeIdempotencyInFlight（key 被毒化）。
	panicked := true
	defer func() {
		if panicked {
			store.Delete(scope, scopedKey)
		}
	}()
	payload := producer()
	panicked = false
	// spec api_spec_base.md「幂等性」：仅缓存成功响应（code == 0）。
	// 失败响应不缓存——删除 processing 预留，让客户端修正后以同一 key 重试
	// 能真正再次执行，而不是在 24h 内反复拿到缓存的失败响应。
	if payload.Code != httpx.CodeOK {
		store.Delete(scope, scopedKey)
		httpx.WriteJSON(w, http.StatusOK, payload)
		return true
	}
	blob, _ := json.Marshal(payload)
	store.Save(Record{Key: scopedKey, Scope: scope, Status: StatusCompleted, Response: append(blob, '\n'), ExpiresAt: expiresAt})
	httpx.WriteJSON(w, http.StatusOK, payload)
	return true
}
