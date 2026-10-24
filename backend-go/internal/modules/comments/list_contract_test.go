package comments

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/testdb"
)

// commentsContractState 契约用例上下文：PG 侧 comments 有 moments/users 外键，
// 预置用户与瞬间行；内存实现无需。
type commentsContractState struct {
	ctx       context.Context
	user      string
	momentID  string
	momentID2 string
}

// TestCommentsRepositoryContract 接入双实现契约一致性基建：
// 创建/顶层列表 keyset 分页/回复递增父评论 replies_count/软删（含重复删除幂等）/
// 回复列表分页/批量取每父评论最新回复。游标必须两实现同格式（AES-GCM 不透明），
// 非法游标统一 ErrInvalidCursor。
func TestCommentsRepositoryContract(t *testing.T) {
	user := "it_comments_contract_user"
	momentID := "it_comments_contract_moment"
	momentID2 := "it_comments_contract_moment_2"

	impls := []testdb.ContractImpl[Repository, commentsContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, commentsContractState, func()) {
				return NewInMemoryRepository(), commentsContractState{ctx: context.Background(), user: user, momentID: momentID, momentID2: momentID2}, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, commentsContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				testdb.InsertUser(t, txFactory, user, "nick_"+user)
				testdb.InsertMoment(t, txFactory, momentID, user)
				testdb.InsertMoment(t, txFactory, momentID2, user)
				return NewPostgresRepository(txFactory), commentsContractState{ctx: context.Background(), user: user, momentID: momentID, momentID2: momentID2}, cleanup
			},
		},
	}

	create := func(t *testing.T, repo Repository, state commentsContractState, momentID, parentID, content string) CommentItem {
		t.Helper()
		item, err := repo.Create(context.Background(), state.user, CreateCommentRequest{MomentID: momentID, Content: content, ParentID: parentID})
		if err != nil {
			t.Fatalf("create comment: %v", err)
		}
		if item.CommentID == "" {
			t.Fatal("created comment id must be non-empty")
		}
		return item
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, commentsContractState]{
		{
			Name: "create top-level comments and list them",
			Run: func(t *testing.T, repo Repository, state commentsContractState) {
				first := create(t, repo, state, state.momentID, "", "first")
				second := create(t, repo, state, state.momentID, "", "second")
				create(t, repo, state, state.momentID2, "", "other moment")
				page, err := repo.List(state.ctx, state.momentID, 10, "")
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(page.Items) != 2 {
					t.Fatalf("list = %d items, want 2", len(page.Items))
				}
				if page.NextCursor != nil {
					t.Fatal("unexpected next cursor")
				}
				seen := map[string]bool{}
				for _, item := range page.Items {
					seen[item.CommentID] = true
					if item.ParentID != nil {
						t.Fatalf("top-level comment must have nil parent_id: %#v", item.ParentID)
					}
					if item.Content == "" || item.User.Nickname == "" {
						t.Fatalf("comment content/nickname must be non-empty: %#v", item)
					}
					if _, err := time.Parse(time.RFC3339, item.CreatedAt); err != nil {
						t.Fatalf("created_at must be RFC3339: %q", item.CreatedAt)
					}
				}
				if !seen[first.CommentID] || !seen[second.CommentID] {
					t.Fatalf("list missing created comments: %#v", seen)
				}
			},
		},
		{
			Name: "reply increments parent replies_count",
			Run: func(t *testing.T, repo Repository, state commentsContractState) {
				parent := create(t, repo, state, state.momentID, "", "parent")
				reply := create(t, repo, state, state.momentID, parent.CommentID, "reply")
				if reply.ParentID == nil || *reply.ParentID != parent.CommentID {
					t.Fatalf("reply parent = %#v", reply.ParentID)
				}
				top, err := repo.List(state.ctx, state.momentID, 10, "")
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(top.Items) != 1 || top.Items[0].RepliesCount != 1 {
					t.Fatalf("top-level with reply = %#v", top.Items)
				}
				replies, err := repo.ListReplies(state.ctx, parent.CommentID, 10, "")
				if err != nil {
					t.Fatalf("list replies: %v", err)
				}
				if len(replies.Items) != 1 || replies.Items[0].CommentID != reply.CommentID {
					t.Fatalf("replies = %#v", replies.Items)
				}
			},
		},
		{
			Name: "soft delete reply decrements parent and is idempotent",
			Run: func(t *testing.T, repo Repository, state commentsContractState) {
				parent := create(t, repo, state, state.momentID, "", "parent")
				reply := create(t, repo, state, state.momentID, parent.CommentID, "reply")
				deleted, err := repo.SoftDelete(state.ctx, reply.CommentID)
				if err != nil || !deleted {
					t.Fatalf("soft delete reply = %v err=%v, want true", deleted, err)
				}
				parentAfter, found, err := repo.Get(state.ctx, parent.CommentID)
				if err != nil || !found {
					t.Fatalf("get parent: %#v err=%v", parentAfter, err)
				}
				if parentAfter.RepliesCount != 0 {
					t.Fatalf("parent replies_count = %d, want 0", parentAfter.RepliesCount)
				}
				deleted, err = repo.SoftDelete(state.ctx, reply.CommentID)
				if err != nil || deleted {
					t.Fatalf("second soft delete = %v err=%v, want false", deleted, err)
				}
				replies, err := repo.ListReplies(state.ctx, parent.CommentID, 10, "")
				if err != nil {
					t.Fatalf("list replies: %v", err)
				}
				if len(replies.Items) != 1 || !replies.Items[0].IsDeleted || replies.Items[0].Content != "" {
					t.Fatalf("deleted reply = %#v", replies.Items)
				}
				parentAfter, _, err = repo.Get(state.ctx, parent.CommentID)
				if err != nil {
					t.Fatalf("get parent again: %v", err)
				}
				if parentAfter.RepliesCount != 0 {
					t.Fatalf("parent replies_count after duplicate delete = %d, want 0 (idempotent)", parentAfter.RepliesCount)
				}
			},
		},
		{
			Name: "soft delete top-level clears content",
			Run: func(t *testing.T, repo Repository, state commentsContractState) {
				item := create(t, repo, state, state.momentID, "", "to delete")
				deleted, err := repo.SoftDelete(state.ctx, item.CommentID)
				if err != nil || !deleted {
					t.Fatalf("soft delete = %v err=%v", deleted, err)
				}
				got, found, err := repo.Get(state.ctx, item.CommentID)
				if err != nil || !found {
					t.Fatalf("get deleted: %#v err=%v", got, err)
				}
				if !got.IsDeleted || got.Content != "" {
					t.Fatalf("deleted top-level = %#v", got)
				}
				missing, err := repo.SoftDelete(state.ctx, "comment_missing")
				if err != nil || missing {
					t.Fatalf("soft delete missing = %v err=%v, want false", missing, err)
				}
			},
		},
		{
			Name: "top-level pagination is stable and disjoint",
			Run: func(t *testing.T, repo Repository, state commentsContractState) {
				for i := 0; i < 5; i++ {
					create(t, repo, state, state.momentID, "", "page")
				}
				seen := map[string]bool{}
				cursor := ""
				pages := 0
				for {
					page, err := repo.List(state.ctx, state.momentID, 2, cursor)
					if err != nil {
						t.Fatalf("page %d: %v", pages, err)
					}
					if len(page.Items) > 2 {
						t.Fatalf("page %d size = %d", pages, len(page.Items))
					}
					for _, item := range page.Items {
						if seen[item.CommentID] {
							t.Fatalf("duplicate comment across pages: %s", item.CommentID)
						}
						seen[item.CommentID] = true
					}
					pages++
					if page.NextCursor == nil {
						break
					}
					cursor = *page.NextCursor
					if pages > 6 {
						t.Fatal("pagination did not terminate")
					}
				}
				if len(seen) != 5 {
					t.Fatalf("seen %d comments, want 5", len(seen))
				}
				if _, err := repo.List(state.ctx, state.momentID, 2, "not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
					t.Fatalf("invalid list cursor err = %v, want ErrInvalidCursor", err)
				}
			},
		},
		{
			Name: "reply pagination is stable and disjoint",
			Run: func(t *testing.T, repo Repository, state commentsContractState) {
				parent := create(t, repo, state, state.momentID, "", "parent")
				for i := 0; i < 3; i++ {
					create(t, repo, state, state.momentID, parent.CommentID, "reply")
				}
				seen := map[string]bool{}
				cursor := ""
				pages := 0
				for {
					page, err := repo.ListReplies(state.ctx, parent.CommentID, 2, cursor)
					if err != nil {
						t.Fatalf("page %d: %v", pages, err)
					}
					if len(page.Items) > 2 {
						t.Fatalf("page %d size = %d", pages, len(page.Items))
					}
					for _, item := range page.Items {
						if seen[item.CommentID] {
							t.Fatalf("duplicate reply across pages: %s", item.CommentID)
						}
						seen[item.CommentID] = true
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
				if len(seen) != 3 {
					t.Fatalf("seen %d replies, want 3", len(seen))
				}
				if _, err := repo.ListReplies(state.ctx, parent.CommentID, 2, "not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
					t.Fatalf("invalid reply cursor err = %v, want ErrInvalidCursor", err)
				}
			},
		},
		{
			Name: "batch top replies per parent",
			Run: func(t *testing.T, repo Repository, state commentsContractState) {
				parentA := create(t, repo, state, state.momentID, "", "parent A")
				parentB := create(t, repo, state, state.momentID, "", "parent B")
				for i := 0; i < 3; i++ {
					create(t, repo, state, state.momentID, parentA.CommentID, "reply A")
				}
				for i := 0; i < 2; i++ {
					create(t, repo, state, state.momentID, parentB.CommentID, "reply B")
				}
				result, err := repo.ListTopRepliesByParentIDs(state.ctx, []string{parentA.CommentID, parentB.CommentID}, 2)
				if err != nil {
					t.Fatalf("batch top replies: %v", err)
				}
				if len(result[parentA.CommentID]) != 2 {
					t.Fatalf("parent A replies = %d, want 2", len(result[parentA.CommentID]))
				}
				if len(result[parentB.CommentID]) != 2 {
					t.Fatalf("parent B replies = %d, want 2", len(result[parentB.CommentID]))
				}
			},
		},
	})
}
