package users

import (
	"context"
	"errors"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"

	"github.com/jackc/pgx/v5"
)

func g3SearchBackend(rows ...[]any) *g3Backend {
	return &g3Backend{
		queryRowByQuery: map[string]persistence.Row{},
		rowsByQuery:     map[string]persistence.Rows{"FROM users u JOIN user_settings us ON us.user_id": &g3Rows{rows: rows}},
		queryErrs:       map[string]error{},
		execErrs:        map[string]error{},
	}
}

func TestG3RepoSearchProfilesNonRankedBranches(t *testing.T) {
	ctx := context.Background()
	radius := 5000
	ref := GeoPoint{Latitude: 34.2, Longitude: 117.1}

	// 查询错误。
	errBackend := &g3Backend{queryRowByQuery: map[string]persistence.Row{}, rowsByQuery: map[string]persistence.Rows{}, queryErrs: map[string]error{"FROM users u JOIN user_settings us ON us.user_id": errors.New("boom")}, execErrs: map[string]error{}}
	if _, err := g3RepoWith(errBackend).SearchProfiles(ctx, SearchUsersRequest{Tag: "t"}); err == nil {
		t.Fatal("expected search query error")
	}

	// 空结果。
	emptyRepo := g3RepoWith(g3SearchBackend())
	page, err := emptyRepo.SearchProfiles(ctx, SearchUsersRequest{Tag: "t"})
	if err != nil || len(page.Items) != 0 || page.NextCursor != "" {
		t.Fatalf("empty search = %#v err=%v", page, err)
	}

	// tag 过滤的 Go 侧双保险：SQL 返回了不匹配标签的行 → 跳过。
	mismatchRepo := g3RepoWith(g3SearchBackend(g3SearchRow("u1", 34.2, 117.1, []string{"other"}, true, true, true, true)))
	if page, err := mismatchRepo.SearchProfiles(ctx, SearchUsersRequest{Tag: "美食"}); err != nil || len(page.Items) != 0 {
		t.Fatalf("tag mismatch search = %#v err=%v", page, err)
	}

	// 半径外的候选被 Go 侧精确距离过滤。
	farRepo := g3RepoWith(g3SearchBackend(g3SearchRow("u_far", 0, 0, nil, false, true, true, true)))
	if page, err := farRepo.SearchProfiles(ctx, SearchUsersRequest{Radius: &radius, ReferencePoint: ref}); err != nil || len(page.Items) != 0 {
		t.Fatalf("radius filter search = %#v err=%v", page, err)
	}

	// keyword+radius（非 ranked 路径）：命中半径内候选，分页后补标签。
	nearRepo := g3RepoWith(g3SearchBackend(g3SearchRow("u_near", 34.2, 117.1, nil, false, true, true, true)))
	page, err = nearRepo.SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick", Radius: &radius, ReferencePoint: ref})
	if err != nil {
		t.Fatalf("near search: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].UserID != "u_near" {
		t.Fatalf("near search page = %#v", page)
	}
	if len(page.Settings) != 1 {
		t.Fatalf("search must return inline settings, got %#v", page.Settings)
	}

	// 分页：limit=1 时第二行作为 next cursor。
	twoRows := g3RepoWith(g3SearchBackend(
		g3SearchRow("u1", 34.2, 117.1, nil, false, true, true, true),
		g3SearchRow("u2", 34.2, 117.1, nil, false, true, true, true),
	))
	page, err = twoRows.SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick", Radius: &radius, ReferencePoint: ref, Pagination: &PaginationParams{Limit: 1}})
	if err != nil || len(page.Items) != 1 || page.NextCursor == "" {
		t.Fatalf("paged search = %#v err=%v", page, err)
	}

	// 手机号格式 keyword + radius：走 auth_accounts 精确匹配分支。
	phoneRepo := g3RepoWith(g3SearchBackend(g3SearchRow("u_phone", 34.2, 117.1, nil, false, true, true, true)))
	if _, err := phoneRepo.SearchProfiles(ctx, SearchUsersRequest{Keyword: "13800000000", Radius: &radius, ReferencePoint: ref}); err != nil {
		t.Fatalf("phone keyword search: %v", err)
	}

	// distance 排序（无半径）走非 ranked 路径，半径缺省不过滤。
	distanceRepo := g3RepoWith(g3SearchBackend(
		g3SearchRow("u_a", 34.21, 117.11, nil, false, true, true, true),
		g3SearchRow("u_b", 34.2, 117.1, nil, false, true, true, true),
	))
	page, err = distanceRepo.SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick", SortBy: "distance", ReferencePoint: ref})
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("distance search = %#v err=%v", page, err)
	}
	if page.Items[0].UserID != "u_b" {
		t.Fatalf("distance order = %v", page.Items)
	}

	// rows scan / rows.Err / 标签批量查询错误。
	scanBackend := g3SearchBackend(g3SearchRow("u1", 34.2, 117.1, []string{"t"}, true, true, true, true))
	scanBackend.rowsByQuery["FROM users u JOIN user_settings us ON us.user_id"] = &g3Rows{rows: [][]any{g3SearchRow("u1", 34.2, 117.1, []string{"t"}, true, true, true, true)}, scanErr: errors.New("scan")}
	if _, err := g3RepoWith(scanBackend).SearchProfiles(ctx, SearchUsersRequest{Tag: "t"}); err == nil {
		t.Fatal("expected search scan error")
	}
	tailBackend := g3SearchBackend(g3SearchRow("u1", 34.2, 117.1, []string{"t"}, true, true, true, true))
	tailBackend.rowsByQuery["FROM users u JOIN user_settings us ON us.user_id"] = &g3Rows{rows: [][]any{g3SearchRow("u1", 34.2, 117.1, []string{"t"}, true, true, true, true)}, tailErr: errors.New("tail")}
	if _, err := g3RepoWith(tailBackend).SearchProfiles(ctx, SearchUsersRequest{Tag: "t"}); err == nil {
		t.Fatal("expected search rows error")
	}
	tagsErrBackend := g3SearchBackend(g3SearchRow("u_near", 34.2, 117.1, nil, false, true, true, true))
	tagsErrBackend.queryErrs[queryListUserProfileTagsBatch] = errors.New("tags boom")
	if _, err := g3RepoWith(tagsErrBackend).SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick", Radius: &radius, ReferencePoint: ref}); err == nil {
		t.Fatal("expected page tags error")
	}
}

func TestG3RepoSearchProfilesRankedBranches(t *testing.T) {
	ctx := context.Background()
	ref := GeoPoint{Latitude: 34.2, Longitude: 117.1}
	_ = ref

	rankedBackend := func(rows ...[]any) *g3Backend {
		b := &g3Backend{
			queryRowByQuery: map[string]persistence.Row{},
			rowsByQuery:     map[string]persistence.Rows{"FROM (SELECT u.user_id": &g3Rows{rows: rows}},
			queryErrs:       map[string]error{},
			execErrs:        map[string]error{},
		}
		return b
	}
	rankedRow := func(userID string) []any {
		return g3SearchRow(userID, 34.2, 117.1, []string{"tag"}, true, true, true, true)
	}

	// 非法游标与游标形态不匹配。
	repo := g3RepoWith(rankedBackend())
	if page, err := repo.SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick", Pagination: &PaginationParams{Cursor: "garbage"}}); err != nil || !page.InvalidCursor {
		t.Fatalf("garbage cursor = %#v err=%v", page, err)
	}
	distanceCursor := encodeSearchCursor(searchProfileCandidate{Profile: MyProfile{UserID: "u1"}, SortBy: "distance"})
	if page, err := repo.SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick", Pagination: &PaginationParams{Cursor: distanceCursor}}); err != nil || !page.InvalidCursor {
		t.Fatalf("sort mismatch cursor = %#v err=%v", page, err)
	}
	relevanceCursor := encodeSearchCursor(searchProfileCandidate{Profile: MyProfile{UserID: "u1"}, SortBy: "relevance"})
	if page, err := repo.SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick", Pagination: &PaginationParams{Cursor: relevanceCursor}}); err != nil || !page.InvalidCursor {
		t.Fatalf("relevance cursor without score = %#v err=%v", page, err)
	}
	newestCursor := encodeSearchCursor(searchProfileCandidate{Profile: MyProfile{UserID: "u1"}, SortBy: "newest"})
	if page, err := repo.SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick", SortBy: "newest", Pagination: &PaginationParams{Cursor: newestCursor}}); err != nil || !page.InvalidCursor {
		t.Fatalf("newest cursor without created_at = %#v err=%v", page, err)
	}

	// 查询错误。
	errBackend := rankedBackend()
	errBackend.queryErrs["FROM (SELECT u.user_id"] = errors.New("boom")
	if _, err := g3RepoWith(errBackend).SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick"}); err == nil {
		t.Fatal("expected ranked query error")
	}
	// scan / rows.Err。
	scanBackend := rankedBackend(rankedRow("u1"))
	scanBackend.rowsByQuery["FROM (SELECT u.user_id"] = &g3Rows{rows: [][]any{rankedRow("u1")}, scanErr: errors.New("scan")}
	if _, err := g3RepoWith(scanBackend).SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick"}); err == nil {
		t.Fatal("expected ranked scan error")
	}
	tailBackend := rankedBackend(rankedRow("u1"))
	tailBackend.rowsByQuery["FROM (SELECT u.user_id"] = &g3Rows{rows: [][]any{rankedRow("u1")}, tailErr: errors.New("tail")}
	if _, err := g3RepoWith(tailBackend).SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick"}); err == nil {
		t.Fatal("expected ranked rows error")
	}

	// 成功（relevance）：两行 + limit=1 → next cursor。
	page, err := g3RepoWith(rankedBackend(rankedRow("u1"), rankedRow("u2"))).SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick", Pagination: &PaginationParams{Limit: 1}})
	if err != nil || len(page.Items) != 1 || page.NextCursor == "" || len(page.Settings) != 2 {
		t.Fatalf("ranked page = %#v err=%v", page, err)
	}
	// 成功（newest）：带上合法 newest 游标走 keyset 分支。
	newestValid := encodeSearchCursor(searchProfileCandidate{Profile: MyProfile{UserID: "u1", CreatedAt: g3Time.Format("2006-01-02T15:04:05Z07:00")}, SortBy: "newest"})
	if _, err := g3RepoWith(rankedBackend(rankedRow("u2"))).SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick", SortBy: "newest", Pagination: &PaginationParams{Cursor: newestValid, Limit: 1}}); err != nil {
		t.Fatalf("newest ranked search: %v", err)
	}
	// relevance 合法游标 + tag 过滤。
	validScore := encodeSearchCursor(searchProfileCandidate{Profile: MyProfile{UserID: "u1", LastOnline: g3Time.Format("2006-01-02T15:04:05Z07:00")}, SortBy: "relevance", RelevanceScore: 3})
	if _, err := g3RepoWith(rankedBackend(rankedRow("u2"))).SearchProfiles(ctx, SearchUsersRequest{Keyword: "nick", Tag: "tag", Pagination: &PaginationParams{Cursor: validScore, Limit: 1}}); err != nil {
		t.Fatalf("relevance ranked search: %v", err)
	}
}

func TestG3RepoWriteBranches(t *testing.T) {
	ctx := context.Background()
	profile := MyProfile{UserID: "u1", Nickname: "Nick", UserType: UserTypeNormal, Location: Location{LocationName: "loc", Point: GeoPoint{Latitude: 34.2, Longitude: 117.1}}}

	// PutProfile：无标签 / 有标签 / 各语句失败。
	backend := &g3Backend{queryRowByQuery: map[string]persistence.Row{}, rowsByQuery: map[string]persistence.Rows{}, queryErrs: map[string]error{}, execErrs: map[string]error{}}
	repo := g3RepoWith(backend)
	if err := repo.PutProfile(ctx, profile); err != nil {
		t.Fatalf("PutProfile without tags: %v", err)
	}
	profile.Tags = []string{"a", "b"}
	if err := repo.PutProfile(ctx, profile); err != nil {
		t.Fatalf("PutProfile with tags: %v", err)
	}
	for _, stmt := range []string{queryUpsertUserProfile, queryReplaceUserProfileTags, queryInsertUserProfileTagsBatch} {
		backend.execErrs = map[string]error{stmt: errors.New("exec boom")}
		if err := repo.PutProfile(ctx, profile); err == nil {
			t.Fatalf("expected PutProfile error for %s", stmt)
		}
	}
	if err := g3EmptyRepo().PutProfile(ctx, profile); err == nil {
		t.Fatal("expected PutProfile not-implemented error")
	}

	// SetAccountBanState。
	backend.execErrs = map[string]error{}
	if err := repo.SetAccountBanState(ctx, "u1", AccountBanState{Status: "banned", Reason: "spam", DurationDays: 3, BannedAt: g3Time}); err != nil {
		t.Fatalf("SetAccountBanState: %v", err)
	}
	backend.execErrs = map[string]error{querySetAccountBanState: errors.New("boom")}
	if err := repo.SetAccountBanState(ctx, "u1", AccountBanState{}); err == nil {
		t.Fatal("expected ban state error")
	}

	// DeleteProfile：成功与首条语句失败。
	backend.execErrs = map[string]error{}
	if err := repo.DeleteProfile(ctx, "u1"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
	backend.execErrs = map[string]error{queryAnonymizeUserProfile: errors.New("boom")}
	if err := repo.DeleteProfile(ctx, "u1"); err == nil {
		t.Fatal("expected DeleteProfile error")
	}

	// UpdateProfile：仅标签（存在性探测）、列级更新、行缺失、查询失败、标签写入失败。
	backend.execErrs = map[string]error{}
	backend.queryRowByQuery[queryUserExists] = testdb.RowValues{Values: []any{1}}
	tags := []string{"a"}
	if found, err := repo.UpdateProfile(ctx, "u1", ProfilePatch{}, &tags); err != nil || !found {
		t.Fatalf("tags-only update found=%v err=%v", found, err)
	}
	emptyTags := []string{}
	if found, err := repo.UpdateProfile(ctx, "u1", ProfilePatch{}, &emptyTags); err != nil || !found {
		t.Fatalf("empty tags update found=%v err=%v", found, err)
	}
	nick := "NewNick"
	updateSQL, _ := buildUpdateProfileSQL("u1", ProfilePatch{Nickname: &nick})
	backend.queryRowByQuery[updateSQL] = testdb.RowValues{Values: []any{1}}
	if found, err := repo.UpdateProfile(ctx, "u1", ProfilePatch{Nickname: &nick}, nil); err != nil || !found {
		t.Fatalf("column update found=%v err=%v", found, err)
	}
	backend.queryRowByQuery[updateSQL] = testdb.RowValues{Err: pgx.ErrNoRows}
	if found, err := repo.UpdateProfile(ctx, "u1", ProfilePatch{Nickname: &nick}, nil); err != nil || found {
		t.Fatalf("missing row update found=%v err=%v", found, err)
	}
	backend.queryRowByQuery[updateSQL] = testdb.RowValues{Err: errors.New("boom")}
	if _, err := repo.UpdateProfile(ctx, "u1", ProfilePatch{Nickname: &nick}, nil); err == nil {
		t.Fatal("expected column update error")
	}
	backend.queryRowByQuery[queryUserExists] = testdb.RowValues{Err: errors.New("boom")}
	if _, err := repo.UpdateProfile(ctx, "u1", ProfilePatch{}, &tags); err == nil {
		t.Fatal("expected user exists error")
	}
	backend.queryRowByQuery[queryUserExists] = testdb.RowValues{Err: pgx.ErrNoRows}
	if found, err := repo.UpdateProfile(ctx, "u1", ProfilePatch{}, &tags); err != nil || found {
		t.Fatalf("missing user tags-only found=%v err=%v", found, err)
	}
	backend.queryRowByQuery[updateSQL] = testdb.RowValues{Values: []any{1}}
	backend.execErrs = map[string]error{queryReplaceUserProfileTags: errors.New("boom")}
	if _, err := repo.UpdateProfile(ctx, "u1", ProfilePatch{Nickname: &nick}, &tags); err == nil {
		t.Fatal("expected tags replace error")
	}
	backend.execErrs = map[string]error{queryInsertUserProfileTagsBatch: errors.New("boom")}
	if _, err := repo.UpdateProfile(ctx, "u1", ProfilePatch{Nickname: &nick}, &tags); err == nil {
		t.Fatal("expected tags insert error")
	}
	backend.execErrs = map[string]error{}

	// UpdateProfileLocation。
	backend.queryRowByQuery[queryUpdateProfileLocation] = testdb.RowValues{Values: []any{1}}
	if found, err := repo.UpdateProfileLocation(ctx, "u1", profile.Location); err != nil || !found {
		t.Fatalf("update location found=%v err=%v", found, err)
	}
	backend.queryRowByQuery[queryUpdateProfileLocation] = testdb.RowValues{Err: pgx.ErrNoRows}
	if found, err := repo.UpdateProfileLocation(ctx, "u1", profile.Location); err != nil || found {
		t.Fatalf("missing location row found=%v err=%v", found, err)
	}
	backend.queryRowByQuery[queryUpdateProfileLocation] = testdb.RowValues{Err: errors.New("boom")}
	if _, err := repo.UpdateProfileLocation(ctx, "u1", profile.Location); err == nil {
		t.Fatal("expected update location error")
	}

	// UpdateSettings：全字段 patch 命中动态 SQL。
	private := "private"
	visible := true
	perm := ChatPermNobody
	fullPatch := PartialUserSettings{
		PrivacyLevel: &private, LocationVisible: &visible, PresenceVisible: &visible,
		SocialListsVisible: &visible, GroupsVisible: &visible, ReactedMomentsVisible: &visible,
		StatsVisible: &visible, AllowFollow: &visible, AllowPrivateChat: &perm,
		Searchable: &visible, PhoneSearchable: &visible, NicknameSearchable: &visible,
		LocationSearchable: &visible, NearbyMomentNotify: &visible,
	}
	settingsSQL, _, ok := buildUpdateSettingsSQL("u1", fullPatch)
	if !ok {
		t.Fatal("expected full settings SQL")
	}
	backend.queryRowByQuery[settingsSQL] = testdb.RowValues{Values: []any{1}}
	if found, err := repo.UpdateSettings(ctx, "u1", fullPatch); err != nil || !found {
		t.Fatalf("update settings found=%v err=%v", found, err)
	}
	backend.queryRowByQuery[settingsSQL] = testdb.RowValues{Err: pgx.ErrNoRows}
	if found, err := repo.UpdateSettings(ctx, "u1", fullPatch); err != nil || found {
		t.Fatalf("missing settings row found=%v err=%v", found, err)
	}
	backend.queryRowByQuery[settingsSQL] = testdb.RowValues{Err: errors.New("boom")}
	if _, err := repo.UpdateSettings(ctx, "u1", fullPatch); err == nil {
		t.Fatal("expected update settings error")
	}
	if _, err := g3EmptyRepo().UpdateSettings(ctx, "u1", fullPatch); err == nil {
		t.Fatal("expected update settings not-implemented error")
	}

	// PutSettings。
	if err := repo.PutSettings(ctx, "u1", defaultSettings()); err != nil {
		t.Fatalf("PutSettings: %v", err)
	}
	backend.execErrs = map[string]error{queryUpsertUserSettings: errors.New("boom")}
	if err := repo.PutSettings(ctx, "u1", defaultSettings()); err == nil {
		t.Fatal("expected PutSettings error")
	}
	backend.execErrs = map[string]error{}

	// Follow no-op（ErrNoRows）。
	backend.queryRowByQuery[queryUpsertUserRelationFollow] = testdb.RowValues{Err: pgx.ErrNoRows}
	if changed, err := repo.Follow(ctx, "u1", "u2", FollowSourceProfile, ""); err != nil || changed {
		t.Fatalf("follow no-op changed=%v err=%v", changed, err)
	}

	// 关系写方法成功与失败。
	for _, tc := range []struct {
		name string
		call func() error
		sql  string
	}{
		{"unfollow", func() error { return repo.Unfollow(ctx, "u1", "u2") }, queryUpsertUserRelationUnfollow},
		{"block", func() error { return repo.Block(ctx, "u1", "u2", "spam") }, queryUpsertUserRelationBlock},
		{"unblock", func() error { return repo.Unblock(ctx, "u1", "u2") }, queryUpsertUserRelationUnblock},
		{"remark", func() error { return repo.SetRemark(ctx, "u1", "u2", "buddy") }, queryUpsertUserRelationRemark},
		{"presence", func() error { return repo.SetPresenceStatus(ctx, "u1", UserPresenceOnline) }, queryUpdatePresenceStatus},
	} {
		if err := tc.call(); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		backend.execErrs = map[string]error{tc.sql: errors.New("boom")}
		if err := tc.call(); err == nil {
			t.Fatalf("%s: expected error", tc.name)
		}
		backend.execErrs = map[string]error{}
	}

	// CheckBlockedTags：空列表 / 查询错误 / scan / rows.Err / 命中。
	if blocked, err := repo.CheckBlockedTags(ctx, nil); err != nil || len(blocked) != 0 {
		t.Fatalf("empty blocked tags = %#v err=%v", blocked, err)
	}
	backend.rowsByQuery[queryListBlockedUserTags] = &g3Rows{rows: [][]any{{"bad"}}}
	if blocked, err := repo.CheckBlockedTags(ctx, []string{"bad", "ok"}); err != nil || len(blocked) != 1 || blocked[0] != "bad" {
		t.Fatalf("blocked tags = %#v err=%v", blocked, err)
	}
	backend.queryErrs = map[string]error{queryListBlockedUserTags: errors.New("boom")}
	if _, err := repo.CheckBlockedTags(ctx, []string{"bad"}); err == nil {
		t.Fatal("expected blocked tags query error")
	}
	backend.queryErrs = map[string]error{}
	backend.rowsByQuery[queryListBlockedUserTags] = &g3Rows{rows: [][]any{{"bad"}}, scanErr: errors.New("scan")}
	if _, err := repo.CheckBlockedTags(ctx, []string{"bad"}); err == nil {
		t.Fatal("expected blocked tags scan error")
	}
	backend.rowsByQuery[queryListBlockedUserTags] = &g3Rows{rows: nil, tailErr: errors.New("tail")}
	if _, err := repo.CheckBlockedTags(ctx, []string{"bad"}); err == nil {
		t.Fatal("expected blocked tags rows error")
	}
	if _, err := g3EmptyRepo().CheckBlockedTags(ctx, []string{"bad"}); err == nil {
		t.Fatal("expected blocked tags not-implemented error")
	}

	// 未接 PG 时列表/存在性查询返回 ErrNotImplemented。
	empty := g3EmptyRepo()
	if _, err := empty.ListBlocked(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListBlocked not-implemented error")
	}
	if _, err := empty.ListFriends(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListFriends not-implemented error")
	}
	if _, err := empty.ListFollowing(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListFollowing not-implemented error")
	}
	if _, err := empty.ListFollowers(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListFollowers not-implemented error")
	}
	if _, err := empty.UserExists(ctx, "u1"); err == nil {
		t.Fatal("expected UserExists not-implemented error")
	}

	// UserExists：ErrNoRows → false；其他错误 → 上抛。
	backend.queryRowByQuery[queryUserExists] = testdb.RowValues{Err: pgx.ErrNoRows}
	if exists, err := repo.UserExists(ctx, "u1"); err != nil || exists {
		t.Fatalf("UserExists no rows exists=%v err=%v", exists, err)
	}
	backend.queryRowByQuery[queryUserExists] = testdb.RowValues{Err: errors.New("boom")}
	if _, err := repo.UserExists(ctx, "u1"); err == nil {
		t.Fatal("expected UserExists error")
	}

	// NearbyMomentNotifyTargets：成功 / scan 错误 / rows.Err / 查询错误 / limit<=0。
	backend.rowsByQuery[queryNearbyMomentNotifyTargets] = &g3Rows{rows: [][]any{{"u2", 34.2, 117.1}}}
	targets, err := repo.NearbyMomentNotifyTargets(ctx, 34.2, 117.1, 5000, "u1", 10)
	if err != nil || len(targets) != 1 || targets[0].UserID != "u2" {
		t.Fatalf("nearby targets = %#v err=%v", targets, err)
	}
	if _, err := repo.NearbyMomentNotifyTargets(ctx, 34.2, 117.1, 5000, "u1", 0); err != nil {
		t.Fatalf("nearby targets without limit: %v", err)
	}
	backend.rowsByQuery[queryNearbyMomentNotifyTargets] = &g3Rows{rows: [][]any{{"u2", 34.2, 117.1}}, scanErr: errors.New("scan")}
	if _, err := repo.NearbyMomentNotifyTargets(ctx, 34.2, 117.1, 5000, "u1", 10); err == nil {
		t.Fatal("expected nearby scan error")
	}
	backend.rowsByQuery[queryNearbyMomentNotifyTargets] = &g3Rows{rows: nil, tailErr: errors.New("tail")}
	if _, err := repo.NearbyMomentNotifyTargets(ctx, 34.2, 117.1, 5000, "u1", 10); err == nil {
		t.Fatal("expected nearby rows error")
	}
	backend.queryErrs = map[string]error{queryNearbyMomentNotifyTargets: errors.New("boom")}
	if _, err := repo.NearbyMomentNotifyTargets(ctx, 34.2, 117.1, 5000, "u1", 10); err == nil {
		t.Fatal("expected nearby query error")
	}
}

func TestG3RepoMopUpBranches(t *testing.T) {
	ctx := context.Background()
	empty := g3EmptyRepo()
	if _, err := empty.SearchProfiles(ctx, SearchUsersRequest{Tag: "t"}); err == nil {
		t.Fatal("expected SearchProfiles not-implemented error")
	}
	if err := empty.Unfollow(ctx, "u1", "u2"); err == nil {
		t.Fatal("expected Unfollow not-implemented error")
	}
	if err := empty.Block(ctx, "u1", "u2", "spam"); err == nil {
		t.Fatal("expected Block not-implemented error")
	}
	if err := empty.BlockCascade(ctx, "u1", "u2", "spam"); err == nil {
		t.Fatal("expected BlockCascade not-implemented error")
	}
	if err := empty.Unblock(ctx, "u1", "u2"); err == nil {
		t.Fatal("expected Unblock not-implemented error")
	}
	if err := empty.SetRemark(ctx, "u1", "u2", "buddy"); err == nil {
		t.Fatal("expected SetRemark not-implemented error")
	}
	if isPhoneFormatKeyword("") {
		t.Fatal("empty keyword must not be phone format")
	}

	// buildUpdateProfileSQL 的 avatar/bio/gender 分支经由 UpdateProfile 全字段 patch 覆盖。
	backend := &g3Backend{queryRowByQuery: map[string]persistence.Row{}, rowsByQuery: map[string]persistence.Rows{}, queryErrs: map[string]error{}, execErrs: map[string]error{}}
	repo := g3RepoWith(backend)
	nickname, avatar, bio := "Nick", "avatar", "bio"
	gender := string(GenderMale)
	fullPatch := ProfilePatch{Nickname: &nickname, AvatarURL: &avatar, Bio: &bio, Gender: &gender}
	fullSQL, fullArgs := buildUpdateProfileSQL("u1", fullPatch)
	if len(fullArgs) != 5 || !strings.Contains(fullSQL, "gender") {
		t.Fatalf("full profile SQL = %q args=%v", fullSQL, fullArgs)
	}
	backend.queryRowByQuery[fullSQL] = testdb.RowValues{Values: []any{1}}
	if found, err := repo.UpdateProfile(ctx, "u1", fullPatch, nil); err != nil || !found {
		t.Fatalf("full profile update found=%v err=%v", found, err)
	}

	// BlockCascade 三步写：成功与各步失败。
	backend.execErrs = map[string]error{}
	if err := repo.BlockCascade(ctx, "u1", "u2", "spam"); err != nil {
		t.Fatalf("BlockCascade: %v", err)
	}
	for _, tc := range []struct {
		sql  string
		call func() error
	}{
		{queryUpsertUserRelationUnfollow, func() error { return repo.BlockCascade(ctx, "u1", "u2", "spam") }},
		{queryUpsertUserRelationBlock, func() error { return repo.BlockCascade(ctx, "u1", "u2", "spam") }},
	} {
		backend.execErrs = map[string]error{tc.sql: errors.New("boom")}
		if err := tc.call(); err == nil {
			t.Fatalf("expected BlockCascade error for %s", tc.sql)
		}
	}
}
