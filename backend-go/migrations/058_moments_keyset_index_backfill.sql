-- 058_moments_keyset_index_backfill.sql
-- 补建存量库「基线漏执行」的 idx_moments_keyset 索引（2026-11-01 线上排查）。
--
-- 背景：056（2026-10-07）声明了 moments 时间线 keyset 翻页索引，但线上库在
-- 2026-10-08 做过「存量库基线」——internal/platform/migrate/embedded.go 对
-- 「已有全部核心表、schema_migrations 为空」的库只登记文件、不执行 SQL。线上
-- 数据快照早于 056，基线把 056 登记为已应用后，其中的 CREATE INDEX 永远不会
-- 再执行：线上 82 个声明索引里恰好缺这一个（其余齐全、无 INVALID）。
-- 影响：moments/fetch、moments/list 等 WHERE is_deleted = FALSE
-- ORDER BY created_at DESC, moment_id DESC 的 keyset 查询没有匹配索引，数据量
-- 上来后从 Index Only Scan 退回 Seq Scan + top-N Sort（056 自带 5 万行压测：
-- 7.96ms → 0.037ms）。
--
-- 修法：新增迁移重放 056 的建索引语句（IF NOT EXISTS 幂等）。索引定义必须与
-- 056 逐字一致（表/列/排序方向/部分索引谓词），否则「新库顺序执行（056 建）」
-- 与「存量库增量执行（本文件建）」会得到语义不同的索引；embed_test.go 的
-- TestDuplicateIndexDeclarationsShareDefinition 锁定了这条一致性。
--
-- 幂等：索引已存在时为 no-op，重复执行无副作用（data_migrations 旗标无需使用）。
-- 兼容：纯新增索引，旧版二进制查询无需感知；普通 CREATE INDEX（迁移整文件
-- 单消息执行会落隐式事务，禁止 CONCURRENTLY，见 skill hotspot-backend-migrations）。
-- 防复发：scripts/check-migration-indexes.sh 按迁移声明比对库中实际索引，
-- 可检出「基线登记但未执行」造成的缺失（用法见脚本头）。
CREATE INDEX IF NOT EXISTS idx_moments_keyset
  ON moments (created_at DESC, moment_id DESC)
  WHERE is_deleted = FALSE;
