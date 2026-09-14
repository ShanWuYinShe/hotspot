package admin

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresIT_IsBannedHonorsExpiry 限期封禁到期在 PG 实现同样视为解封
// （与内存实现 + UserBanState.Expired 单测同口径，防止双实现漂移）。
func TestPostgresIT_IsBannedHonorsExpiry(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		seedBanUser := func(userID string) {
			t.Helper()
			q, ok := persistence.RepoQueryable(ctx, txFactory)
			if !ok {
				t.Fatal("no tx queryable")
			}
			if err := q.Exec(ctx, `INSERT INTO users (user_id, nickname, location_name, latitude, longitude) VALUES ($1, $2, '', 0, 0)`, userID, userID); err != nil {
				t.Fatalf("seed user: %v", err)
			}
		}

		// 已到期（7 天限期，BannedAt 8 天前）→ 视为未封禁。
		seedBanUser("user_expired")
		if err := repo.SetUserBanState(ctx, "user_expired", UserBanState{Status: UserAccountStatusBanned, Reason: "test", DurationDays: 7, BannedAt: time.Now().UTC().Add(-8 * 24 * time.Hour)}); err != nil {
			t.Fatalf("SetUserBanState: %v", err)
		}
		if banned, err := repo.IsBanned(ctx, "user_expired"); err != nil || banned {
			t.Fatalf("expected expired ban to read unbanned, banned=%v err=%v", banned, err)
		}

		// 限期内 → 仍封禁。
		seedBanUser("user_active")
		if err := repo.SetUserBanState(ctx, "user_active", UserBanState{Status: UserAccountStatusBanned, DurationDays: 7, BannedAt: time.Now().UTC()}); err != nil {
			t.Fatalf("SetUserBanState: %v", err)
		}
		if banned, err := repo.IsBanned(ctx, "user_active"); err != nil || !banned {
			t.Fatalf("expected active ban, banned=%v err=%v", banned, err)
		}

		// 永久封禁 → 仍封禁。
		seedBanUser("user_perm")
		if err := repo.SetUserBanState(ctx, "user_perm", UserBanState{Status: UserAccountStatusBanned, BannedAt: time.Now().UTC().Add(-365 * 24 * time.Hour)}); err != nil {
			t.Fatalf("SetUserBanState: %v", err)
		}
		if banned, err := repo.IsBanned(ctx, "user_perm"); err != nil || !banned {
			t.Fatalf("expected permanent ban to persist, banned=%v err=%v", banned, err)
		}
	})
}
