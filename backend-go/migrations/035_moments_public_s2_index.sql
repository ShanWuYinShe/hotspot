-- 为隐私级别为 public 的瞬间补充部分索引。
-- 地理 feed（feed / map / share / search / moments）高频使用
-- `WHERE m.privacy_level = 'public' AND s2_cell_id BETWEEN ...` 的查询路径，
-- 窄索引比既有的 idx_moments_non_private_s2_cell_id 更利于热路径扫描。
CREATE INDEX IF NOT EXISTS idx_moments_public_s2_cell_id ON moments(s2_cell_id) WHERE privacy_level = 'public';
