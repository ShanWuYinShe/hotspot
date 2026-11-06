-- moments/groups 搜索短关键词（1-2 字符 CJK）同 users/search 问题：
-- pg_trgm 需 >=3 字符模式，短关键词退化为全表 seq scan。迁移 063 已启用
-- pgroonga 扩展；此处为 description/name 建 PGroonga GIN 索引，SQL 侧用
-- &@ 子串预过滤 + 原 ILIKE 精确复核（语义不变）。
CREATE INDEX IF NOT EXISTS idx_moments_description_pgroonga
    ON moments USING pgroonga (description pgroonga_text_full_text_search_ops_v2);

CREATE INDEX IF NOT EXISTS idx_groups_name_pgroonga
    ON groups USING pgroonga (name pgroonga_text_full_text_search_ops_v2);

CREATE INDEX IF NOT EXISTS idx_groups_description_pgroonga
    ON groups USING pgroonga (description pgroonga_text_full_text_search_ops_v2);
