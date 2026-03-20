package groups

const (
	queryGetGroup = `
SELECT group_id, name, avatar_url, description, has_password, location_name, latitude, longitude, join_method, is_searchable, max_members
FROM groups
WHERE group_id = $1`

	querySearchGroups = `
SELECT group_id, name, avatar_url, description, has_password, location_name, latitude, longitude, join_method, is_searchable, max_members
FROM groups
WHERE ($1 = '' OR name = $1)
ORDER BY group_id ASC`
)
