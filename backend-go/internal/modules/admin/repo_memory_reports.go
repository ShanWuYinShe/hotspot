package admin

import (
	"context"

	"hotspot/backend-go/internal/platform/types"
	"sort"

	"time"
)

func cloneAdminReportItem(item AdminReportItem) AdminReportItem {
	cloned := item
	if item.HandledBy != nil {
		handledBy := *item.HandledBy
		cloned.HandledBy = &handledBy
	}
	return cloned
}

func cloneReportHandleRecord(record reportHandleRecord) reportHandleRecord {
	cloned := record
	cloned.Item = cloneAdminReportItem(record.Item)
	return cloned
}

func (r *InMemoryRepository) SearchReports(_ context.Context, req AdminReportListRequest) (adminReportPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cursor, apiErr := decodeAdminReportCursor(types.PaginationCursor(req.Pagination))
	if apiErr != nil {
		return adminReportPage{}, apiErr
	}
	items := make([]AdminReportItem, 0, len(r.reports))
	for _, record := range r.reports {
		if req.Status != "" && record.Item.Status != req.Status {
			continue
		}
		if req.TargetType != "" && record.Item.TargetType != req.TargetType {
			continue
		}
		items = append(items, cloneAdminReportItem(record.Item))
	}
	sort.Slice(items, func(i, j int) bool {
		it, jt := timeFromRFC3339(items[i].CreatedAt), timeFromRFC3339(items[j].CreatedAt)
		if it.Equal(jt) {
			return items[i].ReportID > items[j].ReportID
		}
		return it.After(jt)
	})
	filtered := make([]AdminReportItem, 0, len(items))
	for _, item := range items {
		if types.PaginationCursor(req.Pagination) != "" {
			createdAt := timeFromRFC3339(item.CreatedAt)
			if createdAt.After(cursor.CreatedAt) {
				continue
			}
			if createdAt.Equal(cursor.CreatedAt) && item.ReportID >= cursor.ReportID {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	limit := types.NormalizeLimit(types.PaginationLimit(req.Pagination))
	paged, hasMore := limitPageItems(filtered, limit)
	page := adminReportPage{Items: append(make([]AdminReportItem, 0, len(paged)), paged...)}
	if hasMore {
		last := paged[len(paged)-1]
		page.Next = &adminReportCursor{CreatedAt: timeFromRFC3339(last.CreatedAt), ReportID: last.ReportID}
	}
	return page, nil
}

func (r *InMemoryRepository) GetReportHandleRecord(_ context.Context, reportID string) (reportHandleRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.reports[reportID]
	if !ok {
		return reportHandleRecord{}, false, nil
	}
	return cloneReportHandleRecord(record), true, nil
}

func (r *InMemoryRepository) HandleReport(_ context.Context, reportID string, action ReportAction, note string, handledBy *UserInfo) (AdminReportItem, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.reports[reportID]
	if !ok {
		return AdminReportItem{}, false, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	switch action {
	case ReportActionResolve:
		record.Item.Status = ReportStatusResolved
	case ReportActionDismiss:
		record.Item.Status = ReportStatusDismissed
	}
	record.Item.HandledAt = &now
	record.Item.HandleNote = &note
	if handledBy != nil {
		h := *handledBy
		record.Item.HandledBy = &h
	}
	r.reports[reportID] = record
	return cloneAdminReportItem(record.Item), true, nil
}

func (r *InMemoryRepository) RecallGroupMessage(_ context.Context, messageID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	msg, ok := r.groupMessages[messageID]
	if !ok {
		return nil
	}
	msg.Recalled = true
	r.groupMessages[messageID] = msg
	return nil
}

func (r *InMemoryRepository) RecallPrivateMessage(_ context.Context, messageID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	msg, ok := r.privateMessages[messageID]
	if !ok {
		return nil
	}
	msg.Recalled = true
	r.privateMessages[messageID] = msg
	return nil
}

func (r *InMemoryRepository) DeleteMoment(_ context.Context, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.moments, momentID)
	return nil
}

func (r *InMemoryRepository) DeleteComment(_ context.Context, commentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.comments, commentID)
	return nil
}

func (r *InMemoryRepository) CreateMoment(_ context.Context, id, content, authorID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.moments[id] = momentRecord{
		ID:        id,
		Content:   content,
		AuthorID:  authorID,
		CreatedAt: time.Now().UTC(),
	}
}

func (r *InMemoryRepository) CreateComment(_ context.Context, id, content, authorID, momentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.comments[id] = commentRecord{
		ID:        id,
		Content:   content,
		AuthorID:  authorID,
		MomentID:  momentID,
		CreatedAt: time.Now().UTC(),
	}
}

func (r *InMemoryRepository) SendGroupMessage(_ context.Context, id, content, senderID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.groupMessages[id] = messageRecord{
		ID:        id,
		Content:   content,
		SenderID:  senderID,
		CreatedAt: time.Now().UTC(),
	}
}

func (r *InMemoryRepository) HasMoment(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.moments[id]
	return ok
}

func (r *InMemoryRepository) HasComment(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.comments[id]
	return ok
}

func (r *InMemoryRepository) IsGroupMessageRecalled(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	msg, ok := r.groupMessages[id]
	if !ok {
		return false
	}
	return msg.Recalled
}
