package feed

const (
	queryFeedUsers = `
SELECT u.user_id,
       u.nickname,
       COALESCE(u.avatar_url, ''),
       COALESCE(u.bio, ''),
       u.location_name,
       u.latitude,
       u.longitude,
       EXISTS(SELECT 1 FROM user_relations mine WHERE mine.user_id = $3 AND mine.target_user_id = u.user_id AND mine.is_following = TRUE),
       EXISTS(SELECT 1 FROM user_relations rev WHERE rev.user_id = u.user_id AND rev.target_user_id = $3 AND rev.is_following = TRUE),
       COALESCE((
         SELECT COUNT(*)
         FROM user_relations mine
         JOIN user_relations theirs ON theirs.user_id = u.user_id AND theirs.target_user_id = mine.target_user_id AND theirs.is_following = TRUE
         WHERE mine.user_id = $3 AND mine.is_following = TRUE
       ), 0),
       COALESCE((
         SELECT COUNT(*)
         FROM group_members gm1
         JOIN group_members gm2 ON gm2.group_id = gm1.group_id AND gm2.user_id = u.user_id
         JOIN groups g ON g.group_id = gm1.group_id
         WHERE gm1.user_id = $3 AND g.deleted_at IS NULL
       ), 0),
       u.presence_status,
       u.last_online,
       COALESCE((
         SELECT ARRAY_AGG(tag_name ORDER BY tag_name)
         FROM user_profile_tags t WHERE t.user_id = u.user_id
       ), '{}'),
       COALESCE((SELECT COUNT(*) FROM moments m WHERE m.user_id = u.user_id AND m.is_deleted = FALSE), 0),
       COALESCE((
         SELECT COUNT(*) FROM user_relations ur
         JOIN user_relations rev ON rev.user_id = ur.target_user_id AND rev.target_user_id = ur.user_id AND rev.is_following = TRUE
         WHERE ur.user_id = u.user_id AND ur.is_following = TRUE
       ), 0),
       COALESCE((SELECT COUNT(*) FROM user_relations ur WHERE ur.target_user_id = u.user_id AND ur.is_following = TRUE), 0),
       COALESCE((SELECT COUNT(*) FROM user_relations ur WHERE ur.user_id = u.user_id AND ur.is_following = TRUE), 0),
       COALESCE((SELECT COUNT(*) FROM group_members gm JOIN groups g ON g.group_id = gm.group_id WHERE gm.user_id = u.user_id AND g.deleted_at IS NULL), 0)
FROM users u
JOIN user_settings us ON us.user_id = u.user_id
WHERE us.searchable = TRUE
  AND u.is_deleted = FALSE
  AND us.location_visible = TRUE
  AND us.privacy_level = 'public'
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE u.s2_cell_id BETWEEN t.lo AND t.hi)`

	queryFeedGroups = `
SELECT g.group_id,
       g.name,
       COALESCE(g.avatar_url, ''),
       COALESCE(g.description, ''),
       g.has_password,
       g.location_name,
       g.latitude,
       g.longitude,
       COUNT(gm.user_id) AS members_count,
       EXISTS(SELECT 1 FROM group_members viewer_gm WHERE viewer_gm.group_id = g.group_id AND viewer_gm.user_id = $3),
       COALESCE((
         SELECT COUNT(*)
         FROM group_members gm_friend
         JOIN user_relations ur ON ur.target_user_id = gm_friend.user_id AND ur.user_id = $3 AND ur.is_following = TRUE
         WHERE gm_friend.group_id = g.group_id
       ), 0),
       g.join_method,
       g.requires_approval,
       g.tags,
       g.next_activity_at,
       g.is_searchable,
       g.max_members,
       g.created_at,
       g.owner_id,
       COALESCE((
         SELECT COUNT(*)
         FROM group_members om
         JOIN users ou ON ou.user_id = om.user_id AND ou.presence_status = 'online'
         WHERE om.group_id = g.group_id
       ), 0),
       viewer_gm2.role,
       viewer_gm2.joined_at,
       viewer_gm2.muted_until,
       viewer_gm2.is_pinned,
       COALESCE((
         SELECT COUNT(*) FROM group_messages m
         WHERE m.group_id = g.group_id
           AND (grs.last_read_sent_at IS NULL OR m.sent_at > grs.last_read_sent_at)
       ), 0)
FROM groups g
LEFT JOIN group_members gm ON gm.group_id = g.group_id
LEFT JOIN group_members viewer_gm2 ON viewer_gm2.group_id = g.group_id AND viewer_gm2.user_id = $3
LEFT JOIN group_message_read_state grs ON grs.group_id = g.group_id AND grs.user_id = $3
WHERE g.deleted_at IS NULL
  AND g.is_searchable = TRUE
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE g.s2_cell_id BETWEEN t.lo AND t.hi)
GROUP BY g.group_id, g.name, g.avatar_url, g.description, g.has_password, g.location_name, g.latitude, g.longitude,
         g.join_method, g.requires_approval, g.tags, g.next_activity_at, g.is_searchable, g.max_members, g.created_at, g.owner_id,
         viewer_gm2.role, viewer_gm2.joined_at, viewer_gm2.muted_until, viewer_gm2.is_pinned, grs.last_read_sent_at`

	queryFeedMomentsFollowing = `
SELECT m.moment_id,
       m.user_id,
       COALESCE(u.nickname, ''),
       COALESCE(u.avatar_url, ''),
       m.description,
       m.created_at,
       m.likes_count,
       m.comments_count,
       m.shares_count,
       m.location_name,
       m.latitude,
       m.longitude,
       EXISTS(SELECT 1 FROM reactions r WHERE r.target_type = 'moment' AND r.target_id = m.moment_id AND r.user_id = $1),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $1)
FROM moments m
JOIN users u ON u.user_id = m.user_id
JOIN user_relations ur ON ur.target_user_id = m.user_id AND ur.user_id = $1 AND ur.is_following = TRUE
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE`

	queryFeedMomentsNearby = `
SELECT m.moment_id,
       m.user_id,
       COALESCE(u.nickname, ''),
       COALESCE(u.avatar_url, ''),
       m.description,
       m.created_at,
       m.likes_count,
       m.comments_count,
       m.shares_count,
       m.location_name,
       m.latitude,
       m.longitude,
       EXISTS(SELECT 1 FROM reactions r WHERE r.target_type = 'moment' AND r.target_id = m.moment_id AND r.user_id = $3),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $3),
       EXISTS(SELECT 1 FROM user_relations ur WHERE ur.target_user_id = m.user_id AND ur.user_id = $3 AND ur.is_following = TRUE)
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)`

	queryFeedMomentTags = `
SELECT tag_name
FROM moment_tags
WHERE moment_id = $1
ORDER BY tag_name ASC`

	queryFeedMomentTagsByMomentIDs = `
SELECT moment_id, tag_name
FROM moment_tags
WHERE moment_id = ANY($1)
ORDER BY moment_id, tag_name ASC`

	queryFeedTrendingHotMoments = `
SELECT m.moment_id,
       m.user_id,
       COALESCE(u.nickname, ''),
       COALESCE(u.avatar_url, ''),
       m.description,
       m.created_at,
       m.likes_count,
       m.comments_count,
       m.shares_count,
       m.location_name,
       m.latitude,
       m.longitude,
       EXISTS(SELECT 1 FROM reactions r WHERE r.target_type = 'moment' AND r.target_id = m.moment_id AND r.user_id = $1),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $1)
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE`
)
