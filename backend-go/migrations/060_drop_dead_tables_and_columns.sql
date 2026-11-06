-- allow-destructive: 删除 0 引用的历史死表与死列（持久化审计 P1-2）
--
--  - moment_likes / comment_likes：reactions（迁移 031）已完全取代其语义，
--    Go 代码 0 读取；保留只会让用户/瞬间删除的 CASCADE 白扫行、并误导维护者。
--  - feed_dismissals：spec 预留、从未实现（0 引用、dev 库 0 行）。
--  - comments.likes_count：写路径恒写 0、契约 CommentItem 不含该字段，仅 repo
--    内部扫描；评论点赞的真实来源是 reactions(target_type='comment')。
DROP TABLE IF EXISTS moment_likes;
DROP TABLE IF EXISTS comment_likes;
DROP TABLE IF EXISTS feed_dismissals;
ALTER TABLE comments DROP COLUMN IF EXISTS likes_count;
