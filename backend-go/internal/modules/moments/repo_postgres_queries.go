package moments

const (
	queryGetMoment = `
SELECT moment_id, user_id, description, created_at, likes_count, comments_count, shares_count, location_name, latitude, longitude, privacy_level
FROM moments
WHERE moment_id = $1`

	queryListMyMoments = `
SELECT moment_id, user_id, description, created_at, likes_count, comments_count, shares_count, location_name, latitude, longitude, privacy_level
FROM moments
WHERE user_id = $1
ORDER BY created_at DESC, moment_id DESC`

	queryInsertMoment = `
INSERT INTO moments (moment_id, user_id, description, created_at, likes_count, comments_count, shares_count, location_name, latitude, longitude, privacy_level)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

	queryInsertMomentTagAssignment = `
INSERT INTO moment_tags (moment_id, tag_name, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (moment_id, tag_name) DO NOTHING`

	queryListMomentTagsByMomentID = `
SELECT tag_name
FROM moment_tags
WHERE moment_id = $1
ORDER BY tag_name ASC`

	queryIncrementMomentCommentCount = `
UPDATE moments
SET comments_count = comments_count + 1
WHERE moment_id = $1`
)
