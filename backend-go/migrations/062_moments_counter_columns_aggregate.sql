-- allow-destructive: moments 三个计数冗余列改为读时聚合 + 内部互动热度列
--
-- 旧口径：moments.likes_count 由 reactions 写路径维护、comments_count 由评论写
-- 路径维护、shares_count 由分享创建 +1（撤销不回减），三者失败仅 Warn 且除
-- likes 外无自愈 → 同一对象跨端点数值不一致、热门瞬间写热点。
--
-- 新口径：
--   - comments_count / shares_count（契约字段）改为读时聚合（评论/未撤销分享）；
--   - 删除三列；新增 interaction_count = reactions + 未删评论 + 未撤销分享，
--     仅用于 nearby/popular 排序（不对外），由 reactions/comments/share 写路径
--     增量维护 + 失败容忍（排序启发式允许弱一致，不再承载契约值）。
ALTER TABLE moments ADD COLUMN IF NOT EXISTS interaction_count integer NOT NULL DEFAULT 0;

UPDATE moments m
SET interaction_count =
      (SELECT COUNT(*) FROM reactions r
        WHERE r.target_type = 'moment' AND r.target_id = m.moment_id)
    + (SELECT COUNT(*) FROM comments c
        WHERE c.moment_id = m.moment_id AND c.is_deleted = FALSE)
    + (SELECT COUNT(*) FROM shares s
        WHERE s.share_type = 'moment' AND s.is_revoked = FALSE
          AND s.payload->>'shared_moment_id' = m.moment_id);

-- shares_count 聚合的支撑表达式索引（分享表按 payload 关联瞬间，历史无索引）。
CREATE INDEX IF NOT EXISTS idx_shares_live_moment
  ON shares ((payload->>'shared_moment_id'))
  WHERE share_type = 'moment' AND is_revoked = FALSE;

ALTER TABLE moments DROP COLUMN IF EXISTS likes_count;
ALTER TABLE moments DROP COLUMN IF EXISTS comments_count;
ALTER TABLE moments DROP COLUMN IF EXISTS shares_count;
