package reports

const (
	queryListReports = `
SELECT report_id, target_type, target_id, reason, description, status, created_at
FROM reports
WHERE creator_user_id = $1
ORDER BY created_at DESC, report_id DESC`
)
