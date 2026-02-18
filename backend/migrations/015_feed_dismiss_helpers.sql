-- Helper functions to avoid repeating feed dismissal subqueries
CREATE OR REPLACE FUNCTION feed_moment_not_dismissed(
    p_user_id bigint,
    p_moment_id uuid,
    p_author_id bigint
) RETURNS boolean AS $$
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
    p_user_id bigint,
    p_target_type dismiss_target_type,
    p_target_id text
) RETURNS boolean AS $$
    SELECT NOT EXISTS (
        SELECT 1
        FROM feed_dismissals fd
        WHERE fd.user_id = p_user_id
          AND fd.target_type = p_target_type
          AND fd.target_id = p_target_id
    );
$$ LANGUAGE SQL STABLE;
