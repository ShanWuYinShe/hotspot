package users

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresIT_GetRecentLocation 轻量位置查询：只认存在、未注销且有有效
// 坐标的用户；无坐标/未注销用户返回 found=false（不触发完整画像查询）。
func TestPostgresIT_GetRecentLocation(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		alice := testdb.InsertUser(t, txFactory, "u_loc_alice", "Alice")
		mustExecIT(t, txFactory, `UPDATE users SET latitude = 34.2604, longitude = 117.1855 WHERE user_id = $1`, alice)

		pt, found, err := repo.GetRecentLocation(ctx, alice)
		if err != nil || !found || pt.Latitude != 34.2604 || pt.Longitude != 117.1855 {
			t.Fatalf("recent location = %#v found=%v err=%v", pt, found, err)
		}

		// 无坐标用户：found=false（回退到默认位置由 service 层处理）。
		bob := testdb.InsertUser(t, txFactory, "u_loc_bob", "Bob")
		if _, found, err := repo.GetRecentLocation(ctx, bob); err != nil || found {
			t.Fatalf("user without location must not be found: found=%v err=%v", found, err)
		}

		// 已注销用户：即使有坐标也不返回。
		mustExecIT(t, txFactory, `UPDATE users SET is_deleted = TRUE WHERE user_id = $1`, alice)
		if _, found, err := repo.GetRecentLocation(ctx, alice); err != nil || found {
			t.Fatalf("deleted user must not be found: found=%v err=%v", found, err)
		}
	})
}
