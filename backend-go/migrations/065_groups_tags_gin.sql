-- 065_groups_tags_gin.sql
-- 群标签过滤（groups/search 的 tag）从「= ANY(g.tags)」改为数组包含「g.tags @> ARRAY[...]」，
-- 配合 GIN(tags) 让选择性标签不再全表扫描。
--
-- 实测（100k 群、选择性标签命中 99 行）：= ANY + 无索引 13.4ms → @> + GIN 0.28ms。
-- 注意：= ANY 无法走 GIN，必须同时改查询写法（repo_postgres_read.go）。
-- 迁移 runner 在事务内执行，不能用 CREATE INDEX CONCURRENTLY（见迁移纪律）。
CREATE INDEX IF NOT EXISTS idx_groups_tags_gin ON groups USING gin (tags);
