package resourceguard

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

const maxGuardBodyBytes = 64 << 10

type guardRule struct {
	suffix string
	refs   []guardRef
}

type guardRef struct {
	field string
	kind  ResourceKind
}

var guardRules = []guardRule{
	{suffix: "/api/v1/users/open_the_door", refs: []guardRef{{field: "user_id", kind: ResourceUser}}},
	{suffix: "/api/v1/users/fetch_groups", refs: []guardRef{{field: "user_id", kind: ResourceUser}}},
	{suffix: "/api/v1/users/fetch_moments", refs: []guardRef{{field: "user_id", kind: ResourceUser}}},
	{suffix: "/api/v1/users/fetch_reacted_moments", refs: []guardRef{{field: "user_id", kind: ResourceUser}}},
	{suffix: "/api/v1/groups/open_the_door", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/members/fetch", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/members/search", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	// 群管理端点实际挂在 /groups/admin/* 子树（modules/groups/module.go）；
	// 旧规则 /groups/update、/groups/dismiss 从未匹配（HasSuffix 不命中，死配置）。
	// 全量对齐 admin 子树携带 group_id 的端点：资源存在性预检 + 404 语义统一。
	{suffix: "/api/v1/groups/admin/update", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/settings", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/invites/create", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/invites/fetch_sent", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/invites/revoke", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/members/fetch", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/members/set_role", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/members/set_mute", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/members/set_ban", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/members/fetch_banned", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/members/remove", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/applications/fetch", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/applications/approve", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/applications/reject", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/announcements/create", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/announcements/update", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/announcements/delete", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/messages/recall", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/dismiss", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/admin/transfer_ownership", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/groups/enter", refs: []guardRef{{field: "group_id", kind: ResourceGroup}}},
	{suffix: "/api/v1/moments/open_the_door", refs: []guardRef{{field: "moment_id", kind: ResourceMoment}}},
	{suffix: "/api/v1/moments/update", refs: []guardRef{{field: "moment_id", kind: ResourceMoment}}},
	{suffix: "/api/v1/moments/delete", refs: []guardRef{{field: "moment_id", kind: ResourceMoment}}},
	{suffix: "/api/v1/moments/set_favorite", refs: []guardRef{{field: "moment_id", kind: ResourceMoment}}},
	{suffix: "/api/v1/comments/fetch", refs: []guardRef{{field: "moment_id", kind: ResourceMoment}}},
	{suffix: "/api/v1/comments/create", refs: []guardRef{{field: "moment_id", kind: ResourceMoment}}},
}

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
	for _, rule := range guardRules {
		if strings.HasSuffix(path, rule.suffix) {
			return rule, true
		}
	}
	return guardRule{}, false
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
	refs := make([]resolvedRef, 0, len(rule.refs))
	for _, spec := range rule.refs {
		raw, ok := payload[spec.field]
		if !ok {
			continue
		}
		var id string
		if err := json.Unmarshal(raw, &id); err != nil || id == "" {
			continue
		}
		refs = append(refs, resolvedRef{kind: spec.kind, id: id})
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
