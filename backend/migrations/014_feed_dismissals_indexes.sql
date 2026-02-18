-- Improve feed_dismissals lookup performance for feed filters
CREATE INDEX IF NOT EXISTS idx_feed_dismissals_user_moment
    ON feed_dismissals (user_id, target_id)
    WHERE target_type = 'moment';

CREATE INDEX IF NOT EXISTS idx_feed_dismissals_user_user
    ON feed_dismissals (user_id, target_id)
    WHERE target_type = 'user';

CREATE INDEX IF NOT EXISTS idx_feed_dismissals_user_group
    ON feed_dismissals (user_id, target_id)
    WHERE target_type = 'group';
