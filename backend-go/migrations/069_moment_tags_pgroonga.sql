-- moment_tags.tag_name 此前只有 pg_trgm GIN 索引：pg_trgm 需要 >=3 字符的
-- 模式，1-2 字（尤其中文标签）的标签搜索/联想只能全表顺序扫描。PGroonga
-- 1 字符起即可走 GIN（与 063/064 的 users/moments/groups 搜索索引同一方案），
-- 本迁移补齐 tags 的短关键词索引；SQL 使用 tag_name &@ $keyword。
CREATE INDEX IF NOT EXISTS idx_moment_tags_name_pgroonga
    ON moment_tags USING pgroonga (tag_name pgroonga_text_full_text_search_ops_v2);
