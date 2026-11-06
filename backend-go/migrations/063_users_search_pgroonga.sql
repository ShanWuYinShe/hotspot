-- 用户搜索短关键词（1-2 字符，尤其中文姓名）原先无可用索引：pg_trgm 需要
-- >=3 字符模式，短关键词退化为 users 全表 seq scan（100k 用户实测 85ms）。
-- 引入 PGroonga 扩展（数据库镜像内置）：&@ 子串操作符可走该 GIN 索引，
-- 1 字符起有效；SQL 仍保留 ILIKE 精确复核，语义不变。
CREATE EXTENSION IF NOT EXISTS pgroonga;

CREATE INDEX IF NOT EXISTS idx_users_nickname_pgroonga
    ON users USING pgroonga (nickname pgroonga_text_full_text_search_ops_v2);

CREATE INDEX IF NOT EXISTS idx_users_bio_pgroonga
    ON users USING pgroonga (bio pgroonga_text_full_text_search_ops_v2);
