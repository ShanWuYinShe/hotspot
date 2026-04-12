package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type authRow struct {
	values []any
	err    error
}

func (r authRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case **string:
			if r.values[i] == nil {
				*d = nil
			} else if v, ok := r.values[i].(*string); ok {
				*d = v
			} else {
				v := r.values[i].(string)
				*d = &v
			}
		case *time.Time:
			if v, ok := r.values[i].(*time.Time); ok {
				*d = *v
			} else {
				*d = r.values[i].(time.Time)
			}
		case **time.Time:
			if r.values[i] == nil {
				*d = nil
			} else if v, ok := r.values[i].(*time.Time); ok {
				*d = v
			} else {
				v := r.values[i].(time.Time)
				*d = &v
			}
		}
	}
	return nil
}

type authRows struct {
	rows [][]any
	idx  int
	err  error
}

func (r *authRows) Next() bool { return r.idx < len(r.rows) }
func (r *authRows) Scan(dest ...any) error {
	row := authRow{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *authRows) Close() error { return nil }
func (r *authRows) Err() error   { return r.err }

type authBackendStub struct {
	sessions map[string]SessionRecord
	refresh  map[string]RefreshSession
	execErrs map[string]error
}

func newAuthBackendStub() *authBackendStub {
	return &authBackendStub{
		sessions: map[string]SessionRecord{},
		refresh:  map[string]RefreshSession{},
		execErrs: map[string]error{},
	}
}

func (b *authBackendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	switch query {
	case queryUpsertSession:
		record := SessionRecord{
			SessionID:    args[0].(string),
			UserID:       args[1].(string),
			CreatedAt:    args[2].(time.Time),
			LastActiveAt: args[3].(time.Time),
			DeviceType:   args[5].(string),
			DeviceName:   args[6].(string),
			IPAddress:    args[7].(string),
			IPLocation:   args[8].(string),
		}
		if args[4] != nil {
			revokedAt := args[4].(*time.Time)
			record.RevokedAt = revokedAt
		}
		b.sessions[record.SessionID] = record
	case queryTouchSession:
		id := args[0].(string)
		record, ok := b.sessions[id]
		if ok && record.RevokedAt == nil {
			record.LastActiveAt = time.Now().UTC()
			b.sessions[id] = record
		}
	case queryRevokeSession:
		id := args[0].(string)
		record, ok := b.sessions[id]
		if ok && record.RevokedAt == nil {
			now := time.Now().UTC()
			record.RevokedAt = &now
			b.sessions[id] = record
		}
	case queryRevokeOtherSessions:
		userID := args[0].(string)
		currentSessionID := args[1].(string)
		now := time.Now().UTC()
		for id, record := range b.sessions {
			if record.UserID != userID || id == currentSessionID || record.RevokedAt != nil {
				continue
			}
			record.RevokedAt = &now
			b.sessions[id] = record
		}
	case queryUpsertRefreshToken:
		session := RefreshSession{
			RefreshToken: args[0].(string),
			UserID:       args[1].(string),
			SessionID:    args[2].(string),
			ExpiresAt:    args[5].(time.Time),
		}
		if rotatedTo, ok := args[3].(*string); ok && rotatedTo != nil {
			session.RotatedTo = *rotatedTo
		}
		if graceUntil, ok := args[4].(*time.Time); ok && graceUntil != nil {
			session.GraceUntil = *graceUntil
		}
		b.refresh[session.RefreshToken] = session
	case queryRotateRefreshToken:
		oldToken := args[0].(string)
		nextToken := args[1].(string)
		graceUntil := args[2].(time.Time)
		next := RefreshSession{
			RefreshToken: nextToken,
			UserID:       args[3].(string),
			SessionID:    args[4].(string),
			ExpiresAt:    args[5].(time.Time),
		}
		current, ok := b.refresh[oldToken]
		if !ok {
			return pgx.ErrNoRows
		}
		current.RotatedTo = nextToken
		current.GraceUntil = graceUntil
		b.refresh[oldToken] = current
		b.refresh[next.RefreshToken] = next
	}
	return nil
}

func (b *authBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if query != queryListActiveSessionsByUser {
		return &persistence.InMemoryRows{}, nil
	}
	userID := args[0].(string)
	rows := [][]any{}
	for _, record := range b.sessions {
		if record.UserID == userID && record.RevokedAt == nil {
			rows = append(rows, []any{record.SessionID, record.UserID, record.CreatedAt, record.LastActiveAt, record.RevokedAt, record.DeviceType, record.DeviceName, record.IPAddress, record.IPLocation})
		}
	}
	return &authRows{rows: rows}, nil
}

func (b *authBackendStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	switch query {
	case queryGetSession:
		if record, ok := b.sessions[args[0].(string)]; ok {
			return authRow{values: []any{record.SessionID, record.UserID, record.CreatedAt, record.LastActiveAt, record.RevokedAt, record.DeviceType, record.DeviceName, record.IPAddress, record.IPLocation}}
		}
		return authRow{err: pgx.ErrNoRows}
	case queryGetRefreshToken:
		if session, ok := b.refresh[args[0].(string)]; ok {
			var rotatedTo *string
			if session.RotatedTo != "" {
				rotatedTo = &session.RotatedTo
			}
			var graceUntil *time.Time
			if !session.GraceUntil.IsZero() {
				graceUntil = &session.GraceUntil
			}
			return authRow{values: []any{session.UserID, session.SessionID, session.RefreshToken, rotatedTo, graceUntil, session.ExpiresAt}}
		}
		return authRow{err: pgx.ErrNoRows}
	default:
		return authRow{err: persistence.ErrNotImplemented}
	}
}

func newAuthFactory(backend *authBackendStub) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresSessionStoreWritesAndReadsQueryableDB(t *testing.T) {
	backend := newAuthBackendStub()
	store := NewPostgresSessionStore(newAuthFactory(backend), nil)
	now := time.Now().UTC().Round(time.Second)
	record := SessionRecord{SessionID: "sess_1", UserID: "user_alice", CreatedAt: now, LastActiveAt: now, DeviceType: "app", DeviceName: "HotSpot App", IPAddress: "203.0.113.1", IPLocation: "US"}
	if err := store.Save(record); err != nil {
		t.Fatalf("save session: %v", err)
	}
	stored, ok, err := store.Get("sess_1")
	if err != nil || !ok || stored.UserID != "user_alice" {
		t.Fatalf("unexpected session: %#v ok=%v err=%v", stored, ok, err)
	}
	if stored.DeviceType != "app" || stored.DeviceName != "HotSpot App" || stored.IPAddress != "203.0.113.1" || stored.IPLocation != "US" {
		t.Fatalf("unexpected stored metadata: %#v", stored)
	}
	items, err := store.ListActiveByUser("user_alice")
	if err != nil || len(items) != 1 || items[0].SessionID != "sess_1" {
		t.Fatalf("unexpected items: %#v err=%v", items, err)
	}
	if err := store.Touch("sess_1"); err != nil {
		t.Fatalf("touch session: %v", err)
	}
	if revoked, err := store.Revoke("sess_1"); err != nil || !revoked {
		t.Fatalf("unexpected revoke result: revoked=%v err=%v", revoked, err)
	}
	stored, ok, err = store.Get("sess_1")
	if err != nil || !ok || stored.RevokedAt == nil {
		t.Fatalf("expected revoked session, got %#v ok=%v err=%v", stored, ok, err)
	}
}

func TestPostgresSessionStoreRevokeOthers(t *testing.T) {
	backend := newAuthBackendStub()
	store := NewPostgresSessionStore(newAuthFactory(backend), nil)
	now := time.Now().UTC()
	for _, sessionID := range []string{"sess_1", "sess_2", "sess_3"} {
		if err := store.Save(SessionRecord{SessionID: sessionID, UserID: "user_alice", CreatedAt: now, LastActiveAt: now}); err != nil {
			t.Fatalf("save session %s: %v", sessionID, err)
		}
	}
	count, err := store.RevokeOthers("user_alice", "sess_2")
	if err != nil || count != 2 {
		t.Fatalf("unexpected revoke others result: count=%d err=%v", count, err)
	}
	for _, sessionID := range []string{"sess_1", "sess_3"} {
		record, ok, err := store.Get(sessionID)
		if err != nil || !ok || record.RevokedAt == nil {
			t.Fatalf("expected revoked session %s, got %#v ok=%v err=%v", sessionID, record, ok, err)
		}
	}
	record, ok, err := store.Get("sess_2")
	if err != nil || !ok || record.RevokedAt != nil {
		t.Fatalf("expected current session to stay active, got %#v ok=%v err=%v", record, ok, err)
	}
}

func TestPostgresRefreshStoreWritesReadsAndRotatesQueryableDB(t *testing.T) {
	backend := newAuthBackendStub()
	store := NewPostgresRefreshStore(newAuthFactory(backend), nil)
	now := time.Now().UTC().Round(time.Second)
	session := RefreshSession{UserID: "user_alice", SessionID: "sess_1", RefreshToken: "refresh_1", ExpiresAt: now.Add(time.Hour)}
	if err := store.Save(session); err != nil {
		t.Fatalf("save refresh: %v", err)
	}
	stored, ok, err := store.Get("refresh_1")
	if err != nil || !ok || stored.UserID != "user_alice" {
		t.Fatalf("unexpected refresh session: %#v ok=%v err=%v", stored, ok, err)
	}
	next := RefreshSession{UserID: "user_alice", SessionID: "sess_1", RefreshToken: "refresh_2", ExpiresAt: now.Add(2 * time.Hour)}
	if err := store.Rotate("refresh_1", stored, next); err != nil {
		t.Fatalf("rotate refresh: %v", err)
	}
	rotated, ok, err := store.Get("refresh_1")
	if err != nil || !ok || rotated.RotatedTo != "refresh_2" || rotated.GraceUntil.IsZero() {
		t.Fatalf("unexpected rotated current session: %#v ok=%v err=%v", rotated, ok, err)
	}
	nextStored, ok, err := store.Get("refresh_2")
	if err != nil || !ok || nextStored.ExpiresAt.IsZero() {
		t.Fatalf("unexpected rotated next session: %#v ok=%v err=%v", nextStored, ok, err)
	}
}

func TestPostgresStoresReturnWriteErrors(t *testing.T) {
	backend := newAuthBackendStub()
	backend.execErrs[queryUpsertSession] = errors.New("session write failed")
	backend.execErrs[queryUpsertRefreshToken] = errors.New("refresh write failed")
	factory := newAuthFactory(backend)
	sessionStore := NewPostgresSessionStore(factory, nil)
	refreshStore := NewPostgresRefreshStore(factory, nil)
	if err := sessionStore.Save(SessionRecord{SessionID: "sess_1", UserID: "user_alice", CreatedAt: time.Now().UTC(), LastActiveAt: time.Now().UTC()}); err == nil {
		t.Fatal("expected session save error")
	}
	if err := refreshStore.Save(RefreshSession{UserID: "user_alice", SessionID: "sess_1", RefreshToken: "refresh_1", ExpiresAt: time.Now().UTC().Add(time.Hour)}); err == nil {
		t.Fatal("expected refresh save error")
	}
}
