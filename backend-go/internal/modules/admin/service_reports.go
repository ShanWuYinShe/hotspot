package admin

import (
	"context"
	"errors"
	usersmodule "hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/aftercommit"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/outbox"
	"hotspot/backend-go/internal/platform/realtime"
	"strings"
	"time"
)

func (s *Service) SearchReports(ctx context.Context, req AdminReportListRequest) (PaginatedAdminReports, *httpx.APIError, error) {
	if _, apiErr, err := s.requirePermission(ctx, permissionReportsRead); apiErr != nil || err != nil {
		return PaginatedAdminReports{}, apiErr, err
	}
	if req.Pagination != nil {
		if _, apiErr := decodeAdminReportCursor(req.Pagination.Cursor); apiErr != nil {
			return PaginatedAdminReports{}, apiErr, nil
		}
	}
	page, err := s.repo.SearchReports(ctx, req)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return PaginatedAdminReports{}, apiErr, nil
		}
		return PaginatedAdminReports{}, nil, err
	}
	resp := PaginatedAdminReports{Items: page.Items}
	// 「待处理」角标等计数场景：status 过滤时透出该状态总数（AdminReportsPage.total）
	if req.Status != "" {
		total := int64(page.Total)
		resp.Total = &total
	}
	if page.Next != nil {
		c := encodeAdminReportCursor(*page.Next)
		resp.NextCursor = &c
	}
	return resp, nil, nil
}

// HandleReport 是举报裁决入口：跨模块编排写（内容删除/封禁/举报终态）与裁决
// 审计在同一事务内提交（审计写失败整体回滚）；举报人通知的 WS 广播经 outbox
// 在 COMMIT 后补投，未装配 outbox 时的同步通知回退经 aftercommit 在 COMMIT 后
// 执行，回滚时丢弃。未注入 TxRunner（单测/内存装配）时直跑，无回滚语义。
func (s *Service) HandleReport(ctx context.Context, req JudgeAdminReportRequest) (*httpx.APIError, error) {
	return s.runDecisionInTx(ctx, func(txCtx context.Context) (*httpx.APIError, error) {
		return s.handleReport(txCtx, req)
	})
}

func (s *Service) handleReport(ctx context.Context, req JudgeAdminReportRequest) (*httpx.APIError, error) {
	actor, apiErr, err := s.requirePermission(ctx, permissionReportsWrite)
	if apiErr != nil || err != nil {
		return apiErr, err
	}
	note := strings.TrimSpace(req.Note)
	if strings.TrimSpace(req.ReportID) == "" || !isValidReportAction(req.Action) || len(note) > 500 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if req.BanDurationDays != nil && *req.BanDurationDays < 0 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	record, found, err := s.repo.GetReportHandleRecord(ctx, req.ReportID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "举报不存在"}, nil
	}
	if record.Item.Status != ReportStatusPending {
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "举报已处理"}, nil
	}
	// 权限蕴含（spec admin_moderation.md → reports/judge）：resolve 且携带
	// ban_duration_days 时实际执行封禁，须同时具备 admin.users.write，
	// 防止低权限管理员经举报通道绕过用户封禁权限边界。
	if req.Action == ReportActionResolve && req.BanDurationDays != nil && !actor.HasPermission(permissionUsersWrite) {
		return httpx.Forbidden(httpx.ReasonInsufficientAdminPermission, "平台管理员权限不足: "+permissionUsersWrite), nil
	}
	if req.Action == ReportActionResolve {
		switch record.Item.TargetType {
		case ReportTargetTypeGroup:
			// 与 groups/dismiss 端点共用同一解散链路（成员 GroupDismissed 广播 +
			// group.dismiss 审计），避免经举报通道解散时群成员无感知、操作不可追溯。
			if _, apiErr, err := s.dismissGroupAndTrace(ctx, actor, record.Item.TargetID, note); apiErr != nil || err != nil {
				return apiErr, err
			}
		case ReportTargetTypeUser:
			if apiErr, err := s.banReportedUser(ctx, actor, record.TargetUser.UserID, note, req.BanDurationDays); apiErr != nil || err != nil {
				return apiErr, err
			}
		case ReportTargetTypeMoment:
			if s.momentModerator == nil {
				return nil, errors.New("admin: moment moderator not configured")
			}
			if err := s.momentModerator.DeleteAsAdmin(ctx, record.Item.TargetID); err != nil {
				return nil, err
			}
			if apiErr, err := s.banReportedUser(ctx, actor, record.TargetUser.UserID, note, req.BanDurationDays); apiErr != nil || err != nil {
				return apiErr, err
			}
		case ReportTargetTypeComment:
			if s.commentModerator == nil {
				return nil, errors.New("admin: comment moderator not configured")
			}
			if err := s.commentModerator.DeleteAsAdmin(ctx, record.Item.TargetID); err != nil {
				return nil, err
			}
			if apiErr, err := s.banReportedUser(ctx, actor, record.TargetUser.UserID, note, req.BanDurationDays); apiErr != nil || err != nil {
				return apiErr, err
			}
		case ReportTargetTypeMessage:
			// 消息表 group/private 分离且举报记录不区分来源：先试群表，未命中再试
			// 私聊表（撤回走归属模块服务并广播撤回事件）。两表都未命中（消息已
			// 删除/已撤回）时内容已不可见，记 Warn 后继续裁决流程，不阻断。
			if s.groupMessageModerator == nil || s.privateMessageModerator == nil {
				return nil, errors.New("admin: message moderators not configured")
			}
			groupHit, err := s.groupMessageModerator.RecallMessageByIDAsAdmin(ctx, actor.UserID, record.Item.TargetID)
			if err != nil {
				return nil, err
			}
			if !groupHit {
				privateHit, err := s.privateMessageModerator.RecallMessageByIDAsAdmin(ctx, actor.UserID, record.Item.TargetID)
				if err != nil {
					return nil, err
				}
				if !privateHit {
					observability.FromContext(ctx).Warn("report resolve: message not found in either table, already removed?",
						"message_id", record.Item.TargetID, "report_id", req.ReportID)
				}
			}
			if apiErr, err := s.banReportedUser(ctx, actor, record.TargetUser.UserID, note, req.BanDurationDays); apiErr != nil || err != nil {
				return apiErr, err
			}
		}
	}
	// 裁决落库归位 reports 模块（pending 守卫 + RETURNING 在其仓储实现内；
	// 并发裁决败者返回 false → 已处理冲突）。
	status := ReportStatusPending
	switch req.Action {
	case ReportActionResolve:
		status = ReportStatusResolved
	case ReportActionDismiss:
		status = ReportStatusDismissed
	}
	if s.reportAdmin == nil {
		return nil, errors.New("admin: report admin store not configured")
	}
	notePtr := note
	updated, err := s.reportAdmin.MarkHandledAsAdmin(ctx, req.ReportID, string(status), time.Now().UTC(), &actor.UserID, &notePtr)
	if err != nil {
		if errors.Is(err, errReportAlreadyHandled) {
			// 并发裁决败者：副作用（解散/封禁/删内容）可能已由获胜方执行，
			// 直接返回已处理冲突，不再重复执行也不覆盖裁决结果。
			return &httpx.APIError{Code: httpx.CodeConflict, Message: "举报已处理"}, nil
		}
		if apiErr := asAPIError(err); apiErr != nil {
			return apiErr, nil
		}
		return nil, err
	}
	if !updated {
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "举报已处理"}, nil
	}
	judgeDetails := map[string]any{"type": "report.judge", "report_id": req.ReportID, "action": string(req.Action), "note": note}
	if req.BanDurationDays != nil {
		// 封禁时长进裁决审计：与 user.ban 审计互为印证，裁决记录自身可回答「禁了多久」。
		judgeDetails["ban_duration_days"] = *req.BanDurationDays
	}
	// 裁决审计在裁决事务内写入（与举报终态、内容删除/封禁同事务）：审计写失败
	// 整体回滚并返回错误，管理员可安全重试（举报仍 pending）。此前走 aftercommit
	// best-effort，COMMIT 与回调间崩溃/审计写失败会永久丢审计，且与同一次裁决里
	// 已在事务内写入的 user.ban 审计口径不一致。
	if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("report.judge"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.Nickname}, TargetType: AuditTargetType("report"), TargetID: req.ReportID, TargetName: req.ReportID, Details: judgeDetails}); err != nil {
		return nil, err
	}
	// 举报结果通知的落库在裁决事务内完成（与举报终态同事务）；WS 广播走 outbox
	// 在 COMMIT 后补投，进程崩溃不会丢广播。未装配 outbox 时保留同步回退。
	notifyReporter := req.Action == ReportActionResolve || req.Action == ReportActionDismiss
	useOutboxForReport := notifyReporter && s.eventOutbox != nil && s.pushService != nil && record.Item.Reporter.UserID != ""
	if useOutboxForReport {
		occurredAt := time.Now().UTC().Format(time.RFC3339)
		resolved := req.Action == ReportActionResolve
		notificationID, notifyErr := s.appendReportNotification(ctx, record.Item.Reporter.UserID, req.ReportID, string(record.Item.TargetType), record.Item.TargetID, note, resolved)
		if notifyErr != nil {
			return nil, notifyErr
		}
		event := ReportHandledBroadcastPayload{
			NotifyID:       notificationID,
			ReporterUserID: record.Item.Reporter.UserID,
			ReportID:       req.ReportID,
			TargetType:     string(record.Item.TargetType),
			TargetID:       record.Item.TargetID,
			Note:           note,
			Resolved:       resolved,
			OccurredAt:     occurredAt,
		}
		if enqueueErr := s.eventOutbox.Enqueue(ctx, outbox.TopicAdminReportHandled, event); enqueueErr != nil {
			return nil, enqueueErr
		}
	}
	// 未装配 outbox 时的举报人通知回退仍在事务 COMMIT 后执行（aftercommit）：
	// 回滚时不会发出「裁决未生效但举报人已收到通知」的幻影副作用；无事务时
	// Register 立即执行，语义与旧实现一致。审计已移入事务，不再在此重复写入。
	if notifyReporter && !useOutboxForReport {
		aftercommit.Register(ctx, func(postCtx context.Context) {
			s.notifyReportHandled(postCtx, record.Item.Reporter.UserID, req.ReportID, string(record.Item.TargetType), record.Item.TargetID, note, req.Action == ReportActionResolve)
		})
	}
	return nil, nil
}

// banReportedUser 按举报处理附带的天数封禁目标用户（nil = 不封禁，跳过）。
// appendAudit=true：经举报通道的封禁与 users/set_ban 端点同权同责，必须留痕
// （spec api_spec_admin.md：封禁操作均记录审计日志），否则「谁禁的、禁多久」无从追溯。
func (s *Service) banReportedUser(ctx context.Context, actor AdminActor, targetUserID, note string, banDays *int) (*httpx.APIError, error) {
	if banDays == nil {
		return nil, nil
	}
	return s.banUser(ctx, actor, targetUserID, note, banDays, true)
}

// appendReportNotification 在裁决事务内落库举报结果通知（与举报终态同事务，
// 回滚时不会出现「举报未处理但举报人已收到通知」）。空 reporter/pushService 跳过。
func (s *Service) appendReportNotification(ctx context.Context, reporterUserID, reportID, targetType, targetID, note string, resolved bool) (string, error) {
	if reporterUserID == "" || s.pushService == nil {
		return "", nil
	}
	if resolved {
		return s.pushService.AppendReportResolved(ctx, reporterUserID, reportID, targetType, targetID, note)
	}
	return s.pushService.AppendReportDismissed(ctx, reporterUserID, reportID, targetType, targetID, note)
}

// reportHandledWSPayload 构造 ReportResolved/ReportDismissed 的 WS payload
// （note 为空时省略键；时间键按 resolved/dismissed 区分）。
func reportHandledWSPayload(notifyID, reportID, targetType, targetID, note string, resolved bool, occurredAt string) map[string]any {
	payload := map[string]any{
		"notify_id":   notifyID,
		"report_id":   reportID,
		"target_type": targetType,
		"target_id":   targetID,
	}
	if note != "" {
		payload["note"] = note
	}
	if resolved {
		payload["resolved_at"] = occurredAt
	} else {
		payload["dismissed_at"] = occurredAt
	}
	return payload
}

func (s *Service) broadcastReportHandled(ctx context.Context, reporterUserID string, payload map[string]any, resolved bool) {
	eventType := realtime.EventTypeReportResolved
	if !resolved {
		eventType = realtime.EventTypeReportDismissed
	}
	realtime.HubFromContext(ctx).BroadcastToUser(reporterUserID, realtime.Event{Type: eventType, Payload: payload})
}

// notifyReportHandled 是未装配 outbox 时的同步回退：事务提交后 append 通知并
// WS 推送（bootstrap 生产路径走 outbox，本函数仅供单测/内存装配）。
func (s *Service) notifyReportHandled(ctx context.Context, reporterUserID, reportID, targetType, targetID, note string, resolved bool) {
	if reporterUserID == "" || s.pushService == nil {
		return
	}
	occurredAt := time.Now().UTC().Format(time.RFC3339)
	notificationID, err := s.appendReportNotification(ctx, reporterUserID, reportID, targetType, targetID, note, resolved)
	if err != nil {
		observability.FromContext(ctx).Error("admin.report notify append failed", "error", err, "report_id", reportID)
	}
	s.broadcastReportHandled(ctx, reporterUserID, reportHandledWSPayload(notificationID, reportID, targetType, targetID, note, resolved, occurredAt), resolved)
}

func (s *Service) banUser(ctx context.Context, actor AdminActor, userID, reason string, durationDays *int, appendAudit bool) (*httpx.APIError, error) {
	if strings.TrimSpace(userID) == "" || len(reason) > 500 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	// duration_days 仅拒绝负数（spec api_spec_admin.md users/set_ban：不提供或
	// 0 = 永久封禁；正数无上限约束，不引入规范外的额外上限）。
	if durationDays != nil && *durationDays < 0 {
		return httpx.InvalidRequest("参数错误"), nil
	}
	if actor.UserID == userID {
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "不能封禁自己"}, nil
	}
	item, found, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "用户不存在"}, nil
	}
	_, isAdmin, err := s.repo.GetActor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if isAdmin && !actor.IsSuperAdmin() {
		return httpx.Forbidden(httpx.ReasonAdminProtectedTarget, "目标为平台管理员或超管，不能封禁"), nil
	}
	duration := 0
	if durationDays != nil {
		duration = *durationDays
	}
	bannedAt := time.Now().UTC()
	if s.accountStateWriter == nil {
		return nil, errors.New("admin: account state writer not configured")
	}
	state := usersmodule.AccountBanState{Status: string(UserAccountStatusBanned), Reason: strings.TrimSpace(reason), DurationDays: duration, BannedAt: bannedAt}
	if err := s.accountStateWriter.SetAccountBanState(ctx, userID, state); err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return apiErr, nil
		}
		return nil, err
	}
	s.invalidateBanCache(userID)
	payload := map[string]any{"reason": strings.TrimSpace(reason), "duration_days": duration, "banned_at": bannedAt.Format(time.RFC3339)}
	if s.sessionAdmin == nil {
		return nil, errors.New("admin: session admin not configured")
	}
	sessionIDs, err := s.sessionAdmin.ListActiveSessionIDsAsAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	// 逐条撤销但失败不中断：封禁主流程（ban state 已落库、WS 已断开）不应因
	// 单条会话撤销失败整体报错——已断开的 WS 连接下该会话无法再使用，残留
	// 记录由逐条 Warn 留痕便于排查；此前失败即 return 会让部分会话漏撤。
	failedSessions := 0
	for _, sessionID := range sessionIDs {
		// 踢连接广播在事务 COMMIT 后执行（封禁回滚时不应误踢）；无事务时立即执行。
		aftercommit.Register(ctx, func(context.Context) {
			realtime.HubFromContext(ctx).CloseSession(sessionID, realtime.Event{Type: realtime.EventTypeAccountBanned, Payload: payload})
		})
		if _, err := s.sessionAdmin.RevokeSessionAsAdmin(ctx, sessionID); err != nil {
			failedSessions++
			observability.FromContext(ctx).Warn("admin: revoke session after ban failed",
				"user_id", userID, "session_id", sessionID, "error", err)
		}
	}
	// 审计先于「会话撤销失败」判定落库（审查 P2-8）：封禁状态已生效，spec
	// 要求「封禁操作均记录审计日志」——此前失败分支 return 在 appendAudit 之前，
	// 部分会话撤销失败会导致封禁无审计、且返回 10600 诱导管理员重试（重复封禁/
	// 重复审计）。撤销失败降级为 Warn（逐条已留痕），不阻断封禁结果。
	if appendAudit {
		if err := s.appendAudit(ctx, AuditLogEntry{Action: AuditAction("user.ban"), Operator: UserInfo{UserID: actor.UserID, Nickname: actor.Nickname}, TargetType: AuditTargetType("user"), TargetID: userID, TargetName: item.User.Nickname, Details: map[string]any{"type": "user.ban", "reason": strings.TrimSpace(reason), "duration_days": duration}}); err != nil {
			return nil, err
		}
	}
	if failedSessions > 0 {
		observability.FromContext(ctx).Warn("admin: ban applied but some sessions not revoked",
			"user_id", userID, "failed", failedSessions, "total", len(sessionIDs))
	}
	return nil, nil
}

func (s *Service) appendAudit(ctx context.Context, entry AuditLogEntry) error {
	actor := AdminActor{UserID: entry.Operator.UserID}
	return s.repo.AppendAuditLog(ctx, newAuditEntry(ctx, actor, entry))
}

func isValidReportAction(action ReportAction) bool {
	switch action {
	case ReportActionResolve, ReportActionDismiss:
		return true
	default:
		return false
	}
}
