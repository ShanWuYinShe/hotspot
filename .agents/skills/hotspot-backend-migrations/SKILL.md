---
name: hotspot-backend-migrations
description: hotspot 后端数据库迁移纪律：文件命名与幂等要求、向后兼容（只加不删 / expand-contract）、data_migrations 旗标、禁止 CREATE INDEX CONCURRENTLY 及原因、INVALID 索引处理。触发：新增或修改 backend-go/migrations 下的 SQL；改表结构或索引；回填数据；排查启动时迁移或建索引失败。
---

# hotspot-backend-migrations

## 数据库迁移

`migrations/` 下 `NNN_描述.sql` 格式，嵌入二进制由启动时自动执行（字典序全量重跑，幂等靠各自 `IF NOT EXISTS`）。新增 migration 使用下一个可用序号（CI 校验序号唯一且无重复——历史 009 重号在脚本白名单放行，见仓库根 `scripts/check-migrations.sh`）。

**向后兼容纪律（硬性）**：迁移必须兼容「上一个已发布版本」的二进制——发布是无停机产物替换，旧版后端会短暂与新 schema 共存：

- 只加不删：新迁移内不 DROP/RENAME 仍在被上一版本读写的列与表；废弃列先停止写入，跨一个版本再清理（expand-contract 两阶段）。
- 改列语义/回填数据用 `DO $$` 自检式幂等写法（参照 `050`），不依赖「只跑一次」。
- 数据性一次性迁移（无法按格式判幂等的）必须用 `data_migrations` 旗标表包裹（参照 `053`）。
- 无 down 迁移：回滚版本 = 旧镜像 + 保留新 schema，因此 schema 变更必须对旧代码向后兼容，这是唯一防线。
- **迁移内禁止 `CREATE INDEX CONCURRENTLY`**：迁移经 pgx 简单协议（`QueryExecModeSimpleProtocol`）把整个文件作为一条消息执行，多语句文件落入 PostgreSQL 隐式事务块，CONCURRENTLY 以 25001 拒绝（2026-09-20 实测，`cmd/migrate` 与服务端 `ensureSchema` 两条路径均如此，会导致启动失败）。普通 CREATE INDEX 的 SHARE 锁会短暂阻塞读写是已知代价；要消除它需先建 `schema_migrations` 版本表并把建索引拆成单语句迁移文件（遗留项）。并发构建失败会留下 INVALID 索引且 `IF NOT EXISTS` 不会重建——启动日志见索引报错时先 DROP INVALID 索引再重启。
