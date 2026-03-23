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
)
