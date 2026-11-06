-- 066_comments_mentions_column.sql
-- CommentItem.mentions 契约字段此前只参与通知、从未持久化：读接口恒返回空数组。
-- 直接在 comments 行内存 text[]（评论量远小于消息，读路径无需 join）：
-- 创建时随评论同事务写入原始 mention 用户 ID 列表，读接口原样返回。
ALTER TABLE comments ADD COLUMN IF NOT EXISTS mentions TEXT[] NOT NULL DEFAULT '{}';
