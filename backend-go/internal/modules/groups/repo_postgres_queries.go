package groups

const (
	queryInsertGroup = `
	INSERT INTO groups (group_id, name, avatar_url, description, has_password, location_name, latitude, longitude, join_method, requires_approval, next_activity, is_searchable, max_members, owner_id, created_at, last_activity_at, tags, s2_cell_token, s2_cell_id)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`

	queryGetGroup = `
	SELECT group_id, name, avatar_url, description, has_password, location_name, latitude, longitude, join_method, requires_approval, next_activity, is_searchable, max_members, owner_id, created_at, last_activity_at, COALESCE(tags, '{}'),
	       (SELECT COUNT(*)::int FROM group_members gm JOIN users u ON u.user_id = gm.user_id WHERE gm.group_id = groups.group_id AND u.presence_status = 'online') AS online_members_count
	FROM groups
	WHERE group_id = $1 AND deleted_at IS NULL`

	querySearchGroups = `
	SELECT g.group_id, g.name, g.avatar_url, g.description, g.has_password, g.location_name, g.latitude, g.longitude, g.join_method, g.requires_approval, g.next_activity, g.is_searchable, g.max_members, g.owner_id, g.created_at, g.last_activity_at, COALESCE(g.tags, '{}'),
	       (SELECT COUNT(*)::int FROM group_members gm JOIN users u ON u.user_id = gm.user_id WHERE gm.group_id = g.group_id AND u.presence_status = 'online') AS online_members_count
	FROM groups g
	WHERE g.deleted_at IS NULL`

	queryGetGroupMembership = `
SELECT role, joined_at, is_muted, muted_until
FROM group_members
WHERE group_id = $1 AND user_id = $2`

	queryGetGroupMemberships = `
SELECT group_id, role, joined_at, is_muted, muted_until
FROM group_members
WHERE user_id = $1 AND group_id = ANY($2)`

	queryCountGroupMembers = `
SELECT COUNT(*)
FROM group_members
WHERE group_id = $1`

	queryListGroupMembers = `
SELECT gm.user_id, COALESCE(u.nickname, gm.user_id), gm.role, gm.joined_at
FROM group_members gm
LEFT JOIN users u ON u.user_id = gm.user_id
WHERE gm.group_id = $1 AND ($2 = '' OR gm.role = $2)
ORDER BY gm.joined_at ASC, gm.user_id ASC`

	// queryListMyGroupsBase is the static prefix of the my-groups query. The
	// keyset cursor predicate, ORDER BY and LIMIT are appended dynamically
	// inside ListMyGroups so pagination is done at the DB layer (no OFFSET, no
	// Go-side full scan). members_count is computed via a single aggregated
	// LEFT JOIN instead of a per-row COUNT(*) (which was an N+1).
	queryListMyGroupsBase = `
		SELECT g.group_id, name, avatar_url, description, has_password, location_name, latitude, longitude, join_method, g.requires_approval, g.next_activity, is_searchable, max_members, g.owner_id, g.created_at, g.last_activity_at, COALESCE(g.tags, '{}'),
		       COALESCE(mem.members_count, 0),
		       (SELECT COUNT(*)::int FROM group_members gm JOIN users u ON u.user_id = gm.user_id WHERE gm.group_id = g.group_id AND u.presence_status = 'online') AS online_members_count,
		       gm.role, gm.joined_at, gm.is_muted, gm.muted_until, gm.is_pinned,
		       COALESCE((
		         SELECT COUNT(*) FROM group_messages m
		         WHERE m.group_id = g.group_id
		           AND (grs.last_read_sent_at IS NULL OR m.sent_at > grs.last_read_sent_at)
		       ), 0) AS unread_count,
		       lm.message_id, lm.type, lm.content, lm.sender_user_id, lm.sender_nickname, lm.sender_avatar_url, lm.sent_at
		FROM groups g
	JOIN group_members gm ON gm.group_id = g.group_id
	LEFT JOIN group_message_read_state grs ON grs.group_id = g.group_id AND grs.user_id = $1
	LEFT JOIN (
		SELECT group_id, COUNT(*)::int AS members_count FROM group_members GROUP BY group_id
	) mem ON mem.group_id = g.group_id
	LEFT JOIN LATERAL (
		SELECT m.message_id, m.type, m.content, u.user_id AS sender_user_id, u.nickname AS sender_nickname, u.avatar_url AS sender_avatar_url, m.sent_at
		FROM group_messages m
		LEFT JOIN users u ON u.user_id = m.sender_user_id
		WHERE m.group_id = g.group_id AND m.is_recalled = FALSE
		ORDER BY m.sent_at DESC, m.message_id DESC
		LIMIT 1
	) lm ON TRUE
WHERE gm.user_id = $1 AND g.deleted_at IS NULL`

	queryCountOnlineGroupMembers = `
SELECT COUNT(*)
FROM group_members gm
JOIN users u ON u.user_id = gm.user_id
WHERE gm.group_id = $1 AND u.presence_status = 'online'`

	queryInsertGroupMember = `
INSERT INTO group_members (group_id, user_id, role, joined_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (group_id, user_id) DO NOTHING`

	queryInsertGroupApplication = `
INSERT INTO group_applications (application_id, group_id, applicant_user_id, message, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6)`

	queryFindPendingGroupApplication = `
SELECT application_id
FROM group_applications
WHERE group_id = $1 AND applicant_user_id = $2 AND status = 'pending'
ORDER BY created_at DESC, application_id DESC
LIMIT 1`

	queryGetGroupApplication = `
SELECT application_id, group_id, applicant_user_id, COALESCE(message, ''), status, created_at, handled_at, handled_by_user_id
FROM group_applications
WHERE application_id = $1`

	queryListGroupApplications = `
SELECT application_id, applicant_user_id, COALESCE(message, ''), status, created_at, handled_at, handled_by_user_id
FROM group_applications
WHERE group_id = $1 AND ($2 = '' OR status = $2)
ORDER BY created_at DESC, application_id DESC`

	queryListMyGroupApplications = `
SELECT application_id, group_id, COALESCE(message, ''), status, created_at, handled_at, handled_by_user_id
FROM group_applications
WHERE applicant_user_id = $1 AND ($2 = '' OR status = $2)
ORDER BY created_at DESC, application_id DESC`

	queryUpdateGroupApplication = `
UPDATE group_applications
SET status = $2, handled_at = $3, handled_by_user_id = $4
WHERE application_id = $1`

	queryInsertGroupInvite = `
INSERT INTO group_invites (invite_id, group_id, inviter_user_id, invitee_user_id, message, status, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	queryGetGroupInvite = `
SELECT invite_id, group_id, inviter_user_id, invitee_user_id, COALESCE(message, ''), status, created_at, expires_at, responded_at
FROM group_invites
WHERE invite_id = $1`

	queryFindPendingGroupInvite = `
SELECT invite_id, expires_at
FROM group_invites
WHERE group_id = $1 AND invitee_user_id = $2 AND status = 'pending'
ORDER BY created_at DESC, invite_id DESC
LIMIT 1`

	queryListSentInvites = `
SELECT invite_id, invitee_user_id, COALESCE(message, ''), status, created_at, expires_at, responded_at
FROM group_invites
WHERE group_id = $1 AND ($2 = '' OR status = $2)
ORDER BY created_at DESC, invite_id DESC`

	queryListMyInvites = `
SELECT gi.invite_id, gi.group_id, g.name, COALESCE(g.avatar_url, ''),
       (SELECT COUNT(*)::int FROM group_members gm WHERE gm.group_id = gi.group_id) AS members_count,
       gi.inviter_user_id, COALESCE(gi.message, ''), gi.status, gi.created_at, gi.expires_at, gi.responded_at
FROM group_invites gi
JOIN groups g ON g.group_id = gi.group_id
WHERE gi.invitee_user_id = $1 AND ($2 = '' OR gi.status = $2)
ORDER BY gi.created_at DESC, gi.invite_id DESC`

	queryUpdateGroupInvite = `
UPDATE group_invites
SET status = $2, responded_at = $3
WHERE invite_id = $1`

	queryUpdateGroup = `
	UPDATE groups
	SET name = COALESCE($2, name),
	    avatar_url = COALESCE($3, avatar_url),
	    description = COALESCE($4, description),
	    has_password = COALESCE($5, has_password),
	    join_method = COALESCE($6, join_method),
	    requires_approval = COALESCE($7, requires_approval),
	    next_activity = COALESCE($8, next_activity),
	    is_searchable = COALESCE($9, is_searchable),
	    max_members = COALESCE($10, max_members),
	    location_name = COALESCE($11, location_name),
	    latitude = COALESCE($12, latitude),
	    longitude = COALESCE($13, longitude),
	    tags = COALESCE($14, tags),
	    s2_cell_token = COALESCE($15, s2_cell_token),
	    s2_cell_id = COALESCE($16, s2_cell_id)
	WHERE group_id = $1`

	queryDeleteGroup = `
DELETE FROM groups
WHERE group_id = $1`

	queryInsertGroupAnnouncement = `
INSERT INTO group_announcements (announcement_id, group_id, title, content, created_by_user_id, is_pinned, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`

	queryListGroupAnnouncements = `
SELECT announcement_id, group_id, title, content, created_by_user_id, is_pinned, created_at
FROM group_announcements
WHERE group_id = $1 AND deleted_at IS NULL
ORDER BY is_pinned DESC, created_at DESC, announcement_id DESC`

	queryUpdateGroupAnnouncement = `
UPDATE group_announcements
SET title = COALESCE($2, title),
    content = COALESCE($3, content),
    is_pinned = COALESCE($4, is_pinned),
    updated_at = $5
WHERE announcement_id = $1 AND deleted_at IS NULL`

	queryDeleteGroupAnnouncement = `
UPDATE group_announcements
SET deleted_at = $2
WHERE announcement_id = $1 AND deleted_at IS NULL`

	queryUpdateGroupMemberSettings = `
UPDATE group_members
SET is_muted = COALESCE($3, is_muted),
    is_pinned = COALESCE($4, is_pinned)
WHERE group_id = $1 AND user_id = $2`

	queryDeleteGroupMember = `
DELETE FROM group_members
WHERE group_id = $1 AND user_id = $2`

	queryUpdateGroupMemberRole = `
UPDATE group_members
SET role = $3
WHERE group_id = $1 AND user_id = $2`

	queryMuteGroupMember = `
UPDATE group_members
SET is_muted = TRUE, muted_until = $3
WHERE group_id = $1 AND user_id = $2`

	queryUnmuteGroupMember = `
UPDATE group_members
SET is_muted = FALSE, muted_until = NULL
WHERE group_id = $1 AND user_id = $2`

	queryInsertGroupBan = `
INSERT INTO group_bans (group_id, user_id, banned_by_user_id, reason, banned_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (group_id, user_id) DO UPDATE SET banned_by_user_id = $3, reason = $4, banned_at = $5`

	queryDeleteGroupBan = `
DELETE FROM group_bans
WHERE group_id = $1 AND user_id = $2`

	queryListGroupBans = `
SELECT gb.user_id, gb.banned_at, gb.reason, gb.banned_by_user_id
FROM group_bans gb
WHERE gb.group_id = $1
ORDER BY gb.banned_at DESC, gb.user_id ASC`

	queryUpdateGroupOwner = `
UPDATE groups
SET owner_id = $2
WHERE group_id = $1`

	querySearchGroupMembers = `
SELECT gm.user_id, COALESCE(u.nickname, gm.user_id), gm.role, gm.joined_at
FROM group_members gm
LEFT JOIN users u ON u.user_id = gm.user_id
WHERE gm.group_id = $1 AND ($2 = '' OR gm.role = $2)
  AND ($3 = '' OR gm.user_id ILIKE '%' || $3 || '%' OR u.nickname ILIKE '%' || $3 || '%')
ORDER BY gm.joined_at ASC, gm.user_id ASC`
)
