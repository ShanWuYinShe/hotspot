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

	queryFilterAllowedMomentTags = `
SELECT candidate.tag_name
FROM unnest($1::text[]) AS candidate(tag_name)
LEFT JOIN admin_blocked_tags bt ON bt.name = candidate.tag_name
WHERE bt.name IS NULL
ORDER BY candidate.tag_name ASC`

	queryListMomentTagsByMomentID = `
SELECT tag_name
FROM moment_tags
WHERE moment_id = $1
ORDER BY tag_name ASC`

	queryListMomentTagsByMomentIDs = `
SELECT moment_id, tag_name
FROM moment_tags
WHERE moment_id = ANY($1)
ORDER BY moment_id, tag_name ASC`

	queryDeleteMomentTagsByMomentID = `
DELETE FROM moment_tags
WHERE moment_id = $1`

	queryInsertMomentMention = `
INSERT INTO moment_mentions (moment_id, mentioned_user_id, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (moment_id, mentioned_user_id) DO NOTHING`

	queryListMomentMentionsByMomentID = `
SELECT mentioned_user_id
FROM moment_mentions
WHERE moment_id = $1
ORDER BY mentioned_user_id ASC`

	queryListMomentMentionsByMomentIDs = `
SELECT moment_id, mentioned_user_id
FROM moment_mentions
WHERE moment_id = ANY($1)
ORDER BY moment_id, mentioned_user_id ASC`

	queryDeleteMomentMentionsByMomentID = `
DELETE FROM moment_mentions
WHERE moment_id = $1`

	queryIncrementMomentCommentCount = `
UPDATE moments
SET comments_count = comments_count + 1
	WHERE moment_id = $1`

	queryDecrementMomentCommentCount = `
UPDATE moments
SET comments_count = GREATEST(comments_count - 1, 0)
WHERE moment_id = $1`

	queryIncrementMomentShareCount = `
UPDATE moments
SET shares_count = shares_count + 1
WHERE moment_id = $1`

	queryUpdateMoment = `
UPDATE moments
SET description = COALESCE($2, description),
    privacy_level = COALESCE($3, privacy_level),
    location_name = COALESCE($4, location_name),
    latitude      = COALESCE($5, latitude),
    longitude     = COALESCE($6, longitude)
WHERE moment_id = $1`

	queryDeleteMoment = `
DELETE FROM moments WHERE moment_id = $1`

	querySearchMoments = `
SELECT moment_id, user_id, description, created_at, likes_count, comments_count, shares_count, location_name, latitude, longitude, privacy_level
FROM moments
WHERE privacy_level != 'private'
  AND ($1::text IS NULL OR description ILIKE '%' || $1 || '%')
  AND ($2::text IS NULL OR moment_id IN (SELECT moment_id FROM moment_tags WHERE tag_name = $2))
ORDER BY created_at DESC, moment_id DESC
LIMIT $3`

	querySearchMomentsAll = `
SELECT moment_id, user_id, description, created_at, likes_count, comments_count, shares_count, location_name, latitude, longitude, privacy_level
FROM moments
WHERE privacy_level != 'private'
  AND ($1::text IS NULL OR description ILIKE '%' || $1 || '%')
  AND ($2::text IS NULL OR moment_id IN (SELECT moment_id FROM moment_tags WHERE tag_name = $2))`

	queryInsertMomentLike = `
INSERT INTO moment_likes (moment_id, user_id, liked_at)
VALUES ($1, $2, $3)
ON CONFLICT (moment_id, user_id) DO NOTHING`

	queryDeleteMomentLike = `
DELETE FROM moment_likes
WHERE moment_id = $1 AND user_id = $2`

	queryGetMomentLike = `
SELECT 1
FROM moment_likes
WHERE moment_id = $1 AND user_id = $2`

	queryIncrementMomentLikeCount = `
UPDATE moments
SET likes_count = likes_count + 1
WHERE moment_id = $1`

	queryDecrementMomentLikeCount = `
UPDATE moments
SET likes_count = GREATEST(likes_count - 1, 0)
WHERE moment_id = $1`

	queryInsertMomentFavorite = `
INSERT INTO moment_favorites (user_id, moment_id, favorited_at)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, moment_id) DO NOTHING`

	queryDeleteMomentFavorite = `
DELETE FROM moment_favorites
WHERE user_id = $1 AND moment_id = $2`

	queryListMomentLikes = `
SELECT user_id, liked_at
FROM moment_likes
WHERE moment_id = $1
  AND (
    $2 IS NULL
    OR liked_at < $2
    OR (liked_at = $2 AND user_id > $3)
  )
ORDER BY liked_at DESC, user_id ASC
LIMIT $4`

	queryListFavoriteMoments = `
SELECT m.moment_id, m.user_id, m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level, mf.favorited_at
FROM moment_favorites mf
JOIN moments m ON m.moment_id = mf.moment_id
WHERE mf.user_id = $1
  AND (
    $2 IS NULL
    OR mf.favorited_at < $2
    OR (mf.favorited_at = $2 AND m.moment_id < $3)
  )
ORDER BY mf.favorited_at DESC, m.moment_id DESC
LIMIT $4`
)
