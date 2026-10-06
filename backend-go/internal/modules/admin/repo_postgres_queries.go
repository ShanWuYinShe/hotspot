package admin

const (
	queryRoleInsert = `
INSERT INTO admin_roles (role_id, name, description, created_at, updated_at, deleted_at)
VALUES ($1,$2,$3,$4,$5,NULL)`

	queryRoleUpdate = `
UPDATE admin_roles
SET name = $2, description = $3, updated_at = $4
WHERE role_id = $1`

	queryRoleDelete = `
UPDATE admin_roles
SET deleted_at = $2, updated_at = $2
WHERE role_id = $1 AND deleted_at IS NULL`

	queryRoleDeleteAssignments = `
DELETE FROM admin_user_roles
WHERE role_id = $1`

	queryRoleReplacePermissions = `
DELETE FROM admin_role_permissions
WHERE role_id = $1`

	queryRoleInsertPermission = `
INSERT INTO admin_role_permissions (role_id, permission_key, created_at)
VALUES ($1,$2,$3)`

	// queryRoleInsertPermissionsBatch inserts all permissions for a role in one
	// round trip via unnest, replacing the previous per-permission loop.
	queryRoleInsertPermissionsBatch = `
INSERT INTO admin_role_permissions (role_id, permission_key, created_at)
SELECT $1, perm, $2 FROM unnest($3::text[]) AS perm`

	queryRoleByID = `
SELECT role_id, name, COALESCE(description, ''), created_at, updated_at, deleted_at IS NOT NULL
FROM admin_roles
WHERE role_id = $1`

	queryRoleByName = `
SELECT role_id, name, COALESCE(description, ''), created_at, updated_at, deleted_at IS NOT NULL
FROM admin_roles
WHERE lower(name) = lower($1)
ORDER BY deleted_at NULLS FIRST, updated_at DESC
LIMIT 1`

	queryRolePermissions = `
SELECT permission_key
FROM admin_role_permissions
WHERE role_id = $1
ORDER BY permission_key ASC`

	queryRoleList = `
SELECT r.role_id, r.name, COALESCE(r.description, ''), COALESCE(array_agg(rp.permission_key ORDER BY rp.permission_key) FILTER (WHERE rp.permission_key IS NOT NULL), ARRAY[]::text[])
FROM admin_roles r
LEFT JOIN admin_role_permissions rp ON rp.role_id = r.role_id
WHERE r.deleted_at IS NULL
GROUP BY r.role_id, r.name, r.description
ORDER BY r.name ASC, r.role_id ASC`

	queryAdminUpsert = `
INSERT INTO admin_users (user_id, grant_all_permissions, granted_at, revoked_at)
VALUES ($1,$2,$3,NULL)
ON CONFLICT (user_id) DO UPDATE
SET grant_all_permissions = EXCLUDED.grant_all_permissions,
    granted_at = EXCLUDED.granted_at,
    revoked_at = NULL`

	queryAdminReplaceRoles = `
DELETE FROM admin_user_roles
WHERE user_id = $1`

	// 角色批量插入（单次往返）：RoleIDs 经 unnest 展开为行，替代逐条 EXEC。
	queryAdminInsertRoles = `
INSERT INTO admin_user_roles (user_id, role_id, created_at)
SELECT $1, role_id, $3 FROM unnest($2::text[]) AS t(role_id)`

	queryAdminRevoke = `
UPDATE admin_users
SET revoked_at = $2
WHERE user_id = $1 AND revoked_at IS NULL`

	queryAdminActorDetail = `
SELECT au.user_id,
       COALESCE(u.nickname, au.user_id),
       COALESCE(u.avatar_url, ''),
       au.grant_all_permissions,
       au.granted_at,
       COALESCE(ar.role_id, ''),
       COALESCE(ar.name, ''),
       COALESCE(array_agg(arp.permission_key ORDER BY arp.permission_key) FILTER (WHERE arp.permission_key IS NOT NULL), ARRAY[]::text[])
FROM admin_users au
LEFT JOIN users u ON u.user_id = au.user_id
LEFT JOIN admin_user_roles aur ON aur.user_id = au.user_id
LEFT JOIN admin_roles ar ON ar.role_id = aur.role_id AND ar.deleted_at IS NULL
LEFT JOIN admin_role_permissions arp ON arp.role_id = ar.role_id
WHERE au.revoked_at IS NULL AND au.user_id = $1
GROUP BY au.user_id, u.nickname, u.avatar_url, au.grant_all_permissions, au.granted_at, ar.role_id, ar.name
ORDER BY ar.name ASC, ar.role_id ASC`

	queryAdminList = `
SELECT au.user_id,
       COALESCE(u.nickname, au.user_id),
       COALESCE(u.avatar_url, ''),
       au.grant_all_permissions,
       au.granted_at,
       COALESCE(ar.role_id, ''),
       COALESCE(ar.name, ''),
       COALESCE(array_agg(arp.permission_key ORDER BY arp.permission_key) FILTER (WHERE arp.permission_key IS NOT NULL), ARRAY[]::text[])
FROM admin_users au
LEFT JOIN users u ON u.user_id = au.user_id
LEFT JOIN admin_user_roles aur ON aur.user_id = au.user_id
LEFT JOIN admin_roles ar ON ar.role_id = aur.role_id AND ar.deleted_at IS NULL
LEFT JOIN admin_role_permissions arp ON arp.role_id = ar.role_id
WHERE au.revoked_at IS NULL
GROUP BY au.user_id, u.nickname, u.avatar_url, au.grant_all_permissions, au.granted_at, ar.role_id, ar.name
ORDER BY au.granted_at DESC, au.user_id ASC, ar.name ASC, ar.role_id ASC`

	queryUserExists = `
SELECT user_id, nickname, COALESCE(avatar_url, '')
FROM users
WHERE user_id = $1`

	queryAuditInsert = `
INSERT INTO admin_audit_logs (log_id, action, operator_id, operator_name, operator_avatar_url, target_type, target_id, target_name, details_json, ip_location, ip_address, device_summary, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`

	queryAuditList = `
SELECT log_id, action, operator_id, operator_name, COALESCE(operator_avatar_url, ''), target_type, target_id, COALESCE(target_name, ''), COALESCE(details_json, '{}'), COALESCE(ip_location, ''), COALESCE(ip_address, ''), COALESCE(device_summary, ''), created_at
FROM admin_audit_logs
ORDER BY created_at DESC, log_id DESC`

	// AdminUserItem 读取（spec api_spec_admin.md）：banned_until = 封禁到期时间
	// （banned_at + duration；null = 未封禁/永久封禁），privacy_level 来自
	// user_settings（平台管理视角豁免隐私过滤、恒返回真值）。
	queryAdminSearchUsers = `
SELECT u.user_id,
       u.nickname,
       COALESCE(u.avatar_url, ''),
       COALESCE(u.bio, ''),
       u.presence_status,
       u.last_online,
       u.location_name,
       u.latitude,
       u.longitude,
       u.user_type,
       u.created_at,
       COALESCE(u.status, 'normal'),
       COALESCE(a.login_id, ''),
       COALESCE(u.register_ip, ''),
       COALESCE(u.register_location, ''),
       COALESCE(u.last_login_ip, ''),
       COALESCE(u.last_login_location, ''),
       COALESCE(u.email, ''),
       COALESCE(u.phone, ''),
       CASE WHEN COALESCE(u.status, 'normal') = 'banned'
                 AND u.banned_at IS NOT NULL
                 AND COALESCE(u.banned_duration_days, 0) > 0
            THEN u.banned_at + make_interval(days => COALESCE(u.banned_duration_days, 0))
            ELSE NULL END AS banned_until,
       COALESCE(us.privacy_level, 'public')
FROM users u
LEFT JOIN auth_accounts a ON a.user_id = u.user_id
LEFT JOIN user_settings us ON us.user_id = u.user_id
WHERE ($1 = '' OR u.user_id = $1)
  AND ($2 = '' OR lower(u.nickname) LIKE '%' || lower($2) || '%')
  AND ($3 = '' OR lower(COALESCE(a.login_id, '')) LIKE '%' || lower($3) || '%')
  AND ($4 = '' OR COALESCE(u.status, 'normal') = $4)
  AND ($5 = '' OR u.user_type = $5)
ORDER BY u.created_at DESC, u.user_id ASC`
	queryAdminGetUser = `
SELECT u.user_id,
       u.nickname,
       COALESCE(u.avatar_url, ''),
       COALESCE(u.bio, ''),
       u.presence_status,
       u.last_online,
       u.location_name,
       u.latitude,
       u.longitude,
       u.user_type,
       u.created_at,
       COALESCE(u.status, 'normal'),
       COALESCE(a.login_id, ''),
       COALESCE(u.register_ip, ''),
       COALESCE(u.register_location, ''),
       COALESCE(u.last_login_ip, ''),
       COALESCE(u.last_login_location, ''),
       COALESCE(u.email, ''),
       COALESCE(u.phone, ''),
       CASE WHEN COALESCE(u.status, 'normal') = 'banned'
                 AND u.banned_at IS NOT NULL
                 AND COALESCE(u.banned_duration_days, 0) > 0
            THEN u.banned_at + make_interval(days => COALESCE(u.banned_duration_days, 0))
            ELSE NULL END AS banned_until,
       COALESCE(us.privacy_level, 'public')
FROM users u
LEFT JOIN auth_accounts a ON a.user_id = u.user_id
LEFT JOIN user_settings us ON us.user_id = u.user_id
WHERE u.user_id = $1`
	queryAdminGetUserBanState = `
SELECT COALESCE(status, 'normal'), COALESCE(banned_reason, ''), COALESCE(banned_duration_days, 0), banned_at
FROM users
WHERE user_id = $1`
	queryAdminSetUserBanState = `
UPDATE users
SET status = $2,
    banned_reason = $3,
    banned_duration_days = $4,
    banned_at = $5
WHERE user_id = $1`
	queryAdminListUserSessions = `
SELECT session_id
FROM auth_sessions
WHERE user_id = $1 AND revoked_at IS NULL
ORDER BY last_active_at DESC, session_id ASC`
	queryAdminRevokeSession = `
UPDATE auth_sessions
SET revoked_at = NOW()
WHERE session_id = $1 AND revoked_at IS NULL`
	queryAdminSearchGroups = `
SELECT g.group_id,
       g.name,
       COALESCE(g.avatar_url, ''),
       COALESCE(g.description, ''),
       g.has_password,
       g.location_name,
       g.latitude,
       g.longitude,
       g.join_method,
       g.is_searchable,
       g.max_members,
       g.created_at,
       COALESCE(g.owner_id, ''),
       COALESCE(owner.nickname, COALESCE(g.owner_id, '')),
       COALESCE(owner.avatar_url, ''),
       COALESCE(members.members_count, 0),
       COALESCE(last_message.last_activity_at, g.created_at)
FROM groups g
LEFT JOIN users owner ON owner.user_id = g.owner_id
LEFT JOIN (
    SELECT group_id, COUNT(*)::int AS members_count
    FROM group_members
    GROUP BY group_id
) members ON members.group_id = g.group_id
LEFT JOIN (
    SELECT group_id, MAX(sent_at) AS last_activity_at
    FROM group_messages
    GROUP BY group_id
) last_message ON last_message.group_id = g.group_id
WHERE g.deleted_at IS NULL
  AND ($1 = '' OR g.group_id = $1)
  AND ($2 = '' OR lower(g.name) LIKE '%' || lower($2) || '%')
  AND ($3 = '' OR COALESCE(g.owner_id, '') = $3)
ORDER BY g.created_at DESC, g.group_id ASC`
	queryAdminGetGroup = `
SELECT g.group_id,
       g.name,
       COALESCE(g.avatar_url, ''),
       COALESCE(g.description, ''),
       g.has_password,
       g.location_name,
       g.latitude,
       g.longitude,
       g.join_method,
       g.is_searchable,
       g.max_members,
       g.created_at,
       COALESCE(g.owner_id, ''),
       COALESCE(owner.nickname, COALESCE(g.owner_id, '')),
       COALESCE(owner.avatar_url, ''),
       COALESCE(members.members_count, 0),
       COALESCE(last_message.last_activity_at, g.created_at)
FROM groups g
LEFT JOIN users owner ON owner.user_id = g.owner_id
LEFT JOIN (
    SELECT group_id, COUNT(*)::int AS members_count
    FROM group_members
    GROUP BY group_id
) members ON members.group_id = g.group_id
LEFT JOIN (
    SELECT group_id, MAX(sent_at) AS last_activity_at
    FROM group_messages
    GROUP BY group_id
) last_message ON last_message.group_id = g.group_id
WHERE g.deleted_at IS NULL AND g.group_id = $1`
	queryAdminDismissGroup = `
UPDATE groups
SET deleted_at = NOW()
WHERE group_id = $1 AND deleted_at IS NULL`
	// 解散级联（spec api_spec_admin.md groups/dismiss 契约级）：该群全部 pending
	// 申请/邀请自动置 revoked，字段口径与 groups 模块 BanMember 的级联一致
	//（申请记 handled_at、邀请记 responded_at）。与 queryAdminDismissGroup 同事务执行。
	queryAdminRevokePendingGroupApplications = `
UPDATE group_applications
SET status = 'revoked', handled_at = $2
WHERE group_id = $1 AND status = 'pending'`
	queryAdminRevokePendingGroupInvites = `
UPDATE group_invites
SET status = 'revoked', responded_at = $2
WHERE group_id = $1 AND status = 'pending'`
	queryAdminListGroupMemberIDs = `
SELECT user_id
FROM group_members
WHERE group_id = $1
ORDER BY user_id ASC`
	queryAdminSearchReports = `
SELECT r.report_id,
       r.target_type,
       r.target_id,
       r.reason,
       COALESCE(r.description, ''),
       r.status,
       r.created_at,
       r.handled_at,
       reporter.user_id,
       COALESCE(reporter.nickname, reporter.user_id),
       COALESCE(reporter.avatar_url, ''),
       COALESCE(handler.user_id, ''),
       COALESCE(handler.nickname, ''),
       COALESCE(handler.avatar_url, ''),
       COALESCE(r.handle_note, ''),
       r.message_snapshot::text
FROM reports r
JOIN users reporter ON reporter.user_id = r.creator_user_id
LEFT JOIN users handler ON handler.user_id = r.handled_by_user_id
WHERE ($1 = '' OR r.status = $1)
  AND ($2 = '' OR r.target_type = $2)
ORDER BY r.created_at DESC, r.report_id DESC`
	queryAdminGetReportHandleRecord = `
SELECT r.report_id,
       r.target_type,
       r.target_id,
       r.reason,
       COALESCE(r.description, ''),
       r.status,
       r.created_at,
       r.handled_at,
       reporter.user_id,
       COALESCE(reporter.nickname, reporter.user_id),
       COALESCE(reporter.avatar_url, ''),
       COALESCE(handler.user_id, ''),
       COALESCE(handler.nickname, ''),
       COALESCE(handler.avatar_url, ''),
       COALESCE(r.handle_note, ''),
       r.message_snapshot::text,
       COALESCE(target_user.user_id, ''),
       COALESCE(target_user.nickname, ''),
       COALESCE(target_user.avatar_url, '')
FROM reports r
JOIN users reporter ON reporter.user_id = r.creator_user_id
LEFT JOIN users handler ON handler.user_id = r.handled_by_user_id
LEFT JOIN users target_user ON target_user.user_id = (
    CASE
        WHEN r.target_type = 'user' THEN r.target_id
        WHEN r.target_type = 'group' THEN (SELECT owner_id FROM groups WHERE group_id = r.target_id)
        WHEN r.target_type = 'moment' THEN (SELECT user_id FROM moments WHERE moment_id = r.target_id)
        WHEN r.target_type = 'comment' THEN (SELECT user_id FROM comments WHERE comment_id = r.target_id)
        WHEN r.target_type = 'message' THEN (
            SELECT sender_user_id FROM group_messages WHERE message_id = r.target_id
            UNION ALL
            SELECT sender_user_id FROM private_messages WHERE message_id = r.target_id
            LIMIT 1
        )
        ELSE NULL
    END
)
WHERE r.report_id = $1`
	// pending 守卫 + RETURNING：两管理员并发处理同一举报时只有一个能命中，
	// 防止双重解散/封禁等副作用；0 行返回由 repo 映射为「已处理」冲突。
	queryAdminHandleReport = `
UPDATE reports
SET status = $2,
    handled_at = $3,
    handled_by_user_id = $4,
    handle_note = $5
WHERE report_id = $1 AND status = 'pending'
RETURNING report_id`
	// === 统计分析（stats/geo）===
	// 口径（spec api_spec_admin.md「统计分析（stats）API」）：平台管理视角豁免
	// 客户端隐私过滤——用户未注销、群组未解散、动态未删除即计入，不限可见性。

	// 全站存量总数：三条标量 COUNT 一次往返取回；不受时间窗影响。
	queryAdminStatsTotals = `
SELECT
  (SELECT COUNT(*) FROM users WHERE is_deleted = FALSE),
  (SELECT COUNT(*) FROM groups WHERE deleted_at IS NULL),
  (SELECT COUNT(*) FROM moments WHERE is_deleted = FALSE)`

	// 区域排名：bucket = s2_cell_id >> $2（$2 = geo.CellLevelShift(level)）把叶级
	// cell 归到 level 层祖先桶（算术右移对 face 4/5 的负 int64 同样成立）；
	// 三实体 UNION ALL 后按桶合并 total + 三分类 FILTER 拆分。
	// $1 = limit、$3/$4 = 可空时间窗（各表 created_at）；s2_cell_id = 0（无效
	// 位置）不参与地理统计。tie-breaker 取 bucket 升序保证确定性。
	queryAdminStatsRegions = `
SELECT t.bucket,
       SUM(t.n)::bigint AS total,
       COALESCE(SUM(t.n) FILTER (WHERE t.kind = 'user'), 0)::bigint AS user_n,
       COALESCE(SUM(t.n) FILTER (WHERE t.kind = 'group'), 0)::bigint AS group_n,
       COALESCE(SUM(t.n) FILTER (WHERE t.kind = 'moment'), 0)::bigint AS moment_n
FROM (
  SELECT s2_cell_id >> $2 AS bucket, COUNT(*) AS n, 'user' AS kind
  FROM users
  WHERE is_deleted = FALSE AND s2_cell_id <> 0
    AND ($3::timestamptz IS NULL OR created_at >= $3)
    AND ($4::timestamptz IS NULL OR created_at <= $4)
  GROUP BY 1
  UNION ALL
  SELECT s2_cell_id >> $2, COUNT(*), 'group'
  FROM groups
  WHERE deleted_at IS NULL AND s2_cell_id <> 0
    AND ($3::timestamptz IS NULL OR created_at >= $3)
    AND ($4::timestamptz IS NULL OR created_at <= $4)
  GROUP BY 1
  UNION ALL
  SELECT s2_cell_id >> $2, COUNT(*), 'moment'
  FROM moments
  WHERE is_deleted = FALSE AND s2_cell_id <> 0
    AND ($3::timestamptz IS NULL OR created_at >= $3)
    AND ($4::timestamptz IS NULL OR created_at <= $4)
  GROUP BY 1
) t
GROUP BY t.bucket
ORDER BY total DESC, t.bucket ASC
LIMIT $1`

	// 热门地点排名：location_name 精确文本聚合（同名合并，自由文本不做行政区
	// 归一化）。口径与区域排名一致；空地名不计入地点维度（坐标仍参与区域）。
	queryAdminStatsPlaces = `
SELECT t.location_name,
       SUM(t.n)::bigint AS total,
       COALESCE(SUM(t.n) FILTER (WHERE t.kind = 'user'), 0)::bigint AS user_n,
       COALESCE(SUM(t.n) FILTER (WHERE t.kind = 'group'), 0)::bigint AS group_n,
       COALESCE(SUM(t.n) FILTER (WHERE t.kind = 'moment'), 0)::bigint AS moment_n
FROM (
  SELECT location_name, COUNT(*) AS n, 'user' AS kind
  FROM users
  WHERE is_deleted = FALSE AND location_name <> ''
    AND ($2::timestamptz IS NULL OR created_at >= $2)
    AND ($3::timestamptz IS NULL OR created_at <= $3)
  GROUP BY 1
  UNION ALL
  SELECT location_name, COUNT(*), 'group'
  FROM groups
  WHERE deleted_at IS NULL AND location_name <> ''
    AND ($2::timestamptz IS NULL OR created_at >= $2)
    AND ($3::timestamptz IS NULL OR created_at <= $3)
  GROUP BY 1
  UNION ALL
  SELECT location_name, COUNT(*), 'moment'
  FROM moments
  WHERE is_deleted = FALSE AND location_name <> ''
    AND ($2::timestamptz IS NULL OR created_at >= $2)
    AND ($3::timestamptz IS NULL OR created_at <= $3)
  GROUP BY 1
) t
GROUP BY t.location_name
ORDER BY total DESC, t.location_name ASC
LIMIT $1`

	// 每日新增分布（trend_days）：created_at 按 UTC+8 日界分桶。用 + INTERVAL
	// '8 hours' 而非 AT TIME ZONE '+08:00'——POSIX 时区串符号反转（实测 +8
	// 得 UTC-8），区间加法无歧义；中国无夏令时，固定偏移即日历日。
	// 窗口 [start, end) 由 Service 按 +08:00 日界换算传入。
	// $1 = 窗口起点、$2 = 窗口终点（均 timestamptz）。
	queryAdminStatsTrends = `
SELECT t.day,
       COALESCE(SUM(t.n) FILTER (WHERE t.kind = 'user'), 0)::bigint AS user_n,
       COALESCE(SUM(t.n) FILTER (WHERE t.kind = 'group'), 0)::bigint AS group_n,
       COALESCE(SUM(t.n) FILTER (WHERE t.kind = 'moment'), 0)::bigint AS moment_n
FROM (
  SELECT ((created_at + INTERVAL '8 hours')::date) AS day, COUNT(*) AS n, 'user' AS kind
  FROM users
  WHERE is_deleted = FALSE AND created_at >= $1 AND created_at < $2
  GROUP BY 1
  UNION ALL
  SELECT ((created_at + INTERVAL '8 hours')::date), COUNT(*), 'group'
  FROM groups
  WHERE deleted_at IS NULL AND created_at >= $1 AND created_at < $2
  GROUP BY 1
  UNION ALL
  SELECT ((created_at + INTERVAL '8 hours')::date), COUNT(*), 'moment'
  FROM moments
  WHERE is_deleted = FALSE AND created_at >= $1 AND created_at < $2
  GROUP BY 1
) t
GROUP BY t.day
ORDER BY t.day`

	// 区域下钻明细（stats/geo/drilldown）：单网格桶内实体明细，created_at DESC +
	// id DESC keyset。$1 = bucket、$2 = shift、$3/$4 = 可空时间窗、
	// $5/$6 = 可空游标（行值比较；$5 IS NULL 短路守首页，避免 NULL 比较全排除）、
	// $7 = limit。实体类型为白名单三选一，分别对应三条独立常量。
	queryAdminStatsDrilldownUsers = `
SELECT u.user_id, u.nickname, NULLIF(u.location_name, ''), u.latitude, u.longitude, u.created_at
FROM users u
WHERE u.is_deleted = FALSE AND u.s2_cell_id <> 0 AND (u.s2_cell_id >> $2) = $1
  AND ($3::timestamptz IS NULL OR u.created_at >= $3)
  AND ($4::timestamptz IS NULL OR u.created_at <= $4)
  AND ($5::timestamptz IS NULL OR (u.created_at, u.user_id) < ($5, $6))
ORDER BY u.created_at DESC, u.user_id DESC
LIMIT $7`

	queryAdminStatsDrilldownGroups = `
SELECT g.group_id, g.name, NULLIF(g.location_name, ''), g.latitude, g.longitude, g.created_at
FROM groups g
WHERE g.deleted_at IS NULL AND g.s2_cell_id <> 0 AND (g.s2_cell_id >> $2) = $1
  AND ($3::timestamptz IS NULL OR g.created_at >= $3)
  AND ($4::timestamptz IS NULL OR g.created_at <= $4)
  AND ($5::timestamptz IS NULL OR (g.created_at, g.group_id) < ($5, $6))
ORDER BY g.created_at DESC, g.group_id DESC
LIMIT $7`

	queryAdminStatsDrilldownMoments = `
SELECT m.moment_id, m.description, NULLIF(m.location_name, ''), m.latitude, m.longitude, m.created_at
FROM moments m
WHERE m.is_deleted = FALSE AND m.s2_cell_id <> 0 AND (m.s2_cell_id >> $2) = $1
  AND ($3::timestamptz IS NULL OR m.created_at >= $3)
  AND ($4::timestamptz IS NULL OR m.created_at <= $4)
  AND ($5::timestamptz IS NULL OR (m.created_at, m.moment_id) < ($5, $6))
ORDER BY m.created_at DESC, m.moment_id DESC
LIMIT $7`
)
