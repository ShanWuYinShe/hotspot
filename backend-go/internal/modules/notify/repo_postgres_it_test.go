package notify

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresIT_AppendBatchPerUser 落库后各用户经 List 看到各自独立的
// notify_id 与 payload（@ 提及批量路径的六数组 unnest 单往返 INSERT，
// 含 jsonb[] 参数编码与复合主键 ON CONFLICT 的真实数据库验证）。
func TestPostgresIT_AppendBatchPerUser(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		user1 := testdb.InsertUser(t, txFactory, "u_batch_mention_1", "BatchUser1")
		user2 := testdb.InsertUser(t, txFactory, "u_batch_mention_2", "BatchUser2")

		err := repo.AppendBatchPerUser(ctx,
			[]string{user1, user2, ""},
			[]NotifyItem{
				{NotifyID: "notif_batch_1", Type: "mentioned", Payload: map[string]any{"type": "mentioned", "moment_id": "m1", "comment_id": "c1"}},
				{NotifyID: "notif_batch_2", Type: "mentioned", Payload: map[string]any{"type": "mentioned", "moment_id": "m1"}},
				{NotifyID: "notif_batch_skip", Type: "mentioned", Payload: map[string]any{}},
			})
		if err != nil {
			t.Fatalf("AppendBatchPerUser: %v", err)
		}

		page1, err := repo.List(ctx, user1, NotifyListRequest{})
		if err != nil {
			t.Fatalf("List(%s): %v", user1, err)
		}
		if len(page1.Items) != 1 || page1.Items[0].NotifyID != "notif_batch_1" {
			t.Fatalf("user1 items = %#v, want single notif_batch_1", page1.Items)
		}
		if page1.Items[0].Payload["comment_id"] != "c1" {
			t.Fatalf("user1 payload = %#v, want comment_id=c1", page1.Items[0].Payload)
		}

		page2, err := repo.List(ctx, user2, NotifyListRequest{})
		if err != nil {
			t.Fatalf("List(%s): %v", user2, err)
		}
		if len(page2.Items) != 1 || page2.Items[0].NotifyID != "notif_batch_2" {
			t.Fatalf("user2 items = %#v, want single notif_batch_2", page2.Items)
		}
		if _, hasComment := page2.Items[0].Payload["comment_id"]; hasComment {
			t.Fatalf("user2 payload = %#v, want no comment_id", page2.Items[0].Payload)
		}
	})
}
