-- 高频路径补索引（2026-09 审计性能批次）：
-- 1) private_chats(user_a_id) / (user_b_id)：会话列表
--    WHERE user_a_id = $1 OR user_b_id = $1 此前零二级索引，每次打开会话列表
--    全表扫描（BitmapOr 合并两个单列索引）；
-- 2) notifications(user_id, is_read)：未读计数 / 全部已读 / 列表 is_read 过滤
--    此前全表扫描；
-- 3) notifications(created_at) WHERE is_read = TRUE：prune sweeper 每日清理
--    「已读且超期」通知的删除范围选择，部分索引与谓词精确匹配；
-- 4) moment_tags(tag_name) gin_trgm_ops：search/suggest tags 的 unanchored
--    ILIKE（045 已覆盖 users/groups，动态标签漏配，表随动态线性增长）。
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_private_chats_user_a ON private_chats(user_a_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_private_chats_user_b ON private_chats(user_b_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_user_read ON notifications(user_id, is_read);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_read_created_at ON notifications(created_at) WHERE is_read = TRUE;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moment_tags_name_trgm ON moment_tags USING GIN (tag_name gin_trgm_ops);
