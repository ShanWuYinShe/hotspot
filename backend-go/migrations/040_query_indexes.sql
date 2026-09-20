-- 查询路径索引补充（架构审查批次 C）：
-- 1) comments(parent_id)：ListReplies/ListTopRepliesByParentIDs 按父评论过滤，
--    此前只有 (moment_id, created_at) 索引，评论量大后回复列表走 seq scan。
-- 2) group_messages(sender_user_id)：按发送者查消息（历史/审计路径）。
-- 3) moments description ILIKE 搜索：pg_trgm GIN 索引，关键词搜索免全表扫。
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_comments_parent_id ON comments(parent_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_group_messages_sender_user_id ON group_messages(sender_user_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moments_description_trgm ON moments USING GIN (description gin_trgm_ops);
