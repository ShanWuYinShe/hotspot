-- 动态时间线 keyset 翻页索引（2026-10-07 性能基线批次，5 万行压测数据实测）：
-- moments/fetch 与 moments/list 系列查询均为
--   WHERE is_deleted = false ORDER BY created_at DESC, moment_id DESC LIMIT n
--   + keyset 谓词 (created_at, moment_id) < (cursor_ts, cursor_id)。
-- 表上唯一的排序可用索引是 idx_moments_user_id_created_at（user_id 前导，
-- 全站时间线用不上）；实测 5 万行下第一页与深翻页均为 Seq Scan + 全表
-- top-N Sort（7.96ms / 5.32ms），建本索引后走 Index Only Scan（0.037ms /
-- 0.040ms，深翻页不再随时长线性退化）。
-- 部分索引谓词与查询过滤条件精确匹配（is_deleted = false）；已删除动态
-- 走回收路径不在此列。
CREATE INDEX IF NOT EXISTS idx_moments_keyset
  ON moments (created_at DESC, moment_id DESC)
  WHERE is_deleted = FALSE;
