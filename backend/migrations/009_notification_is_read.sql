-- 通知已读判断函数
-- 用于 notification API 的 list / unread_count 等
CREATE OR REPLACE FUNCTION notification_is_read(
    p_user_id BIGINT,
    p_notification_id UUID,
    p_notification_type VARCHAR,
    p_created_at TIMESTAMPTZ
) RETURNS BOOLEAN AS $$
BEGIN
    -- 1. 逐条已读：在 notification_read_items 中
    IF EXISTS (
        SELECT 1 FROM notification_read_items
        WHERE user_id = p_user_id AND notification_id = p_notification_id
    ) THEN
        RETURN TRUE;
    END IF;

    -- 2. 全局水位线：created_at 在 last_read_at 之前
    IF EXISTS (
        SELECT 1 FROM notification_read_state
        WHERE user_id = p_user_id AND last_read_at >= p_created_at
    ) THEN
        RETURN TRUE;
    END IF;

    RETURN FALSE;
END;
$$ LANGUAGE plpgsql STABLE;
