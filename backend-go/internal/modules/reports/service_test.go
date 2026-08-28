package reports

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/types"
)

type stubRepo struct {
	created          ReportCreated
	createErr        error
	listResult       PaginatedReports
	listErr          error
	existingReportID string
	existingFound    bool
	findExistingErr  error
	ownerID          string
	ownerFound       bool
	ownerErr         error
	// snapshot 服务端快照生成路径（target_type='message'）的注入桩。
	snapshot     *ReportSnapshot
	snapshotOK   bool
	snapshotErr  error
	lastSnapshot *ReportSnapshot
}

func (r *stubRepo) Create(_ context.Context, _ string, _ CreateReportRequest, snapshot *ReportSnapshot) (ReportCreated, error) {
	r.lastSnapshot = snapshot
	return r.created, r.createErr
}
func (r *stubRepo) List(_ context.Context, _ string, _ ListReportsRequest) (PaginatedReports, error) {
	return r.listResult, r.listErr
}
func (r *stubRepo) FindExistingReport(_ context.Context, _ string, _ ReportTargetType, _ string) (string, bool, error) {
	return r.existingReportID, r.existingFound, r.findExistingErr
}
func (r *stubRepo) TargetOwner(_ context.Context, _ ReportTargetType, _ string) (string, bool, error) {
	return r.ownerID, r.ownerFound, r.ownerErr
}

func ctxWithUser(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func TestServiceCreateRejectsInvalidTargetType(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerFound: true})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: "bad", TargetID: "t1", Reason: ReportReasonSpam})
	if err == nil {
		t.Fatal("expected error for invalid target type")
	}
}

func TestServiceCreateRejectsInvalidReason(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerFound: true})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "t1", Reason: "bad"})
	if err == nil {
		t.Fatal("expected error for invalid reason")
	}
}

func TestServiceCreateRejectsEmptyTargetID(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerFound: true})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "", Reason: ReportReasonSpam})
	if err == nil {
		t.Fatal("expected error for empty target_id")
	}
}

func TestServiceCreateRejectsSelfReport(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerID: "u1", ownerFound: true})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u1", Reason: ReportReasonSpam})
	if err == nil {
		t.Fatal("expected error for self report")
	}
}

func TestServiceCreateRejectsTargetNotFound(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerFound: false})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u2", Reason: ReportReasonSpam})
	if err == nil {
		t.Fatal("expected error for target not found")
	}
}

func TestServiceCreateDuplicateReturnsIsDuplicate(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerID: "other", ownerFound: true, existingFound: true, existingReportID: "r_existing"})
	result, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u2", Reason: ReportReasonSpam})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.IsDuplicate {
		t.Fatal("expected IsDuplicate=true on duplicate report")
	}
	if result.ReportID != "r_existing" {
		t.Fatalf("expected existing report_id r_existing, got %s", result.ReportID)
	}
}

func TestServiceCreateRejectsOwnerSelfReport(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerID: "u1", ownerFound: true})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeMoment, TargetID: "m1", Reason: ReportReasonSpam})
	if err == nil {
		t.Fatal("expected error for self report via owner match")
	}
}

func TestServiceCreateHappyPath(t *testing.T) {
	repo := &stubRepo{ownerID: "u2", ownerFound: true, created: ReportCreated{ReportID: "r1"}}
	service := NewServiceWithRepo(repo)
	result, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u2", Reason: ReportReasonSpam, Description: types.StrPtr("spammy")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ReportID != "r1" {
		t.Fatalf("expected report_id r1, got %s", result.ReportID)
	}
}

func TestServiceCreatePropagatesRepoError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerFound: true, createErr: errors.New("boom")})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u2", Reason: ReportReasonSpam})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceCreatePropagatesTargetOwnerError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerErr: errors.New("boom")})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u2", Reason: ReportReasonSpam})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceCreatePropagatesFindExistingError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerFound: true, findExistingErr: errors.New("boom")})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u2", Reason: ReportReasonSpam})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceCreateRejectsDescriptionTooLong(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerFound: true})
	longDesc := make([]byte, 1001)
	for i := range longDesc {
		longDesc[i] = 'a'
	}
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u2", Reason: ReportReasonSpam, Description: types.StrPtr(string(longDesc))})
	if err == nil {
		t.Fatal("expected error for too long description")
	}
}

func TestServiceListPassesUserID(t *testing.T) {
	repo := &stubRepo{listResult: PaginatedReports{Items: []ReportItem{{ReportID: "r1"}}}}
	service := NewServiceWithRepo(repo)
	result, err := service.List(ctxWithUser("u1"), ListReportsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestServiceListPropagatesError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{listErr: errors.New("boom")})
	_, err := service.List(ctxWithUser("u1"), ListReportsRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// stubSnapshotSource 实现 MessageSnapshotSource（服务端快照生成源）。
type stubSnapshotSource struct {
	snapshot *ReportSnapshot
	found    bool
	err      error
}

func (s stubSnapshotSource) GetMessageSnapshot(_ context.Context, _ string, _ string) (*ReportSnapshot, bool, error) {
	return s.snapshot, s.found, s.err
}

// TestServiceCreateGeneratesMessageSnapshotServerSide：target_type='message' 时
// 服务端基于消息真值生成快照并随举报落库（spec api_spec_types.md：客户端
// 不提交快照，防伪造证据链）。
func TestServiceCreateGeneratesMessageSnapshotServerSide(t *testing.T) {
	repo := &stubRepo{ownerID: "u2", ownerFound: true, created: ReportCreated{ReportID: "r1"}}
	service := NewServiceWithRepo(repo)
	snapshot := &ReportSnapshot{ChatType: "group", TargetChatID: "g1", MessageType: "text", SnapshotContent: types.StrPtr("hi"), SentAt: "2026-01-01T00:00:00Z"}
	service.SetMessageSnapshotSource(stubSnapshotSource{snapshot: snapshot, found: true})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeMessage, TargetID: "m1", Reason: ReportReasonSpam})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastSnapshot != snapshot {
		t.Fatalf("expected server-generated snapshot to be stored, got %#v", repo.lastSnapshot)
	}
}

// TestServiceCreateMessageReportWithoutSnapshotSource：未接线快照源时消息举报
// 按目标不存在处理（不产生无证据的举报）。
func TestServiceCreateMessageReportWithoutSnapshotSource(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerID: "u2", ownerFound: true})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeMessage, TargetID: "m1", Reason: ReportReasonSpam})
	if err == nil || err.Error() != "reports: target not found" {
		t.Fatalf("expected target-not-found error, got %v", err)
	}
}

// TestServiceCreateMessageReportSnapshotNotFound：消息不存在（快照源 found=false）
// 时受理返回 10300（reports: target not found）。
func TestServiceCreateMessageReportSnapshotNotFound(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerID: "u2", ownerFound: true})
	service.SetMessageSnapshotSource(stubSnapshotSource{found: false})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeMessage, TargetID: "missing", Reason: ReportReasonSpam})
	if err == nil || err.Error() != "reports: target not found" {
		t.Fatalf("expected target-not-found error, got %v", err)
	}
}

// TestServiceCreateMessageReportSnapshotError：快照源故障按服务端错误上抛。
func TestServiceCreateMessageReportSnapshotError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerID: "u2", ownerFound: true})
	service.SetMessageSnapshotSource(stubSnapshotSource{err: errors.New("boom")})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeMessage, TargetID: "m1", Reason: ReportReasonSpam})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestServiceCreateNonMessageReportSkipsSnapshot：非消息举报不查快照源、
// 落库快照为 nil。
func TestServiceCreateNonMessageReportSkipsSnapshot(t *testing.T) {
	repo := &stubRepo{ownerID: "u2", ownerFound: true, created: ReportCreated{ReportID: "r1"}}
	service := NewServiceWithRepo(repo)
	// 未接线快照源：非消息举报不应受影响
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeMoment, TargetID: "m1", Reason: ReportReasonSpam})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastSnapshot != nil {
		t.Fatalf("expected nil snapshot for non-message report, got %#v", repo.lastSnapshot)
	}
}

// TestServiceCreatePassesReporterToSnapshotSource：service 必须把举报人身份
// 透传给快照源（参与者校验在快照源实现内做：群聊成员/私聊双方）。
func TestServiceCreatePassesReporterToSnapshotSource(t *testing.T) {
	repo := &stubRepo{ownerID: "u2", ownerFound: true, created: ReportCreated{ReportID: "r9"}}
	service := NewServiceWithRepo(repo)
	source := &recordingSnapshotSource{snapshot: &ReportSnapshot{ChatType: "private", TargetChatID: "u2", MessageType: "text"}, found: true}
	service.SetMessageSnapshotSource(source)
	if _, err := service.Create(ctxWithUser("u_reporter"), CreateReportRequest{TargetType: ReportTargetTypeMessage, TargetID: "m1", Reason: ReportReasonSpam}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source.lastReporter != "u_reporter" {
		t.Fatalf("expected reporter id to be passed to snapshot source, got %q", source.lastReporter)
	}
}

type recordingSnapshotSource struct {
	snapshot     *ReportSnapshot
	found        bool
	lastReporter string
}

func (s *recordingSnapshotSource) GetMessageSnapshot(_ context.Context, _ string, reporterUserID string) (*ReportSnapshot, bool, error) {
	s.lastReporter = reporterUserID
	return s.snapshot, s.found, nil
}
