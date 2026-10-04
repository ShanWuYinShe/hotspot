package groups

const (
	queryInsertGroup = `
	INSERT INTO groups (group_id, name, avatar_url, description, has_password, password_hash, location_name, latitude, longitude, join_method, requires_approval, next_activity_at, is_searchable, members_searchable, max_members, owner_id, created_at, last_activity_at, tags, s2_cell_token, s2_cell_id)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`

	// 入群容量检查：FOR UPDATE 锁群行，使并发入群/审批/受邀加入在同一个
	// max_members 快照下串行判定，杜绝竞态超员。
	queryLockGroupCapacity = `
	SELECT g.max_members,
	       (SELECT COUNT(*)::int FROM group_members gm WHERE gm.group_id = g.group_id)
	FROM groups g
	WHERE g.group_id = $1
	FOR UPDATE`

	queryGetGroupPasswordHash = `
	SELECT password_hash
	FROM groups
	WHERE group_id = $1`

	queryGetGroup = `
	SELECT group_id, name, COALESCE(avatar_url, ''), COALESCE(description, ''), has_password, location_name, latitude, longitude, join_method, requires_approval, next_activity_at, is_searchable, members_searchable, max_members, owner_id, created_at, last_activity_at, COALESCE(tags, '{}'),
	       (SELECT COUNT(*)::int FROM group_members gm JOIN users u ON u.user_id = gm.user_id WHERE gm.group_id = groups.group_id AND u.presence_status = 'online') AS online_members_count
	FROM groups
	WHERE group_id = $1 AND deleted_at IS NULL`

	querySearchGroups = `
	SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.description, ''), g.has_password, g.location_name, g.latitude, g.longitude, g.join_method, g.requires_approval, g.next_activity_at, g.is_searchable, g.members_searchable, g.max_members, g.owner_id, g.created_at, g.last_activity_at, COALESCE(g.tags, '{}'),
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
  AND ($3::timestamptz IS NULL OR gm.joined_at > $3 OR (gm.joined_at = $3 AND gm.user_id > $4))
ORDER BY gm.joined_at ASC, gm.user_id ASC
LIMIT $5`

	// queryListAdminGroupMembers 是管理视角成员列表（含 muted_until 真值，null=未禁言），
	// 仅守卫子树 groups/admin/members/list 使用。
	queryListAdminGroupMembers = `
SELECT gm.user_id, COALESCE(u.nickname, gm.user_id), gm.role, gm.joined_at, gm.muted_until
FROM group_members gm
LEFT JOIN users u ON u.user_id = gm.user_id
WHERE gm.group_id = $1 AND ($2 = '' OR gm.role = $2)
  AND ($3::timestamptz IS NULL OR gm.joined_at > $3 OR (gm.joined_at = $3 AND gm.user_id > $4))
ORDER BY gm.joined_at ASC, gm.user_id ASC
LIMIT $5`

	// queryListMyGroupsBase is the static prefix of the my-groups query. The
	// keyset cursor predicate, ORDER BY and LIMIT are appended dynamically
	// inside ListMyGroups so pagination is done at the DB layer (no OFFSET, no
	// Go-side full scan). members_count is computed via a single aggregated
	// LEFT JOIN instead of a per-row COUNT(*) (which was an N+1).
	queryListMyGroupsBase = `
		SELECT g.group_id, name, COALESCE(avatar_url, ''), COALESCE(description, ''), has_password, location_name, latitude, longitude, join_method, g.requires_approval, g.next_activity_at, is_searchable, members_searchable, max_members, g.owner_id, g.created_at, g.last_activity_at, COALESCE(g.tags, '{}'),
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
	-- members_count 走 LATERAL 逐行计数（PK 前缀索引探测），而非此前对整个
	-- group_members 表的无过滤 GROUP BY 全表聚合 + hash aggregate——后者与用户
	-- 页大小无关、恒为全表扫描，外层 keyset 分页形同虚设（审计 perf P1-3）。
	-- 外层 WHERE gm.user_id = $1 已把行集限定为当前页，LATERAL 只跑页大小次数。
	LEFT JOIN LATERAL (
		SELECT COUNT(*)::int AS members_count FROM group_members gm2 WHERE gm2.group_id = g.group_id
	) mem ON TRUE
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
SELECT ga.application_id, ga.group_id, g.name, COALESCE(g.avatar_url, ''),
       (SELECT COUNT(*)::int FROM group_members gm WHERE gm.group_id = ga.group_id) AS members_count,
       ga.applicant_user_id, COALESCE(au.nickname, ''), COALESCE(ga.message, ''), ga.status, ga.created_at, ga.handled_at, ga.handled_by_user_id
FROM group_applications ga
JOIN groups g ON g.group_id = ga.group_id
LEFT JOIN users au ON au.user_id = ga.applicant_user_id
WHERE ga.application_id = $1`

	// 申请人昵称经 users 表解析（此前占位填 user_id，管理收件箱直接展示裸
	// id；与 ListSentInvites/ListMyInvites 的 LEFT JOIN 口径对齐）。
	queryListGroupApplications = `
SELECT ga.application_id, ga.group_id, g.name, COALESCE(g.avatar_url, ''),
       (SELECT COUNT(*)::int FROM group_members gm WHERE gm.group_id = ga.group_id) AS members_count,
       ga.applicant_user_id, COALESCE(au.nickname, ''), COALESCE(ga.message, ''), ga.status, ga.created_at, ga.handled_at, ga.handled_by_user_id
FROM group_applications ga
JOIN groups g ON g.group_id = ga.group_id
LEFT JOIN users au ON au.user_id = ga.applicant_user_id
WHERE ga.group_id = $1 AND ($2 = '' OR ga.status = $2)
  AND ($3::timestamptz IS NULL OR ga.created_at < $3 OR (ga.created_at = $3 AND ga.application_id < $4))
ORDER BY ga.created_at DESC, ga.application_id DESC
LIMIT $5`

	queryListMyGroupApplications = `
SELECT ga.application_id, ga.group_id, g.name, COALESCE(g.avatar_url, ''),
       (SELECT COUNT(*)::int FROM group_members gm WHERE gm.group_id = ga.group_id) AS members_count,
       ga.applicant_user_id, COALESCE(au.nickname, ''), COALESCE(ga.message, ''), ga.status, ga.created_at, ga.handled_at, ga.handled_by_user_id
FROM group_applications ga
JOIN groups g ON g.group_id = ga.group_id
LEFT JOIN users au ON au.user_id = ga.applicant_user_id
WHERE ga.applicant_user_id = $1 AND ($2 = '' OR ga.status = $2)
ORDER BY ga.created_at DESC, ga.application_id DESC`

	// RETURNING + pending 守卫：状态流转仅对 pending 生效；并发/重复处理时
	// 0 行命中返回 ErrNoRows，repo 据此返回 ErrAlreadyProcessed（防
	// approve/cancel 竞态双写与已处理申请被改回）。
	queryUpdateGroupApplication = `
UPDATE group_applications
SET status = $2, handled_at = $3, handled_by_user_id = $4
WHERE application_id = $1 AND status = 'pending'
RETURNING application_id`

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
  AND (expires_at IS NULL OR expires_at > now())
ORDER BY created_at DESC, invite_id DESC
LIMIT 1`

	queryListSentInvites = `
SELECT gi.invite_id, gi.inviter_user_id, COALESCE(ui.nickname, ''),
       gi.invitee_user_id, COALESCE(ue.nickname, ''),
       COALESCE(gi.message, ''), gi.status, gi.created_at, gi.expires_at, gi.responded_at
FROM group_invites gi
LEFT JOIN users ui ON ui.user_id = gi.inviter_user_id
LEFT JOIN users ue ON ue.user_id = gi.invitee_user_id
WHERE gi.group_id = $1 AND ($2 = '' OR gi.status = $2)
  AND ($3::timestamptz IS NULL OR gi.created_at < $3 OR (gi.created_at = $3 AND gi.invite_id < $4))
ORDER BY gi.created_at DESC, gi.invite_id DESC
LIMIT $5`

	// inviter 昵称经 users 表解析（此前占位填 user_id，通知待办区块直接
	// 把裸 id 展示给用户；与 ListSentInvites 的 LEFT JOIN 口径对齐）。
	queryListMyInvites = `
SELECT gi.invite_id, gi.group_id, g.name, COALESCE(g.avatar_url, ''),
       (SELECT COUNT(*)::int FROM group_members gm WHERE gm.group_id = gi.group_id) AS members_count,
       gi.inviter_user_id, COALESCE(iu.nickname, ''), COALESCE(gi.message, ''), gi.status, gi.created_at, gi.expires_at, gi.responded_at
FROM group_invites gi
JOIN groups g ON g.group_id = gi.group_id
LEFT JOIN users iu ON iu.user_id = gi.inviter_user_id
WHERE gi.invitee_user_id = $1 AND ($2 = '' OR gi.status = $2)
  AND ($3::timestamptz IS NULL OR gi.created_at < $3 OR (gi.created_at = $3 AND gi.invite_id < $4))
ORDER BY gi.created_at DESC, gi.invite_id DESC
LIMIT $5`

	// Accept/Reject 共用：仅 pending 可流转（RETURNING 让 0 行命中可检测）。
	queryUpdateGroupInvite = `
UPDATE group_invites
SET status = $2, responded_at = $3
WHERE invite_id = $1 AND status = 'pending'
RETURNING invite_id`

	// Revoke 专用：归属双键（invite_id + group_id）+ pending 守卫。缺少
	// group_id 条件时任意群管理员可传自己的 group_id 撤销其它群的邀请（IDOR）。
	queryRevokeGroupInvite = `
UPDATE group_invites
SET status = $2, responded_at = $3
WHERE invite_id = $1 AND group_id = $4 AND status = 'pending'
RETURNING invite_id`

	queryUpdateGroup = `
	UPDATE groups
	SET name = COALESCE($2, name),
	    avatar_url = COALESCE($3, avatar_url),
	    description = COALESCE($4, description),
	    has_password = COALESCE($5, has_password),
	    password_hash = COALESCE($6, password_hash),
	    join_method = COALESCE($7, join_method),
	    requires_approval = COALESCE($8, requires_approval),
	    next_activity_at = COALESCE($9, next_activity_at),
	    is_searchable = COALESCE($10, is_searchable),
	    members_searchable = COALESCE($11, members_searchable),
	    max_members = COALESCE($12, max_members),
	    location_name = COALESCE($13, location_name),
	    latitude = COALESCE($14, latitude),
	    longitude = COALESCE($15, longitude),
	    tags = COALESCE($16, tags),
	    s2_cell_token = COALESCE($17, s2_cell_token),
	    s2_cell_id = COALESCE($18, s2_cell_id)
	WHERE group_id = $1`

	queryDeleteGroup = `
DELETE FROM groups
WHERE group_id = $1`

	// 解散群组的事务性清理（repo.Dismiss）：groups 行硬删前必须同事务清空
	// 全部子表——否则残留成员/申请/邀请/封禁/公告行，让已解散群在详情页
	// 呈现幽灵态（可见 + 虚假成员数，且可再次点击加入后被 404 拒绝）。
	// group_messages 按归档语义保留（群不存在后无查询路径触达）。
	// read_state 由 049 迁移的 FK ON DELETE CASCADE 随 group_members 自动清理。
	queryDeleteGroupMembersByGroup = `
DELETE FROM group_members
WHERE group_id = $1`

	queryDeleteGroupApplicationsByGroup = `
DELETE FROM group_applications
WHERE group_id = $1`

	queryDeleteGroupInvitesByGroup = `
DELETE FROM group_invites
WHERE group_id = $1`

	queryDeleteGroupBansByGroup = `
DELETE FROM group_bans
WHERE group_id = $1`

	queryDeleteGroupAnnouncementsByGroup = `
DELETE FROM group_announcements
WHERE group_id = $1`

	queryInsertGroupAnnouncement = `
INSERT INTO group_announcements (announcement_id, group_id, title, content, created_by_user_id, is_pinned, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`

	queryListGroupAnnouncements = `
SELECT announcement_id, group_id, title, content, created_by_user_id, is_pinned, created_at
FROM group_announcements
WHERE group_id = $1 AND deleted_at IS NULL
ORDER BY is_pinned DESC, created_at DESC, announcement_id DESC`

	// 公告更新/删除带归属双键：groupID 是授权锚点，仅按全局唯一
	// announcement_id 定位会让 A 群管理员操作 B 群公告（IDOR）。
	queryUpdateGroupAnnouncement = `
UPDATE group_announcements
SET title = COALESCE($2, title),
    content = COALESCE($3, content),
    is_pinned = COALESCE($4, is_pinned),
    updated_at = $5
WHERE announcement_id = $1 AND group_id = $6 AND deleted_at IS NULL`

	queryDeleteGroupAnnouncement = `
UPDATE group_announcements
SET deleted_at = $2
WHERE announcement_id = $1 AND group_id = $3 AND deleted_at IS NULL`

	queryUpdateGroupMemberSettings = `
UPDATE group_members
SET is_muted = COALESCE($3, is_muted),
    is_pinned = COALESCE($4, is_pinned)
WHERE group_id = $1 AND user_id = $2`

	queryDeleteGroupMember = `
DELETE FROM group_members
WHERE group_id = $1 AND user_id = $2`

	// 角色 SetMemberRole 专用：role <> 'owner' 守卫挡住对 owner 行的任何改动
	// （审计 F-05 防线二，服务层已拒绝 owner 目标）。owner 行唯一合法的变更
	// 是转让群主，走 queryDemoteGroupOwnerRole（同事务内、已验操作者身份）。
	queryUpdateGroupMemberRole = `
UPDATE group_members
SET role = $3
WHERE group_id = $1 AND user_id = $2 AND role <> 'owner'`

	// 转让群主第一步专用：仅当行当前是 owner 才降级，防误伤普通成员。
	queryDemoteGroupOwnerRole = `
UPDATE group_members
SET role = $3
WHERE group_id = $1 AND user_id = $2 AND role = 'owner'`

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

	// 被封禁者与操作者昵称经 users 表解析（此前占位填 user_id，封禁名单
	// 直接展示裸 id；与申请/邀请列表的 LEFT JOIN 口径对齐）。
	queryListGroupBans = `
SELECT gb.user_id, COALESCE(bu.nickname, ''), gb.banned_at, gb.reason, gb.banned_by_user_id, COALESCE(ou.nickname, '')
FROM group_bans gb
LEFT JOIN users bu ON bu.user_id = gb.user_id
LEFT JOIN users ou ON ou.user_id = gb.banned_by_user_id
WHERE gb.group_id = $1
ORDER BY gb.banned_at DESC, gb.user_id ASC`

	queryIsGroupBanned = `
SELECT 1 FROM group_bans WHERE group_id = $1 AND user_id = $2`

	// 封禁级联（契约级，spec api_spec_group.md → set_ban）：
	// 被 ban 用户「发出的」该群 pending 申请、以及「发给」被 ban 用户的该群
	// pending 邀请，均须置 revoked。此前完全缺失级联——BanMember 只插封禁 +
	// 删成员，导致封禁后 pending 申请仍可被管理员审批通过、pending 邀请仍可
	// 被接受，封禁拦截被两条旁路绕过。
	queryRevokeApplicationsByApplicant = `
UPDATE group_applications
SET status = 'revoked', handled_at = $3
WHERE group_id = $1 AND applicant_user_id = $2 AND status = 'pending'`

	queryRevokeInvitesByInvitee = `
UPDATE group_invites
SET status = 'revoked', responded_at = $3
WHERE group_id = $1 AND invitee_user_id = $2 AND status = 'pending'`

	queryUpdateGroupOwner = `
UPDATE groups
SET owner_id = $2
WHERE group_id = $1`

	querySearchGroupMembers = `
SELECT gm.user_id, COALESCE(u.nickname, gm.user_id), gm.role, gm.joined_at
FROM group_members gm
LEFT JOIN users u ON u.user_id = gm.user_id
WHERE gm.group_id = $1 AND ($2 = '' OR gm.role = $2)
  AND ($3 = '' OR gm.user_id ILIKE '%' || $3 || '%' ESCAPE '\' OR u.nickname ILIKE '%' || $3 || '%' ESCAPE '\')
  AND ($4::timestamptz IS NULL OR gm.joined_at > $4 OR (gm.joined_at = $4 AND gm.user_id > $5))
ORDER BY gm.joined_at ASC, gm.user_id ASC
LIMIT $6`

	// queryListGroupsByUser 是 users 模块「某用户的群组列表」聚合读（原实现位于
	// users repo，直查 groups/group_members 表；现已归位本模块，users 经 Service
	// 接口消费）。
	queryListGroupsByUser = `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.description, ''), g.has_password, g.location_name, g.latitude, g.longitude, g.join_method, g.is_searchable, g.max_members, gm.joined_at
FROM group_members gm
JOIN groups g ON g.group_id = gm.group_id
WHERE gm.user_id = $1 AND g.deleted_at IS NULL AND g.is_searchable = TRUE
  AND (
    $2::timestamptz IS NULL
    OR gm.joined_at < $2::timestamptz
    OR (gm.joined_at = $2::timestamptz AND g.group_id > $3)
  )
ORDER BY gm.joined_at DESC, g.group_id ASC
LIMIT $4`

	// queryCountOwnedGroups 统计用户担任群主的未解散群数，供 users 注销守卫：
	// 群主注销会制造无人可管理的孤儿群（退群/降级等路径均已拒绝制造孤儿，
	// 见 SetMemberRole 审计 F-05），注销前必须先转让或解散。
	queryCountOwnedGroups = `
SELECT COUNT(*)
FROM group_members gm
JOIN groups g ON g.group_id = gm.group_id
WHERE gm.user_id = $1 AND gm.role = 'owner' AND g.deleted_at IS NULL`

	queryCountCommonGroups = `
SELECT COUNT(*)
FROM (
	SELECT gm.group_id
	FROM group_members gm
	JOIN groups g ON g.group_id = gm.group_id
	WHERE gm.user_id = $1 AND g.deleted_at IS NULL
	INTERSECT
	SELECT gm.group_id
	FROM group_members gm
	JOIN groups g ON g.group_id = gm.group_id
	WHERE gm.user_id = $2 AND g.deleted_at IS NULL
)`
)
