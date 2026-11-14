package reports

const (
	queryInsertReport = `
INSERT INTO reports (report_id, creator_user_id, target_type, target_id, reason, description, status, created_at, message_snapshot)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)`

	// 举报列表 keyset 分页（spec api_spec_report.md /reports/fetch：created_at
	// DESC，时间相同以 report_id DESC 稳定 tie-breaker；分页契约见
	// api_spec_base.md「分页机制」——cursor 游标、禁止 offset）。cursor 为
	// 不透明 token（platform/cursor 加密编码），($4,$5) 为上一页末行元组。
	queryListReports = `
SELECT report_id, target_type, target_id, reason, description, status, created_at, message_snapshot::text,
       handled_at, COALESCE(handle_note, '')
FROM reports
WHERE creator_user_id = $1
  AND ($2 = '' OR status = $2)
  AND ($3 = '' OR target_type = $3)
  AND ($4::timestamptz IS NULL OR (created_at, report_id) < ($4::timestamptz, $5))
ORDER BY created_at DESC, report_id DESC
LIMIT $6`

	// 去重必须限定举报人（creator_user_id）：此前只按 target 匹配，任何人对
	// 目标有一条 pending 举报就会把其他所有人的举报判为 duplicate，并把
	// 别人的 report_id 泄漏给举报人。
	queryFindExistingReport = `
SELECT report_id
FROM reports
WHERE creator_user_id = $3 AND target_type = $1 AND target_id = $2 AND status = 'pending'
LIMIT 1`

	queryReportTargetOwnerUser = `
SELECT user_id FROM users WHERE user_id = $1`

	queryReportTargetOwnerMoment = `
SELECT user_id FROM moments WHERE moment_id = $1`

	queryReportTargetOwnerComment = `
SELECT user_id FROM comments WHERE comment_id = $1`

	queryReportTargetOwnerGroup = `
SELECT owner_id FROM groups WHERE group_id = $1 AND deleted_at IS NULL`

	queryReportTargetOwnerMessage = `
SELECT sender_user_id FROM group_messages WHERE message_id = $1
UNION ALL
SELECT sender_user_id FROM private_messages WHERE message_id = $1
LIMIT 1`
)

// 管理端裁决/保留期清理（reports 表归属本模块；admin 经 Service 消费，P0-4）。
const (
	queryMarkReportHandledAsAdmin = `
UPDATE reports
SET status = $2,
    handled_at = $3,
    handled_by_user_id = $4,
    handle_note = $5
WHERE report_id = $1 AND status = 'pending'
RETURNING report_id`

	// queryDeleteHandledReportsBefore 保留期清理（单批）：借 idx_reports_handled_at
	// 部分索引（handled_at IS NOT NULL，迁移 070）按 handled_at 顺序取最早的一批
	//（$2 行），再按 ctid 定位删除，避免无界 DELETE 一次锁住大批行。只删已处理
	// 终态，pending 永不清理；CTE + RETURNING 统计删除行数。
	queryDeleteHandledReportsBefore = `
WITH deleted AS (
    DELETE FROM reports
    WHERE ctid IN (
        SELECT ctid FROM reports
        WHERE handled_at IS NOT NULL
          AND handled_at < $1
          AND status IN ('resolved', 'dismissed')
        ORDER BY handled_at
        LIMIT $2
    )
    RETURNING 1
)
SELECT COUNT(*) FROM deleted`
)
