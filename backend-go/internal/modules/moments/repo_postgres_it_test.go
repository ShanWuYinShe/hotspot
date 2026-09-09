package moments

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func createMomentWithPoint(t *testing.T, ctx context.Context, repo *PostgresRepository, userID, desc string, lat, lng float64, privacyLevel string) MomentItem {
	t.Helper()
	loc := &Location{
		LocationName: "Loc",
	}
	loc.Point.Latitude = lat
	loc.Point.Longitude = lng
	item, err := repo.Create(ctx, userID, CreateMomentRequest{
		Description:  desc,
		Location:     loc,
		PrivacyLevel: privacyLevel,
	})
	if err != nil {
		t.Fatalf("Create moment %q: %v", desc, err)
	}
	return item
}

func TestPostgresIT_ListNearbyMoments_S2SpatialFiltering(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		authorID := testdb.InsertUser(t, txFactory, "m_author_s2", "Author")
		viewerID := testdb.InsertUser(t, txFactory, "m_viewer_s2", "Viewer")

		// 隔离参考中心点
		refPoint := GeoPoint{Latitude: 42.1234, Longitude: 88.5678}

		// 1. 附近公开动态 A (~290m)
		mA := createMomentWithPoint(t, ctx, repo, authorID, "Moment A - Close", 42.1260, 88.5678, "public")
		// 2. 附近公开动态 B (~1300m)
		mB := createMomentWithPoint(t, ctx, repo, authorID, "Moment B - Medium", 42.1350, 88.5678, "public")
		// 3. 附近私密动态 C (~290m，同 A 位置但 private)
		mC := createMomentWithPoint(t, ctx, repo, authorID, "Moment C - Private", 42.1260, 88.5678, "private")
		// 4. 附近软删除动态 D (~290m，创建后软删)
		mD := createMomentWithPoint(t, ctx, repo, authorID, "Moment D - Deleted", 42.1260, 88.5678, "public")
		if err := repo.Delete(ctx, mD.MomentID); err != nil {
			t.Fatalf("Delete moment D: %v", err)
		}
		// 5. 远距离动态 E (相距 ~50km 外)
		mE := createMomentWithPoint(t, ctx, repo, authorID, "Moment E - Far", 42.6000, 88.5678, "public")

		_ = mC
		_ = mE

		// 查询半径 3000m (3km) 内的附近瞬间
		resp, err := repo.ListNearbyMoments(ctx, momentsQuery{
			ViewerUserID:      viewerID,
			ReferencePoint:    refPoint,
			HasReferencePoint: true,
			Radius:            3000,
			Limit:             10,
		})
		if err != nil {
			t.Fatalf("ListNearbyMoments: %v", err)
		}

		if len(resp.Items) != 2 {
			t.Fatalf("expected exactly 2 nearby moments, got %d: %#v", len(resp.Items), resp.Items)
		}

		foundIDs := make(map[string]bool)
		for _, item := range resp.Items {
			foundIDs[item.MomentID] = true
			if item.Location == nil || item.Location.Distance == nil || item.Location.BearingDegrees == nil {
				t.Fatalf("expected relative location (distance and bearing) populated, got %#v", item.Location)
			}
			if *item.Location.Distance > 3000 {
				t.Fatalf("moment %s distance %.1f exceeds 3000m radius", item.MomentID, *item.Location.Distance)
			}
		}

		if !foundIDs[mA.MomentID] || !foundIDs[mB.MomentID] {
			t.Fatalf("expected moments A and B in results, got %#v", foundIDs)
		}
		if foundIDs[mC.MomentID] {
			t.Fatal("private moment C leaked into public nearby stream")
		}
		if foundIDs[mD.MomentID] {
			t.Fatal("deleted moment D leaked into nearby stream")
		}
		if foundIDs[mE.MomentID] {
			t.Fatal("far moment E leaked beyond 3km search radius")
		}

		// 验证首位为距离最近的 mA
		if resp.Items[0].MomentID != mA.MomentID {
			t.Fatalf("expected mA as first item due to closer distance, got %s", resp.Items[0].MomentID)
		}
	})
}

func TestPostgresIT_ListNearbyMoments_Pagination(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		authorID := testdb.InsertUser(t, txFactory, "m_author_page", "Author")
		viewerID := testdb.InsertUser(t, txFactory, "m_viewer_page", "Viewer")

		refPoint := GeoPoint{Latitude: 42.1234, Longitude: 88.5678}

		// 插入 3 条不同距离的附近公开动态
		m1 := createMomentWithPoint(t, ctx, repo, authorID, "Page 1", 42.1240, 88.5678, "public")
		m2 := createMomentWithPoint(t, ctx, repo, authorID, "Page 2", 42.1260, 88.5678, "public")
		m3 := createMomentWithPoint(t, ctx, repo, authorID, "Page 3", 42.1280, 88.5678, "public")
		_ = m1
		_ = m2
		_ = m3

		// 请求第 1 页 (limit=2)
		page1, err := repo.ListNearbyMoments(ctx, momentsQuery{
			ViewerUserID:      viewerID,
			ReferencePoint:    refPoint,
			HasReferencePoint: true,
			Radius:            5000,
			Limit:             2,
		})
		if err != nil {
			t.Fatalf("ListNearbyMoments page 1: %v", err)
		}
		if len(page1.Items) != 2 {
			t.Fatalf("expected 2 items in page 1, got %d", len(page1.Items))
		}
		if page1.NextCursor == nil || *page1.NextCursor == "" {
			t.Fatal("expected non-empty next cursor on page 1")
		}

		// 请求第 2 页
		page2, err := repo.ListNearbyMoments(ctx, momentsQuery{
			ViewerUserID:      viewerID,
			ReferencePoint:    refPoint,
			HasReferencePoint: true,
			Radius:            5000,
			Limit:             2,
			Cursor:            *page1.NextCursor,
		})
		if err != nil {
			t.Fatalf("ListNearbyMoments page 2: %v", err)
		}
		if len(page2.Items) != 1 {
			t.Fatalf("expected 1 item in page 2, got %d", len(page2.Items))
		}

		// 确保两页数据互斥无重复
		page1IDs := map[string]bool{page1.Items[0].MomentID: true, page1.Items[1].MomentID: true}
		if page1IDs[page2.Items[0].MomentID] {
			t.Fatalf("page 2 item %s duplicated from page 1", page2.Items[0].MomentID)
		}
	})
}
