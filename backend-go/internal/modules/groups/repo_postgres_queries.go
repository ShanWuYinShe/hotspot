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

	queryUpdateGroup = `
UPDATE groups
SET name = COALESCE($2, name),
    avatar_url = COALESCE($3, avatar_url),
    description = COALESCE($4, description),
    has_password = COALESCE($5, has_password),
    join_method = COALESCE($6, join_method),
    is_searchable = COALESCE($7, is_searchable),
    max_members = COALESCE($8, max_members),
    location_name = COALESCE($9, location_name),
    latitude = COALESCE($10, latitude),
    longitude = COALESCE($11, longitude)
WHERE group_id = $1`

	queryDeleteGroup = `
DELETE FROM groups
WHERE group_id = $1`
)
