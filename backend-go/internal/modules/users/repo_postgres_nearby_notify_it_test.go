package users

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// 附近新瞬间通知目标查询此前无 LIMIT：moments 创建链路先全量读入内存再截断
// （扇出上限 200），密集区域候选行随用户量线性增长。锁定 limit 参数在 SQL 侧
// 限量（limit<=0 = 不限制）与排除作者/圈外/未开启通知的过滤语义。
func TestPostgresIT_NearbyMomentNotifyTargetsLimit(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		q, ok := persistence.RepoQueryable(ctx, txFactory)
		if !ok {
			t.Fatal("no tx queryable")
		}

		// 中心 35.0976,112.6124：u1 作者（排除）、u2/u3 圈内开启、u4 圈外开启。
		seed := []struct {
			userID   string
			lat, lng float64
			notify   bool
		}{
			{"it_nearby_u1", 35.0976, 112.6124, true},
			{"it_nearby_u2", 35.0980, 112.6130, true},
			{"it_nearby_u3", 35.0970, 112.6120, true},
			{"it_nearby_u4", 40.0000, 116.0000, true},
			{"it_nearby_u5", 35.0981, 112.6131, false},
		}
		for _, u := range seed {
			testdb.InsertUser(t, txFactory, u.userID, u.userID)
			if err := q.Exec(ctx, "UPDATE users SET location_name = 'IT', latitude = $2, longitude = $3 WHERE user_id = $1", u.userID, u.lat, u.lng); err != nil {
				t.Fatalf("update location %s: %v", u.userID, err)
			}
			if err := q.Exec(ctx, "INSERT INTO user_settings (user_id, nearby_moment_notify) VALUES ($1,$2) ON CONFLICT (user_id) DO UPDATE SET nearby_moment_notify = EXCLUDED.nearby_moment_notify", u.userID, u.notify); err != nil {
				t.Fatalf("settings %s: %v", u.userID, err)
			}
		}

		ids := func(targets []NearbyNotifyTarget) map[string]bool {
			out := map[string]bool{}
			for _, target := range targets {
				out[target.UserID] = true
			}
			return out
		}

		// limit<=0：不限制，返回圈内开启通知的 u2/u3（u1 作者排除、u4 圈外、u5 未开启）。
		all, err := repo.NearbyMomentNotifyTargets(ctx, 35.0976, 112.6124, 5000, "it_nearby_u1", 0)
		if err != nil {
			t.Fatalf("unlimited: %v", err)
		}
		if len(all) != 2 || !ids(all)["it_nearby_u2"] || !ids(all)["it_nearby_u3"] {
			t.Fatalf("unlimited targets = %#v, want u2/u3", ids(all))
		}

		// limit=1：SQL 侧限量（不再先全量读再截断）。
		limited, err := repo.NearbyMomentNotifyTargets(ctx, 35.0976, 112.6124, 5000, "it_nearby_u1", 1)
		if err != nil {
			t.Fatalf("limited: %v", err)
		}
		if len(limited) != 1 || (!ids(limited)["it_nearby_u2"] && !ids(limited)["it_nearby_u3"]) {
			t.Fatalf("limited targets = %#v, want exactly 1 of u2/u3", ids(limited))
		}

		// 作者排除：exclude 换成别人后 u1 也应出现。
		other, err := repo.NearbyMomentNotifyTargets(ctx, 35.0976, 112.6124, 5000, "it_nearby_u2", 10)
		if err != nil {
			t.Fatalf("other exclude: %v", err)
		}
		if len(other) != 2 || !ids(other)["it_nearby_u1"] || !ids(other)["it_nearby_u3"] {
			t.Fatalf("other exclude targets = %#v, want u1/u3", ids(other))
		}
	})
}
