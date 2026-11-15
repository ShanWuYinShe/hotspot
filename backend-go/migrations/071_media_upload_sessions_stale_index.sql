-- 媒体保留期 GC（media/retention.go 每小时一轮的 ListStaleUploads）按
-- "status <> 'completed' AND created_at < cutoff" 取超期未完成会话，并按
-- created_at 升序 LIMIT 批量；015 的 (user_id, created_at DESC) 与 045 的
-- file_id 部分索引都没有 created_at 维度，此前每轮都退化为对只增不减的
-- media_upload_sessions 全表 Seq Scan + top-N Sort（审计 C-P2-1：30 万行约
-- 50ms / 5000 buffers，行数随历史上传次数线性增长，GC 每轮却只取 500 行）。
-- 部分索引谓词与查询谓词完全一致（completed 行是永久业务数据、不参与 GC，
-- 不进索引），同时满足 created_at 范围过滤与 ORDER BY 排序。
CREATE INDEX IF NOT EXISTS idx_media_upload_sessions_stale
    ON media_upload_sessions (created_at) WHERE status <> 'completed';
