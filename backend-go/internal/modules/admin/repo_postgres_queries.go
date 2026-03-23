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

	queryAdminInsertRole = `
INSERT INTO admin_user_roles (user_id, role_id, created_at)
VALUES ($1,$2,$3)`

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

	queryAuditByID = `
SELECT log_id, action, operator_id, operator_name, COALESCE(operator_avatar_url, ''), target_type, target_id, COALESCE(target_name, ''), COALESCE(details_json, '{}'), COALESCE(ip_location, ''), COALESCE(ip_address, ''), COALESCE(device_summary, ''), created_at
FROM admin_audit_logs
WHERE log_id = $1`
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
       COALESCE(u.phone, '')
FROM users u
LEFT JOIN auth_accounts a ON a.user_id = u.user_id
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
       COALESCE(u.phone, '')
FROM users u
LEFT JOIN auth_accounts a ON a.user_id = u.user_id
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
       COALESCE(r.handle_note, '')
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
	queryAdminHandleReport = `
UPDATE reports
SET status = $2,
    handled_at = $3,
    handled_by_user_id = $4,
    handle_note = $5
WHERE report_id = $1`
)
