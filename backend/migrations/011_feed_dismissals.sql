-- Feed dismissals tracking
CREATE TYPE dismiss_target_type AS ENUM ('moment', 'user', 'group');

CREATE TABLE feed_dismissals (
    dismiss_id      BIGSERIAL PRIMARY KEY,
    user_id         BIGINT      NOT NULL,
    target_type     dismiss_target_type NOT NULL,
    target_id       TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, target_type, target_id)
);

CREATE INDEX idx_feed_dismissals_user ON feed_dismissals(user_id);
