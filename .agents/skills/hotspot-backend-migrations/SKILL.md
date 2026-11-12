---
name: hotspot-backend-migrations
description: hotspot 后端数据库迁移纪律：文件命名、版本表 checksum/原子执行、schema manifest 校验与重建、破坏性 DDL 声明、data_migrations 旗标、禁止 CREATE INDEX CONCURRENTLY 及原因。触发：新增或修改 backend-go/migrations 下的 SQL；重建 schema manifest；从空库初始化；改表结构或索引；回填数据；排查启动或迁移校验失败。
---

# hotspot-backend-migrations

## 运行模型

- `migrations/NNN_描述.sql`，字典序执行，序号唯一（历史 009 重号在 `scripts/check-migrations.sh` 白名单放行，新增不得重复）。
- 应用启动时 `platform/migrate.EnsureSchema` 执行（与 `go run ./cmd/migrate` 同一实现）：
  1. 确保/升级版本表 `schema_migrations(name PK, checksum, execution_ms, applied_at)`；
  2. 读取已应用集合，为旧版 runner 写入的 checksum 空行补写当前文件哈希（一次性过渡，记 WARN）；
  3. 校验「已应用文件名都存在且当前内容哈希 == 记录」（不符即 fail-fast，启动失败）；
  4. 逐个执行 pending：**同一事务内**先执行整文件 SQL、再插入版本记录、提交（失败整体回滚，不会留下「已执行未登记」或半应用状态）；
  5. 用 `migrations/schema_manifest.json` 校验期望表/列/索引是否存在（缺失致命、库内多余对象仅告警）。
- **已应用迁移文件禁止修改/删除**：checksum 会阻止演进；要改 schema 一律新增迁移。
- **不做基线、不做存量库接管**：库内已有业务表但版本表为空时启动直接失败；必须从空库按文件顺序执行全量迁移。禁止恢复「只登记不执行」的基线（历史事故：线上索引被基线吞掉）。

## 常用命令

```bash
go run ./cmd/migrate                      # 应用 pending（幂等）
go run ./cmd/migrate -verify-only         # 只读校验：checksum / pending / schema manifest
go run ./cmd/migrate -write-manifest=migrations/schema_manifest.json      # 在全新库重建 manifest
```

## schema manifest

- `migrations/schema_manifest.json` 是「全部迁移执行后应存在的表/列/索引名」快照（排除
  `schema_migrations`/`data_migrations` 簿记表）。
- **新增或修改迁移后必须重建 manifest**：在全新库执行 `-write-manifest`，与迁移文件一起提交。
  CI 的 `backend-integration` job（本地可用 `verify-backend-ci.sh --with-db` 复现）会用临时库重放全部迁移并 diff manifest，忘记重建即失败。
- 校验方向只检查「期望对象缺失」，允许运维手工加索引/临时表。

## 破坏性 DDL 与幂等

- 本项目已不考虑「旧二进制 + 新 schema」兼容；`DROP/RENAME/TYPE 变更` 允许，但迁移文件顶部必须写
  `-- allow-destructive: <理由>`，`scripts/check-migrations.sh` 强制该声明。
- 仍建议幂等写法（`IF NOT EXISTS` / `DO $$` 自检），用于手工重跑与本地反复初始化；
  无法按格式判幂等的一次性数据迁移用 `data_migrations` 旗标表包裹（参照 053）。
- 无 down 迁移：回滚以数据库备份/快照为准；破坏性迁移一旦发行，旧镜像不能直接跑在新 schema 上。

## 禁止 CREATE INDEX CONCURRENTLY

迁移整文件在**显式事务**内执行（原子性要求），CONCURRENTLY 在事务内报 25001 导致启动失败。
普通 `CREATE INDEX` 的 SHARE 锁会短暂阻塞读写，是当前已知代价；失败会随事务整体回滚，不留 INVALID 索引。
如将来需要在线建索引，需在 runner 增加「单语句文件 + 非事务执行」的显式模式（尚未支持）。
