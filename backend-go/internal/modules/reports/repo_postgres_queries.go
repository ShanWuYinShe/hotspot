package reports

const (
	queryInsertReport = `
INSERT INTO reports (report_id, creator_user_id, target_type, target_id, reason, description, status, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

	queryListReports = `
SELECT report_id, target_type, target_id, reason, description, status, created_at
FROM reports
WHERE creator_user_id = $1
  AND ($2 = '' OR status = $2)
  AND ($3 = '' OR target_type = $3)
ORDER BY created_at DESC, report_id DESC`

	queryFindExistingReport = `
SELECT report_id
FROM reports
WHERE target_type = $1 AND target_id = $2 AND status = 'pending'
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
