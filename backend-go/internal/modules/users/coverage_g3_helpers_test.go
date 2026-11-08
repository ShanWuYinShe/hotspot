package users

import (
	"context"
	"errors"
	"testing"

	"time"

	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

func g3IntPtr(v int) *int { return &v }

func g3FixedTime() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

// g3ActorProvider 是 actor.Provider 的可注入桩（deps.go 的降级分支用）。
type g3ActorProvider struct {
	info actor.Info
	err  error
}

func (p g3ActorProvider) GetActorInfo(context.Context, string) (actor.Info, error) {
	return p.info, p.err
}

func TestG3GetActorInfoErrorFallback(t *testing.T) {
	boom := errors.New("actor lookup failed")
	got := GetActorInfo(context.Background(), g3ActorProvider{err: boom}, "user_x")
	if got.UserID != "user_x" || got.Nickname != "user_x" {
		t.Fatalf("expected fallback identity, got %#v", got)
	}
	want := actor.Info{UserID: "user_y", Nickname: "Y", AvatarURL: "avatar"}
	if got := GetActorInfo(context.Background(), g3ActorProvider{info: want}, "user_y"); got != want {
		t.Fatalf("expected provider info, got %#v", got)
	}
}

func TestG3NormalizeSearchRadiusAndSort(t *testing.T) {
	cases := []struct {
		radius *int
		want   int
	}{
		{nil, 5000},
		{g3IntPtr(50), 100},
		{g3IntPtr(100), 100},
		{g3IntPtr(3000), 3000},
		{g3IntPtr(60000), 50000},
	}
	for _, tc := range cases {
		if got := normalizeSearchRadius(tc.radius); got != tc.want {
			t.Fatalf("normalizeSearchRadius(%v) = %d, want %d", tc.radius, got, tc.want)
		}
	}
	for input, want := range map[string]string{"distance": "distance", "newest": "newest", "": "relevance", "bogus": "relevance"} {
		if got := normalizeSearchSort(input); got != want {
			t.Fatalf("normalizeSearchSort(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestG3SortSearchCandidatesAllModes(t *testing.T) {
	profile := func(userID, lastOnline, createdAt string) MyProfile {
		return MyProfile{UserID: userID, LastOnline: lastOnline, CreatedAt: createdAt}
	}
	early := "2026-01-01T00:00:00Z"
	late := "2026-01-01T00:00:09Z"

	distanceItems := []searchProfileCandidate{
		{Profile: profile("u_d", early, early), Distance: 10, RelevanceScore: 7},
		{Profile: profile("u_b", early, early), Distance: 10, RelevanceScore: 9},
		{Profile: profile("u_e", late, early), Distance: 10, RelevanceScore: 7},
		{Profile: profile("u_a", early, early), Distance: 5},
		{Profile: profile("u_c", early, early), Distance: 10, RelevanceScore: 7},
	}
	sortSearchCandidates(distanceItems, "distance")
	wantDistance := []string{"u_a", "u_b", "u_e", "u_c", "u_d"}
	for i, want := range wantDistance {
		if distanceItems[i].Profile.UserID != want {
			t.Fatalf("distance sort[%d] = %s, want %s", i, distanceItems[i].Profile.UserID, want)
		}
	}

	newestItems := []searchProfileCandidate{
		{Profile: profile("u_a", early, early)},
		{Profile: profile("u_c", early, late)},
		{Profile: profile("u_b", early, late)},
	}
	sortSearchCandidates(newestItems, "newest")
	wantNewest := []string{"u_b", "u_c", "u_a"}
	for i, want := range wantNewest {
		if newestItems[i].Profile.UserID != want {
			t.Fatalf("newest sort[%d] = %s, want %s", i, newestItems[i].Profile.UserID, want)
		}
	}

	relevanceItems := []searchProfileCandidate{
		{Profile: profile("u_a", early, early), RelevanceScore: 5},
		{Profile: profile("u_b", early, early), RelevanceScore: 10},
		{Profile: profile("u_d", late, early), RelevanceScore: 10},
		{Profile: profile("u_c", late, early), RelevanceScore: 10},
	}
	sortSearchCandidates(relevanceItems, "relevance")
	wantRelevance := []string{"u_c", "u_d", "u_b", "u_a"}
	for i, want := range wantRelevance {
		if relevanceItems[i].Profile.UserID != want {
			t.Fatalf("relevance sort[%d] = %s, want %s", i, relevanceItems[i].Profile.UserID, want)
		}
	}
}

func TestG3SearchRelevanceScore(t *testing.T) {
	exact := MyProfile{Nickname: "alice", Bio: "alice fan", Tags: []string{"美食"}}
	if got := searchRelevanceScore(exact, "alice", ""); got != 6+2 {
		t.Fatalf("exact nickname+bio score = %d, want 8", got)
	}
	prefix := MyProfile{Nickname: "alicex", Bio: ""}
	if got := searchRelevanceScore(prefix, "alice", ""); got != 4 {
		t.Fatalf("prefix nickname score = %d, want 4", got)
	}
	contains := MyProfile{Nickname: "xalicex", Bio: ""}
	if got := searchRelevanceScore(contains, "alice", ""); got != 3 {
		t.Fatalf("contains nickname score = %d, want 3", got)
	}
	tagOnly := MyProfile{Nickname: "bob", Tags: []string{"美食"}}
	if got := searchRelevanceScore(tagOnly, "", "美食"); got != 3 {
		t.Fatalf("tag score = %d, want 3", got)
	}
	none := MyProfile{Nickname: "bob"}
	if got := searchRelevanceScore(none, "zzz", "other"); got != 0 {
		t.Fatalf("no-match score = %d, want 0", got)
	}
}

func TestG3PaginateSearchCandidates(t *testing.T) {
	items := []searchProfileCandidate{
		{Profile: MyProfile{UserID: "u1", CreatedAt: "2026-01-01T00:00:01Z"}, Distance: 1, RelevanceScore: 5, SortBy: "relevance"},
		{Profile: MyProfile{UserID: "u2", CreatedAt: "2026-01-01T00:00:02Z"}, Distance: 2, RelevanceScore: 4, SortBy: "relevance"},
		{Profile: MyProfile{UserID: "u3", CreatedAt: "2026-01-01T00:00:03Z"}, Distance: 3, RelevanceScore: 3, SortBy: "relevance"},
	}
	first := paginateSearchCandidates(items, &types.PaginationParams{Limit: 2})
	if len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("first page = %#v, want 2 items and cursor", first)
	}
	if first.Items[0].UserID != "u1" || first.Items[1].UserID != "u2" {
		t.Fatalf("first page order = %v", first.Items)
	}
	second := paginateSearchCandidates(items, &types.PaginationParams{Limit: 2, Cursor: first.NextCursor})
	if len(second.Items) != 1 || second.Items[0].UserID != "u3" {
		t.Fatalf("second page = %#v, want only u3", second)
	}
	if second.NextCursor != "" {
		t.Fatalf("second page must not have cursor, got %q", second.NextCursor)
	}

	if page := paginateSearchCandidates(items, &types.PaginationParams{Limit: -1}); len(page.Items) != 3 {
		t.Fatalf("negative limit must fall back to default, got %#v", page)
	}
	if page := paginateSearchCandidates(items, nil); len(page.Items) != 3 {
		t.Fatalf("nil pagination must return all items, got %#v", page)
	}
	invalid := paginateSearchCandidates(items, &types.PaginationParams{Limit: 2, Cursor: "not-a-cursor"})
	if !invalid.InvalidCursor {
		t.Fatal("expected invalid cursor flag for garbage cursor")
	}
	// 合法但不在候选集内的游标同样视为无效（翻页位置已漂移）。
	staleCursor := encodeSearchCursor(searchProfileCandidate{Profile: MyProfile{UserID: "u_missing"}, SortBy: "relevance"})
	if staleCursor == "" {
		t.Fatal("expected encoded cursor")
	}
	stale := paginateSearchCandidates(items, &types.PaginationParams{Cursor: staleCursor})
	if !stale.InvalidCursor {
		t.Fatal("expected invalid cursor flag for unmatched cursor")
	}
}

func TestG3SearchCursorEncoding(t *testing.T) {
	item := searchProfileCandidate{
		Profile:        MyProfile{UserID: "user_cursor", CreatedAt: "2026-01-02T03:04:05Z", LastOnline: "2026-01-02T03:04:06Z"},
		Distance:       42,
		SortBy:         "distance",
		RelevanceScore: 3,
	}
	encoded := encodeSearchCursor(item)
	if encoded == "" {
		t.Fatal("expected non-empty cursor")
	}
	decoded, err := decodeSearchCursor(encoded)
	if err != nil {
		t.Fatalf("decodeSearchCursor: %v", err)
	}
	if decoded.UserID != item.Profile.UserID || decoded.SortBy != item.SortBy || decoded.Distance != item.Distance {
		t.Fatalf("decoded cursor = %#v", decoded)
	}
	if _, err := decodeSearchCursor("garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
	missingFields, err := usersCursorCodec.Encode(searchCursor{UserID: "u_only"})
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	if _, err := decodeSearchCursor(missingFields); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("cursor missing sort_by must be invalid, got %v", err)
	}
}

func TestG3ContainsFoldMaxIntAndMomentCursor(t *testing.T) {
	if !containsFold([]string{"Foo", "bar"}, "foo") {
		t.Fatal("expected case-insensitive match")
	}
	if containsFold(nil, "foo") {
		t.Fatal("empty slice must not match")
	}
	if maxInt(3, 2) != 3 || maxInt(2, 3) != 3 {
		t.Fatal("maxInt returned wrong value")
	}
	// 游标解析：缺失字段/非法时间格式/非法 base64 均归为 ErrInvalidCursor。
	badAt, err := usersCursorCodec.Encode(userMomentCursor{At: "not-a-time", ID: "m1"})
	if err != nil {
		t.Fatalf("encode moment cursor: %v", err)
	}
	if _, _, err := decodeUserMomentCursor(badAt); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("bad time must be invalid cursor, got %v", err)
	}
	missingID, err := usersCursorCodec.Encode(userMomentCursor{At: "2026-01-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("encode moment cursor: %v", err)
	}
	if _, _, err := decodeUserMomentCursor(missingID); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("missing id must be invalid cursor, got %v", err)
	}
	if _, _, err := decodeUserMomentCursor("garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("garbage must be invalid cursor, got %v", err)
	}
	at, id, err := decodeUserMomentCursor("")
	if err != nil || at != nil || id != "" {
		t.Fatalf("empty cursor = %v %v %v", at, id, err)
	}
}

func TestG3APIErrorFromServiceMapping(t *testing.T) {
	cases := []struct {
		err  error
		code httpx.Code
	}{
		{ErrUnauthorized, httpx.CodeUnauthorized},
		{ErrPermissionDenied, httpx.CodeForbidden},
		{ErrUserNotFound, httpx.CodeNotFound},
		{ErrBlockedTag, httpx.CodeConflict},
		{ErrOwnsGroups, httpx.CodeConflict},
		{ErrLoginIDTaken, httpx.CodeConflict},
		{ErrAvatarNotOwned, httpx.CodeNotFound},
		{ErrInvalidProfileField, httpx.CodeInvalidRequest},
		{ErrConflict, httpx.CodeConflict},
		{ErrInvalidRequest, httpx.CodeInvalidRequest},
		{ErrInvalidCursor, httpx.CodeInvalidRequest},
		{errors.New("unexpected"), httpx.CodeInternalUnavailable},
	}
	for _, tc := range cases {
		apiErr := apiErrorFromService(tc.err)
		if apiErr.Code != tc.code {
			t.Fatalf("apiErrorFromService(%v).Code = %d, want %d", tc.err, apiErr.Code, tc.code)
		}
	}
	forbidden := apiErrorFromService(ErrPermissionDenied)
	if forbidden.Data == nil {
		t.Fatal("permission denied must carry forbidden_reason data")
	}
}

// g3ExecRecorderBackend 记录 Exec/QueryRow 调用；Exec 恒成功，用于覆盖
// profile writer 成功分支。
type g3ExecRecorderBackend struct {
	execs int
}

func (b *g3ExecRecorderBackend) Ping(context.Context) error { return nil }
func (b *g3ExecRecorderBackend) Close() error               { return nil }
func (b *g3ExecRecorderBackend) Exec(context.Context, string, ...any) error {
	b.execs++
	return nil
}
func (b *g3ExecRecorderBackend) QueryRow(context.Context, string, ...any) persistence.Row {
	return g3ErrorRow{}
}
func (b *g3ExecRecorderBackend) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}

type g3ErrorRow struct{}

func (g3ErrorRow) Scan(...any) error { return errors.New("scan failed") }

func TestG3ProfileWriterService(t *testing.T) {
	// 内存仓储不实现 profileWriterStore → 返回不支持错误。
	memSvc := NewServiceWithRepo(NewInMemoryRepository())
	if err := memSvc.ProvisionUser(context.Background(), ProvisionUserInput{UserID: "u1"}); !errors.Is(err, errProfileWriterUnsupported) {
		t.Fatalf("expected unsupported error, got %v", err)
	}
	if err := memSvc.RecordLogin(context.Background(), "u1", "1.2.3.4", "x"); !errors.Is(err, errProfileWriterUnsupported) {
		t.Fatalf("expected unsupported error, got %v", err)
	}

	backend := &g3ExecRecorderBackend{}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://g3"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	svc := NewServiceWithRepo(NewPostgresRepository(factory))
	input := ProvisionUserInput{
		UserID:           "g3_user_1",
		Nickname:         "G3",
		Email:            "g3@example.com",
		Phone:            "13800000000",
		LocationName:     "徐州",
		Latitude:         34.2,
		Longitude:        117.1,
		RegisterIP:       "1.2.3.4",
		RegisterLocation: "x",
	}
	input.CreatedAt = g3FixedTime()
	if err := svc.ProvisionUser(context.Background(), input); err != nil {
		t.Fatalf("provision user: %v", err)
	}
	// 空 email/phone 走 nullIfEmpty 的 NULL 分支。
	empty := input
	empty.Email = ""
	empty.Phone = ""
	if err := svc.ProvisionUser(context.Background(), empty); err != nil {
		t.Fatalf("provision user without contacts: %v", err)
	}
	if err := svc.RecordLogin(context.Background(), "g3_user_1", "5.6.7.8", "y"); err != nil {
		t.Fatalf("record login: %v", err)
	}
	if backend.execs != 5 {
		t.Fatalf("expected 5 exec calls, got %d", backend.execs)
	}

	// Exec 错误原样上抛。
	errBackend := &userBackendStub{
		queryRowByQuery:  map[string]persistence.Row{},
		queryRowsByQuery: map[string][][]any{},
		queryErrs:        map[string]error{},
		execErrs:         map[string]error{queryProvisionUserProfile: errors.New("insert failed")},
	}
	errFactory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://g3"})
	errFactory.DB.(*persistence.PostgresDB).Backend = errBackend
	errSvc := NewServiceWithRepo(NewPostgresRepository(errFactory))
	if err := errSvc.ProvisionUser(context.Background(), input); err == nil {
		t.Fatal("expected provision user error")
	}
	errBackend.execErrs = map[string]error{queryRecordLoginMeta: errors.New("update failed")}
	if err := errSvc.RecordLogin(context.Background(), "g3_user_1", "ip", "loc"); err == nil {
		t.Fatal("expected record login error")
	}
}
