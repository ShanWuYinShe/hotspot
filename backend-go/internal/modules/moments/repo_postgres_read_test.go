package moments

import (
	"context"
	"errors"
	"sort"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
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
		case *float64:
			*d = r.values[i].(float64)
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

func (r *fallbackRepo) Create(context.Context, string, CreateMomentRequest) (MomentItem, error) {
	r.used = true
	return MomentItem{}, nil
}

func (r *fallbackRepo) Get(context.Context, string) (MomentItem, bool, error) {
	r.used = true
	return MomentItem{}, false, nil
}

func (r *fallbackRepo) ListMine(context.Context, string) (PaginatedMoments, error) {
	r.used = true
	return PaginatedMoments{}, nil
}

func (r *fallbackRepo) Exists(context.Context, string) (bool, error) {
	r.used = true
	return false, nil
}

func (r *fallbackRepo) IncrementCommentCount(context.Context, string) error {
	r.used = true
	return nil
}

func (r *fallbackRepo) Search(context.Context, SearchMomentsRequest) (PaginatedMoments, error) {
	r.used = true
	return PaginatedMoments{}, nil
}

func (r *fallbackRepo) Update(context.Context, string, EditMomentRequest) error {
	r.used = true
	return nil
}

func (r *fallbackRepo) Delete(context.Context, string) error {
	r.used = true
	return nil
}

type backendStub struct {
	moments      map[string]MomentItem
	blockedTags  map[string]struct{}
	execErrs     map[string]error
	queryErr     error
	queryRowErrs map[string]error
}

func newBackendStub() *backendStub {
	return &backendStub{moments: make(map[string]MomentItem), blockedTags: make(map[string]struct{}), execErrs: make(map[string]error), queryRowErrs: make(map[string]error)}
}

func (b *backendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	switch query {
	case queryInsertMoment:
		item := MomentItem{
			MomentID:      args[0].(string),
			Description:   args[2].(string),
			CreatedAt:     args[3].(string),
			LikesCount:    args[4].(int),
			CommentsCount: args[5].(int),
			SharesCount:   args[6].(int),
			User:          UserInfo{UserID: args[1].(string), Nickname: args[1].(string)},
			PrivacyLevel:  args[10].(string),
			MyState:       &MyMomentState{IsLiked: false, IsFavorited: false},
		}
		item.Location.LocationName = args[7].(string)
		item.Location.Point.Latitude = args[8].(float64)
		item.Location.Point.Longitude = args[9].(float64)
		b.moments[item.MomentID] = item
	case queryInsertMomentTagAssignment:
		momentID := args[0].(string)
		item := b.moments[momentID]
		tag := args[1].(string)
		item.Tags = append(item.Tags, tag)
		sort.Strings(item.Tags)
		b.moments[momentID] = item
	case queryIncrementMomentCommentCount:
		momentID := args[0].(string)
		item := b.moments[momentID]
		item.CommentsCount++
		b.moments[momentID] = item
	}
	return nil
}

func (b *backendStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	if query != queryGetMoment {
		return rowValues{err: persistence.ErrNotImplemented}
	}
	if err := b.queryRowErrs[query]; err != nil {
		return rowValues{err: err}
	}
	item, ok := b.moments[args[0].(string)]
	if !ok {
		return rowValues{err: pgx.ErrNoRows}
	}
	return rowValues{values: []any{item.MomentID, item.User.UserID, item.Description, item.CreatedAt, item.LikesCount, item.CommentsCount, item.SharesCount, item.Location.LocationName, item.Location.Point.Latitude, item.Location.Point.Longitude, item.PrivacyLevel}}
}

func (b *backendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if b.queryErr != nil {
		return nil, b.queryErr
	}
	switch query {
	case queryListMyMoments:
		userID := args[0].(string)
		items := make([]MomentItem, 0)
		for _, item := range b.moments {
			if item.User.UserID == userID {
				items = append(items, item)
			}
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].CreatedAt == items[j].CreatedAt {
				return items[i].MomentID > items[j].MomentID
			}
			return items[i].CreatedAt > items[j].CreatedAt
		})
		rows := make([][]any, 0, len(items))
		for _, item := range items {
			rows = append(rows, []any{item.MomentID, item.User.UserID, item.Description, item.CreatedAt, item.LikesCount, item.CommentsCount, item.SharesCount, item.Location.LocationName, item.Location.Point.Latitude, item.Location.Point.Longitude, item.PrivacyLevel})
		}
		return &rowsValues{rows: rows}, nil
	case queryFilterAllowedMomentTags:
		input := args[0].([]string)
		rows := make([][]any, 0, len(input))
		for _, tag := range input {
			if _, blocked := b.blockedTags[tag]; blocked {
				continue
			}
			rows = append(rows, []any{tag})
		}
		return &rowsValues{rows: rows}, nil
	case queryListMomentTagsByMomentID:
		momentID := args[0].(string)
		item, ok := b.moments[momentID]
		if !ok {
			return &rowsValues{rows: nil}, nil
		}
		rows := make([][]any, 0, len(item.Tags))
		for _, tag := range item.Tags {
			rows = append(rows, []any{tag})
		}
		return &rowsValues{rows: rows}, nil
	default:
		return &persistence.InMemoryRows{}, nil
	}
}

func TestPostgresRepositoryReadsQueryableDB(t *testing.T) {
	backend := newBackendStub()
	backend.moments["moment_1"] = MomentItem{MomentID: "moment_1", Description: "hello", CreatedAt: "2026-03-20T00:00:00Z", LikesCount: 1, CommentsCount: 2, SharesCount: 3, User: UserInfo{UserID: "user_alice", Nickname: "user_alice"}, PrivacyLevel: "public", Location: Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: 35.0, Longitude: 112.0}, LocationName: "南天门"}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &fallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)
	item, ok, err := repo.Get(context.Background(), "moment_1")
	if err != nil || !ok || item.MomentID != "moment_1" || item.Description != "hello" {
		t.Fatalf("unexpected item: %#v ok=%v err=%v", item, ok, err)
	}
	exists, err := repo.Exists(context.Background(), "moment_1")
	if err != nil || !exists {
		t.Fatalf("unexpected exists result: exists=%v err=%v", exists, err)
	}
	list, err := repo.ListMine(context.Background(), "user_alice")
	if err != nil || len(list.Items) != 1 || list.Items[0].MomentID != "moment_1" {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryWritesQueryableDB(t *testing.T) {
	backend := newBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &fallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)
	created, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Tags: []string{" travel ", "hotspot", "travel"}, Location: Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: 35.0, Longitude: 112.0}, LocationName: "南天门"}})
	if err != nil || created.MomentID == "" || created.Description != "hello" {
		t.Fatalf("unexpected created item: %#v err=%v", created, err)
	}
	if len(created.Tags) != 2 || created.Tags[0] != "travel" || created.Tags[1] != "hotspot" {
		t.Fatalf("unexpected persisted tags: %#v", created.Tags)
	}
	if len(created.Mentions) != 0 {
		t.Fatalf("postgres create should not return non-persisted mentions: %#v", created)
	}
	if err := repo.IncrementCommentCount(context.Background(), created.MomentID); err != nil {
		t.Fatalf("increment comment count: %v", err)
	}
	opened, ok, err := repo.Get(context.Background(), created.MomentID)
	if err != nil || !ok || opened.CommentsCount != 1 {
		t.Fatalf("unexpected opened item: %#v ok=%v err=%v", opened, ok, err)
	}
	list, err := repo.ListMine(context.Background(), "user_alice")
	if err != nil || len(list.Items) != 1 || list.Items[0].MomentID != created.MomentID || list.Items[0].CommentsCount != 1 {
		t.Fatalf("unexpected list after write: %#v err=%v", list, err)
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryGetReturnsNotFoundOnNoRows(t *testing.T) {
	backend := newBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &fallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)
	_, ok, err := repo.Get(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ok {
		t.Fatal("expected not found")
	}
	exists, err := repo.Exists(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected nil error from exists, got %v", err)
	}
	if exists {
		t.Fatal("expected missing moment to report exists=false")
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryCreateReturnsErrorOnExecFailure(t *testing.T) {
	backend := newBackendStub()
	backend.execErrs[queryInsertMoment] = errors.New("write failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &fallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)
	_, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Location: Location{LocationName: "南天门"}})
	if err == nil {
		t.Fatal("expected create error")
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryIncrementCommentCountReturnsErrorOnExecFailure(t *testing.T) {
	backend := newBackendStub()
	backend.execErrs[queryIncrementMomentCommentCount] = errors.New("update failed")
	backend.moments["moment_1"] = MomentItem{MomentID: "moment_1", User: UserInfo{UserID: "user_alice"}}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &fallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)
	if err := repo.IncrementCommentCount(context.Background(), "moment_1"); err == nil {
		t.Fatal("expected increment error")
	}
	if backend.moments["moment_1"].CommentsCount != 0 {
		t.Fatalf("unexpected comments count: %#v", backend.moments["moment_1"])
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryCreateFiltersBlockedTags(t *testing.T) {
	backend := newBackendStub()
	backend.blockedTags["blocked"] = struct{}{}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &fallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)
	created, err := repo.Create(context.Background(), "user_alice", CreateMomentRequest{Description: "hello", Tags: []string{"travel", "blocked", "hotspot"}, Location: Location{LocationName: "南天门"}})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if len(created.Tags) != 2 || created.Tags[0] != "travel" || created.Tags[1] != "hotspot" {
		t.Fatalf("unexpected filtered tags: %#v", created.Tags)
	}
	opened, ok, err := repo.Get(context.Background(), created.MomentID)
	if err != nil || !ok {
		t.Fatalf("unexpected opened moment: %#v ok=%v err=%v", opened, ok, err)
	}
	if len(opened.Tags) != 2 || opened.Tags[0] != "hotspot" && opened.Tags[0] != "travel" {
		t.Fatalf("unexpected stored tags: %#v", opened.Tags)
	}
	for _, tag := range opened.Tags {
		if tag == "blocked" {
			t.Fatalf("blocked tag should not be persisted: %#v", opened.Tags)
		}
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}
