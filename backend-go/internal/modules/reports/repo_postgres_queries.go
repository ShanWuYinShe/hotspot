package reports

const (
	queryInsertReport = `
INSERT INTO reports (report_id, creator_user_id, target_type, target_id, reason, description, status, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

	queryListReports = `
SELECT report_id, target_type, target_id, reason, description, status, created_at
FROM reports
WHERE creator_user_id = $1
ORDER BY created_at DESC, report_id DESC`
)
