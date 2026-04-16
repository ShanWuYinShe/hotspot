package reports

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

type stubRepo struct {
	created       ReportCreated
	createErr     error
	listResult    PaginatedReports
	listErr       error
	existsOpen    bool
	existsOpenErr error
	ownerID       string
	ownerFound    bool
	ownerErr      error
}

func (r *stubRepo) Create(_ context.Context, _ string, _ CreateReportRequest) (ReportCreated, error) {
	return r.created, r.createErr
}
func (r *stubRepo) List(_ context.Context, _ string, _ MyReportsRequest) (PaginatedReports, error) {
	return r.listResult, r.listErr
}
func (r *stubRepo) ExistsOpenReport(_ context.Context, _ ReportTargetType, _ string) (bool, error) {
	return r.existsOpen, r.existsOpenErr
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

func TestServiceCreateRejectsDuplicateReport(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerID: "other", ownerFound: true, existsOpen: true})
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u2", Reason: ReportReasonSpam})
	if err == nil {
		t.Fatal("expected error for duplicate report")
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
	result, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u2", Reason: ReportReasonSpam, Description: "spammy"})
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

func TestServiceCreatePropagatesExistsOpenError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{ownerFound: true, existsOpenErr: errors.New("boom")})
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
	_, err := service.Create(ctxWithUser("u1"), CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u2", Reason: ReportReasonSpam, Description: string(longDesc)})
	if err == nil {
		t.Fatal("expected error for too long description")
	}
}

func TestServiceListPassesUserID(t *testing.T) {
	repo := &stubRepo{listResult: PaginatedReports{Items: []ReportItem{{ReportID: "r1"}}}}
	service := NewServiceWithRepo(repo)
	result, err := service.List(ctxWithUser("u1"), MyReportsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestServiceListPropagatesError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{listErr: errors.New("boom")})
	_, err := service.List(ctxWithUser("u1"), MyReportsRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}
