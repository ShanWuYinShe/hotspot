package admin

// report_ws_payloads.go — 举报裁决结果的 WebSocket 出站负载
// （spec api_spec_websocket.md → ReportResolvedPayload / ReportDismissedPayload）。
//
// 同步路径（admin/service_reports.go 事务提交后）与 outbox 补投路径
// （bootstrap/app.go 消费 adminReportHandledOutboxHandler）此前各自用
// map[string]any 条件写键：note 为空时整键缺失，而 spec 定义为恒定
// nullable（string | null）。两处实现还容易再次漂移，故收敛为唯一构造器。

// ReportResolvedWSPayload 举报确认有效（resolved）。
type ReportResolvedWSPayload struct {
	NotifyID   string  `json:"notify_id"`
	ReportID   string  `json:"report_id"`
	TargetType string  `json:"target_type"`
	TargetID   string  `json:"target_id"`
	ResolvedAt string  `json:"resolved_at"`
	Note       *string `json:"note"`
}

// ReportDismissedWSPayload 举报被驳回（dismissed）。
type ReportDismissedWSPayload struct {
	NotifyID    string  `json:"notify_id"`
	ReportID    string  `json:"report_id"`
	TargetType  string  `json:"target_type"`
	TargetID    string  `json:"target_id"`
	DismissedAt string  `json:"dismissed_at"`
	Note        *string `json:"note"`
}

// BuildReportHandledWSPayload 依 resolved 选择事件形状；note 空串归一为 null，
// 保证「无处理说明」在两种事件里都是显式 null（而非缺键）。
func BuildReportHandledWSPayload(notifyID, reportID, targetType, targetID, note string, resolved bool, occurredAt string) any {
	var notePtr *string
	if note != "" {
		notePtr = &note
	}
	if resolved {
		return ReportResolvedWSPayload{
			NotifyID:   notifyID,
			ReportID:   reportID,
			TargetType: targetType,
			TargetID:   targetID,
			ResolvedAt: occurredAt,
			Note:       notePtr,
		}
	}
	return ReportDismissedWSPayload{
		NotifyID:    notifyID,
		ReportID:    reportID,
		TargetType:  targetType,
		TargetID:    targetID,
		DismissedAt: occurredAt,
		Note:        notePtr,
	}
}
