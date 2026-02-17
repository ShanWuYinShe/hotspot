-- Additional indexes for feed & share performance

-- 优化分享列表（按分享者/类型/时间排序）
CREATE INDEX IF NOT EXISTS idx_shares_creator_type_created
    ON shares (sharer_id, share_type, created_at DESC, share_id DESC);

-- 优化按目标统计/查询 feed dismiss 记录
CREATE INDEX IF NOT EXISTS idx_feed_dismissals_target
    ON feed_dismissals (target_type, target_id, user_id);
