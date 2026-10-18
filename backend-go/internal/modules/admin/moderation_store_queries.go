package admin

const (
	// querySearchTags 返回标签管理列表：**包含已屏蔽标签**并带 is_blocked 标记
	// （spec api_spec_admin.md tags/search「屏蔽标签的可见性」）——若排除已屏蔽标签，
	// 管理端将无法再选中它解屏蔽，屏蔽会变成不可逆操作。
	querySearchTags = `
SELECT mt.tag_name,
       COUNT(*) AS usage_count,
       MIN(mt.created_at) AS created_at,
       (bt.name IS NOT NULL) AS is_blocked
FROM moment_tags mt
LEFT JOIN admin_blocked_tags bt ON bt.name = mt.tag_name
WHERE ($1 = '' OR mt.tag_name ILIKE '%' || $1 || '%')
GROUP BY mt.tag_name, bt.name
HAVING ($2::int IS NULL OR COUNT(*) < $2 OR (COUNT(*) = $2 AND mt.tag_name > $3))
ORDER BY usage_count DESC, mt.tag_name ASC
LIMIT $4`

	queryBlockTag = `
INSERT INTO admin_blocked_tags (name, blocked_at)
VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE
SET blocked_at = EXCLUDED.blocked_at`

	// queryUnblockTag 解屏蔽：DELETE 未命中行（未屏蔽标签）为 no-op 成功。
	queryUnblockTag = `
DELETE FROM admin_blocked_tags
WHERE name = $1`

	queryDeleteMomentTagAssignments = `
DELETE FROM moment_tags
WHERE tag_name = $1`

	queryTagDeleteState = `
SELECT EXISTS(SELECT 1 FROM admin_blocked_tags WHERE name = $1) AS is_blocked,
       EXISTS(SELECT 1 FROM moment_tags WHERE tag_name = $1) AS has_moment_refs`
)
