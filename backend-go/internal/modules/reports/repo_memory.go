package reports

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu           sync.RWMutex
	reports      map[string][]ReportItem
	targetOwners map[ReportTargetType]map[string]string
	// seq 保证同纳秒内连续创建的 ID 不碰撞（UnixNano 时间戳在快速循环下会重复）。
	seq uint64
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		reports: make(map[string][]ReportItem),
		targetOwners: map[ReportTargetType]map[string]string{
			ReportTargetTypeUser:    {},
			ReportTargetTypeMoment:  {},
			ReportTargetTypeComment: {},
			ReportTargetTypeMessage: {},
			ReportTargetTypeGroup:   {},
		},
	}
}

func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateReportRequest) (ReportCreated, error) {
	now := time.Now().UTC()
	r.seq++
	report := ReportItem{ReportID: fmt.Sprintf("report_%d_%d", now.UnixNano(), r.seq), TargetType: req.TargetType, TargetID: req.TargetID, Reason: req.Reason, Description: req.Description, Status: ReportStatusPending, CreatedAt: now.Format(time.RFC3339), MessageSnapshot: req.MessageSnapshot}
	r.mu.Lock()
	r.reports[userID] = append(r.reports[userID], report)
	r.mu.Unlock()
	return ReportCreated{ReportID: report.ReportID}, nil
}

func (r *InMemoryRepository) List(_ context.Context, userID string, req MyReportsRequest) (PaginatedReports, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ReportItem, 0, len(r.reports[userID]))
	for _, item := range r.reports[userID] {
		if req.Status != "" && item.Status != req.Status {
			continue
		}
		if req.TargetType != "" && item.TargetType != req.TargetType {
			continue
		}
		items = append(items, item)
	}
	return PaginatedReports{Items: items}, nil
}

func (r *InMemoryRepository) FindExistingReport(_ context.Context, _ string, targetType ReportTargetType, targetID string) (string, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, items := range r.reports {
		for _, item := range items {
			if item.TargetType == targetType && item.TargetID == targetID && item.Status == ReportStatusPending {
				return item.ReportID, true, nil
			}
		}
	}
	return "", false, nil
}

func (r *InMemoryRepository) TargetOwner(_ context.Context, targetType ReportTargetType, targetID string) (string, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	owners, ok := r.targetOwners[targetType]
	if !ok {
		return "", false, nil
	}
	ownerUserID, found := owners[targetID]
	return ownerUserID, found, nil
}

func (r *InMemoryRepository) SeedTargetOwner(targetType ReportTargetType, targetID string, ownerUserID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	owners, ok := r.targetOwners[targetType]
	if !ok {
		owners = make(map[string]string)
		r.targetOwners[targetType] = owners
	}
	owners[targetID] = ownerUserID
}
