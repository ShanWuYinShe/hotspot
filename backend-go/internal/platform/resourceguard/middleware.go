package resourceguard

import (
	"bytes"
	"encoding/json"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/routemeta"
	"io"
	"net/http"
)

const maxGuardBodyBytes = 64 << 10

// 路由元数据（守卫规则表）统一声明在 platform/routemeta（审查 P2-架构）；
// 本包只做请求体解析与存在性预检。类型别名保持既有调用点不变。
type guardRule = routemeta.GuardRule
type guardRef = routemeta.GuardRef

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rule, ok := matchGuardRule(r.URL.Path)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		checker := DefaultChecker()
		if checker == nil || !checker.Ready(r.Context()) {
			observability.L().Debug("resourceguard bypass: checker not ready", "path", r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}
		refs, restored := extractRefs(r, rule)
		r = restored
		if len(refs) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		for _, ref := range refs {
			allowed, err := checker.MayExist(r.Context(), ref.kind, ref.id)
			if err != nil {
				observability.L().Warn("resourceguard bypass: checker query failed", "path", r.URL.Path, "kind", ref.kind, "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				httpx.WriteJSON(w, http.StatusOK, httpx.Error(httpx.CodeNotFound, notFoundMessage(ref.kind)))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type resolvedRef struct {
	kind ResourceKind
	id   string
}

func matchGuardRule(path string) (guardRule, bool) {
	return routemeta.MatchGuard(path)
}

func extractRefs(r *http.Request, rule guardRule) ([]resolvedRef, *http.Request) {
	if r.Body == nil {
		return nil, r
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, cloneRequestWithBody(r, nil)
	}
	cloned := cloneRequestWithBody(r, body)
	if len(body) == 0 || len(body) > maxGuardBodyBytes {
		return nil, cloned
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, cloned
	}
	refs := make([]resolvedRef, 0, len(rule.Refs))
	for _, spec := range rule.Refs {
		raw, ok := payload[spec.Field]
		if !ok {
			continue
		}
		var id string
		if err := json.Unmarshal(raw, &id); err != nil || id == "" {
			continue
		}
		refs = append(refs, resolvedRef{kind: spec.Kind, id: id})
	}
	return refs, cloned
}

func cloneRequestWithBody(r *http.Request, body []byte) *http.Request {
	clone := r.Clone(r.Context())
	clone.Body = io.NopCloser(bytes.NewReader(body))
	return clone
}

func notFoundMessage(kind ResourceKind) string {
	switch kind {
	case ResourceUser:
		return "用户不存在"
	case ResourceGroup:
		return "群组不存在"
	case ResourceMoment:
		return "瞬间不存在"
	default:
		return "资源不存在"
	}
}
