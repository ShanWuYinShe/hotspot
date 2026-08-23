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
