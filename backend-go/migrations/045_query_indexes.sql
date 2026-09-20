-- 高频查询补索引：
-- 1) media_upload_sessions.file_id：moments/create、moments/update 的图片归属
--    校验（file_id = ANY(...)）与 moments/feed 带图读取的 JOIN 都按 file_id
--    命中，此前全表扫描；
-- 2) users.nickname / users.bio / groups.name 的 pg_trgm GIN：search/suggest 是
--    高频联想接口（挂 /public），unanchored ILIKE 此前无索引可用；
-- 3) moment_favorites(moment_id)：moments 删除广播路径按 moment_id 反查
--    收藏者列表。
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_upload_sessions_file_id
  ON media_upload_sessions(file_id) WHERE status = 'completed';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_nickname_trgm ON users USING GIN (nickname gin_trgm_ops);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_bio_trgm ON users USING GIN (bio gin_trgm_ops);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_groups_name_trgm ON groups USING GIN (name gin_trgm_ops);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moment_favorites_moment_id ON moment_favorites(moment_id);
