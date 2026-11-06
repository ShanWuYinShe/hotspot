package moments

const (
	queryGetUserNickname = `
SELECT COALESCE(nickname, '')
FROM users
WHERE user_id = $1`

	// queryGetMoment 一次往返取齐 moment + tags + mentions（审查 P2-16：此前
	// 详情路径 3 次往返——主查询 + loadMomentTags + loadMomentMentions）。
	// 两个 LATERAL 的 ARRAY_AGG 排序与共享 loader（tag_name ASC /
	// mentioned_user_id ASC）一致，避免详情与列表两条路径的 tag/mention 顺序
	// 出现口径差异。
	queryGetMoment = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, (SELECT COUNT(*)::int FROM comments c WHERE c.moment_id = m.moment_id AND c.is_deleted = FALSE), (SELECT COUNT(*)::int FROM shares s WHERE s.share_type = 'moment' AND s.is_revoked = FALSE AND s.payload->>'shared_moment_id' = m.moment_id), m.interaction_count, m.location_name, m.latitude, m.longitude, m.privacy_level, m.is_deleted,
       COALESCE(t.tags, '{}'),
       COALESCE(mn.mentions, '{}')
FROM moments m
LEFT JOIN users u ON u.user_id = m.user_id
LEFT JOIN LATERAL (
    SELECT ARRAY_AGG(mt.tag_name ORDER BY mt.tag_name) AS tags
    FROM moment_tags mt
    WHERE mt.moment_id = m.moment_id
) t ON TRUE
LEFT JOIN LATERAL (
    SELECT ARRAY_AGG(mm.mentioned_user_id ORDER BY mm.mentioned_user_id) AS mentions
    FROM moment_mentions mm
    WHERE mm.moment_id = m.moment_id
) mn ON TRUE
WHERE m.moment_id = $1`

	queryListMyMoments = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, (SELECT COUNT(*)::int FROM comments c WHERE c.moment_id = m.moment_id AND c.is_deleted = FALSE), (SELECT COUNT(*)::int FROM shares s WHERE s.share_type = 'moment' AND s.is_revoked = FALSE AND s.payload->>'shared_moment_id' = m.moment_id), m.interaction_count, m.location_name, m.latitude, m.longitude, m.privacy_level, m.is_deleted
FROM moments m
LEFT JOIN users u ON u.user_id = m.user_id
WHERE m.user_id = $1
  AND ($2::timestamptz IS NULL
       OR (m.created_at, m.moment_id) < ($2::timestamptz, $3))
ORDER BY m.created_at DESC, m.moment_id DESC
LIMIT $4`

	queryInsertMoment = `
INSERT INTO moments (moment_id, user_id, description, created_at, location_name, latitude, longitude, privacy_level, s2_cell_id, interaction_count)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,0)`

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

	queryMomentFavoritesByIDs = `
SELECT moment_id
FROM moment_favorites
WHERE user_id = $1 AND moment_id = ANY($2)`

	queryListMomentTagsByMomentIDs = `
SELECT moment_id, tag_name
FROM moment_tags
WHERE moment_id = ANY($1)
ORDER BY moment_id, tag_name ASC`

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

	queryDeleteMomentTagsByMomentID = `
DELETE FROM moment_tags
WHERE moment_id = $1`

	queryInsertMomentMention = `
INSERT INTO moment_mentions (moment_id, mentioned_user_id, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (moment_id, mentioned_user_id) DO NOTHING`
	// 仅作者且未软删的瞬间可编辑（spec api_spec_moment.md update：仅作者；
	// 删除语义——删除后其余场景按不存在处理 10300）。owner/软删条件在 SQL 内
	// 收口（service 层预检之外的纵深防御），RETURNING moment_id 供调用方以
	// ErrNoRows 判定未命中（检查-使用窗口内被并发软删/删除）。
	queryUpdateMoment = `
UPDATE moments
SET description = COALESCE($3, description),
    privacy_level = COALESCE($4, privacy_level),
    location_name = COALESCE($5, location_name),
    latitude      = COALESCE($6, latitude),
    longitude     = COALESCE($7, longitude),
    s2_cell_id    = COALESCE($8, s2_cell_id)
WHERE moment_id = $1 AND user_id = $2 AND is_deleted = FALSE
RETURNING moment_id`

	// queryDeleteMoment 带 RETURNING：0 行命中（不存在）→ repo 返回 ErrMomentNotFound，
	// 与内存实现一致（此前 Exec 丢弃行数，删除不存在瞬间静默成功）。
	queryDeleteMoment = `
	UPDATE moments
	SET is_deleted = TRUE, description = ''
	WHERE moment_id = $1
	RETURNING moment_id`

	queryCheckMomentFavorite = `
SELECT 1
FROM moment_favorites
WHERE user_id = $1 AND moment_id = $2`

	queryInsertMomentFavorite = `
INSERT INTO moment_favorites (user_id, moment_id, favorited_at)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, moment_id) DO NOTHING`

	queryDeleteMomentFavorite = `
DELETE FROM moment_favorites
WHERE user_id = $1 AND moment_id = $2`

	// queryListMomentFavoriteUserIDs 带 LIMIT：调用方传 fanout 上限（NULL = 不限制）。
	queryListMomentFavoriteUserIDs = `
	SELECT user_id
	FROM moment_favorites
	WHERE moment_id = $1
	LIMIT $2`

	queryListFavoriteMoments = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, (SELECT COUNT(*)::int FROM comments c WHERE c.moment_id = m.moment_id AND c.is_deleted = FALSE), (SELECT COUNT(*)::int FROM shares s WHERE s.share_type = 'moment' AND s.is_revoked = FALSE AND s.payload->>'shared_moment_id' = m.moment_id), m.interaction_count, m.location_name, m.latitude, m.longitude, m.privacy_level, m.is_deleted, mf.favorited_at
FROM moment_favorites mf
JOIN moments m ON m.moment_id = mf.moment_id
LEFT JOIN users u ON u.user_id = m.user_id
WHERE mf.user_id = $1
  AND (
    $2::timestamptz IS NULL
    OR mf.favorited_at < $2
    OR (mf.favorited_at = $2 AND m.moment_id < $3)
  )
ORDER BY mf.favorited_at DESC, m.moment_id DESC
LIMIT $4`

	// queryFollowingMoments 关注动态（spec moment.md → moments/fetch_following，
	// 原 feed/following 迁移）：keyset 谓词与 LIMIT 直接下推 SQL。
	queryFollowingMoments = `
SELECT m.moment_id,
       m.user_id,
       COALESCE(u.nickname, ''),
       COALESCE(u.avatar_url, ''),
       m.description,
       m.created_at,
       (SELECT COUNT(*)::int FROM comments c WHERE c.moment_id = m.moment_id AND c.is_deleted = FALSE), (SELECT COUNT(*)::int FROM shares s WHERE s.share_type = 'moment' AND s.is_revoked = FALSE AND s.payload->>'shared_moment_id' = m.moment_id), m.interaction_count,
       m.location_name,
       m.latitude,
       m.longitude,
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $1),
       COALESCE((SELECT array_agg(mt.tag_name ORDER BY mt.tag_name)
                 FROM moment_tags mt WHERE mt.moment_id = m.moment_id), ARRAY[]::text[])
FROM moments m
JOIN users u ON u.user_id = m.user_id
JOIN user_relations ur ON ur.target_user_id = m.user_id AND ur.user_id = $1 AND ur.is_following = TRUE
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE
  -- 双向拉黑过滤（对齐私聊/评论口径）：任何一方拉黑即互不可见
  AND NOT EXISTS(SELECT 1 FROM user_relations blk WHERE blk.user_id = $1 AND blk.target_user_id = m.user_id AND blk.is_blocked = TRUE)
  AND NOT EXISTS(SELECT 1 FROM user_relations blk WHERE blk.user_id = m.user_id AND blk.target_user_id = $1 AND blk.is_blocked = TRUE)
  AND ($2::timestamptz IS NULL OR m.created_at < $2::timestamptz OR (m.created_at = $2::timestamptz AND m.moment_id < $3))
ORDER BY m.created_at DESC, m.moment_id DESC
LIMIT $4`

	// queryNearbyMoments 附近瞬间（spec moment.md → moments/fetch_nearby，
	// 原 feed/nearby_moments 迁移）：S2 覆盖区间粗筛，精筛在 Go 层（geo.SearchPlan）。
	// 终序由 Go 层 compareNearbyCandidates（全序，momentID 兜底）确定，SQL 行序
	// 不影响排序结果；ORDER BY + LIMIT 仅作为确定性防御截断——区间行数无界时
	// 避免 DoS/内存无界（截断表现为"没有更多"，分页不错乱；正常规模不触达）。
	// queryNearbyMomentDetails 一次查询补全附近流页面内的展示字段、标签数组与收藏态
	// （20 行以内的 LATERAL/EXISTS 成本可忽略，比三次往返更省 context/网络开销）。
	queryNearbyMomentDetails = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), COALESCE(u.avatar_url, ''), m.description, (SELECT COUNT(*)::int FROM comments c WHERE c.moment_id = m.moment_id AND c.is_deleted = FALSE), (SELECT COUNT(*)::int FROM shares s WHERE s.share_type = 'moment' AND s.is_revoked = FALSE AND s.payload->>'shared_moment_id' = m.moment_id), m.interaction_count, m.location_name,
       COALESCE((SELECT array_agg(mt.tag_name ORDER BY mt.tag_name) FROM moment_tags mt WHERE mt.moment_id = m.moment_id), ARRAY[]::text[]),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $2),
       -- 图片行随详情查询一并取回（LATERAL 仅作用于 20 行页内），服务层只做
       -- presign，不再多发一次图片批量查询（架构审计 perf D2）。
       COALESCE(img.rows, '[]'::jsonb)
FROM moments m
JOIN users u ON u.user_id = m.user_id
LEFT JOIN LATERAL (
  SELECT jsonb_agg(jsonb_build_object(
           'file_id', mus.file_id,
           'storage_key', mus.storage_key,
           'thumbnail_url', COALESCE(mus.thumbnail_url, ''),
           'width', COALESCE(mus.width, 0),
           'height', COALESCE(mus.height, 0),
           'mime_type', COALESCE(mus.final_mime_type, '')
         ) ORDER BY mi.position) AS rows
  FROM moment_images mi
  JOIN media_upload_sessions mus ON mus.file_id = mi.file_id AND mus.status = 'completed'
  WHERE mi.moment_id = m.moment_id
) img ON TRUE
WHERE m.moment_id = ANY($1)`
	// queryNearbyMoments 候选粗筛只返回排序所需列（无逐行子查询、无 users join）：
	// 热度/关注改为候选集批量聚合，展示字段分页后批量补全。
	queryNearbyMoments = `
-- blocked：查看者双向拉黑集合一次物化（通常 0–几十行）。此前两个相关
-- NOT EXISTS 让优化器对每个候选行各扫一遍 user_relations（478 候选 × 2 次
-- 全表扫描，压测下 nearby p99 明显劣化），改为 hash/anti-join 一次构建。
WITH blocked AS (
  SELECT target_user_id AS user_id
  FROM user_relations
  WHERE user_id = $3 AND is_blocked = TRUE
  UNION
  SELECT user_id
  FROM user_relations
  WHERE target_user_id = $3 AND is_blocked = TRUE
)
SELECT m.moment_id,
       m.user_id,
       m.created_at,
       m.interaction_count,
       m.latitude,
       m.longitude,
       -- 关注加分输入随候选粗筛一并返回：查看者关注集合通常很小，PG 会把它
       -- 规划成 hash 子计划（一次构建 + 每候选行 hash 探测），替代原「候选查询
       -- 后再发一次批量关注查询」的额外 DB 往返（架构审计 perf D2）。
       ($3 <> '' AND m.user_id IN (
         SELECT target_user_id FROM user_relations
         WHERE user_id = $3 AND is_following = TRUE
       ))
FROM moments m
WHERE m.privacy_level = 'public'
  AND m.is_deleted = FALSE
  -- 双向拉黑过滤（对齐私聊/评论口径）：任何一方拉黑即互不可见
  AND m.user_id NOT IN (SELECT user_id FROM blocked)
  AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi)
              WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)
ORDER BY m.created_at DESC
LIMIT 2000`
)

// momentCountSelectColumns 是 moment 行集的统一计数片段：契约字段 comments_count /
// shares_count 读时聚合（评论未删 / 分享未撤销），interaction_count 为内部排序热度
// （reactions + 未删评论 + 未撤销分享，由写路径增量维护）。动态 SQL（搜索）拼接使用。
const momentCountSelectColumns = `(SELECT COUNT(*)::int FROM comments c WHERE c.moment_id = m.moment_id AND c.is_deleted = FALSE), (SELECT COUNT(*)::int FROM shares s WHERE s.share_type = 'moment' AND s.is_revoked = FALSE AND s.payload->>'shared_moment_id' = m.moment_id), m.interaction_count`

// queryAdjustMomentInteractionCount 维护内部互动热度列（仅供排序，弱一致）。
const queryAdjustMomentInteractionCount = `
UPDATE moments
SET interaction_count = GREATEST(interaction_count + $1, 0)
WHERE moment_id = $2`

// ── users 模块「按作者聚合的瞬间列表」读能力（原实现位于 users repo，直查
// moments/reactions 表；现已归位本模块，users 经 Service 接口消费）──
const (
	// 行内计数统一由 momentCountSelectColumns 读时聚合（comments/shares），
	// interaction_count 为内部热度列（排序用途，不对外）。
	queryListMomentsByAuthor = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, (SELECT COUNT(*)::int FROM comments c WHERE c.moment_id = m.moment_id AND c.is_deleted = FALSE), (SELECT COUNT(*)::int FROM shares s WHERE s.share_type = 'moment' AND s.is_revoked = FALSE AND s.payload->>'shared_moment_id' = m.moment_id), m.interaction_count, m.location_name, m.latitude, m.longitude, m.privacy_level
FROM moments m
LEFT JOIN users u ON u.user_id = m.user_id
WHERE m.user_id = $1
  AND m.is_deleted = FALSE
  AND ($2 OR m.privacy_level <> 'private')
  AND (
    $3::timestamptz IS NULL
    OR m.created_at < $3::timestamptz
    OR (m.created_at = $3::timestamptz AND m.moment_id < $4)
  )
ORDER BY m.created_at DESC, m.moment_id DESC
LIMIT $5`

	// 尾列 EXISTS(moment_favorites) 承载查看者 is_favorited（spec
	// api_spec_types.md MomentItem：当前用户是否收藏——与 reaction 无关，
	// 此前硬编码 true 属语义错位）；匿名 viewer（$2=''）EXISTS 恒 false，
	// 与 buildSearchMomentsQuery 同模式。reacted_at 为 upsert 切换 emoji 时
	// 会刷新的「最近反应时刻」（spec：last_reacted_at 排序），列以 time.Time
	// 全精度写入——游标必须以 RFC3339Nano 编码（AGENTS 分页纪律）。
	queryListReactedByUser = `
SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, (SELECT COUNT(*)::int FROM comments c WHERE c.moment_id = m.moment_id AND c.is_deleted = FALSE), (SELECT COUNT(*)::int FROM shares s WHERE s.share_type = 'moment' AND s.is_revoked = FALSE AND s.payload->>'shared_moment_id' = m.moment_id), m.interaction_count, m.location_name, m.latitude, m.longitude, m.privacy_level, r.reacted_at,
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $2)
FROM reactions r
JOIN moments m ON m.moment_id = r.target_id
LEFT JOIN users u ON u.user_id = m.user_id
WHERE r.target_type = 'moment' AND r.user_id = $1
  AND m.is_deleted = FALSE
  AND ($2 = '' OR m.user_id = $2 OR m.privacy_level <> 'private')
  AND (
    $3::timestamptz IS NULL
    OR r.reacted_at < $3::timestamptz
    OR (r.reacted_at = $3::timestamptz AND m.moment_id < $4)
  )
ORDER BY r.reacted_at DESC, m.moment_id DESC
LIMIT $5`
)

// 管理端标签管理（moment_tags 归属本模块；admin 经 Service 消费，P0-4）。
const (
	queryMomentTagAssignmentsExist = `
SELECT EXISTS(SELECT 1 FROM moment_tags WHERE tag_name = $1)`
	queryDeleteMomentTagAssignments = `
DELETE FROM moment_tags
WHERE tag_name = $1`
)
