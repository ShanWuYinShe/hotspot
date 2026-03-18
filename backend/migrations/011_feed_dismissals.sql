-- 011_feed_dismissals.sql
-- 信息流屏蔽记录：表结构 + 索引 + 帮助函数

CREATE TYPE dismiss_target_type AS ENUM ('moment', 'user', 'group');

CREATE TABLE IF NOT EXISTS feed_dismissals (
    dismiss_id      BIGSERIAL PRIMARY KEY,
    user_id         BIGINT              NOT NULL,
    target_type     dismiss_target_type NOT NULL,
    target_id       TEXT                NOT NULL,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, target_type, target_id)
);

-- 按用户维度查询屏蔽记录（基础索引）
CREATE INDEX IF NOT EXISTS idx_feed_dismissals_user
    ON feed_dismissals(user_id);

-- 按目标统计/查询屏蔽记录
CREATE INDEX IF NOT EXISTS idx_feed_dismissals_target
    ON feed_dismissals (target_type, target_id, user_id);

-- 针对 feed 过滤的典型查询场景的部分索引
CREATE INDEX IF NOT EXISTS idx_feed_dismissals_user_moment
    ON feed_dismissals (user_id, target_id)
    WHERE target_type = 'moment';

CREATE INDEX IF NOT EXISTS idx_feed_dismissals_user_user
    ON feed_dismissals (user_id, target_id)
    WHERE target_type = 'user';

CREATE INDEX IF NOT EXISTS idx_feed_dismissals_user_group
    ON feed_dismissals (user_id, target_id)
    WHERE target_type = 'group';

-- Helper functions to avoid repeating feed dismissal subqueries
CREATE OR REPLACE FUNCTION feed_moment_not_dismissed(
    p_user_id   BIGINT,
    p_moment_id UUID,
    p_author_id BIGINT
) RETURNS BOOLEAN AS $$
    SELECT NOT EXISTS (
        SELECT 1
        FROM feed_dismissals fd
        WHERE fd.user_id = p_user_id
          AND fd.target_type = 'moment'
          AND fd.target_id = p_moment_id::text
    ) AND NOT EXISTS (
        SELECT 1
        FROM feed_dismissals fd
        WHERE fd.user_id = p_user_id
          AND fd.target_type = 'user'
          AND fd.target_id = p_author_id::text
    );
$$ LANGUAGE SQL STABLE;

CREATE OR REPLACE FUNCTION feed_entity_not_dismissed(
    p_user_id     BIGINT,
    p_target_type dismiss_target_type,
    p_target_id   TEXT
) RETURNS BOOLEAN AS $$
    SELECT NOT EXISTS (
        SELECT 1
        FROM feed_dismissals fd
        WHERE fd.user_id = p_user_id
          AND fd.target_type = p_target_type
          AND fd.target_id = p_target_id
    );
$$ LANGUAGE SQL STABLE;
