package moments

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresITListNearbyMomentsMergedDetails：附近流单 SQL 详情查询（批量
// 聚合 comments/shares/tags/images/reactions + 作者关系）在真实库上的行为：
// 聚合按 moment_id 正确归属、空集合返回空数组/零值。
func TestPostgresITListNearbyMomentsMergedDetails(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		qdb, ok := txFactory.DB.(persistence.QueryableDB)
		if !ok {
			t.Fatal("factory.DB is not QueryableDB")
		}
		userID := testdb.InsertUser(t, txFactory, "it_nearby_detail_u1", "Nearby Author")
		point := geo.Point{Latitude: 34.2604, Longitude: 117.1855}
		cellID := geo.LeafCellID(point)
		momentID := "it_nearby_detail_m1"
		if err := qdb.Exec(ctx, "INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level, s2_cell_id, interaction_count) VALUES ($1,$2,$3,'',$4,$5,'public',$6,3)", momentID, userID, "nearby detail", point.Latitude, point.Longitude, cellID); err != nil {
			t.Fatalf("insert moment: %v", err)
		}
		if err := qdb.Exec(ctx, "INSERT INTO moment_tags (moment_id, tag_name) VALUES ($1,$2), ($1,$3)", momentID, "美食", "旅行"); err != nil {
			t.Fatalf("insert tags: %v", err)
		}
		if err := qdb.Exec(ctx, "INSERT INTO reactions (target_type, target_id, user_id, emoji) VALUES ('moment',$1,$2,$3)", momentID, userID, "❤️"); err != nil {
			t.Fatalf("insert reaction: %v", err)
		}
		repo := NewPostgresRepository(txFactory)
		page, err := repo.ListNearbyMoments(ctx, momentsQuery{
			ViewerUserID:      "",
			ReferencePoint:    GeoPoint{Latitude: point.Latitude, Longitude: point.Longitude},
			HasReferencePoint: true,
			Radius:            5000,
			Limit:             20,
		})
		if err != nil {
			t.Fatalf("ListNearbyMoments: %v", err)
		}
		var got *MomentItem
		for i := range page.Items {
			if page.Items[i].MomentID == momentID {
				got = &page.Items[i]
				break
			}
		}
		if got == nil {
			t.Fatalf("moment %s missing from nearby page: %#v", momentID, page.Items)
		}
		if got.User.Nickname != "Nearby Author" {
			t.Fatalf("nickname = %q", got.User.Nickname)
		}
		if len(got.Tags) != 2 {
			t.Fatalf("tags = %v, want 2", got.Tags)
		}
		if len(got.Reactions) != 1 || got.Reactions[0].Emoji != "❤️" || got.Reactions[0].Count != 1 {
			t.Fatalf("reactions = %#v", got.Reactions)
		}
		if got.CommentsCount != 0 || got.SharesCount != 0 {
			t.Fatalf("counts = %d/%d, want 0/0", got.CommentsCount, got.SharesCount)
		}
		if len(got.Images) != 0 {
			t.Fatalf("images = %#v, want empty", got.Images)
		}
	})
}
