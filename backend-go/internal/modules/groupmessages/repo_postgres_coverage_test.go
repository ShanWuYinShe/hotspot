package groupmessages

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// gmStub 提供 QueryRow + Exec 的 stub。
type gmStub struct {
	rowValues []any
	execErr   map[string]error
	execs     int
}

func (s *gmStub) QueryRow(_ context.Context, _ string, _ ...any) persistence.Row {
	return gmRow{values: s.rowValues}
}

func (s *gmStub) Query(_ context.Context, _ string, _ ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}

func (s *gmStub) Exec(_ context.Context, query string, _ ...any) error {
	s.execs++
	return s.execErr[query]
}

type gmRow struct {
	values []any
}

func (r gmRow) Scan(dest ...any) error {
	if r.values == nil {
		return errNoRows
	}
	for i := range dest {
		if i >= len(r.values) {
			break
		}
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *time.Time:
			*d = r.values[i].(time.Time)
		case **time.Time:
			if r.values[i] == nil {
				*d = nil
			} else if v, ok := r.values[i].(time.Time); ok {
				*d = &v
			}
		case *bool:
			*d = r.values[i].(bool)
		case *int:
			*d = r.values[i].(int)
		}
	}
	return nil
}

func newGmFactory(backend persistence.Queryable) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresSendAndUpdateAndRecall(t *testing.T) {
	backend := &gmStub{
		rowValues: []any{"g1", "u1", "text", false, time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")},
	}
	repo := NewPostgresRepository(newGmFactory(backend))
	ctx := context.Background()

	sent, err := repo.Send(ctx, SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u1", Type: "text", Content: "hi"})
	if err != nil || sent.MessageID == "" {
		t.Fatalf("send: %#v err=%v", sent, err)
	}

	updated, err := repo.Update(ctx, UpdateGroupMessageRequest{MessageID: sent.MessageID, UserID: "u1", Content: "hi2"})
	if err != nil || updated.GroupID != "g1" || updated.Content != "hi2" {
		t.Fatalf("update: %#v err=%v", updated, err)
	}

	recalled, err := repo.Recall(ctx, 2*time.Minute, RecallGroupMessageRequest{MessageID: sent.MessageID, UserID: "u1"})
	if err != nil || recalled.MessageID != sent.MessageID {
		t.Fatalf("recall: %#v err=%v", recalled, err)
	}
	// Update/Recall 已改走 QueryRow（RETURNING 行数检测），Exec 仅剩 Send。
	if backend.execs < 1 {
		t.Fatalf("expected execs, got %d", backend.execs)
	}
}

func TestPostgresUpdateRejectsRecalled(t *testing.T) {
	backend := &gmStub{
		rowValues: []any{"g1", "u1", "text", true, time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")},
	}
	repo := NewPostgresRepository(newGmFactory(backend))
	if _, err := repo.Update(context.Background(), UpdateGroupMessageRequest{MessageID: "m1", UserID: "u1", Content: "x"}); err != ErrMessageRecalled {
		t.Fatalf("expected recalled error, got %v", err)
	}
}

func TestPostgresUpdateRejectsNonText(t *testing.T) {
	backend := &gmStub{
		rowValues: []any{"g1", "u1", "image", false, time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")},
	}
	repo := NewPostgresRepository(newGmFactory(backend))
	if _, err := repo.Update(context.Background(), UpdateGroupMessageRequest{MessageID: "m1", UserID: "u1", Content: "x"}); err != ErrMessageNotEditable {
		t.Fatalf("expected not editable, got %v", err)
	}
}

func TestPostgresGroupMessagesNotConfigured(t *testing.T) {
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	ctx := context.Background()
	if _, err := repo.Send(ctx, SendGroupMessageRequest{}); err != persistence.ErrNotImplemented {
		t.Fatalf("send: %v", err)
	}
	if _, err := repo.Update(ctx, UpdateGroupMessageRequest{}); err != persistence.ErrNotImplemented {
		t.Fatalf("update: %v", err)
	}
	if _, err := repo.Recall(ctx, 2*time.Minute, RecallGroupMessageRequest{}); err != persistence.ErrNotImplemented {
		t.Fatalf("recall: %v", err)
	}
}

func TestPostgresQueryMethods(t *testing.T) {
	ctx := context.Background()

	// 1. GetReadCount
	backendCount := &gmStub{rowValues: []any{5}}
	repoCount := NewPostgresRepository(newGmFactory(backendCount))
	count, err := repoCount.GetReadCount(ctx, "g1", "m1")
	if err != nil || count != 5 {
		t.Fatalf("GetReadCount: %d err=%v", count, err)
	}

	// 2. LatestMessageID
	backendLatest := &gmStub{rowValues: []any{"m_latest"}}
	repoLatest := NewPostgresRepository(newGmFactory(backendLatest))
	latest, err := repoLatest.LatestMessageID(ctx, "g1")
	if err != nil || latest != "m_latest" {
		t.Fatalf("LatestMessageID: %q err=%v", latest, err)
	}

	// 3. IsMember
	backendMember := &gmStub{rowValues: []any{1}}
	repoMember := NewPostgresRepository(newGmFactory(backendMember))
	isMember, err := repoMember.IsMember(ctx, "g1", "u1")
	if err != nil || !isMember {
		t.Fatalf("IsMember: %v err=%v", isMember, err)
	}

	// 4. MessageGroupID
	backendGroupID := &gmStub{rowValues: []any{"g1"}}
	repoGroupID := NewPostgresRepository(newGmFactory(backendGroupID))
	gid, found, err := repoGroupID.MessageGroupID(ctx, "m1")
	if err != nil || !found || gid != "g1" {
		t.Fatalf("MessageGroupID: %q found=%v err=%v", gid, found, err)
	}

	// 5. MessageExists
	backendExists := &gmStub{rowValues: []any{"u_sender", "nick", "avatar", "text", "hi", "", "2026-01-01T00:00:00Z"}}
	repoExists := NewPostgresRepository(newGmFactory(backendExists))
	found, sender, err := repoExists.MessageExists(ctx, "m1")
	if err != nil || !found || sender != "u_sender" {
		t.Fatalf("MessageExists: found=%v sender=%q err=%v", found, sender, err)
	}

	// 6. GetGroupName
	backendName := &gmStub{rowValues: []any{"群名"}}
	repoName := NewPostgresRepository(newGmFactory(backendName))
	name, found, err := repoName.GetGroupName(ctx, "g1")
	if err != nil || !found || name != "群名" {
		t.Fatalf("GetGroupName: %q found=%v err=%v", name, found, err)
	}
}

var errNoRows = errors.New("no rows")
