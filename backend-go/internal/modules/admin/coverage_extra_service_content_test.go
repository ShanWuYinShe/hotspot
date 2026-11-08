package admin

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
)

// 本文件覆盖 tags/announcements/reports/stats 四组 Service 的输入校验、
// 跨模块依赖缺失、仓储错误与事件投递失败分支。

// ── tags ──────────────────────────────────────────────────────────────

func TestServiceSearchTagsErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	mods := &covModStore{InMemoryModerationStore: NewInMemoryModerationStore()}
	svc := covPermsService(NewInMemoryRepository(), mods, nil)

	if _, apiErr, err := svc.SearchTags(ctx, AdminTagSearchRequest{Keyword: strings.Repeat("k", 101)}); err != nil || apiErr == nil {
		t.Fatalf("long keyword: apiErr=%v err=%v", apiErr, err)
	}
	if _, apiErr, err := svc.SearchTags(ctx, AdminTagSearchRequest{Pagination: &PaginationParams{Cursor: "not-a-cursor"}}); err != nil || apiErr == nil {
		t.Fatalf("invalid cursor: apiErr=%v err=%v", apiErr, err)
	}
	mods.searchTagsErr = errCovAdmin
	if _, apiErr, err := svc.SearchTags(ctx, AdminTagSearchRequest{Keyword: "hot"}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("store error: apiErr=%v err=%v", apiErr, err)
	}
	mods.searchTagsErr = nil

	createdAt := time.Now().UTC().Truncate(time.Second)
	mods.searchTagsPage = &TagPage{
		Items: []TagRecord{{Name: "hotspot", UsageCount: 2, CreatedAt: createdAt}},
		Next:  &tagCursor{Name: "hotspot", UsageCount: 2, CreatedAt: createdAt},
	}
	cursor := encodeTagCursor(tagCursor{Name: "travel", UsageCount: 3, CreatedAt: createdAt})
	resp, apiErr, err := svc.SearchTags(ctx, AdminTagSearchRequest{Pagination: &PaginationParams{Cursor: cursor}})
	if apiErr != nil || err != nil {
		t.Fatalf("cursor search: apiErr=%v err=%v", apiErr, err)
	}
	if resp.NextCursor == nil || *resp.NextCursor == "" {
		t.Fatalf("expected encoded next cursor: %#v", resp)
	}
}

func TestServiceSetTagBanBranches(t *testing.T) {
	ctx := adminCtx("admin")
	mods := &covModStore{InMemoryModerationStore: NewInMemoryModerationStore()}
	svc := covPermsService(NewInMemoryRepository(), mods, nil)

	// hard_delete 详情落审计。
	if apiErr, err := svc.SetTagBan(ctx, AdminSetTagBanRequest{Name: "hotspot", Active: true, HardDelete: true}); apiErr != nil || err != nil {
		t.Fatalf("hard delete tag ban: apiErr=%v err=%v", apiErr, err)
	}
	// 存储层错误。
	mods.setTagBanErr = errCovAdmin
	if _, err := svc.SetTagBan(ctx, AdminSetTagBanRequest{Name: "travel", Active: true}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("store error: %v", err)
	}
	mods.setTagBanErr = nil
	// 审计写失败。
	base := NewInMemoryRepository()
	repo := &covRepo{InMemoryRepository: base, auditErr: errCovAdmin}
	auditSvc := covPermsService(repo, mods, nil)
	if _, err := auditSvc.SetTagBan(ctx, AdminSetTagBanRequest{Name: "travel", Active: true}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("audit error: %v", err)
	}
}

func TestServiceListAdminAnnouncementsBranches(t *testing.T) {
	ctx := adminCtx("admin")
	mods := &covModStore{InMemoryModerationStore: NewInMemoryModerationStore()}
	svc := covPermsService(NewInMemoryRepository(), mods, nil)

	if _, apiErr, err := svc.ListAdminAnnouncements(ctx, ListAdminAnnouncementsRequest{Pagination: &PaginationParams{Cursor: "not-a-cursor"}}); err != nil || apiErr == nil {
		t.Fatalf("invalid cursor: apiErr=%v err=%v", apiErr, err)
	}
	mods.searchAnnErr = errCovAdmin
	if _, apiErr, err := svc.ListAdminAnnouncements(ctx, ListAdminAnnouncementsRequest{}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("store error: apiErr=%v err=%v", apiErr, err)
	}
	mods.searchAnnErr = nil

	createdAt := time.Now().UTC().Truncate(time.Second)
	mods.searchAnnPage = &systemmodule.AnnouncementPage{
		Items: []systemmodule.AnnouncementRecord{{AnnouncementID: "ann_cov", Title: "t", Content: "c", Severity: systemmodule.AnnounceLevelInfo, CreatedAt: createdAt}},
		Next:  &systemmodule.AnnouncementCursor{CreatedAt: createdAt, AnnouncementID: "ann_cov"},
	}
	cursor := encodeAnnouncementCursor(announcementCursor{CreatedAt: createdAt, AnnouncementID: "ann_cov"})
	resp, apiErr, err := svc.ListAdminAnnouncements(ctx, ListAdminAnnouncementsRequest{Pagination: &PaginationParams{Cursor: cursor}})
	if apiErr != nil || err != nil {
		t.Fatalf("cursor list: apiErr=%v err=%v", apiErr, err)
	}
	if resp.NextCursor == nil || *resp.NextCursor == "" {
		t.Fatalf("expected encoded next cursor: %#v", resp)
	}
}

func TestServiceCreateAnnouncementBranches(t *testing.T) {
	ctx := adminCtx("admin")
	mods := &covModStore{InMemoryModerationStore: NewInMemoryModerationStore()}
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := covPermsService(repo, mods, covPush{})

	// 非法内容。
	if _, apiErr, err := svc.CreateAnnouncement(ctx, CreateAnnouncementRequest{Title: "", Content: "c", Severity: systemmodule.AnnounceLevelInfo}); err != nil || apiErr == nil {
		t.Fatalf("invalid create: apiErr=%v err=%v", apiErr, err)
	}

	// 非事务路径的存储错误 + 审计失败。
	mods.createAnnErr = errCovAdmin
	if _, apiErr, err := svc.CreateAnnouncement(ctx, CreateAnnouncementRequest{Title: "t", Content: "c", Severity: systemmodule.AnnounceLevelInfo}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("create store error: apiErr=%v err=%v", apiErr, err)
	}
	mods.createAnnErr = nil
	repo.auditErr = errCovAdmin
	if _, apiErr, err := svc.CreateAnnouncement(ctx, CreateAnnouncementRequest{Title: "t", Content: "c", Severity: systemmodule.AnnounceLevelInfo}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("create audit error: apiErr=%v err=%v", apiErr, err)
	}
	repo.auditErr = nil

	// 事务 + outbox 路径：入队成功。
	svc.SetTxRunner(covRunInTx)
	svc.SetEventOutbox(covOutbox{})
	if _, apiErr, err := svc.CreateAnnouncement(ctx, CreateAnnouncementRequest{Title: "tx", Content: "c", Severity: systemmodule.AnnounceLevelInfo, SendPush: true}); apiErr != nil || err != nil {
		t.Fatalf("tx create: apiErr=%v err=%v", apiErr, err)
	}

	// 事务内创建失败。
	mods.createAnnErr = errCovAdmin
	if _, apiErr, err := svc.CreateAnnouncement(ctx, CreateAnnouncementRequest{Title: "tx2", Content: "c", Severity: systemmodule.AnnounceLevelInfo, SendPush: true}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("tx create error: apiErr=%v err=%v", apiErr, err)
	}
	mods.createAnnErr = nil

	// 事务内 outbox 入队失败。
	svc.SetEventOutbox(covOutbox{err: errCovAdmin})
	if _, apiErr, err := svc.CreateAnnouncement(ctx, CreateAnnouncementRequest{Title: "tx3", Content: "c", Severity: systemmodule.AnnounceLevelInfo, SendPush: true}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("outbox error: apiErr=%v err=%v", apiErr, err)
	}
}

func TestServiceUpdateAnnouncementBranches(t *testing.T) {
	ctx := adminCtx("admin")
	mods := &covModStore{InMemoryModerationStore: NewInMemoryModerationStore()}
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := covPermsService(repo, mods, nil)

	// 非法更新请求。
	if _, apiErr, err := svc.UpdateAnnouncement(ctx, UpdateAnnouncementRequest{AnnouncementID: "ann_cov"}); err != nil || apiErr == nil {
		t.Fatalf("invalid update: apiErr=%v err=%v", apiErr, err)
	}

	// 读取失败。
	mods.getAnnErr = errCovAdmin
	title := "new title"
	if _, apiErr, err := svc.UpdateAnnouncement(ctx, UpdateAnnouncementRequest{AnnouncementID: "ann_cov", Title: &title}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("get error: apiErr=%v err=%v", apiErr, err)
	}
	mods.getAnnErr = nil

	// 存储层 not found / 原始错误。
	created, apiErr, err := svc.CreateAnnouncement(ctx, CreateAnnouncementRequest{Title: "orig", Content: "c", Severity: systemmodule.AnnounceLevelInfo})
	if apiErr != nil || err != nil {
		t.Fatalf("seed announcement: apiErr=%v err=%v", apiErr, err)
	}
	mods.updateAnnErr = systemmodule.ErrAnnouncementNotFound()
	if _, apiErr, err := svc.UpdateAnnouncement(ctx, UpdateAnnouncementRequest{AnnouncementID: created.AnnouncementID, Title: &title}); err != nil || apiErr == nil {
		t.Fatalf("update not found: apiErr=%v err=%v", apiErr, err)
	}
	mods.updateAnnErr = errCovAdmin
	if _, apiErr, err := svc.UpdateAnnouncement(ctx, UpdateAnnouncementRequest{AnnouncementID: created.AnnouncementID, Title: &title}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("update store error: apiErr=%v err=%v", apiErr, err)
	}
	mods.updateAnnErr = nil

	// 成功更新但审计失败。
	repo.auditErr = errCovAdmin
	if _, apiErr, err := svc.UpdateAnnouncement(ctx, UpdateAnnouncementRequest{AnnouncementID: created.AnnouncementID, Title: &title}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("update audit error: apiErr=%v err=%v", apiErr, err)
	}
}

func TestServiceDeleteAnnouncementBranches(t *testing.T) {
	ctx := adminCtx("admin")
	mods := &covModStore{InMemoryModerationStore: NewInMemoryModerationStore()}
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := covPermsService(repo, mods, nil)

	mods.getAnnErr = errCovAdmin
	if _, err := svc.DeleteAnnouncement(ctx, "ann_cov"); !errors.Is(err, errCovAdmin) {
		t.Fatalf("get error: %v", err)
	}
	mods.getAnnErr = nil

	// 已实删/不存在的公告：存储层返回 not found。
	mods.deleteAnnErr = systemmodule.ErrAnnouncementNotFound()
	if apiErr, err := svc.DeleteAnnouncement(ctx, "ann_missing"); err != nil || apiErr == nil {
		t.Fatalf("delete not found: apiErr=%v err=%v", apiErr, err)
	}
	mods.deleteAnnErr = nil

	created, _, _ := svc.CreateAnnouncement(ctx, CreateAnnouncementRequest{Title: "t", Content: "c", Severity: systemmodule.AnnounceLevelInfo})
	mods.deleteAnnErr = errCovAdmin
	if _, err := svc.DeleteAnnouncement(ctx, created.AnnouncementID); !errors.Is(err, errCovAdmin) {
		t.Fatalf("delete store error: %v", err)
	}
	mods.deleteAnnErr = nil

	repo.auditErr = errCovAdmin
	if _, err := svc.DeleteAnnouncement(ctx, created.AnnouncementID); !errors.Is(err, errCovAdmin) {
		t.Fatalf("delete audit error: %v", err)
	}
}

// ── reports ───────────────────────────────────────────────────────────

func TestServiceSearchReportsErrorBranches(t *testing.T) {
	ctx := adminCtx("admin")
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := covPermsService(repo, NewInMemoryModerationStore(), nil)

	if _, apiErr, err := svc.SearchReports(ctx, AdminReportListRequest{Pagination: &PaginationParams{Cursor: "not-a-cursor"}}); err != nil || apiErr == nil {
		t.Fatalf("invalid cursor: apiErr=%v err=%v", apiErr, err)
	}
	repo.searchReportsErr = &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "boom"}
	if _, apiErr, err := svc.SearchReports(ctx, AdminReportListRequest{}); apiErr == nil || err != nil {
		t.Fatalf("search api error: apiErr=%v err=%v", apiErr, err)
	}
	repo.searchReportsErr = errCovAdmin
	if _, apiErr, err := svc.SearchReports(ctx, AdminReportListRequest{}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("search raw error: apiErr=%v err=%v", apiErr, err)
	}
	repo.searchReportsErr = nil
	repo.searchReportsPage = &adminReportPage{
		Items: []AdminReportItem{},
		Next:  &adminReportCursor{CreatedAt: time.Now().UTC(), ReportID: "report_cov"},
		Total: 3,
	}
	resp, apiErr, err := svc.SearchReports(ctx, AdminReportListRequest{Status: ReportStatusPending})
	if apiErr != nil || err != nil || resp.NextCursor == nil || resp.Total == nil {
		t.Fatalf("next cursor/total: resp=%#v apiErr=%v err=%v", resp, apiErr, err)
	}
}

func covNewReportService(mods ModerationStore, push announcementPushService) (*Service, *covRepo) {
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := NewService(repo, &stubProvider{config: defaultConfig()}, mods, push)
	return svc, repo
}

func TestServiceHandleReportValidationBranches(t *testing.T) {
	svc, repo := covNewReportService(NewInMemoryModerationStore(), nil)
	ctx := adminCtx("admin")
	seedPendingReport(t, repo.InMemoryRepository, "report_cov", "user_reporter", "user", "user_target")

	// 非法动作 / 空 ID。
	if apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_cov", Action: ReportAction("bogus")}); err != nil || apiErr == nil {
		t.Fatalf("invalid action: apiErr=%v err=%v", apiErr, err)
	}
	if apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: " ", Action: ReportActionResolve}); err != nil || apiErr == nil {
		t.Fatalf("empty report id: apiErr=%v err=%v", apiErr, err)
	}

	repo.getReportRecordErr = errCovAdmin
	if _, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_cov", Action: ReportActionResolve}); !errors.Is(err, errCovAdmin) {
		t.Fatalf("record error: %v", err)
	}
	repo.getReportRecordErr = nil

	if apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_missing", Action: ReportActionResolve}); err != nil || apiErr == nil {
		t.Fatalf("report not found: apiErr=%v err=%v", apiErr, err)
	}
}

func TestServiceHandleReportTargetActions(t *testing.T) {
	newSvc := func() (*Service, *covRepo) {
		svc, repo := covNewReportService(NewInMemoryModerationStore(), nil)
		svc.SetReportAdminStore(testReportAdminStore{store: repo.InMemoryRepository})
		return svc, repo
	}
	ctx := adminCtx("admin")

	t.Run("moment moderator missing", func(t *testing.T) {
		svc, repo := newSvc()
		seedPendingReport(t, repo.InMemoryRepository, "report_moment", "user_reporter", "moment", "moment_cov")
		if _, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_moment", Action: ReportActionResolve}); err == nil {
			t.Fatal("expected moment moderator missing error")
		}
	})

	t.Run("moment delete error", func(t *testing.T) {
		svc, repo := newSvc()
		seedPendingReport(t, repo.InMemoryRepository, "report_moment2", "user_reporter", "moment", "moment_cov")
		svc.SetMomentModerator(covMomentModerator{err: errCovAdmin})
		if _, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_moment2", Action: ReportActionResolve}); !errors.Is(err, errCovAdmin) {
			t.Fatalf("moment delete error: %v", err)
		}
	})

	t.Run("comment moderator missing", func(t *testing.T) {
		svc, repo := newSvc()
		seedPendingReport(t, repo.InMemoryRepository, "report_comment", "user_reporter", "comment", "comment_cov")
		if _, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_comment", Action: ReportActionResolve}); err == nil {
			t.Fatal("expected comment moderator missing error")
		}
	})

	t.Run("comment delete error", func(t *testing.T) {
		svc, repo := newSvc()
		seedPendingReport(t, repo.InMemoryRepository, "report_comment2", "user_reporter", "comment", "comment_cov")
		svc.SetCommentModerator(covCommentModerator{err: errCovAdmin})
		if _, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_comment2", Action: ReportActionResolve}); !errors.Is(err, errCovAdmin) {
			t.Fatalf("comment delete error: %v", err)
		}
	})

	t.Run("message moderators missing", func(t *testing.T) {
		svc, repo := newSvc()
		seedPendingReport(t, repo.InMemoryRepository, "report_message", "user_reporter", "message", "message_cov")
		if _, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_message", Action: ReportActionResolve}); err == nil {
			t.Fatal("expected message moderators missing error")
		}
	})

	t.Run("group message recall error", func(t *testing.T) {
		svc, repo := newSvc()
		seedPendingReport(t, repo.InMemoryRepository, "report_message2", "user_reporter", "message", "message_cov")
		svc.SetMessageModerators(covMessageModerator{err: errCovAdmin}, covMessageModerator{})
		if _, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_message2", Action: ReportActionResolve}); !errors.Is(err, errCovAdmin) {
			t.Fatalf("group recall error: %v", err)
		}
	})

	t.Run("private message recall error and miss", func(t *testing.T) {
		svc, repo := newSvc()
		seedPendingReport(t, repo.InMemoryRepository, "report_message3", "user_reporter", "message", "message_cov")
		svc.SetMessageModerators(covMessageModerator{}, covMessageModerator{err: errCovAdmin})
		if _, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_message3", Action: ReportActionResolve}); !errors.Is(err, errCovAdmin) {
			t.Fatalf("private recall error: %v", err)
		}

		seedPendingReport(t, repo.InMemoryRepository, "report_message4", "user_reporter", "message", "message_cov")
		svc.SetMessageModerators(covMessageModerator{}, covMessageModerator{})
		if apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_message4", Action: ReportActionResolve}); apiErr != nil || err != nil {
			t.Fatalf("both tables miss: apiErr=%v err=%v", apiErr, err)
		}
	})

	t.Run("report admin store missing", func(t *testing.T) {
		svc, repo := covNewReportService(NewInMemoryModerationStore(), nil)
		seedPendingReport(t, repo.InMemoryRepository, "report_user", "user_reporter", "user", "user_target")
		if _, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_user", Action: ReportActionResolve}); err == nil {
			t.Fatal("expected report admin store missing error")
		}
	})
}

func TestServiceHandleReportPersistenceErrors(t *testing.T) {
	ctx := adminCtx("admin")
	seedAndRun := func(t *testing.T, store reportAdminStore, push announcementPushService, configure func(*Service, *covRepo)) (*Service, *covRepo) {
		t.Helper()
		svc, repo := covNewReportService(NewInMemoryModerationStore(), push)
		svc.SetReportAdminStore(store)
		seedPendingReport(t, repo.InMemoryRepository, "report_cov", "user_reporter", "user", "user_target")
		if configure != nil {
			configure(svc, repo)
		}
		return svc, repo
	}

	t.Run("already handled", func(t *testing.T) {
		svc, _ := seedAndRun(t, covReportAdminStore{err: errReportAlreadyHandled}, nil, nil)
		apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_cov", Action: ReportActionResolve})
		if err != nil || apiErr == nil || apiErr.Code != httpx.CodeConflict {
			t.Fatalf("already handled: apiErr=%v err=%v", apiErr, err)
		}
	})

	t.Run("store api error", func(t *testing.T) {
		storeErr := &httpx.APIError{Code: httpx.CodeInternalUnavailable, Message: "boom"}
		svc, _ := seedAndRun(t, covReportAdminStore{err: storeErr}, nil, nil)
		apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_cov", Action: ReportActionResolve})
		if err != nil || apiErr == nil {
			t.Fatalf("store api error: apiErr=%v err=%v", apiErr, err)
		}
	})

	t.Run("store raw error", func(t *testing.T) {
		svc, _ := seedAndRun(t, covReportAdminStore{err: errCovAdmin}, nil, nil)
		if _, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_cov", Action: ReportActionResolve}); !errors.Is(err, errCovAdmin) {
			t.Fatalf("store raw error: %v", err)
		}
	})

	t.Run("not updated", func(t *testing.T) {
		svc, _ := seedAndRun(t, covReportAdminStore{updated: false}, nil, nil)
		apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_cov", Action: ReportActionResolve})
		if err != nil || apiErr == nil || apiErr.Code != httpx.CodeConflict {
			t.Fatalf("not updated: apiErr=%v err=%v", apiErr, err)
		}
	})

	t.Run("audit failure is best effort", func(t *testing.T) {
		svc, repo := seedAndRun(t, testReportAdminStore{}, nil, func(s *Service, r *covRepo) {
			s.SetReportAdminStore(testReportAdminStore{store: r.InMemoryRepository})
			r.auditErr = errCovAdmin
		})
		apiErr, err := svc.HandleReport(ctx, JudgeAdminReportRequest{ReportID: "report_cov", Action: ReportActionResolve})
		if apiErr != nil || err != nil {
			t.Fatalf("audit failure should not fail decision: apiErr=%v err=%v", apiErr, err)
		}
		_ = repo
	})

	t.Run("outbox notify error", func(t *testing.T) {
		svc, _ := seedAndRun(t, testReportAdminStore{}, covPush{resolvedErr: errCovAdmin}, func(s *Service, r *covRepo) {
			s.SetReportAdminStore(testReportAdminStore{store: r.InMemoryRepository})
			s.SetEventOutbox(covOutbox{})
		})
		if _, err := svc.HandleReport(ctx, judgeResolveRequest()); !errors.Is(err, errCovAdmin) {
			t.Fatalf("outbox notify error: %v", err)
		}
	})

	t.Run("outbox enqueue error", func(t *testing.T) {
		svc, _ := seedAndRun(t, testReportAdminStore{}, covPush{}, func(s *Service, r *covRepo) {
			s.SetReportAdminStore(testReportAdminStore{store: r.InMemoryRepository})
			s.SetEventOutbox(covOutbox{err: errCovAdmin})
		})
		if _, err := svc.HandleReport(ctx, judgeResolveRequest()); !errors.Is(err, errCovAdmin) {
			t.Fatalf("outbox enqueue error: %v", err)
		}
	})
}

func judgeResolveRequest() JudgeAdminReportRequest {
	return JudgeAdminReportRequest{ReportID: "report_cov", Action: ReportActionResolve}
}

func TestServiceHandleReportPermissionInheritance(t *testing.T) {
	// resolve + ban_duration_days 需要 admin.users.write；仅 reports.write 时拒绝。
	mods := NewInMemoryModerationStore()
	svc, repo := covNewReportService(mods, nil)
	seedPendingReport(t, repo.InMemoryRepository, "report_perm", "user_reporter", "user", "user_target")
	covGrantPermissions(t, repo.InMemoryRepository, "user_alice", "cov_report_only", permissionReportsWrite)
	days := 1
	apiErr, err := svc.HandleReport(adminCtx("user_alice"), JudgeAdminReportRequest{ReportID: "report_perm", Action: ReportActionResolve, BanDurationDays: &days})
	if err != nil || apiErr == nil {
		t.Fatalf("expected forbidden, apiErr=%v err=%v", apiErr, err)
	}
}

func TestServiceNotifyReportHandledBranches(t *testing.T) {
	svc, _ := covNewReportService(NewInMemoryModerationStore(), covPush{resolvedErr: errCovAdmin})
	// 空 reporter：appendReportNotification 直接返回空通知 ID。
	if id, err := svc.appendReportNotification(context.Background(), "", "report_cov", "user", "user_target", "note", true); id != "" || err != nil {
		t.Fatalf("empty reporter: id=%q err=%v", id, err)
	}
	// 通知落库失败：notifyReportHandled 记错误日志但不 panic。
	svc.notifyReportHandled(context.Background(), "user_reporter", "report_cov", "user", "user_target", "note", true)
	if id, err := svc.appendReportNotification(context.Background(), "user_reporter", "report_cov", "user", "user_target", "note", true); err == nil {
		t.Fatalf("expected notify error, got id=%q", id)
	} else if id != "" {
		t.Fatalf("expected empty id on error, got %q", id)
	}
}

// ── stats ─────────────────────────────────────────────────────────────

func TestServiceStatsGeoNormalization(t *testing.T) {
	svc, _, _, _ := newTestService()
	ctx := adminCtx("admin")

	// limit=0 → 默认 20；limit 超上限 → 100。
	if _, apiErr, err := svc.StatsGeo(ctx, AdminGeoStatsRequest{Limit: intPtr(0)}); apiErr != nil || err != nil {
		t.Fatalf("limit 0: apiErr=%v err=%v", apiErr, err)
	}
	if _, apiErr, err := svc.StatsGeo(ctx, AdminGeoStatsRequest{Limit: intPtr(500)}); apiErr != nil || err != nil {
		t.Fatalf("limit max: apiErr=%v err=%v", apiErr, err)
	}
	// 时间窗解析失败。
	if _, apiErr, err := svc.StatsGeo(ctx, AdminGeoStatsRequest{TimeRange: &StatsTimeRangeParams{StartTime: "yesterday"}}); err != nil || apiErr == nil {
		t.Fatalf("bad time range: apiErr=%v err=%v", apiErr, err)
	}
	// 仓储错误。
	repo := &covRepo{InMemoryRepository: NewInMemoryRepository(), statsGeoErr: errCovAdmin}
	svcErr := covPermsService(repo, NewInMemoryModerationStore(), nil)
	if _, apiErr, err := svcErr.StatsGeo(ctx, AdminGeoStatsRequest{}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("stats repo error: apiErr=%v err=%v", apiErr, err)
	}
}

func TestServiceStatsGeoDrilldownNormalization(t *testing.T) {
	svc, _, _, _ := newTestService()
	ctx := adminCtx("admin")
	token := covDrilldownToken(t)

	if _, apiErr, err := svc.StatsGeoDrilldown(ctx, AdminGeoDrilldownRequest{CellToken: token, EntityType: "user", TimeRange: &StatsTimeRangeParams{StartTime: "2026-01-01T00:00:00Z", EndTime: "2026-01-02T00:00:00Z"}}); apiErr != nil || err != nil {
		t.Fatalf("valid time range: apiErr=%v err=%v", apiErr, err)
	}
	if _, apiErr, err := svc.StatsGeoDrilldown(ctx, AdminGeoDrilldownRequest{CellToken: token, EntityType: "user", TimeRange: &StatsTimeRangeParams{StartTime: "bogus"}}); err != nil || apiErr == nil {
		t.Fatalf("bad start: apiErr=%v err=%v", apiErr, err)
	}
	if _, apiErr, err := svc.StatsGeoDrilldown(ctx, AdminGeoDrilldownRequest{CellToken: token, EntityType: "user", TimeRange: &StatsTimeRangeParams{EndTime: "bogus"}}); err != nil || apiErr == nil {
		t.Fatalf("bad end: apiErr=%v err=%v", apiErr, err)
	}
	if _, apiErr, err := svc.StatsGeoDrilldown(ctx, AdminGeoDrilldownRequest{CellToken: token, EntityType: "user", TimeRange: &StatsTimeRangeParams{StartTime: "2026-01-02T00:00:00Z", EndTime: "2026-01-01T00:00:00Z"}}); err != nil || apiErr == nil {
		t.Fatalf("reversed range: apiErr=%v err=%v", apiErr, err)
	}
	if _, apiErr, err := svc.StatsGeoDrilldown(ctx, AdminGeoDrilldownRequest{CellToken: token, EntityType: "user", Pagination: &PaginationParams{Limit: 500}}); apiErr != nil || err != nil {
		t.Fatalf("limit max: apiErr=%v err=%v", apiErr, err)
	}

	repo := &covRepo{InMemoryRepository: NewInMemoryRepository(), statsDrilldownErr: errCovAdmin}
	svcErr := covPermsService(repo, NewInMemoryModerationStore(), nil)
	if _, apiErr, err := svcErr.StatsGeoDrilldown(ctx, AdminGeoDrilldownRequest{CellToken: token, EntityType: "user"}); !errors.Is(err, errCovAdmin) || apiErr != nil {
		t.Fatalf("drilldown repo error: apiErr=%v err=%v", apiErr, err)
	}
}

// covDrilldownToken 生成合法 8 层 S2 token。
func covDrilldownToken(t *testing.T) string {
	t.Helper()
	token := geo.CellTokenAtLevel(geo.Point{Latitude: 31.2304, Longitude: 121.4737}, 8)
	if token == "" {
		t.Fatal("empty s2 cell token")
	}
	return token
}
