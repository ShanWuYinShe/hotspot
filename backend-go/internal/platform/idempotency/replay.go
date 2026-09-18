package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/requestmeta"
)

// hashRequestBody 计算请求体指纹（SHA-256）。仅在携带 Idempotency-Key 时调用，
// 请求体此前已被 LimitRequestBody 中间件封顶（1MiB），缓冲内存有界。
func hashRequestBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// defaultStore is read on every request and written once at boot, so guard it
// with an atomic pointer instead of a bare global (which would race under
// -race and on reconfigure). The pointer holds any Store implementation:
// InMemoryStore for single-instance, RedisStore when Redis is configured
// (bootstrap wires it — previously SetStore silently dropped non-in-memory
// stores, breaking cross-instance idempotency).
var defaultStore atomic.Pointer[Store]

func init() {
	SetStore(NewInMemoryStore())
}

// currentStore returns the active store.
func currentStore() Store {
	if p := defaultStore.Load(); p != nil {
		return *p
	}
	return NewInMemoryStore()
}

func SetStore(store Store) {
	if store == nil {
		return
	}
	defaultStore.Store(&store)
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

	// 幂等上下文一致性（spec api_spec_errors.md 22000「幂等上下文不匹配」）：
	// 同一 key 换请求体重放会静默拿到首次响应且本次不执行，掩盖客户端串键
	// 缺陷（如发错内容却"成功"）。缓冲请求体计算指纹并回填 body 供 producer
	// 读取；命中完成态记录时比对，存量记录（无哈希）跳过比对以兼容滚动升级。
	body, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeInvalidRequest, "参数错误"))
		return true
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	bodyHash := hashRequestBody(body)

	// Reserve atomically: returns (record, true) when we claimed it (now
	// processing), or (existing, false) when a record already exists.
	// Expired hits return StatusExpired so we can emit 22001 instead of
	// silently reusing the key as a brand-new request.
	existing, claimed := store.Reserve(scope, scopedKey, expiresAt)
	if !claimed {
		if existing.Status == StatusExpired {
			httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeIdempotencyExpired, "幂等缓存已过期，请使用新的幂等键重试"))
			return true
		}
		if existing.Status == StatusCompleted {
			if existing.BodyHash != "" && existing.BodyHash != bodyHash {
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeIdempotencyConflict, "幂等上下文不匹配"))
				return true
			}
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
	store.Save(Record{Key: scopedKey, Scope: scope, Status: StatusCompleted, Response: append(blob, '\n'), ExpiresAt: expiresAt, BodyHash: bodyHash})
	httpx.WriteJSON(w, http.StatusOK, payload)
	return true
}
