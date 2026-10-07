package bootstrap

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/admin"
)

// realtimeBanAuthorizer.BanDetails 把封禁状态换算成 WS AccountBanned 事件负载
// （spec api_spec_websocket.md）。这条换算此前零覆盖，而它是客户端封禁页唯一的
// 数据来源：限期封禁必须给出 BannedUntil，永久封禁必须为 nil（否则客户端会把
// 永久封禁显示成「到期自动解封」）。
func TestRealtimeBanAuthorizerBanDetails(t *testing.T) {
	ctx := context.Background()
	bannedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	t.Run("限期封禁给出 BannedUntil", func(t *testing.T) {
		repo := admin.NewInMemoryRepository()
		auth := realtimeBanAuthorizer{repo: repo}
		if err := repo.SetUserBanState(ctx, "user_alice", admin.UserBanState{
			Status:       admin.UserAccountStatusBanned,
			Reason:       "spam",
			DurationDays: 7,
			BannedAt:     bannedAt,
		}); err != nil {
			t.Fatalf("seed temp ban: %v", err)
		}

		detail, found, err := auth.BanDetails(ctx, "user_alice")
		if err != nil || !found {
			t.Fatalf("expected found ban detail, found=%v err=%v", found, err)
		}
		if detail.Reason != "spam" || detail.DurationDays != 7 {
			t.Fatalf("unexpected detail: %#v", detail)
		}
		if detail.BannedUntil == nil {
			t.Fatal("限期封禁必须给出 BannedUntil")
		}
		wantUntil := bannedAt.Add(7 * 24 * time.Hour)
		if !detail.BannedUntil.Equal(wantUntil) {
			t.Fatalf("BannedUntil = %v, want %v", detail.BannedUntil, wantUntil)
		}
	})

	t.Run("永久封禁 BannedUntil 为 nil", func(t *testing.T) {
		repo := admin.NewInMemoryRepository()
		auth := realtimeBanAuthorizer{repo: repo}
		if err := repo.SetUserBanState(ctx, "user_bob", admin.UserBanState{
			Status:   admin.UserAccountStatusBanned,
			Reason:   "abuse",
			BannedAt: bannedAt,
		}); err != nil {
			t.Fatalf("seed perm ban: %v", err)
		}

		detail, found, err := auth.BanDetails(ctx, "user_bob")
		if err != nil || !found {
			t.Fatalf("expected found perm ban, found=%v err=%v", found, err)
		}
		if detail.DurationDays != 0 {
			t.Fatalf("永久封禁 DurationDays 应为 0，得到 %d", detail.DurationDays)
		}
		if detail.BannedUntil != nil {
			t.Fatalf("永久封禁 BannedUntil 必须为 nil，得到 %v", detail.BannedUntil)
		}
	})

	// 未封禁用户：repo 返回 found=true 但 Status 为 normal（GetUserBanState 对
	// 无封禁记录的用户同样返回 ok=true），BanDetails 只做字段换算、不重复判定
	// 是否封禁——发不发 AccountBanned 由调用方看 IsBanned/Status 决定。
	t.Run("未封禁用户给出 normal 状态且 BannedUntil 为 nil", func(t *testing.T) {
		repo := admin.NewInMemoryRepository()
		auth := realtimeBanAuthorizer{repo: repo}

		detail, found, err := auth.BanDetails(ctx, "user_cindy")
		if err != nil || !found {
			t.Fatalf("expected found=true err=nil，得到 found=%v err=%v", found, err)
		}
		if detail.BannedUntil != nil {
			t.Fatalf("未封禁用户 BannedUntil 必须为 nil，得到 %v", detail.BannedUntil)
		}
		if detail.DurationDays != 0 {
			t.Fatalf("未封禁用户 DurationDays 应为 0，得到 %d", detail.DurationDays)
		}
	})
}
