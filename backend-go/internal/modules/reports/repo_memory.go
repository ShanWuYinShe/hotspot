package reports

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/types"
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

func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateReportRequest, messageSnapshot *ReportSnapshot) (ReportCreated, error) {
	now := time.Now().UTC()
	r.seq++
	report := ReportItem{ReportID: fmt.Sprintf("report_%d_%d", now.UnixNano(), r.seq), TargetType: req.TargetType, TargetID: req.TargetID, Reason: req.Reason, Description: req.Description, Status: ReportStatusPending, CreatedAt: now.Format(time.RFC3339), MessageSnapshot: messageSnapshot}
	r.mu.Lock()
	r.reports[userID] = append(r.reports[userID], report)
	r.mu.Unlock()
	return ReportCreated{ReportID: report.ReportID}, nil
}

func (r *InMemoryRepository) List(_ context.Context, userID string, req ListReportsRequest) (PaginatedReports, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// 与 postgres 语义对齐（spec api_spec_report.md）：created_at DESC，
	// report_id DESC 稳定 tie-breaker；keyset 游标分页（api_spec_base.md）。
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
	sort.Slice(items, func(i, j int) bool {
		it, jt := timeFromRFC3339(items[i].CreatedAt), timeFromRFC3339(items[j].CreatedAt)
		if it.Equal(jt) {
			return items[i].ReportID > items[j].ReportID
		}
		return it.After(jt)
	})
	cursorAt, cursorID, apiErr := decodeReportListCursor(types.PaginationCursor(req.Pagination))
	if apiErr != nil {
		return PaginatedReports{}, apiErr
	}
	limit := types.NormalizeLimit(types.PaginationLimit(req.Pagination))
	filtered := make([]ReportItem, 0, len(items))
	for _, item := range items {
		if !cursorAt.IsZero() {
			createdAt := timeFromRFC3339(item.CreatedAt)
			if createdAt.After(cursorAt) {
				continue
			}
			if createdAt.Equal(cursorAt) && item.ReportID >= cursorID {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	page := PaginatedReports{Items: filtered}
	if len(filtered) > limit {
		page.Items = filtered[:limit]
		last := page.Items[len(page.Items)-1]
		next := encodeReportListCursor(timeFromRFC3339(last.CreatedAt), last.ReportID)
		if next != "" {
			page.NextCursor = &next
		}
	}
	return page, nil
}

// FindExistingReport 天然幂等去重键（spec api_spec_report.md /reports/create）：
// (reporter_id, target_type, target_id, pending_status)。必须限定举报人——
// 否则任何人对目标的一条 pending 举报会把其他人的新举报判为 duplicate，
// 并把别人的 report_id 泄漏给举报人。
func (r *InMemoryRepository) FindExistingReport(_ context.Context, userID string, targetType ReportTargetType, targetID string) (string, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.reports[userID] {
		if item.TargetType == targetType && item.TargetID == targetID && item.Status == ReportStatusPending {
			return item.ReportID, true, nil
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
