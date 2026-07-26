package comments

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type rowValues struct {
	values []any
	err    error
}

func (r rowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *int:
			*d = r.values[i].(int)
		case *time.Time:
			switch v := r.values[i].(type) {
			case string:
				parsed, _ := time.Parse(time.RFC3339, v)
				*d = parsed
			case time.Time:
				*d = v
			}
		case *bool:
			if v, ok := r.values[i].(bool); ok {
				*d = v
			}
		}
	}
	return nil
}

type rowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *rowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *rowsValues) Scan(dest ...any) error {
	row := rowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *rowsValues) Close() error { return nil }
func (r *rowsValues) Err() error   { return r.err }

type backendStub struct {
	comments map[string][]CommentItem
	execErrs map[string]error
}

type backendCommentCursor struct {
	commentID string
	likedAt   string
	userID    string
}

func newBackendStub() *backendStub {
	return &backendStub{comments: make(map[string][]CommentItem), execErrs: make(map[string]error)}
}

func (b *backendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	switch query {
	case queryInsertComment:
		item := CommentItem{
			CommentID:    args[0].(string),
			MomentID:     args[1].(string),
			Content:      args[4].(string),
			CreatedAt:    args[5].(string),
			LikesCount:   args[6].(int),
			RepliesCount: args[7].(int),
			User:         UserInfo{UserID: args[2].(string), Nickname: args[2].(string)},
			MyState:      &MyCommentState{},
		}
		if parentID, ok := args[3].(string); ok {
			item.ParentID = parentID
		}
		momentID := args[1].(string)
		b.comments[momentID] = append(b.comments[momentID], item)
	case querySoftDeleteComment:
		commentID := args[0].(string)
		for momentID, items := range b.comments {
			for idx, item := range items {
				if item.CommentID != commentID {
					continue
				}
				item.IsDeleted = true
				item.Content = ""
				b.comments[momentID][idx] = item
				break
			}
		}
	}
	return nil
}

func (b *backendStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	switch query {
	case queryGetComment:
		commentID := args[0].(string)
		for _, items := range b.comments {
			for _, item := range items {
				if item.CommentID == commentID {
					return rowValues{values: []any{item.CommentID, item.MomentID, item.User.UserID, item.User.Nickname, item.ParentID, item.Content, item.CreatedAt, item.LikesCount, item.RepliesCount, item.IsDeleted}}
				}
			}
		}
		return rowValues{err: persistence.ErrNotImplemented}
	default:
		return rowValues{err: persistence.ErrNotImplemented}
	}
}

func (b *backendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	switch query {
	case queryListComments:
		momentID := args[0].(string)
		cursor := backendCommentCursor{}
		if args[1] != nil {
			cursor.likedAt = args[1].(interface{ Format(string) string }).Format("2006-01-02T15:04:05Z07:00")
			cursor.commentID = args[2].(string)
		}
		limit := args[3].(int)
		items := append([]CommentItem(nil), b.comments[momentID]...)
		filtered := items[:0]
		for _, item := range items {
			if item.ParentID == "" {
				filtered = append(filtered, item)
			}
		}
		items = filtered
		sort.Slice(items, func(i, j int) bool {
			if items[i].CreatedAt == items[j].CreatedAt {
				return items[i].CommentID > items[j].CommentID
			}
			return items[i].CreatedAt > items[j].CreatedAt
		})
		if cursor.commentID != "" {
			after := items[:0]
			for _, item := range items {
				if item.CreatedAt < cursor.likedAt || (item.CreatedAt == cursor.likedAt && item.CommentID < cursor.commentID) {
					after = append(after, item)
				}
			}
			items = after
		}
		if len(items) > limit {
			items = items[:limit]
		}
		rows := make([][]any, 0, len(items))
		for _, item := range items {
			rows = append(rows, []any{item.CommentID, item.User.UserID, item.User.Nickname, item.ParentID, item.Content, item.CreatedAt, item.LikesCount, item.RepliesCount, item.IsDeleted})
		}
		return &rowsValues{rows: rows}, nil
	case queryListCommentReplies:
		parentID := args[0].(string)
		cursor := backendCommentCursor{}
		if args[1] != nil {
			cursor.likedAt = args[1].(interface{ Format(string) string }).Format("2006-01-02T15:04:05Z07:00")
			cursor.commentID = args[2].(string)
		}
		limit := args[3].(int)
		items := make([]CommentItem, 0)
		for _, momentItems := range b.comments {
			for _, item := range momentItems {
				if item.ParentID == parentID {
					items = append(items, item)
				}
			}
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].CreatedAt == items[j].CreatedAt {
				return items[i].CommentID < items[j].CommentID
			}
			return items[i].CreatedAt < items[j].CreatedAt
		})
		if cursor.commentID != "" {
			filtered := items[:0]
			for _, item := range items {
				if item.CreatedAt > cursor.likedAt || (item.CreatedAt == cursor.likedAt && item.CommentID > cursor.commentID) {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		if len(items) > limit {
			items = items[:limit]
		}
		rows := make([][]any, 0, len(items))
		for _, item := range items {
			rows = append(rows, []any{item.CommentID, item.User.UserID, item.User.Nickname, item.ParentID, item.Content, item.CreatedAt, item.LikesCount, item.RepliesCount, item.IsDeleted})
		}
		return &rowsValues{rows: rows}, nil
	default:
		return &persistence.InMemoryRows{}, nil
	}
}

func TestNewPersistenceRepositoryUsesPostgresWhenDSNConfigured(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	repo := NewPersistenceRepository(factory)
	if _, ok := repo.impl.(*PostgresRepository); !ok {
		t.Fatalf("expected postgres impl, got %T", repo.impl)
	}
}

func TestPostgresRepositoryWritesAndReadsQueryableDB(t *testing.T) {
	backend := newBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)
	created, err := repo.Create(context.Background(), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "nice"})
	if err != nil || created.CommentID == "" || created.Content != "nice" {
		t.Fatalf("unexpected created comment: %#v err=%v", created, err)
	}
	if created.ParentID != "" || len(created.Mentions) != 0 {
		t.Fatalf("postgres create should not return non-persisted fields: %#v", created)
	}
	list, err := repo.List(context.Background(), "moment_1", 20, "")
	if err != nil || len(list.Items) != 1 || list.Items[0].CommentID != created.CommentID {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}
}

func TestPostgresRepositoryCreateReturnsErrorOnExecFailure(t *testing.T) {
	backend := newBackendStub()
	backend.execErrs[queryInsertComment] = errors.New("write failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)
	_, err := repo.Create(context.Background(), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "nice"})
	if err == nil {
		t.Fatal("expected create error")
	}
}

func TestPostgresRepositoryGetReturnsMomentID(t *testing.T) {
	backend := newBackendStub()
	backend.comments["moment_1"] = []CommentItem{{
		CommentID:    "comment_1",
		MomentID:     "moment_1",
		Content:      "hello",
		CreatedAt:    "2026-03-28T10:00:00Z",
		LikesCount:   1,
		RepliesCount: 0,
		User:         UserInfo{UserID: "user_alice", Nickname: "user_alice"},
		MyState:      &MyCommentState{},
	}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	item, found, err := repo.Get(context.Background(), "comment_1")
	if err != nil || !found {
		t.Fatalf("unexpected get result: item=%#v found=%v err=%v", item, found, err)
	}
	if item.MomentID != "moment_1" {
		t.Fatalf("expected moment_id to be populated, got %#v", item)
	}
}

func TestPostgresRepositoryListRepliesUsesCursor(t *testing.T) {
	backend := newBackendStub()
	backend.comments["moment_1"] = []CommentItem{
		{CommentID: "reply_3", MomentID: "moment_1", ParentID: "comment_0", Content: "c", CreatedAt: "2026-03-28T10:02:00Z", User: UserInfo{UserID: "u3", Nickname: "u3"}, MyState: &MyCommentState{}},
		{CommentID: "reply_2", MomentID: "moment_1", ParentID: "comment_0", Content: "b", CreatedAt: "2026-03-28T10:01:00Z", User: UserInfo{UserID: "u2", Nickname: "u2"}, MyState: &MyCommentState{}},
		{CommentID: "reply_1", MomentID: "moment_1", ParentID: "comment_0", Content: "a", CreatedAt: "2026-03-28T10:00:00Z", User: UserInfo{UserID: "u1", Nickname: "u1"}, MyState: &MyCommentState{}},
	}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	first, err := repo.ListReplies(context.Background(), "comment_0", 2, "")
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Items) != 2 || first.Items[0].CommentID != "reply_1" || first.Items[1].CommentID != "reply_2" || first.NextCursor == "" {
		t.Fatalf("unexpected first page: %#v", first)
	}

	second, err := repo.ListReplies(context.Background(), "comment_0", 2, first.NextCursor)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].CommentID != "reply_3" || second.NextCursor != "" {
		t.Fatalf("unexpected second page: %#v", second)
	}
}
