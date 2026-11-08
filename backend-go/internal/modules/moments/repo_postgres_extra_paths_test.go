package moments

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

// pathsBackend 是按 SQL 常量注入固定行的 Queryable 测试替身，用于驱动
// PostgresRepository 常量 SQL 读/写路径的分支（成功、查询错误、扫描错误、
// 迭代错误、Exec 错误）。所有断言都针对仓储返回值本身。
type pathsBackend struct {
	queryRows     map[string][][]any
	queryErr      map[string]error
	rowsErr       map[string]error
	scanErr       map[string]error
	rowValues     map[string][]any
	rowErr        map[string]error
	execErr       map[string]error
	execErrPrefix []string
	queries       []string
	execs         []string
}

func newPathsBackend() *pathsBackend {
	return &pathsBackend{
		queryRows: make(map[string][][]any),
		queryErr:  make(map[string]error),
		rowsErr:   make(map[string]error),
		scanErr:   make(map[string]error),
		rowValues: make(map[string][]any),
		rowErr:    make(map[string]error),
		execErr:   make(map[string]error),
	}
}

func (b *pathsBackend) Exec(_ context.Context, query string, _ ...any) error {
	b.execs = append(b.execs, query)
	if err := b.execErr[query]; err != nil {
		return err
	}
	for _, prefix := range b.execErrPrefix {
		if strings.HasPrefix(query, prefix) {
			return errors.New("exec prefix failure")
		}
	}
	return nil
}

func (b *pathsBackend) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	if err := b.rowErr[query]; err != nil {
		return rowValues{err: err}
	}
	if values, ok := b.rowValues[query]; ok {
		return rowValues{values: values}
	}
	return rowValues{err: pgx.ErrNoRows}
}

func (b *pathsBackend) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	b.queries = append(b.queries, query)
	if err := b.queryErr[query]; err != nil {
		return nil, err
	}
	return &pathsRows{rows: b.queryRows[query], err: b.rowsErr[query], scanErr: b.scanErr[query]}, nil
}

type pathsRows struct {
	rows    [][]any
	idx     int
	err     error
	scanErr error
}

func (r *pathsRows) Next() bool { return r.idx < len(r.rows) }
func (r *pathsRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := rowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *pathsRows) Close() error { return nil }
func (r *pathsRows) Err() error   { return r.err }

func newPathsRepo(backend *pathsBackend) *PostgresRepository {
	return NewPostgresRepository(newMomentFactory(backend))
}

func pathsMomentRow(momentID, userID, nickname, description string, createdAt time.Time, privacy string) []any {
	return []any{momentID, userID, nickname, "", description, createdAt, 0, 0, 0, "南天门", 35.0, 112.0, privacy, false}
}

func pathsReactedRow(momentID, userID, nickname string, createdAt, reactedAt time.Time, favorited bool) []any {
	return []any{momentID, userID, nickname, "", "desc", createdAt, 0, 0, 0, "南天门", 35.0, 112.0, "public", reactedAt, favorited}
}

// TestPostgresRepositoryListReactedByUser 覆盖已反应列表：分页游标
// （reacted_at 全精度）、标签补全、昵称回退与非法游标/未配置出口。
func TestPostgresRepositoryListReactedByUser(t *testing.T) {
	base := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	reactedAt := base.Add(37 * time.Minute)
	backend := newPathsBackend()
	backend.queryRows[queryListReactedByUser] = [][]any{
		pathsReactedRow("m_reacted_1", "u_author", "", base, reactedAt, true),
		pathsReactedRow("m_reacted_2", "u_author", "nick", base.Add(-time.Minute), reactedAt.Add(-time.Minute), false),
		pathsReactedRow("m_reacted_3", "u_author", "nick", base.Add(-2*time.Minute), reactedAt.Add(-2*time.Minute), false),
	}
	backend.queryRows[queryListMomentTagsByMomentIDs] = [][]any{
		{"m_reacted_1", "旅行"},
		{"m_reacted_2", "美食"},
	}
	repo := newPathsRepo(backend)

	page, err := repo.ListReactedByUser(context.Background(), "u_author", "u_viewer", 2, "")
	if err != nil {
		t.Fatalf("ListReactedByUser: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Items))
	}
	if page.Items[0].MomentID != "m_reacted_1" || page.Items[1].MomentID != "m_reacted_2" {
		t.Fatalf("unexpected items: %#v", page.Items)
	}
	if page.Items[0].User.Nickname != "u_author" {
		t.Fatalf("empty nickname must fall back to user id, got %q", page.Items[0].User.Nickname)
	}
	if !page.Items[0].IsFavorited || page.Items[1].IsFavorited {
		t.Fatalf("is_favorited must come from scan: %#v", page.Items)
	}
	if len(page.Items[0].Tags) != 1 || page.Items[0].Tags[0] != "旅行" {
		t.Fatalf("tags not filled: %#v", page.Items[0].Tags)
	}
	if page.NextCursor == nil {
		t.Fatal("expected next cursor when a limit+1 row exists")
	}
	// 游标锚定页尾项 reacted_at 全精度（纳秒），解码往返一致。
	at, id, err := decodeMomentsByUserCursor(*page.NextCursor)
	if err != nil || id != "m_reacted_2" || !at.Equal(reactedAt.Add(-time.Minute)) {
		t.Fatalf("unexpected cursor anchor: at=%v id=%s err=%v", at, id, err)
	}

	// 第二页（无多余行）不返回游标。
	backend.queryRows[queryListReactedByUser] = [][]any{
		pathsReactedRow("m_reacted_3", "u_author", "nick", base.Add(-2*time.Minute), reactedAt.Add(-2*time.Minute), false),
	}
	page2, err := repo.ListReactedByUser(context.Background(), "u_author", "u_viewer", 2, *page.NextCursor)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(page2.Items) != 1 || page2.Items[0].MomentID != "m_reacted_3" || page2.NextCursor != nil {
		t.Fatalf("unexpected second page: %#v cursor=%v", page2.Items, page2.NextCursor)
	}

	// 非法游标 → ErrInvalidCursor（不得当作首页静默重放）。
	if _, err := repo.ListReactedByUser(context.Background(), "u_author", "", 2, "garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("garbage cursor: expected ErrInvalidCursor, got %v", err)
	}

	// 未配置持久化 → ErrNotImplemented。
	if _, err := NewPostgresRepository(persistence.NewFactory(config.Config{})).ListReactedByUser(context.Background(), "u_author", "", 2, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresRepositoryListReactedByUserErrors(t *testing.T) {
	ctx := context.Background()
	base := time.Now().UTC()

	// 查询失败
	backend := newPathsBackend()
	backend.queryErr[queryListReactedByUser] = errMomentFake
	if _, err := newPathsRepo(backend).ListReactedByUser(ctx, "u1", "v1", 2, ""); !errors.Is(err, errMomentFake) {
		t.Fatalf("query error: got %v", err)
	}

	// 行扫描失败
	backend = newPathsBackend()
	backend.queryRows[queryListReactedByUser] = [][]any{pathsReactedRow("m1", "u1", "n", base, base, false)}
	backend.scanErr[queryListReactedByUser] = errMomentFake
	if _, err := newPathsRepo(backend).ListReactedByUser(ctx, "u1", "v1", 2, ""); !errors.Is(err, errMomentFake) {
		t.Fatalf("scan error: got %v", err)
	}

	// 迭代错误
	backend = newPathsBackend()
	backend.queryRows[queryListReactedByUser] = [][]any{
		pathsReactedRow("m1", "u1", "n", base, base, false),
	}
	backend.rowsErr[queryListReactedByUser] = errMomentFake
	if _, err := newPathsRepo(backend).ListReactedByUser(ctx, "u1", "v1", 2, ""); !errors.Is(err, errMomentFake) {
		t.Fatalf("rows error: got %v", err)
	}

	// 标签补全失败（页内路径）
	backend = newPathsBackend()
	backend.queryRows[queryListReactedByUser] = [][]any{
		pathsReactedRow("m1", "u1", "n", base, base, false),
	}
	backend.queryErr[queryListMomentTagsByMomentIDs] = errMomentFake
	if _, err := newPathsRepo(backend).ListReactedByUser(ctx, "u1", "v1", 2, ""); !errors.Is(err, errMomentFake) {
		t.Fatalf("tags error: got %v", err)
	}

	// 标签补全失败（limit+1 溢出页路径）
	backend = newPathsBackend()
	backend.queryRows[queryListReactedByUser] = [][]any{
		pathsReactedRow("m1", "u1", "n", base, base, false),
		pathsReactedRow("m2", "u1", "n", base, base.Add(-time.Minute), false),
		pathsReactedRow("m3", "u1", "n", base, base.Add(-2*time.Minute), false),
	}
	backend.queryErr[queryListMomentTagsByMomentIDs] = errMomentFake
	if _, err := newPathsRepo(backend).ListReactedByUser(ctx, "u1", "v1", 2, ""); !errors.Is(err, errMomentFake) {
		t.Fatalf("overflow tags error: got %v", err)
	}
}

// TestPostgresRepositoryListMomentsByAuthorPaths 覆盖作者动态列表的
// limit+1 游标、标签补全、扫描/迭代错误与未配置分支。
func TestPostgresRepositoryListMomentsByAuthorPaths(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 4, 2, 8, 0, 0, 0, time.UTC)
	backend := newPathsBackend()
	backend.queryRows[queryListMomentsByAuthor] = [][]any{
		pathsMomentRow("m_a1", "u_author", "作者", "first", base, "public"),
		pathsMomentRow("m_a2", "u_author", "", "second", base.Add(-time.Minute), "public"),
	}
	backend.queryRows[queryListMomentTagsByMomentIDs] = [][]any{{"m_a1", "tag1"}}
	repo := newPathsRepo(backend)

	page, err := repo.ListMomentsByAuthor(ctx, "u_author", 1, "", false)
	if err != nil {
		t.Fatalf("ListMomentsByAuthor: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].MomentID != "m_a1" || page.NextCursor == nil {
		t.Fatalf("unexpected page: %#v cursor=%v", page.Items, page.NextCursor)
	}
	if len(page.Items[0].Tags) != 1 || page.Items[0].Tags[0] != "tag1" {
		t.Fatalf("tags not filled: %#v", page.Items[0].Tags)
	}
	if page.Items[0].CreatedAt != base.Format(time.RFC3339) {
		t.Fatalf("created_at format: %q", page.Items[0].CreatedAt)
	}

	// 空结果
	backend.queryRows[queryListMomentsByAuthor] = nil
	empty, err := repo.ListMomentsByAuthor(ctx, "u_author", 10, "", false)
	if err != nil || len(empty.Items) != 0 || empty.NextCursor != nil {
		t.Fatalf("empty page: %#v err=%v", empty, err)
	}

	// 非法游标
	if _, err := repo.ListMomentsByAuthor(ctx, "u_author", 10, "garbage", false); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("garbage cursor: got %v", err)
	}

	// 未配置
	if _, err := NewPostgresRepository(persistence.NewFactory(config.Config{})).ListMomentsByAuthor(ctx, "u_author", 10, "", false); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}

	// 查询 / 扫描 / 迭代错误
	for name, setup := range map[string]func(*pathsBackend){
		"query": func(b *pathsBackend) { b.queryErr[queryListMomentsByAuthor] = errMomentFake },
		"scan": func(b *pathsBackend) {
			b.queryRows[queryListMomentsByAuthor] = [][]any{pathsMomentRow("m1", "u1", "n", "d", base, "public")}
			b.scanErr[queryListMomentsByAuthor] = errMomentFake
		},
		"rows": func(b *pathsBackend) {
			b.queryRows[queryListMomentsByAuthor] = [][]any{pathsMomentRow("m1", "u1", "n", "d", base, "public")}
			b.rowsErr[queryListMomentsByAuthor] = errMomentFake
		},
		"tags": func(b *pathsBackend) {
			b.queryRows[queryListMomentsByAuthor] = [][]any{pathsMomentRow("m1", "u1", "n", "d", base, "public")}
			b.queryErr[queryListMomentTagsByMomentIDs] = errMomentFake
		},
	} {
		b := newPathsBackend()
		setup(b)
		if _, err := newPathsRepo(b).ListMomentsByAuthor(ctx, "u1", 10, "", false); !errors.Is(err, errMomentFake) {
			t.Fatalf("%s: expected propagated error, got %v", name, err)
		}
	}
}

// TestPostgresRepositoryListMinePaths 覆盖我的动态列表的分页/错误分支。
func TestPostgresRepositoryListMinePaths(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 4, 3, 8, 0, 0, 0, time.UTC)
	backend := newPathsBackend()
	backend.queryRows[queryListMyMoments] = [][]any{
		pathsMomentRow("m1", "u1", "n", "one", base, "public"),
		pathsMomentRow("m2", "u1", "n", "two", base.Add(-time.Minute), "public"),
	}
	backend.queryRows[queryListMomentTagsByMomentIDs] = [][]any{{"m1", "t1"}}
	backend.queryRows[queryListMomentMentionsByMomentIDs] = [][]any{{"m1", "u2"}}
	repo := newPathsRepo(backend)

	page, err := repo.ListMine(ctx, "u1", 1, "")
	if err != nil {
		t.Fatalf("ListMine: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].MomentID != "m1" || page.NextCursor == nil {
		t.Fatalf("unexpected page: %#v", page)
	}
	if len(page.Items[0].Mentions) != 1 || page.Items[0].Mentions[0] != "u2" {
		t.Fatalf("mentions not filled: %#v", page.Items[0].Mentions)
	}

	// 非法游标
	if _, err := repo.ListMine(ctx, "u1", 1, "garbage"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("garbage cursor: got %v", err)
	}
	// 未配置
	if _, err := NewPostgresRepository(persistence.NewFactory(config.Config{})).ListMine(ctx, "u1", 1, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}

	for name, setup := range map[string]func(*pathsBackend){
		"query": func(b *pathsBackend) { b.queryErr[queryListMyMoments] = errMomentFake },
		"scan": func(b *pathsBackend) {
			b.queryRows[queryListMyMoments] = [][]any{pathsMomentRow("m1", "u1", "n", "d", base, "public")}
			b.scanErr[queryListMyMoments] = errMomentFake
		},
		"rows": func(b *pathsBackend) {
			b.queryRows[queryListMyMoments] = [][]any{pathsMomentRow("m1", "u1", "n", "d", base, "public")}
			b.rowsErr[queryListMyMoments] = errMomentFake
		},
		"tags": func(b *pathsBackend) {
			b.queryRows[queryListMyMoments] = [][]any{pathsMomentRow("m1", "u1", "n", "d", base, "public")}
			b.queryErr[queryListMomentTagsByMomentIDs] = errMomentFake
		},
		"mentions": func(b *pathsBackend) {
			b.queryRows[queryListMyMoments] = [][]any{pathsMomentRow("m1", "u1", "n", "d", base, "public")}
			b.queryErr[queryListMomentMentionsByMomentIDs] = errMomentFake
		},
	} {
		b := newPathsBackend()
		setup(b)
		if _, err := newPathsRepo(b).ListMine(ctx, "u1", 10, ""); !errors.Is(err, errMomentFake) {
			t.Fatalf("%s: expected propagated error, got %v", name, err)
		}
	}
}

// TestPostgresRepositoryLoadMyState 覆盖详情页查看者状态的三个出口。
func TestPostgresRepositoryLoadMyState(t *testing.T) {
	ctx := context.Background()
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	if st := repo.LoadMyState(ctx, "m1", "u1"); st != nil {
		t.Fatalf("unconfigured repo must return nil, got %#v", st)
	}

	backend := newPathsBackend()
	repo = newPathsRepo(backend)
	if st := repo.LoadMyState(ctx, "m1", ""); st != nil {
		t.Fatalf("anonymous caller must return nil, got %#v", st)
	}

	backend.rowValues[queryCheckMomentFavorite] = []any{1}
	if st := repo.LoadMyState(ctx, "m1", "u1"); st == nil || !st.IsFavorited {
		t.Fatalf("favorited row must set state: %#v", st)
	}

	backend.rowErr[queryCheckMomentFavorite] = pgx.ErrNoRows
	if st := repo.LoadMyState(ctx, "m1", "u1"); st == nil || st.IsFavorited {
		t.Fatalf("no rows must be treated as not favorited: %#v", st)
	}

	backend.rowErr[queryCheckMomentFavorite] = errMomentFake
	if st := repo.LoadMyState(ctx, "m1", "u1"); st == nil || st.IsFavorited {
		t.Fatalf("db error must degrade to not favorited: %#v", st)
	}
}

// TestLoadMomentFavorites 覆盖批量收藏补全（含匿名/空集合短路与错误包装）。
func TestLoadMomentFavorites(t *testing.T) {
	ctx := context.Background()
	backend := newPathsBackend()
	backend.queryRows[queryMomentFavoritesByIDs] = [][]any{{"m1"}, {"m3"}}

	favorited, err := loadMomentFavorites(ctx, backend, "u1", []string{"m1", "m2", "m3"})
	if err != nil {
		t.Fatalf("loadMomentFavorites: %v", err)
	}
	if !favorited["m1"] || favorited["m2"] || !favorited["m3"] {
		t.Fatalf("unexpected favorites: %#v", favorited)
	}

	// 匿名视角与空集合直接返回空集且不查库
	queriesBefore := len(backend.queries)
	favorited, err = loadMomentFavorites(ctx, backend, "", []string{"m1"})
	if err != nil || len(favorited) != 0 {
		t.Fatalf("anonymous: %#v err=%v", favorited, err)
	}
	favorited, err = loadMomentFavorites(ctx, backend, "u1", nil)
	if err != nil || len(favorited) != 0 {
		t.Fatalf("empty ids: %#v err=%v", favorited, err)
	}
	if len(backend.queries) != queriesBefore {
		t.Fatalf("short-circuit paths must not query, got %v", backend.queries[queriesBefore:])
	}

	backend.queryErr[queryMomentFavoritesByIDs] = errMomentFake
	if _, err := loadMomentFavorites(ctx, backend, "u1", []string{"m1"}); err == nil || !strings.Contains(err.Error(), "loadMomentFavorites query") {
		t.Fatalf("query error must be wrapped, got %v", err)
	}
	delete(backend.queryErr, queryMomentFavoritesByIDs)
	backend.scanErr[queryMomentFavoritesByIDs] = errMomentFake
	if _, err := loadMomentFavorites(ctx, backend, "u1", []string{"m1"}); err == nil || !strings.Contains(err.Error(), "loadMomentFavorites scan") {
		t.Fatalf("scan error must be wrapped, got %v", err)
	}
	delete(backend.scanErr, queryMomentFavoritesByIDs)
	backend.queryRows[queryMomentFavoritesByIDs] = [][]any{{"m1"}}
	backend.rowsErr[queryMomentFavoritesByIDs] = errMomentFake
	if _, err := loadMomentFavorites(ctx, backend, "u1", []string{"m1"}); err == nil || !strings.Contains(err.Error(), "loadMomentFavorites rows") {
		t.Fatalf("rows error must be wrapped, got %v", err)
	}
}

// TestPostgresRepositoryMomentImageRows 覆盖图片行批量读取与仓储出口。
func TestPostgresRepositoryMomentImageRows(t *testing.T) {
	ctx := context.Background()
	backend := newPathsBackend()
	backend.queryRows[queryListMomentImagesByMomentIDs] = [][]any{
		{"m1", "f1", "uploads/1.jpg", "", 100, 200, "image/jpeg"},
		{"m1", "f2", "uploads/2.jpg", "thumb/2.jpg", 0, 0, ""},
		{"m2", "f3", "uploads/3.jpg", "", 10, 10, "image/png"},
	}
	repo := newPathsRepo(backend)

	rows, err := repo.FetchMomentImageRows(ctx, []string{"m1", "m2"})
	if err != nil {
		t.Fatalf("FetchMomentImageRows: %v", err)
	}
	if len(rows["m1"]) != 2 || rows["m1"][0].FileID != "f1" || rows["m1"][1].StorageKey != "uploads/2.jpg" {
		t.Fatalf("unexpected m1 rows: %#v", rows["m1"])
	}
	if len(rows["m2"]) != 1 || rows["m2"][0].MimeType != "image/png" {
		t.Fatalf("unexpected m2 rows: %#v", rows["m2"])
	}

	// 空集合不查库
	queriesBefore := len(backend.queries)
	empty, err := loadMomentImageRows(ctx, backend, nil)
	if err != nil || len(empty) != 0 || len(backend.queries) != queriesBefore {
		t.Fatalf("empty ids: %#v err=%v queries=%d", empty, err, len(backend.queries))
	}

	// 未配置持久化：仓储出口返回空集 + nil（归属 hydrate 尽力而为语义）
	unconfigured := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	if rows, err := unconfigured.FetchMomentImageRows(ctx, []string{"m1"}); err != nil || len(rows) != 0 {
		t.Fatalf("unconfigured fetch: %#v err=%v", rows, err)
	}

	backend.queryErr[queryListMomentImagesByMomentIDs] = errMomentFake
	if _, err := loadMomentImageRows(ctx, backend, []string{"m1"}); !errors.Is(err, errMomentFake) {
		t.Fatalf("query error: got %v", err)
	}
	delete(backend.queryErr, queryListMomentImagesByMomentIDs)
	backend.scanErr[queryListMomentImagesByMomentIDs] = errMomentFake
	if _, err := loadMomentImageRows(ctx, backend, []string{"m1"}); !errors.Is(err, errMomentFake) {
		t.Fatalf("scan error: got %v", err)
	}
	delete(backend.scanErr, queryListMomentImagesByMomentIDs)
	backend.rowsErr[queryListMomentImagesByMomentIDs] = errMomentFake
	if _, err := loadMomentImageRows(ctx, backend, []string{"m1"}); !errors.Is(err, errMomentFake) {
		t.Fatalf("rows error: got %v", err)
	}
}

// TestPostgresRepositoryTagAssignments 覆盖管理端标签关联查询/删除。
func TestPostgresRepositoryTagAssignments(t *testing.T) {
	ctx := context.Background()
	backend := newPathsBackend()
	repo := newPathsRepo(backend)

	backend.rowValues[queryMomentTagAssignmentsExist] = []any{true}
	has, err := repo.HasTagAssignments(ctx, "旅行")
	if err != nil || !has {
		t.Fatalf("HasTagAssignments true: %v %v", has, err)
	}
	backend.rowValues[queryMomentTagAssignmentsExist] = []any{false}
	has, err = repo.HasTagAssignments(ctx, "不存在")
	if err != nil || has {
		t.Fatalf("HasTagAssignments false: %v %v", has, err)
	}
	backend.rowErr[queryMomentTagAssignmentsExist] = errMomentFake
	if _, err := repo.HasTagAssignments(ctx, "x"); !errors.Is(err, errMomentFake) {
		t.Fatalf("HasTagAssignments scan error: %v", err)
	}

	if err := repo.DeleteTagAssignments(ctx, "旅行"); err != nil {
		t.Fatalf("DeleteTagAssignments: %v", err)
	}
	if len(backend.execs) != 1 || backend.execs[0] != queryDeleteMomentTagAssignments {
		t.Fatalf("unexpected execs: %v", backend.execs)
	}
	backend.execErr[queryDeleteMomentTagAssignments] = errMomentFake
	if err := repo.DeleteTagAssignments(ctx, "旅行"); !errors.Is(err, errMomentFake) {
		t.Fatalf("DeleteTagAssignments exec error: %v", err)
	}

	unconfigured := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	if _, err := unconfigured.HasTagAssignments(ctx, "x"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("unconfigured HasTagAssignments: %v", err)
	}
	if err := unconfigured.DeleteTagAssignments(ctx, "x"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("unconfigured DeleteTagAssignments: %v", err)
	}
}

// TestPostgresRepositoryPurgeUserData 覆盖账户注销级联删除。
func TestPostgresRepositoryPurgeUserData(t *testing.T) {
	ctx := context.Background()
	backend := newPathsBackend()
	repo := newPathsRepo(backend)
	if err := repo.PurgeUserData(ctx, "u1"); err != nil {
		t.Fatalf("PurgeUserData: %v", err)
	}
	if len(backend.execs) != 1 || backend.execs[0] != queryPurgeUserMomentFavorites {
		t.Fatalf("unexpected execs: %v", backend.execs)
	}
	backend.execErr[queryPurgeUserMomentFavorites] = errMomentFake
	if err := repo.PurgeUserData(ctx, "u1"); !errors.Is(err, errMomentFake) {
		t.Fatalf("PurgeUserData exec error: %v", err)
	}
	unconfigured := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	if err := unconfigured.PurgeUserData(ctx, "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("unconfigured PurgeUserData: %v", err)
	}
}

// pathsDynamicBackend 按查询特征返回固定行的动态 SQL 替身（Search/附近流
// 的 SQL 由 builder 生成，无法按常量键匹配）。
type pathsDynamicBackend struct {
	searchViewerRows [][]any
	searchAnonRows   [][]any
	nearbyRows       [][]any
	nearbyRowsErr    error
	nearbyScanErr    error
	nearbyQueryErr   error
	detailRows       map[string][]any
	detailsRowsErr   error
	detailsQueryErr  error
	tagsRows         [][]any
	queries          []string
	queryErr         error
	rowsErr          error
	execs            []string
	execErr          error
}

func (b *pathsDynamicBackend) Exec(_ context.Context, query string, _ ...any) error {
	b.execs = append(b.execs, query)
	return b.execErr
}

func (b *pathsDynamicBackend) QueryRow(context.Context, string, ...any) persistence.Row {
	return rowValues{err: pgx.ErrNoRows}
}

func (b *pathsDynamicBackend) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	b.queries = append(b.queries, query)
	if b.queryErr != nil {
		return nil, b.queryErr
	}
	switch {
	case query == queryListMomentTagsByMomentIDs:
		return &pathsRows{rows: b.tagsRows}, nil
	case query == queryNearbyMomentDetails:
		if b.detailsQueryErr != nil {
			return nil, b.detailsQueryErr
		}
		ids, _ := args[0].([]string)
		rows := make([][]any, 0, len(ids))
		for _, id := range ids {
			if row, ok := b.detailRows[id]; ok {
				rows = append(rows, row)
			}
		}
		return &pathsRows{rows: rows, err: b.detailsRowsErr}, nil
	case strings.Contains(query, "WITH blocked AS"):
		if b.nearbyQueryErr != nil {
			return nil, b.nearbyQueryErr
		}
		return &pathsRows{rows: b.nearbyRows, err: b.nearbyRowsErr, scanErr: b.nearbyScanErr}, nil
	case strings.Contains(query, "FROM moments m LEFT JOIN users u"):
		if strings.Contains(query, "EXISTS(SELECT 1 FROM moment_favorites mf") {
			return &pathsRows{rows: b.searchViewerRows, err: b.rowsErr}, nil
		}
		return &pathsRows{rows: b.searchAnonRows, err: b.rowsErr}, nil
	default:
		return &persistence.InMemoryRows{}, nil
	}
}

// TestPostgresRepositorySearchBranches 覆盖搜索的查看者收藏列、溢出行游标与错误出口，
// 以及 searchScoreSQL 的三种排序表达式分支。
func TestPostgresRepositorySearchBranches(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 4, 4, 8, 0, 0, 0, time.UTC)

	dynamic := &pathsDynamicBackend{
		searchViewerRows: [][]any{
			{"m1", "u1", "n", "", "hello", base, 0, 0, 0, "南天门", 35.0, 112.0, "public", false, true},
			{"m2", "u1", "n", "", "hello2", base.Add(-time.Minute), 0, 0, 0, "南天门", 35.0, 112.0, "public", false, false},
		},
		tagsRows: [][]any{{"m1", "tag"}},
	}
	repo := NewPostgresRepository(newMomentFactory(dynamic))
	keyword := "hello"
	page, err := repo.Search(ctx, SearchMomentsRequest{
		Keyword:    &keyword,
		SortBy:     "newest",
		Pagination: &PaginationParams{Limit: 1},
	}, "u_viewer")
	if err != nil {
		t.Fatalf("Search viewer: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].MomentID != "m1" || !page.Items[0].IsFavorited {
		t.Fatalf("unexpected viewer search page: %#v", page.Items)
	}
	if page.NextCursor == nil || len(page.Items[0].Tags) != 1 {
		t.Fatalf("expected next cursor and tags, got %#v cursor=%v", page.Items[0].Tags, page.NextCursor)
	}
	if !strings.Contains(dynamic.queries[0], "EXISTS(SELECT 1 FROM moment_favorites mf") {
		t.Fatalf("viewer query must resolve favorite state: %s", dynamic.queries[0])
	}

	// 匿名搜索（无查看者列）走 scanMomentRow 分支。
	dynamic.searchAnonRows = [][]any{
		{"m3", "u1", "n", "", "hello3", base, 0, 0, 0, "南天门", 35.0, 112.0, "public", false},
	}
	anonPage, err := repo.Search(ctx, SearchMomentsRequest{Keyword: &keyword, Pagination: &PaginationParams{Limit: 5}}, "")
	if err != nil {
		t.Fatalf("Search anonymous: %v", err)
	}
	if len(anonPage.Items) != 1 || anonPage.Items[0].MomentID != "m3" || anonPage.NextCursor != nil {
		t.Fatalf("unexpected anonymous page: %#v", anonPage)
	}

	// 非法搜索游标 → ErrInvalidCursor
	if _, err := repo.Search(ctx, SearchMomentsRequest{Keyword: &keyword, Pagination: &PaginationParams{Cursor: "garbage"}}, ""); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("garbage search cursor: %v", err)
	}

	// 查询错误
	dynamic.queryErr = errMomentFake
	if _, err := repo.Search(ctx, SearchMomentsRequest{Keyword: &keyword}, ""); !errors.Is(err, errMomentFake) {
		t.Fatalf("search query error: %v", err)
	}
	dynamic.queryErr = nil

	// 未配置
	if _, err := NewPostgresRepository(persistence.NewFactory(config.Config{})).Search(ctx, SearchMomentsRequest{}, ""); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("unconfigured search: %v", err)
	}

	// searchScoreSQL 分支
	if got := searchScoreSQL("popular", 0, 0, 3); !strings.Contains(got, "POWER(") || !strings.Contains(got, "$3::float8") {
		t.Fatalf("popular score expr: %s", got)
	}
	if got := searchScoreSQL("newest", 0, 0, 0); got != "0::float8" {
		t.Fatalf("newest score expr: %s", got)
	}
	if got := searchScoreSQL("", 0, 0, 0); got != "0::float8" {
		t.Fatalf("empty relevance expr: %s", got)
	}
	if got := searchScoreSQL("relevance", 2, 0, 0); !strings.Contains(got, "ILIKE") || !strings.Contains(got, "$2") {
		t.Fatalf("keyword relevance expr: %s", got)
	}
	if got := searchScoreSQL("relevance", 0, 4, 0); got != "1" {
		t.Fatalf("tag relevance expr: %s", got)
	}
	if got := searchScoreSQL("relevance", 2, 4, 0); !strings.Contains(got, " + 1") {
		t.Fatalf("keyword+tag expr: %s", got)
	}
}

func TestPostgresEscapeLikePattern(t *testing.T) {
	if got := escapeLikePattern("50%_off\\path"); got != "50\\%\\_off\\\\path" {
		t.Fatalf("escapeLikePattern: %q", got)
	}
	if got := escapeLikePattern(""); got != "" {
		t.Fatalf("empty input: %q", got)
	}
}

// TestPostgresRepositoryListFollowingMoments 覆盖关注流的参考点填充、排序、
// 游标、错误出口与未配置分支。
func TestPostgresRepositoryListFollowingMoments(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 4, 5, 8, 0, 0, 0, time.UTC)
	backend := newPathsBackend()
	backend.queryRows[queryFollowingMoments] = [][]any{
		{"m1", "u1", "n1", "", "newer", base, 0, 0, 0, "南天门", 35.0, 112.0, false, []string{"t1"}},
		{"m2", "u2", "n2", "", "older", base.Add(-time.Minute), 0, 0, 0, "南天门", 35.1, 112.0, true, []string(nil)},
	}
	repo := newPathsRepo(backend)

	page, err := repo.ListFollowingMoments(ctx, momentsQuery{
		ViewerUserID:      "viewer",
		ReferencePoint:    GeoPoint{Latitude: 35.0, Longitude: 112.0},
		HasReferencePoint: true,
		Limit:             1,
	})
	if err != nil {
		t.Fatalf("ListFollowingMoments: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].MomentID != "m1" || page.NextCursor == nil {
		t.Fatalf("unexpected page: %#v", page)
	}
	if page.Items[0].Location.Distance == nil || page.Items[0].Location.BearingDegrees == nil {
		t.Fatalf("relative location must be filled: %#v", page.Items[0].Location)
	}
	if page.Items[0].User.Nickname != "n1" || len(page.Items[0].Tags) != 1 {
		t.Fatalf("item fields: %#v", page.Items[0])
	}

	// 第二页：无参考点时不填充 distance；游标继续从页尾摘要开始。
	backend.queryRows[queryFollowingMoments] = [][]any{
		{"m2", "u2", "n2", "", "older", base.Add(-time.Minute), 0, 0, 0, "南天门", 35.1, 112.0, true, []string{}},
	}
	page2, err := repo.ListFollowingMoments(ctx, momentsQuery{ViewerUserID: "viewer", Limit: 1, Cursor: *page.NextCursor})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(page2.Items) != 1 || page2.Items[0].MomentID != "m2" || page2.NextCursor != nil {
		t.Fatalf("unexpected second page: %#v", page2)
	}
	if page2.Items[0].Location.Distance != nil {
		t.Fatalf("no reference point must leave distance nil: %#v", page2.Items[0].Location)
	}

	// 非法游标
	if _, err := repo.ListFollowingMoments(ctx, momentsQuery{Cursor: "garbage"}); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("garbage cursor: %v", err)
	}
	// 未配置
	if _, err := NewPostgresRepository(persistence.NewFactory(config.Config{})).ListFollowingMoments(ctx, momentsQuery{}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("unconfigured: %v", err)
	}

	for name, setup := range map[string]func(*pathsBackend){
		"query": func(b *pathsBackend) { b.queryErr[queryFollowingMoments] = errMomentFake },
		"scan": func(b *pathsBackend) {
			b.queryRows[queryFollowingMoments] = [][]any{
				{"m1", "u1", "n", "", "d", base, 0, 0, 0, "L", 35.0, 112.0, false, []string{}},
			}
			b.scanErr[queryFollowingMoments] = errMomentFake
		},
		"rows": func(b *pathsBackend) {
			b.queryRows[queryFollowingMoments] = [][]any{
				{"m1", "u1", "n", "", "d", base, 0, 0, 0, "L", 35.0, 112.0, false, []string{}},
			}
			b.rowsErr[queryFollowingMoments] = errMomentFake
		},
	} {
		b := newPathsBackend()
		setup(b)
		if _, err := newPathsRepo(b).ListFollowingMoments(ctx, momentsQuery{Limit: 5}); !errors.Is(err, errMomentFake) {
			t.Fatalf("%s: expected propagated error, got %v", name, err)
		}
	}

	// 空结果
	empty := newPathsBackend()
	page3, err := newPathsRepo(empty).ListFollowingMoments(ctx, momentsQuery{Limit: 5})
	if err != nil || len(page3.Items) != 0 || page3.NextCursor != nil {
		t.Fatalf("empty following page: %#v err=%v", page3, err)
	}
}

// TestPostgresRepositoryListNearbyMomentsRemainingPaths 覆盖附近流的排序次序、
// 游标定位、详情补全错误与候选查询错误。
func TestPostgresRepositoryListNearbyMomentsRemainingPaths(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	detail := func(momentID, userID string, imagesJSON, reactionsJSON []byte) []any {
		return []any{momentID, userID, userID + "_nick", "", "desc", 0, 0, 0, "南天门", []string{}, false, imagesJSON, reactionsJSON, false, false}
	}
	query := momentsQuery{
		ReferencePoint:    GeoPoint{Latitude: 35.0, Longitude: 112.0},
		HasReferencePoint: true,
		Radius:            5000,
		Limit:             2,
	}

	// 同距离 → 热度降序 → 时间降序 → ID 降序全序，以及游标翻页。
	dynamic := &pathsDynamicBackend{
		nearbyRows: [][]any{
			{"m_low", "u1", now.Add(-3 * time.Minute), 1, 35.0, 112.0, false},
			{"m_high", "u2", now.Add(-2 * time.Minute), 99, 35.0, 112.0, false},
			{"m_newer", "u3", now.Add(-time.Minute), 5, 35.0, 112.0, false},
		},
		detailRows: map[string][]any{
			"m_high":  detail("m_high", "u2", []byte("[]"), []byte("[]")),
			"m_low":   detail("m_low", "u1", []byte("[]"), []byte("[]")),
			"m_newer": detail("m_newer", "u3", []byte("[]"), []byte("[]")),
		},
	}
	repo := NewPostgresRepository(newMomentFactory(dynamic))

	page, err := repo.ListNearbyMoments(ctx, query)
	if err != nil {
		t.Fatalf("ListNearbyMoments: %v", err)
	}
	if len(page.Items) != 2 || page.Items[0].MomentID != "m_high" || page.Items[1].MomentID != "m_newer" {
		t.Fatalf("unexpected nearby order: %#v", page.Items)
	}
	if page.NextCursor == nil {
		t.Fatal("expected next cursor")
	}
	if page.Items[0].Location.Distance == nil || page.Items[0].Location.BearingDegrees == nil {
		t.Fatalf("relative location must be filled: %#v", page.Items[0].Location)
	}

	// 游标翻页：剩余候选从游标项之后开始。
	page2, err := repo.ListNearbyMoments(ctx, momentsQuery{
		ReferencePoint:    query.ReferencePoint,
		HasReferencePoint: true,
		Radius:            5000,
		Limit:             2,
		Cursor:            *page.NextCursor,
	})
	if err != nil {
		t.Fatalf("second nearby page: %v", err)
	}
	if len(page2.Items) != 1 || page2.Items[0].MomentID != "m_low" {
		t.Fatalf("unexpected second nearby page: %#v", page2.Items)
	}

	// 半径外候选被精筛丢弃 → 空页（非 nil items）。
	outDynamic := &pathsDynamicBackend{
		nearbyRows: [][]any{
			{"m_far", "u1", now, 1, 40.0, 112.0, false},
		},
	}
	pageOut, err := NewPostgresRepository(newMomentFactory(outDynamic)).ListNearbyMoments(ctx, query)
	if err != nil || len(pageOut.Items) != 0 {
		t.Fatalf("out-of-radius candidates: %#v err=%v", pageOut, err)
	}

	// 未配置
	if _, err := NewPostgresRepository(persistence.NewFactory(config.Config{})).ListNearbyMoments(ctx, momentsQuery{}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("unconfigured nearby: %v", err)
	}

	// 候选查询/扫描/迭代错误
	candidateErr := &pathsDynamicBackend{nearbyQueryErr: errMomentFake}
	if _, err := NewPostgresRepository(newMomentFactory(candidateErr)).ListNearbyMoments(ctx, query); !errors.Is(err, errMomentFake) {
		t.Fatalf("nearby query error: %v", err)
	}
	scanErr := &pathsDynamicBackend{nearbyRows: [][]any{{"m1", "u1", now, 1, 35.0, 112.0, false}}, nearbyScanErr: errMomentFake}
	if _, err := NewPostgresRepository(newMomentFactory(scanErr)).ListNearbyMoments(ctx, query); !errors.Is(err, errMomentFake) {
		t.Fatalf("nearby scan error: %v", err)
	}
	rowsErr := &pathsDynamicBackend{
		nearbyRows: [][]any{
			{"m1", "u1", now, 1, 35.0, 112.0, false},
		},
		nearbyRowsErr: errMomentFake,
	}
	if _, err := NewPostgresRepository(newMomentFactory(rowsErr)).ListNearbyMoments(ctx, query); !errors.Is(err, errMomentFake) {
		t.Fatalf("nearby rows error: %v", err)
	}

	// 详情查询错误
	detailsErr := &pathsDynamicBackend{
		nearbyRows:      [][]any{{"m_high", "u2", now, 1, 35.0, 112.0, false}},
		detailsQueryErr: errMomentFake,
	}
	if _, err := NewPostgresRepository(newMomentFactory(detailsErr)).ListNearbyMoments(ctx, momentsQuery{ReferencePoint: query.ReferencePoint, HasReferencePoint: true, Radius: 5000, Limit: 2}); !errors.Is(err, errMomentFake) {
		t.Fatalf("nearby details error: %v", err)
	}

	// 详情图片 JSON 损坏 → 错误传播
	badJSON := &pathsDynamicBackend{
		nearbyRows: [][]any{{"m_high", "u2", now, 1, 35.0, 112.0, false}},
		detailRows: map[string][]any{
			"m_high": detail("m_high", "u2", []byte("not-json"), []byte("[]")),
		},
	}
	if _, err := NewPostgresRepository(newMomentFactory(badJSON)).ListNearbyMoments(ctx, momentsQuery{ReferencePoint: query.ReferencePoint, HasReferencePoint: true, Radius: 5000, Limit: 2}); err == nil {
		t.Fatal("invalid images JSON must error")
	}
}

// TestCompareNearbyCandidateBranches 覆盖附近游标比较的全部分支。
func TestCompareNearbyCandidateBranches(t *testing.T) {
	base := time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC)
	candidate := nearbyCandidate{momentID: "m2", distance: 100, score: 5, createdAt: base}
	cursor := &itemCursor{ID: "m2", Distance: 100, Score: 5, CreatedAt: base.Format(time.RFC3339Nano)}

	if got := compareNearbyCandidate(candidate, cursor, base); got != 0 {
		t.Fatalf("equal tuple: %d", got)
	}
	cases := []struct {
		name string
		cand nearbyCandidate
		cur  *itemCursor
		at   time.Time
		want int
	}{
		{"closer distance", nearbyCandidate{distance: 50}, cursor, base, -1},
		{"farther distance", nearbyCandidate{distance: 150}, cursor, base, 1},
		{"higher score", nearbyCandidate{distance: 100, score: 9}, cursor, base, -1},
		{"lower score", nearbyCandidate{distance: 100, score: 1}, cursor, base, 1},
		{"newer time", nearbyCandidate{distance: 100, score: 5, createdAt: base.Add(time.Minute)}, cursor, base, -1},
		{"older time", nearbyCandidate{distance: 100, score: 5, createdAt: base.Add(-time.Minute)}, cursor, base, 1},
		{"larger id", nearbyCandidate{distance: 100, score: 5, createdAt: base, momentID: "m3"}, cursor, base, -1},
		{"smaller id", nearbyCandidate{distance: 100, score: 5, createdAt: base, momentID: "m1"}, cursor, base, 1},
	}
	for _, tc := range cases {
		if got := compareNearbyCandidate(tc.cand, tc.cur, tc.at); got != tc.want {
			t.Fatalf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}

// TestFindNearbyCandidateStartBranches 覆盖游标定位的边界情形。
func TestFindNearbyCandidateStartBranches(t *testing.T) {
	base := time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC)
	candidates := []nearbyCandidate{
		{momentID: "m1", distance: 10, score: 1, createdAt: base},
		{momentID: "m2", distance: 20, score: 2, createdAt: base},
		{momentID: "m3", distance: 30, score: 3, createdAt: base},
	}
	if got := findNearbyCandidateStart(candidates, ""); got != 0 {
		t.Fatalf("empty cursor: %d", got)
	}
	if got := findNearbyCandidateStart(candidates, "garbage"); got != 0 {
		t.Fatalf("garbage cursor: %d", got)
	}
	// 游标命中 m2 → 从下一项开始
	cursor := encodeFetchCursor(itemCursor{ID: "m2", Distance: 20, Score: 2, CreatedAt: base.Format(time.RFC3339Nano)})
	if got := findNearbyCandidateStart(candidates, cursor); got != 2 {
		t.Fatalf("hit cursor: %d", got)
	}
	// 游标位于所有候选之前（距离更小）→ 从 0 开始
	cursorBefore := encodeFetchCursor(itemCursor{ID: "mx", Distance: 5, CreatedAt: base.Format(time.RFC3339Nano)})
	if got := findNearbyCandidateStart(candidates, cursorBefore); got != 0 {
		t.Fatalf("cursor before list: %d", got)
	}
	// 游标位于所有候选之后 → 返回长度
	cursorAfter := encodeFetchCursor(itemCursor{ID: "mx", Distance: 99, CreatedAt: base.Format(time.RFC3339Nano)})
	if got := findNearbyCandidateStart(candidates, cursorAfter); got != len(candidates) {
		t.Fatalf("cursor after list: %d", got)
	}
}

// TestLoadNearbyMomentDetailsBranches 覆盖附近详情批量查询的短路与失败分支。
func TestLoadNearbyMomentDetailsBranches(t *testing.T) {
	ctx := context.Background()
	if details, err := loadNearbyMomentDetails(ctx, &pathsDynamicBackend{}, nil, "u1"); err != nil || len(details) != 0 {
		t.Fatalf("empty ids: %#v err=%v", details, err)
	}
	dynamic := &pathsDynamicBackend{queryErr: errMomentFake}
	if _, err := loadNearbyMomentDetails(ctx, dynamic, []string{"m1"}, "u1"); !errors.Is(err, errMomentFake) {
		t.Fatalf("query error: %v", err)
	}
	dynamic = &pathsDynamicBackend{detailRows: map[string][]any{"m1": {"m1", "u1", "n", "", "d", 0, 0, 0, "L", []string{}, false, []byte("bad"), []byte("[]"), false, false}}}
	if _, err := loadNearbyMomentDetails(ctx, dynamic, []string{"m1"}, "u1"); err == nil {
		t.Fatal("invalid images JSON must fail")
	}
	dynamic = &pathsDynamicBackend{detailRows: map[string][]any{"m1": {"m1", "u1", "n", "", "d", 0, 0, 0, "L", []string{}, false, []byte("[]"), []byte("bad"), false, false}}}
	if _, err := loadNearbyMomentDetails(ctx, dynamic, []string{"m1"}, "u1"); err == nil {
		t.Fatal("invalid reactions JSON must fail")
	}
	dynamic = &pathsDynamicBackend{detailRows: map[string][]any{"m1": {"m1", "u1", "n", "", "d", 0, 0, 0, "L", []string{}, false, []byte("[]"), []byte("[]"), true, true}}}
	details, err := loadNearbyMomentDetails(ctx, dynamic, []string{"m1"}, "u1")
	if err != nil {
		t.Fatalf("valid details: %v", err)
	}
	if details["m1"].authorRelation == nil || !details["m1"].authorRelation.IsFollowing || !details["m1"].authorRelation.IsFollowedBy {
		t.Fatalf("relation not parsed: %#v", details["m1"].authorRelation)
	}
	dynamic = &pathsDynamicBackend{detailRows: map[string][]any{"m1": {"m1", "u1", "n", "", "d", 0, 0, 0, "L", []string{}, false, []byte("[]"), []byte("[]"), false, false}}, detailsRowsErr: errMomentFake}
	if _, err := loadNearbyMomentDetails(ctx, dynamic, []string{"m1"}, "u1"); !errors.Is(err, errMomentFake) {
		t.Fatalf("detail rows error: %v", err)
	}
}
