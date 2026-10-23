package testdb

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/migrate"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/migrations"
)

var (
	poolMu   sync.Mutex
	testPool *pgxpool.Pool
	testDSN  string
)

func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
}

func DSN() string {
	return os.Getenv("POSTGRES_DSN")
}

func Open(t *testing.T) *persistence.Factory {
	t.Helper()
	dsn := DSN()
	if dsn == "" {
		t.Skip("POSTGRES_DSN not set, skipping integration test")
	}

	poolMu.Lock()
	defer poolMu.Unlock()

	if testPool == nil || testDSN != dsn {
		if testPool != nil {
			testPool.Close()
		}
		testDSN = dsn
		testPool = initPool(t, dsn)
	}

	db := persistence.NewPostgresDB(testDSN)
	db.Pool = testPool
	return &persistence.Factory{
		Config: persistence.Config{PostgresDSN: testDSN},
		DB:     db,
	}
}

func initPool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse postgres dsn: %v", err)
	}
	// 与生产 persistence.NewFactory 保持同一 QueryExecMode（simple protocol）：
	// IT 若走 pgx 默认二进制协议，timestamptz 等 OID 无法 Scan 进 string，
	// 会误报生产不存在的错误（反之亦然），测试结果失去代表性。
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping postgres: %v", err)
	}
	db := persistence.NewPostgresDB(dsn)
	db.Pool = pool
	if err := migrate.EnsureSchema(ctx, db, migrations.Files); err != nil {
		pool.Close()
		t.Fatalf("ensure schema: %v", err)
	}
	return pool
}

// BeginScope 打开一个测试事务并返回 txFactory 与回滚清理函数。与 TxScope 的
// 区别：调用方自己控制事务生命周期，适合「先建实现、再跑多个用例、最后统一
// 回滚」的契约测试（见 ContractImpl）。
func BeginScope(t *testing.T) (*persistence.Factory, func()) {
	t.Helper()
	pool := testPool
	if pool == nil {
		t.Fatal("testdb: pool not initialized, call Open first")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	tx, err := pool.Begin(ctx)
	if err != nil {
		cancel()
		t.Fatalf("begin tx: %v", err)
	}
	factory := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: testDSN},
		DB:     &txWrapper{tx: tx},
	}
	cleanup := func() {
		_ = tx.Rollback(context.Background())
		cancel()
	}
	return factory, cleanup
}

func TxScope(t *testing.T, factory *persistence.Factory, fn func(ctx context.Context, txFactory *persistence.Factory)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool := testPool

	if pool == nil {
		t.Fatal("testdb: pool not initialized, call Open first")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	txFactory := &persistence.Factory{
		Config: factory.Config,
		DB:     &txWrapper{tx: tx},
	}
	fn(ctx, txFactory)
}

const queryInsertTestUser = `INSERT INTO users (user_id, nickname, avatar_url, user_type, bio, gender, email, phone, location_name, latitude, longitude, s2_cell_id, presence_status, last_online, created_at, register_ip, register_location, last_login_ip, last_login_location)
VALUES ($1, $2, '', 'normal', '', '', '', '', '', 0, 0, $3, 'online', $4, $5, '', '', '', '')`

func InsertUser(t *testing.T, factory *persistence.Factory, userID, nickname string) string {
	t.Helper()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	now := time.Now().UTC()
	s2CellID := geo.LeafCellID(geo.Point{Latitude: 35.0, Longitude: 112.0})
	if err := qdb.Exec(context.Background(), queryInsertTestUser, userID, nickname, s2CellID, now, now); err != nil {
		t.Fatalf("insert test user %s: %v", userID, err)
	}
	return userID
}
