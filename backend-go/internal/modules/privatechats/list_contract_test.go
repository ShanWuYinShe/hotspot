package privatechats

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
	"hotspot/backend-go/internal/platform/types"
)

// listContractState 契约用例上下文：内存实现无需预置数据；真实库实现需要先
// 插入 users 行（FK 约束），因此准备逻辑放在各 ContractImpl.New 内。
type listContractState struct {
	ctx     context.Context
	viewer  string
	targets []string
}

// TestPrivateChatsListContract 是「双实现契约一致性」基建的首个接入示例：
// 同一组用例分别跑内存实现与真实库实现，任何行为漂移都会以「实现名 + 用例名」
// 定位（本次下推分页后，两实现的排序/游标/limit 语义必须一致）。
func TestPrivateChatsListContract(t *testing.T) {
	viewer := "it_contract_viewer"
	targets := []string{"it_contract_a", "it_contract_b", "it_contract_c"}

	impls := []testdb.ContractImpl[Repository, listContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, listContractState, func()) {
				return NewInMemoryRepository(), listContractState{ctx: context.Background(), viewer: viewer, targets: targets}, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, listContractState, func()) {
				testdb.SkipIfShort(t)
				factory := testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				for _, userID := range append([]string{viewer}, targets...) {
					testdb.InsertUser(t, txFactory, userID, userID)
				}
				_ = factory
				return NewPostgresRepository(txFactory), listContractState{ctx: context.Background(), viewer: viewer, targets: targets}, cleanup
			},
		},
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, listContractState]{
		{
			Name: "empty list",
			Run: func(t *testing.T, repo Repository, state listContractState) {
				items, next, err := repo.List(state.ctx, state.viewer, ListOptions{Limit: 10})
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(items) != 0 || next != "" {
					t.Fatalf("empty list: items=%d next=%q", len(items), next)
				}
			},
		},
		{
			Name: "create and list",
			Run: func(t *testing.T, repo Repository, state listContractState) {
				if _, err := repo.GetOrCreate(state.ctx, state.viewer, state.targets[0]); err != nil {
					t.Fatalf("get_or_create: %v", err)
				}
				items, _, err := repo.List(state.ctx, state.viewer, ListOptions{Limit: 10})
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(items) != 1 || items[0].TargetID != state.targets[0] {
					t.Fatalf("unexpected items: %+v", items)
				}
				if items[0].LastActivityAt.IsZero() {
					t.Fatal("LastActivityAt must be set (message time or created_at fallback)")
				}
			},
		},
		{
			Name: "get_or_create is idempotent",
			Run: func(t *testing.T, repo Repository, state listContractState) {
				for i := 0; i < 2; i++ {
					if _, err := repo.GetOrCreate(state.ctx, state.viewer, state.targets[0]); err != nil {
						t.Fatalf("get_or_create %d: %v", i, err)
					}
				}
				items, _, err := repo.List(state.ctx, state.viewer, ListOptions{Limit: 10})
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(items) != 1 {
					t.Fatalf("duplicate chats: %+v", items)
				}
			},
		},
		{
			Name: "pagination is stable and disjoint",
			Run: func(t *testing.T, repo Repository, state listContractState) {
				for _, target := range state.targets {
					if _, err := repo.GetOrCreate(state.ctx, state.viewer, target); err != nil {
						t.Fatalf("get_or_create %s: %v", target, err)
					}
				}
				page1, next, err := repo.List(state.ctx, state.viewer, ListOptions{Limit: 2})
				if err != nil {
					t.Fatalf("page1: %v", err)
				}
				if len(page1) != 2 || next == "" {
					t.Fatalf("page1: n=%d next=%q", len(page1), next)
				}
				page2, next2, err := repo.List(state.ctx, state.viewer, ListOptions{Limit: 2, Cursor: next})
				if err != nil {
					t.Fatalf("page2: %v", err)
				}
				if len(page2) != 1 || next2 != "" {
					t.Fatalf("page2: n=%d next=%q", len(page2), next2)
				}
				seen := map[string]bool{}
				for _, item := range append(page1, page2...) {
					if seen[item.TargetID] {
						t.Fatalf("duplicate across pages: %s", item.TargetID)
					}
					seen[item.TargetID] = true
				}
				if len(seen) != len(state.targets) {
					t.Fatalf("pages covered %d targets want %d", len(seen), len(state.targets))
				}
			},
		},
		{
			Name: "limit is clamped to MaxPageSize",
			Run: func(t *testing.T, repo Repository, state listContractState) {
				for _, target := range state.targets {
					if _, err := repo.GetOrCreate(state.ctx, state.viewer, target); err != nil {
						t.Fatalf("get_or_create %s: %v", target, err)
					}
				}
				items, _, err := repo.List(state.ctx, state.viewer, ListOptions{Limit: 100000})
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(items) != len(state.targets) {
					t.Fatalf("n=%d want %d", len(items), len(state.targets))
				}
				if len(items) > types.MaxPageSize {
					t.Fatalf("limit not clamped: %d > %d", len(items), types.MaxPageSize)
				}
			},
		},
		{
			Name: "pinned chats sort first",
			Run: func(t *testing.T, repo Repository, state listContractState) {
				if _, err := repo.GetOrCreate(state.ctx, state.viewer, state.targets[0]); err != nil {
					t.Fatalf("get_or_create a: %v", err)
				}
				if _, err := repo.GetOrCreate(state.ctx, state.viewer, state.targets[1]); err != nil {
					t.Fatalf("get_or_create b: %v", err)
				}
				pinned := true
				if err := repo.UpdateSettings(state.ctx, state.viewer, state.targets[0], UpdateChatSettings{IsPinned: &pinned}); err != nil {
					t.Fatalf("pin: %v", err)
				}
				items, _, err := repo.List(state.ctx, state.viewer, ListOptions{Limit: 10})
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(items) != 2 || !items[0].IsPinned || items[0].TargetID != state.targets[0] {
					t.Fatalf("pinned first: %+v", items)
				}
			},
		},
		{
			Name: "invalid cursor is rejected",
			Run: func(t *testing.T, repo Repository, state listContractState) {
				if _, _, err := repo.List(state.ctx, state.viewer, ListOptions{Limit: 10, Cursor: "not-a-valid-cursor"}); !errors.Is(err, privatemessages.ErrInvalidCursor) {
					t.Fatalf("invalid cursor err=%v want ErrInvalidCursor", err)
				}
			},
		},
	})
}

var _ = persistence.ErrNotImplemented
