package migrate

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

// execMigrationSQL 以简单协议执行迁移文件：运行期连接池默认使用预编译语句
// （extended protocol 一条消息只允许单语句），而迁移文件普遍包含多条 DDL，
// 必须以简单协议整文件发送。「整文件单消息执行」语义保持不变（这也是禁止
// CREATE INDEX CONCURRENTLY 的原因——会落隐式事务）。非 Postgres 实现
// （stub/内存）没有 ExecSimple，回退普通 Exec。
func execMigrationSQL(ctx context.Context, db persistence.QueryableDB, sql string) error {
	if simple, ok := db.(interface {
		ExecSimple(context.Context, string) error
	}); ok {
		return simple.ExecSimple(ctx, sql)
	}
	return db.Exec(ctx, sql)
}
