package share

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/testdb"
)

// shareContractState 契约用例上下文：PG 侧 shares 有 users 外键，预置创建者/他人。
// 契约用例只覆盖 location 类型分享：user/group/moment 类型的 preview 依赖跨模块
// 真值聚合（内存实现无用户目录/目标表，天然无法对齐），location 类型两实现均从
// payload 构造 preview，可严格对齐。
type shareContractState struct {
	ctx     context.Context
	creator string
	other   string
}

var shareCodePattern = regexp.MustCompile(`^[A-HJ-NP-Z2-9]{12}$`)

func TestShareRepositoryContract(t *testing.T) {
	creator := "it_share_contract_creator"
	other := "it_share_contract_other"

	impls := []testdb.ContractImpl[Repository, shareContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, shareContractState, func()) {
				return NewInMemoryRepository(), shareContractState{ctx: context.Background(), creator: creator, other: other}, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, shareContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				testdb.InsertUser(t, txFactory, creator, "nick_"+creator)
				testdb.InsertUser(t, txFactory, other, "nick_"+other)
				return NewPostgresRepository(txFactory), shareContractState{ctx: context.Background(), creator: creator, other: other}, cleanup
			},
		},
	}

	locationPayload := func(name string) map[string]any {
		return map[string]any{
			"shared_location": map[string]any{
				"location_name": name,
				"latitude":      35.123,
				"longitude":     112.456,
			},
		}
	}

	create := func(t *testing.T, repo Repository, state shareContractState, name string, expiresAt *time.Time) ShareCreated {
		t.Helper()
		created, err := repo.Create(context.Background(), createShareCommand{
			ShareType:     ShareTypeLocation,
			Payload:       locationPayload(name),
			CreatorUserID: state.creator,
			ExpiresAt:     expiresAt,
		})
		if err != nil {
			t.Fatalf("create share %q: %v", name, err)
		}
		if created.ShareID == "" || created.ShareCode == "" {
			t.Fatalf("created share = %#v", created)
		}
		return created
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, shareContractState]{
		{
			Name: "create and get round-trips location share",
			Run: func(t *testing.T, repo Repository, state shareContractState) {
				created := create(t, repo, state, "Contract Beach", nil)
				if !shareCodePattern.MatchString(created.ShareCode) {
					t.Fatalf("share code = %q, want 12 chars of [A-HJ-NP-Z2-9]", created.ShareCode)
				}
				if created.ExpiresAt != nil {
					t.Fatalf("expires_at = %#v, want nil", created.ExpiresAt)
				}
				detail, err := repo.Get(state.ctx, getShareQuery{ShareCode: created.ShareCode})
				if err != nil {
					t.Fatalf("get: %v", err)
				}
				if detail.Share.ShareID != created.ShareID || detail.Share.ShareCode != created.ShareCode {
					t.Fatalf("share base = %#v", detail.Share)
				}
				if detail.Share.ShareType != ShareTypeLocation {
					t.Fatalf("share type = %q", detail.Share.ShareType)
				}
				if detail.Share.Creator.UserID != state.creator || detail.Share.Creator.Nickname == "" {
					t.Fatalf("creator = %#v", detail.Share.Creator)
				}
				if _, err := time.Parse(time.RFC3339, detail.Share.CreatedAt); err != nil {
					t.Fatalf("created_at must be RFC3339: %q", detail.Share.CreatedAt)
				}
				if detail.Share.ExpiresAt != nil {
					t.Fatalf("detail expires_at = %#v", detail.Share.ExpiresAt)
				}
				if detail.Preview == nil || detail.Preview.Title != "Contract Beach" || detail.Preview.Subtitle == nil || *detail.Preview.Subtitle != "位置" {
					t.Fatalf("preview = %#v", detail.Preview)
				}
				sharedLocation, ok := detail.Payload["shared_location"].(map[string]any)
				if !ok || sharedLocation["location_name"] != "Contract Beach" {
					t.Fatalf("payload = %#v", detail.Payload)
				}
			},
		},
		{
			Name: "share codes are unique",
			Run: func(t *testing.T, repo Repository, state shareContractState) {
				seen := map[string]bool{}
				for i := 0; i < 20; i++ {
					created := create(t, repo, state, "Code Uniq", nil)
					if seen[created.ShareCode] {
						t.Fatalf("duplicate share code: %s", created.ShareCode)
					}
					seen[created.ShareCode] = true
				}
			},
		},
		{
			Name: "unknown code is not found",
			Run: func(t *testing.T, repo Repository, state shareContractState) {
				if _, err := repo.Get(state.ctx, getShareQuery{ShareCode: "AAAAAAAAAAAA"}); !errors.Is(err, ErrShareNotFound) {
					t.Fatalf("get unknown err = %v, want ErrShareNotFound", err)
				}
				if _, found, err := repo.ResolveShareIDByCode(state.ctx, "AAAAAAAAAAAA"); err != nil || found {
					t.Fatalf("resolve unknown = found:%v err:%v", found, err)
				}
			},
		},
		{
			Name: "expired share is unavailable but resolvable and listed as not revoked",
			Run: func(t *testing.T, repo Repository, state shareContractState) {
				past := time.Now().UTC().Add(-time.Hour)
				created := create(t, repo, state, "Expired Place", &past)
				if created.ExpiresAt == nil {
					t.Fatal("created expires_at must be non-nil")
				}
				if _, err := time.Parse(time.RFC3339, *created.ExpiresAt); err != nil {
					t.Fatalf("expires_at must be RFC3339: %q", *created.ExpiresAt)
				}
				// 过期分享：open_the_door 不可用（10300），但解析接口仍可命中（供撤销幂等）。
				if _, err := repo.Get(state.ctx, getShareQuery{ShareCode: created.ShareCode}); !errors.Is(err, ErrShareNotFound) {
					t.Fatalf("get expired err = %v, want ErrShareNotFound", err)
				}
				shareID, found, err := repo.ResolveShareIDByCode(state.ctx, created.ShareCode)
				if err != nil || !found || shareID != created.ShareID {
					t.Fatalf("resolve expired = %q found:%v err:%v", shareID, found, err)
				}
				// 「我的分享」列表口径：过期与撤销都呈现为不可用（is_revoked=true、preview=nil）。
				// spec 只定义 is_revoked 字段、未单独定义 expired 的列表呈现；两实现按同一口径
				// 收敛（PG 显式后处理），契约据此断言一致行为。
				page, err := repo.List(state.ctx, listSharesQuery{CreatorUserID: state.creator, Limit: 20})
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				var item *ShareListItem
				for i := range page.Items {
					if page.Items[i].ShareID == created.ShareID {
						item = &page.Items[i]
						break
					}
				}
				if item == nil {
					t.Fatalf("expired share missing from list: %#v", page.Items)
				}
				if !item.IsRevoked {
					t.Fatalf("expired share must be listed as unavailable (is_revoked=true): %#v", item)
				}
				if item.ExpiresAt == nil {
					t.Fatalf("list expires_at = %#v", item.ExpiresAt)
				}
				if item.Preview != nil {
					t.Fatalf("unavailable share preview must be nil: %#v", item.Preview)
				}
			},
		},
		{
			Name: "revoke is idempotent and ownership-checked",
			Run: func(t *testing.T, repo Repository, state shareContractState) {
				created := create(t, repo, state, "Revoke Place", nil)
				if err := repo.Revoke(state.ctx, revokeShareCommand{ShareID: created.ShareID, CreatorUserID: state.other}); !errors.Is(err, ErrSharePermission) {
					t.Fatalf("revoke by other err = %v, want ErrSharePermission", err)
				}
				if err := repo.Revoke(state.ctx, revokeShareCommand{ShareID: "share_missing", CreatorUserID: state.creator}); !errors.Is(err, ErrShareNotFound) {
					t.Fatalf("revoke missing err = %v, want ErrShareNotFound", err)
				}
				if err := repo.Revoke(state.ctx, revokeShareCommand{ShareID: created.ShareID, CreatorUserID: state.creator}); err != nil {
					t.Fatalf("revoke: %v", err)
				}
				if err := repo.Revoke(state.ctx, revokeShareCommand{ShareID: created.ShareID, CreatorUserID: state.creator}); err != nil {
					t.Fatalf("revoke again must be idempotent: %v", err)
				}
				if _, err := repo.Get(state.ctx, getShareQuery{ShareCode: created.ShareCode}); !errors.Is(err, ErrShareNotFound) {
					t.Fatalf("get revoked err = %v, want ErrShareNotFound", err)
				}
				if shareID, found, err := repo.ResolveShareIDByCode(state.ctx, created.ShareCode); err != nil || !found || shareID != created.ShareID {
					t.Fatalf("resolve revoked = %q found:%v err:%v", shareID, found, err)
				}
				page, err := repo.List(state.ctx, listSharesQuery{CreatorUserID: state.creator, Limit: 20})
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				for _, item := range page.Items {
					if item.ShareID != created.ShareID {
						continue
					}
					if !item.IsRevoked {
						t.Fatalf("revoked share list item = %#v", item)
					}
					if item.Preview != nil {
						t.Fatalf("revoked share preview must be nil: %#v", item.Preview)
					}
				}
			},
		},
		{
			Name: "list is paginated and scoped to creator",
			Run: func(t *testing.T, repo Repository, state shareContractState) {
				for i := 0; i < 5; i++ {
					create(t, repo, state, "List Place", nil)
				}
				seen := map[string]bool{}
				cursor := ""
				pages := 0
				for {
					page, err := repo.List(state.ctx, listSharesQuery{CreatorUserID: state.creator, Limit: 2, Cursor: cursor})
					if err != nil {
						t.Fatalf("page %d: %v", pages, err)
					}
					if len(page.Items) > 2 {
						t.Fatalf("page %d size = %d", pages, len(page.Items))
					}
					for _, item := range page.Items {
						if seen[item.ShareID] {
							t.Fatalf("duplicate share across pages: %s", item.ShareID)
						}
						seen[item.ShareID] = true
					}
					pages++
					if page.NextCursor == nil {
						break
					}
					cursor = *page.NextCursor
					if pages > 5 {
						t.Fatal("pagination did not terminate")
					}
				}
				if len(seen) != 5 {
					t.Fatalf("seen %d shares, want 5", len(seen))
				}
				if _, err := repo.List(state.ctx, listSharesQuery{CreatorUserID: state.creator, Limit: 2, Cursor: "not-a-cursor"}); !errors.Is(err, ErrInvalidShareRequest) {
					t.Fatalf("invalid cursor err = %v, want ErrInvalidShareRequest", err)
				}
				otherPage, err := repo.List(state.ctx, listSharesQuery{CreatorUserID: state.other, Limit: 20})
				if err != nil {
					t.Fatalf("list other: %v", err)
				}
				if len(otherPage.Items) != 0 {
					t.Fatalf("other user shares = %#v, want empty", otherPage.Items)
				}
			},
		},
	})
}
