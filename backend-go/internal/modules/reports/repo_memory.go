package reports

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu      sync.RWMutex
	reports map[string][]ReportItem
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{reports: make(map[string][]ReportItem)}
}

func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateReportRequest) (ReportCreated, error) {
	now := time.Now().UTC()
	report := ReportItem{ReportID: fmt.Sprintf("report_%d", now.UnixNano()), TargetType: req.TargetType, TargetID: req.TargetID, Reason: req.Reason, Description: req.Description, Status: "pending", CreatedAt: now.Format(time.RFC3339)}
	r.mu.Lock()
	r.reports[userID] = append(r.reports[userID], report)
	r.mu.Unlock()
	return ReportCreated{ReportID: report.ReportID}, nil
}

func (r *InMemoryRepository) List(_ context.Context, userID string) (PaginatedReports, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := append([]ReportItem(nil), r.reports[userID]...)
	return PaginatedReports{Items: items}, nil
}
