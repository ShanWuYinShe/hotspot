package system

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type announcementRowValues struct {
	values []any
	err    error
}

func (r announcementRowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *int:
			*d = r.values[i].(int)
		case *bool:
			*d = r.values[i].(bool)
		case *time.Time:
			*d = r.values[i].(time.Time)
		case **time.Time:
			if r.values[i] == nil {
				*d = nil
			} else {
				switch value := r.values[i].(type) {
				case time.Time:
					copied := value
					*d = &copied
				case *time.Time:
					if value == nil {
						*d = nil
					} else {
						copied := *value
						*d = &copied
					}
				}
			}
		}
	}
	return nil
}

type announcementRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *announcementRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *announcementRowsValues) Scan(dest ...any) error {
	row := announcementRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *announcementRowsValues) Close() error { return nil }
func (r *announcementRowsValues) Err() error   { return r.err }

type fallbackAnnouncementRepo struct{ used bool }

func (r *fallbackAnnouncementRepo) GetMeta(context.Context) ApiVersionMeta {
	r.used = true
	return ApiVersionMeta{}
}
func (r *fallbackAnnouncementRepo) Health(context.Context) (int, string, *HealthStatus) {
	r.used = true
	return 0, "", nil
}
func (r *fallbackAnnouncementRepo) ListAnnouncements(context.Context, int) ([]AnnouncementRecord, error) {
	r.used = true
	return nil, nil
}
func (r *fallbackAnnouncementRepo) SearchAnnouncements(context.Context, AnnouncementSearchFilter) (AnnouncementPage, error) {
	r.used = true
	return AnnouncementPage{}, nil
}
func (r *fallbackAnnouncementRepo) GetAnnouncement(context.Context, string) (AnnouncementRecord, bool, error) {
	r.used = true
	return AnnouncementRecord{}, false, nil
}
func (r *fallbackAnnouncementRepo) CreateAnnouncement(context.Context, AnnouncementCreateInput) (AnnouncementRecord, error) {
	r.used = true
	return AnnouncementRecord{}, nil
}
func (r *fallbackAnnouncementRepo) UpdateAnnouncement(context.Context, AnnouncementUpdateInput) (AnnouncementRecord, error) {
	r.used = true
	return AnnouncementRecord{}, nil
}
func (r *fallbackAnnouncementRepo) DeleteAnnouncement(context.Context, string) error {
	r.used = true
	return nil
}

type announcementBackendStub struct {
	announcements map[string]AnnouncementRecord
	queryErrs     map[string]error
	queryRowErrs  map[string]error
	execErrs      map[string]error
}

func newAnnouncementBackendStub() *announcementBackendStub {
	return &announcementBackendStub{
		announcements: make(map[string]AnnouncementRecord),
		queryErrs:     make(map[string]error),
		queryRowErrs:  make(map[string]error),
		execErrs:      make(map[string]error),
	}
}

func (b *announcementBackendStub) Ping(context.Context) error { return nil }
func (b *announcementBackendStub) Close() error               { return nil }

func (b *announcementBackendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	switch query {
	case queryInsertAnnouncement:
		item := AnnouncementRecord{
			AnnouncementID: args[0].(string),
			Title:          args[1].(string),
			Content:        args[2].(string),
			Severity:       AnnouncementSeverity(args[3].(string)),
			SendPush:       args[5].(bool),
			ViewCount:      args[7].(int),
			CreatedBy:      AnnouncementActor{UserID: args[8].(string), Nickname: args[8].(string)},
			CreatedAt:      args[9].(time.Time),
		}
		if args[4] != nil {
			item.ActionURL = args[4].(string)
		}
		if args[6] != nil {
			pushSentAt := args[6].(time.Time)
			item.PushSentAt = &pushSentAt
		}
		b.announcements[item.AnnouncementID] = item
	case queryUpdateAnnouncement:
		announcementID := args[0].(string)
		item, ok := b.announcements[announcementID]
		if !ok || item.IsDeleted() {
			return nil
		}
		item.Title = args[1].(string)
		item.Content = args[2].(string)
		item.Severity = AnnouncementSeverity(args[3].(string))
		if args[4] == nil {
			item.ActionURL = ""
		} else {
			item.ActionURL = args[4].(string)
		}
		b.announcements[announcementID] = item
	case queryDeleteAnnouncement:
		announcementID := args[0].(string)
		item, ok := b.announcements[announcementID]
		if !ok || item.IsDeleted() {
			return nil
		}
		deletedAt := args[1].(time.Time)
		item.DeletedAt = &deletedAt
		b.announcements[announcementID] = item
	}
	return nil
}

func (b *announcementBackendStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	if err := b.queryRowErrs[query]; err != nil {
		return announcementRowValues{err: err}
	}
	if query != queryAnnouncementByID {
		return announcementRowValues{err: persistence.ErrNotImplemented}
	}
	item, ok := b.announcements[args[0].(string)]
	if !ok {
		return announcementRowValues{err: pgx.ErrNoRows}
	}
	return announcementRowValues{values: serializeAnnouncementRow(item, true)}
}

func (b *announcementBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	items := make([]AnnouncementRecord, 0, len(b.announcements))
	for _, item := range b.announcements {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].AnnouncementID > items[j].AnnouncementID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	switch query {
	case queryListPublicAnnouncements:
		limit := args[0].(int)
		rows := make([][]any, 0, limit)
		for _, item := range items {
			if item.IsDeleted() {
				continue
			}
			rows = append(rows, serializeAnnouncementRow(item, false))
			if len(rows) >= limit {
				break
			}
		}
		return &announcementRowsValues{rows: rows}, nil
	case querySearchAnnouncements:
		severity := args[0].(string)
		var cursor *AnnouncementCursor
		if args[1] != nil {
			createdAt := args[1].(time.Time)
			cursor = &AnnouncementCursor{CreatedAt: createdAt, AnnouncementID: args[2].(string)}
		}
		limit := args[3].(int)
		rows := make([][]any, 0, limit)
		for _, item := range items {
			if severity != "" && string(item.Severity) != severity {
				continue
			}
			if cursor != nil {
				if item.CreatedAt.After(cursor.CreatedAt) {
					continue
				}
				if item.CreatedAt.Equal(cursor.CreatedAt) && item.AnnouncementID >= cursor.AnnouncementID {
					continue
				}
			}
			rows = append(rows, serializeAnnouncementRow(item, true))
			if len(rows) >= limit {
				break
			}
		}
		return &announcementRowsValues{rows: rows}, nil
	default:
		return &persistence.InMemoryRows{}, nil
	}
}

func serializeAnnouncementRow(item AnnouncementRecord, includeDeleted bool) []any {
	row := []any{
		item.AnnouncementID,
		item.Title,
		item.Content,
		string(item.Severity),
		item.ActionURL,
		item.CreatedAt,
		item.CreatedBy.UserID,
		item.CreatedBy.Nickname,
		item.CreatedBy.AvatarURL,
		item.SendPush,
		item.PushSentAt,
		item.ViewCount,
	}
	if includeDeleted {
		row = append(row, item.DeletedAt)
	}
	return row
}

func TestPostgresRepositoryAnnouncementReadsQueryableDB(t *testing.T) {
	backend := newAnnouncementBackendStub()
	backend.announcements["ann_2"] = AnnouncementRecord{AnnouncementID: "ann_2", Title: "维护", Content: "今晚维护", Severity: AnnouncementSeverityWarning, ActionURL: "https://example.com/maint", CreatedAt: time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC), CreatedBy: AnnouncementActor{UserID: "admin", Nickname: "Admin"}, SendPush: true, PushSentAt: timePtr(time.Date(2026, 3, 23, 11, 0, 1, 0, time.UTC)), ViewCount: 3}
	backend.announcements["ann_1"] = AnnouncementRecord{AnnouncementID: "ann_1", Title: "提示", Content: "普通提示", Severity: AnnouncementSeverityInfo, CreatedAt: time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC), CreatedBy: AnnouncementActor{UserID: "admin", Nickname: "Admin"}, DeletedAt: timePtr(time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC))}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &fallbackAnnouncementRepo{}
	repo := NewPostgresRepository(factory, fallback)

	items, err := repo.ListAnnouncements(context.Background(), 10)
	if err != nil || len(items) != 1 || items[0].AnnouncementID != "ann_2" {
		t.Fatalf("unexpected public list: %#v err=%v", items, err)
	}
	page, err := repo.SearchAnnouncements(context.Background(), AnnouncementSearchFilter{Limit: 10})
	if err != nil || len(page.Items) != 2 || page.Items[0].AnnouncementID != "ann_2" || page.Items[1].AnnouncementID != "ann_1" || !page.Items[1].IsDeleted() {
		t.Fatalf("unexpected search page: %#v err=%v", page, err)
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryAnnouncementWritesQueryableDB(t *testing.T) {
	backend := newAnnouncementBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &fallbackAnnouncementRepo{}
	repo := NewPostgresRepository(factory, fallback)

	created, err := repo.CreateAnnouncement(context.Background(), AnnouncementCreateInput{Actor: AnnouncementActor{UserID: "admin", Nickname: "Admin"}, Title: "维护", Content: "今晚维护", Severity: AnnouncementSeverityWarning, ActionURL: "https://example.com/maint", SendPush: true})
	if err != nil || created.AnnouncementID == "" || created.PushSentAt == nil {
		t.Fatalf("unexpected created item: %#v err=%v", created, err)
	}
	updatedTitle := "已恢复"
	updatedContent := "服务恢复"
	updatedSeverity := AnnouncementSeverityInfo
	clearedAction := ""
	updated, err := repo.UpdateAnnouncement(context.Background(), AnnouncementUpdateInput{AnnouncementID: created.AnnouncementID, Title: &updatedTitle, Content: &updatedContent, Severity: &updatedSeverity, ActionURL: &clearedAction})
	if err != nil || updated.Title != updatedTitle || updated.ActionURL != "" {
		t.Fatalf("unexpected updated item: %#v err=%v", updated, err)
	}
	if err := repo.DeleteAnnouncement(context.Background(), created.AnnouncementID); err != nil {
		t.Fatalf("delete announcement: %v", err)
	}
	stored := backend.announcements[created.AnnouncementID]
	if stored.DeletedAt == nil {
		t.Fatalf("expected deleted_at to be set: %#v", stored)
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryAnnouncementNotFoundAndErrors(t *testing.T) {
	backend := newAnnouncementBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &fallbackAnnouncementRepo{}
	repo := NewPostgresRepository(factory, fallback)

	title := "x"
	_, err := repo.UpdateAnnouncement(context.Background(), AnnouncementUpdateInput{AnnouncementID: "missing", Title: &title})
	if !errors.Is(err, ErrAnnouncementNotFound()) {
		t.Fatalf("expected not found error, got %v", err)
	}
	if err := repo.DeleteAnnouncement(context.Background(), "missing"); !errors.Is(err, ErrAnnouncementNotFound()) {
		t.Fatalf("expected delete not found error, got %v", err)
	}
	backend.queryErrs[queryListPublicAnnouncements] = errors.New("read failed")
	if _, err := repo.ListAnnouncements(context.Background(), 10); err == nil {
		t.Fatal("expected list error")
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func timePtr(value time.Time) *time.Time { return &value }
