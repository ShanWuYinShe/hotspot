-- allow-destructive: comments.replies_count 改为读时聚合（删除冗余列）
--
-- 旧口径：回复创建 +1、软删 -1 维护 comments.replies_count；跨模块/异常路径
-- （事务外失败、历史数据）会永久漂移且无自愈，同时让「同一字段」存在列值与
-- 实际回复数两个真相。契约含义是「未删除的直接回复数」，可直接由
-- comments.parent_id 聚合得出，故删除列、改为读时标量子查询。
ALTER TABLE comments DROP COLUMN IF EXISTS replies_count;
-- 回复计数子查询的支撑索引：按父评论过滤且只统计未删除行。
CREATE INDEX IF NOT EXISTS idx_comments_parent_id_live
  ON comments(parent_id) WHERE is_deleted = FALSE;
