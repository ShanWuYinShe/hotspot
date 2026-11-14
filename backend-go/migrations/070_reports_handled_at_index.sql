-- reports 保留期清理（DeleteHandledBefore）按 handled_at 过滤已处理终态
--（status IN ('resolved','dismissed')），但 006 建表时只有 creator/status/
-- target_type 三个索引：EXPLAIN 显示该删除条件只能 Bitmap+Filter 扫描，保留期
-- 清理每批都要扫过大量 pending 行。补 handled_at 部分索引（pending 行
-- handled_at IS NULL、不进索引），让清理按索引顺序取最早的一批、单批只锁
-- 少量行；同时让「已处理举报」的时间维度查询可走索引。
CREATE INDEX IF NOT EXISTS idx_reports_handled_at
    ON reports (handled_at) WHERE handled_at IS NOT NULL;
