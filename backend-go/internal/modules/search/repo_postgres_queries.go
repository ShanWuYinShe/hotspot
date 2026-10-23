package search

// maxSuggestRows bounds the result set of each suggest query. The suggest
// queries use leading-wildcard ILIKE (unanchored, non-indexable), so a short
// or common keyword would otherwise scan and return the whole table. This cap
// keeps autocomplete fast and memory bounded.
const maxSuggestRows = 20

const (
	querySearchSuggestTags = `
SELECT mt.tag_name, COUNT(*) AS usage_count
FROM moment_tags mt
JOIN moments m ON m.moment_id = mt.moment_id
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE
  AND mt.tag_name ILIKE '%' || $1 || '%'
GROUP BY mt.tag_name
ORDER BY usage_count DESC
LIMIT $2`

	// querySearchSuggestUsers：原 SELECT 内 6 个 per-row 标量子查询（审计 perf
	// P2-3）收敛为 4 个 LATERAL（tags / moments / 三个关注计数合一个 / groups）。
	// 内层先过滤 + LIMIT 派生表（原查询无 ORDER BY，LIMIT 行集本就任意，语义不变），
	// LATERAL 只对页内 ≤20 行各执行一次；3 个 user_relations 计数合并后单行只扫
	// 一次关系表。三个统计 LATERAL 内部按 u.stats_visible 短路（P2-3 收口）：
	// 匿名 suggest 上下文里统计仅在该开关为真时进入 payload，关闭时跳过
	// moments/relations/groups 三处聚合（CASE 短路标量子查询）。
	querySearchSuggestUsers = `
SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), COALESCE(u.bio, ''), u.location_name, u.latitude, u.longitude,
       u.presence_status, u.last_online,
       COALESCE(t.tags, '{}'),
       COALESCE(m.moments_count, 0),
       COALESCE(rel.mutual_follow_count, 0),
       COALESCE(rel.follower_count, 0),
       COALESCE(rel.following_count, 0),
       COALESCE(gc.groups_count, 0),
       u.location_visible, u.presence_visible, u.stats_visible
FROM (
    SELECT u.user_id, u.nickname, u.avatar_url, u.bio, u.location_name, u.latitude, u.longitude,
           u.presence_status, u.last_online,
           us.location_visible, us.presence_visible, us.stats_visible
    FROM users u
    JOIN user_settings us ON us.user_id = u.user_id
    WHERE us.searchable = TRUE
      AND u.is_deleted = FALSE
      AND us.privacy_level = 'public'
      AND (
        u.nickname ILIKE '%' || $1 || '%'
        OR (us.nickname_searchable = TRUE AND u.bio ILIKE '%' || $1 || '%')
      )
      -- S2 粗筛（审查 P2-10）：此前先 LIMIT 再在 Go 侧按 radius 过滤，小半径下
      -- 候选被无关行占满导致结果系统性偏少；且无 ORDER BY 时 LIMIT 行集任意。
      -- 覆盖区间为空数组表示不做地理过滤（radius 未启用/坐标非法）。
      -- S2 覆盖是半径圆的包围盒超集，精确距离过滤仍在 Go 层 → 无假阴性。
      AND (COALESCE(cardinality($3::bigint[]), 0) = 0
           OR EXISTS (SELECT 1 FROM unnest($3::bigint[], $4::bigint[]) AS t(lo, hi)
                      WHERE u.s2_cell_id BETWEEN t.lo AND t.hi))
    ORDER BY u.user_id
    LIMIT $2
) u
LEFT JOIN LATERAL (
    SELECT ARRAY_AGG(tag_name ORDER BY tag_name) AS tags
    FROM user_profile_tags pt
    WHERE pt.user_id = u.user_id
) t ON TRUE
LEFT JOIN LATERAL (
    SELECT CASE WHEN u.stats_visible THEN
        (SELECT COUNT(*) FROM moments m2
         WHERE m2.user_id = u.user_id AND m2.is_deleted = FALSE)
    ELSE 0 END AS moments_count
) m ON TRUE
LEFT JOIN LATERAL (
    SELECT
        COUNT(*) FILTER (WHERE ur.user_id = u.user_id AND ur.is_following = TRUE AND rev.user_id IS NOT NULL) AS mutual_follow_count,
        COUNT(*) FILTER (WHERE ur.target_user_id = u.user_id AND ur.is_following = TRUE) AS follower_count,
        COUNT(*) FILTER (WHERE ur.user_id = u.user_id AND ur.is_following = TRUE) AS following_count
    FROM user_relations ur
    LEFT JOIN user_relations rev
      ON rev.user_id = ur.target_user_id AND rev.target_user_id = ur.user_id AND rev.is_following = TRUE
    WHERE u.stats_visible AND (ur.user_id = u.user_id OR ur.target_user_id = u.user_id)
) rel ON TRUE
LEFT JOIN LATERAL (
    SELECT CASE WHEN u.stats_visible THEN
        (SELECT COUNT(*) FROM group_members gm3
         JOIN groups g3 ON g3.group_id = gm3.group_id
         WHERE gm3.user_id = u.user_id AND g3.deleted_at IS NULL)
    ELSE 0 END AS groups_count
) gc ON TRUE
LIMIT $2`

	// querySearchSuggestGroups：online 成员数由 SELECT 内标量子查询（审计 perf
	// P2-3）改为 LATERAL，置于 group_members join 之前保证每群只执行一次。
	querySearchSuggestGroups = `
SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.description, ''), g.location_name, g.latitude, g.longitude, COUNT(gm.user_id) AS members_count,
       g.join_method, g.requires_approval, g.tags, g.next_activity_at, g.is_searchable, g.max_members, g.created_at, g.owner_id,
       onc.online_members_count
FROM groups g
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS online_members_count
    FROM group_members om
    JOIN users ou ON ou.user_id = om.user_id AND ou.presence_status = 'online'
    WHERE om.group_id = g.group_id
) onc ON TRUE
LEFT JOIN group_members gm ON gm.group_id = g.group_id
WHERE g.deleted_at IS NULL
  AND g.is_searchable = TRUE
  AND g.name ILIKE '%' || $1 || '%'
  -- S2 粗筛 + 确定性排序，动机同 querySearchSuggestUsers（审查 P2-10）。
  AND (COALESCE(cardinality($3::bigint[]), 0) = 0
       OR EXISTS (SELECT 1 FROM unnest($3::bigint[], $4::bigint[]) AS t(lo, hi)
                  WHERE g.s2_cell_id BETWEEN t.lo AND t.hi))
GROUP BY g.group_id, g.name, g.avatar_url, g.description, g.location_name, g.latitude, g.longitude,
         g.join_method, g.requires_approval, g.tags, g.next_activity_at, g.is_searchable, g.max_members, g.created_at, g.owner_id,
         onc.online_members_count
ORDER BY g.group_id
LIMIT $2`

	// popularity 的点赞数取 reactions 实时聚合：likes_count 自点赞改为 reactions
	// 后已无写路径、恒为 0，继续使用会让点赞对搜索建议热度零贡献（审计 F-08）。
	// 点赞计数由 SELECT 内标量子查询（审计 perf P2-3）改为 LATERAL（ORDER BY
	// popularity 需要对全部候选行求值，执行次数与原相关子查询一致）。
	querySearchSuggestMoments = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), COALESCE(u.avatar_url, ''), m.description, m.location_name, m.latitude, m.longitude,
       COALESCE(cover.cover_url, ''),
       (lk.likes_count + m.comments_count + m.shares_count) AS popularity
FROM moments m
JOIN users u ON u.user_id = m.user_id
LEFT JOIN LATERAL (
    SELECT mi.file_id
    FROM moment_images mi
    WHERE mi.moment_id = m.moment_id
    ORDER BY mi.position
    LIMIT 1
) first_img ON TRUE
LEFT JOIN LATERAL (
    SELECT COALESCE(NULLIF(s.thumbnail_url, ''), s.storage_key) AS cover_url
    FROM media_upload_sessions s
    WHERE s.file_id = first_img.file_id AND s.status = 'completed'
) cover ON TRUE
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS likes_count
    FROM reactions r
    WHERE r.target_type = 'moment' AND r.target_id = m.moment_id
) lk ON TRUE
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE
  AND m.description ILIKE '%' || $1 || '%'
  -- S2 粗筛，动机同 querySearchSuggestUsers（审查 P2-10）；tie-breaker 保证
  -- popularity 相同时的行序确定（弱一致契约要求稳定 tie-breaker）。
  AND (COALESCE(cardinality($3::bigint[]), 0) = 0
       OR EXISTS (SELECT 1 FROM unnest($3::bigint[], $4::bigint[]) AS t(lo, hi)
                  WHERE m.s2_cell_id BETWEEN t.lo AND t.hi))
ORDER BY popularity DESC, m.moment_id
LIMIT $2`

	// querySuggestViewerRelations 批量取查看者对建议用户的关注/拉黑正向边与
	// 被关注反向边（spec api_spec_types.md Relationship：登录视角 relation 填充，
	// 无任何关系时为 null）。口径与 users 模块 RelationshipBatch 的两条边一致。
	querySuggestViewerRelations = `
SELECT target_user_id, is_following, is_blocked
FROM user_relations
WHERE user_id = $1 AND target_user_id = ANY($2)`

	// querySuggestViewerRelationsReverse 反向边（对方是否关注查看者）。
	querySuggestViewerRelationsReverse = `
SELECT user_id, is_following
FROM user_relations
WHERE target_user_id = $1 AND user_id = ANY($2)`
)
