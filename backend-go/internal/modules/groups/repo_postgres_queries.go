package groups

const (
	queryInsertGroup = `
INSERT INTO groups (group_id, name, avatar_url, description, has_password, location_name, latitude, longitude, join_method, is_searchable, max_members)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

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
