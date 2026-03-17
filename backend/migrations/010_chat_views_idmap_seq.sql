-- ============================================================
-- 010_chat_views_idmap_seq.sql
-- 为聊天消息提供可查询视图，并为 ID Map 添加顺序游标列
-- ============================================================

-- ─── 私聊/群聊消息视图（面向查询层，封装分区表）───────────
CREATE OR REPLACE VIEW private_messages AS
SELECT
    message_id,
    chat_id,
    sender_id,
    receiver_id,
    message_type,
    content,
    metadata,
    share_code,
    reply_to_id,
    is_recalled,
    recalled_at,
    edited_at,
    reaction_counts,
    sent_at
FROM private_messages_partitioned;

CREATE OR REPLACE VIEW group_messages AS
SELECT
    message_id,
    group_id,
    sender_id,
    message_type,
    content,
    metadata,
    share_code,
    reply_to_id,
    at_users AS mentions,
    is_recalled,
    recalled_at,
    recalled_by,
    edited_at,
    reaction_counts,
    read_count,
    sent_at
FROM group_messages_partitioned;

-- ─── ID Map 顺序游标列（供 BloomFilter 暖机高效扫描）────────
-- row_id 列已在 008 中的建表语句中定义，此处只创建索引

CREATE UNIQUE INDEX IF NOT EXISTS idx_private_message_id_map_row_id
	    ON private_message_id_map(row_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_group_message_id_map_row_id
	    ON group_message_id_map(row_id);
