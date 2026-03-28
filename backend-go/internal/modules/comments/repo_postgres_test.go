package comments

import (
	"context"
	"errors"
	"sort"
	"testing"

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

type fallbackRepo struct{ used bool }

func (r *fallbackRepo) Create(context.Context, string, CreateCommentRequest) (CommentItem, error) {
	r.used = true
	return CommentItem{}, nil
}

func (r *fallbackRepo) Delete(context.Context, string) error {
	r.used = true
	return nil
}

func (r *fallbackRepo) List(context.Context, string) (PaginatedComments, error) {
	r.used = true
	return PaginatedComments{}, nil
}

func (r *fallbackRepo) Get(context.Context, string) (CommentItem, bool, error) {
	r.used = true
	return CommentItem{}, false, nil
}

func (r *fallbackRepo) ListReplies(context.Context, string, int, string) (PaginatedComments, error) {
	r.used = true
	return PaginatedComments{}, nil
}

func (r *fallbackRepo) Like(context.Context, string, string) error {
	r.used = true
	return nil
}

func (r *fallbackRepo) Unlike(context.Context, string, string) error {
	r.used = true
	return nil
}

func (r *fallbackRepo) ListLikes(context.Context, string, int, string) ([]CommentLikeItem, string, error) {
	r.used = true
	return nil, "", nil
}

type backendStub struct {
	comments map[string][]CommentItem
	execErrs map[string]error
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
			Content:      args[4].(string),
			CreatedAt:    args[5].(string),
			LikesCount:   args[6].(int),
			RepliesCount: args[7].(int),
			User:         UserInfo{UserID: args[2].(string), Nickname: args[2].(string)},
			MyState:      &MyCommentState{IsLiked: false},
		}
		if parentID, ok := args[3].(string); ok {
			item.ParentID = parentID
		}
		momentID := args[1].(string)
		b.comments[momentID] = append(b.comments[momentID], item)
	case queryDeleteComment:
		commentID := args[0].(string)
		for momentID, items := range b.comments {
			for idx, item := range items {
				if item.CommentID != commentID {
					continue
				}
				b.comments[momentID] = append(items[:idx], items[idx+1:]...)
				break
			}
		}
	}
	return nil
}

func (b *backendStub) QueryRow(context.Context, string, ...any) persistence.Row {
	return rowValues{err: persistence.ErrNotImplemented}
}

func (b *backendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if query != queryListComments {
		return &persistence.InMemoryRows{}, nil
	}
	momentID := args[0].(string)
	items := append([]CommentItem(nil), b.comments[momentID]...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].CommentID > items[j].CommentID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	rows := make([][]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, []any{item.CommentID, item.User.UserID, item.ParentID, item.Content, item.CreatedAt, item.LikesCount, item.RepliesCount})
	}
	return &rowsValues{rows: rows}, nil
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
	fallback := &fallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)
	created, err := repo.Create(context.Background(), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "nice", ParentID: "comment_0"})
	if err != nil || created.CommentID == "" || created.Content != "nice" {
		t.Fatalf("unexpected created comment: %#v err=%v", created, err)
	}
	if created.ParentID != "" || len(created.Mentions) != 0 {
		t.Fatalf("postgres create should not return non-persisted fields: %#v", created)
	}
	list, err := repo.List(context.Background(), "moment_1")
	if err != nil || len(list.Items) != 1 || list.Items[0].CommentID != created.CommentID || list.Items[0].ParentID != "comment_0" {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryCreateReturnsErrorOnExecFailure(t *testing.T) {
	backend := newBackendStub()
	backend.execErrs[queryInsertComment] = errors.New("write failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &fallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)
	_, err := repo.Create(context.Background(), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "nice"})
	if err == nil {
		t.Fatal("expected create error")
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}
