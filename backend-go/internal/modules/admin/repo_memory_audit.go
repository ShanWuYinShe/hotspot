package admin

import (
	"context"
	"fmt"
	"hotspot/backend-go/internal/platform/types"
	"sort"

	"time"
)

func (r *InMemoryRepository) ListAuditLogs(_ context.Context, filter AuditLogFilter, includeIPAddress bool) (auditLogPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cursor, apiErr := decodeAuditCursor(filter.Pagination.Cursor)
	if apiErr != nil {
		return auditLogPage{}, apiErr
	}
	logs := r.filteredAuditLogsLocked(filter)
	limit := types.NormalizeLimit(filter.Pagination.Limit)
	page := auditLogPage{Items: make([]AuditLogItem, 0, limit)}
	seenCursor := filter.Pagination.Cursor == ""
	for _, entry := range logs {
		if !seenCursor {
			if entry.CreatedAt.After(cursor.CreatedAt) {
				continue
			}
			if entry.CreatedAt.Equal(cursor.CreatedAt) && entry.LogID >= cursor.LogID {
				continue
			}
			seenCursor = true
		}
		item := auditLogItemFromEntry(entry, includeIPAddress)
		if len(page.Items) >= limit {
			page.Next = &auditCursor{CreatedAt: entry.CreatedAt, LogID: entry.LogID}
			break
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

func (r *InMemoryRepository) AppendAuditLog(_ context.Context, entry AuditLogEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry.LogID == "" {
		r.seq++
		entry.LogID = fmt.Sprintf("audit_%d_%d", time.Now().UTC().UnixNano(), r.seq)
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	r.auditLogs[entry.LogID] = cloneAuditEntry(entry)
	r.auditOrder = append(r.auditOrder, entry.LogID)
	sort.SliceStable(r.auditOrder, func(i, j int) bool {
		left := r.auditLogs[r.auditOrder[i]]
		right := r.auditLogs[r.auditOrder[j]]
		if left.CreatedAt.Equal(right.CreatedAt) {
			return left.LogID > right.LogID
		}
		return left.CreatedAt.After(right.CreatedAt)
	})
	return nil
}

func (r *InMemoryRepository) filteredAuditLogsLocked(filter AuditLogFilter) []AuditLogEntry {
	items := make([]AuditLogEntry, 0, len(r.auditOrder))
	for _, logID := range r.auditOrder {
		entry := r.auditLogs[logID]
		if filter.Action != "" && entry.Action != filter.Action {
			continue
		}
		if filter.OperatorID != "" && entry.Operator.UserID != filter.OperatorID {
			continue
		}
		if filter.TargetType != "" && entry.TargetType != filter.TargetType {
			continue
		}
		if filter.TargetID != "" && entry.TargetID != filter.TargetID {
			continue
		}
		if filter.StartTime != nil && entry.CreatedAt.Before(*filter.StartTime) {
			continue
		}
		if filter.EndTime != nil && entry.CreatedAt.After(*filter.EndTime) {
			continue
		}
		items = append(items, cloneAuditEntry(entry))
	}
	return items
}

func auditLogItemFromEntry(entry AuditLogEntry, includeIPAddress bool) AuditLogItem {
	item := AuditLogItem{
		LogID:         entry.LogID,
		Action:        entry.Action,
		Operator:      entry.Operator,
		TargetType:    entry.TargetType,
		TargetID:      entry.TargetID,
		TargetName:    entry.TargetName,
		Details:       cloneAuditDetails(entry.Details),
		IPLocation:    entry.IPLocation,
		DeviceSummary: entry.DeviceSummary,
		CreatedAt:     entry.CreatedAt.UTC().Format(time.RFC3339),
	}
	if includeIPAddress && entry.IPAddress != "" {
		ip := entry.IPAddress
		item.IPAddress = &ip
	}
	return item
}

func cloneAuditEntry(entry AuditLogEntry) AuditLogEntry {
	entry.Details = cloneAuditDetails(entry.Details)
	return entry
}

func cloneAuditDetails(details AuditDetails) AuditDetails {
	if len(details) == 0 {
		return nil
	}
	cloned := make(AuditDetails, len(details))
	for key, value := range details {
		cloned[key] = value
	}
	return cloned
}

func timeFromRFC3339(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func (r *InMemoryRepository) seedAuditLogs() {
	now := time.Now().UTC()
	logs := []AuditLogEntry{
		{
			LogID:      "audit_seed_002",
			Action:     AuditActionConfigUpdate,
			Operator:   UserInfo{UserID: "admin", Nickname: "admin"},
			TargetType: AuditTargetTypeConfig,
			TargetID:   "response_defaults.chat_recent_messages_count",
			TargetName: "chat_recent_messages_count",
			Details: AuditDetails{
				"type":       string(AuditActionConfigUpdate),
				"config_key": "response_defaults.chat_recent_messages_count",
				"old_value":  20,
				"new_value":  30,
			},
			IPLocation:    "CN-ZJ-Hangzhou",
			IPAddress:     "203.0.113.10",
			DeviceSummary: "Chrome on macOS",
			CreatedAt:     now.Add(-1 * time.Hour),
		},
		{
			LogID:      "audit_seed_001",
			Action:     AuditActionAdminAward,
			Operator:   UserInfo{UserID: "admin", Nickname: "admin"},
			TargetType: AuditTargetTypeAdmin,
			TargetID:   "user_alice",
			TargetName: "Alice",
			Details: AuditDetails{
				"type":                  string(AuditActionAdminAward),
				"target_user_id":        "user_alice",
				"role_ids":              []string{"role_ops"},
				"award_all_permissions": false,
			},
			IPLocation:    "CN-BJ-Beijing",
			IPAddress:     "198.51.100.20",
			DeviceSummary: "Safari on iPhone",
			CreatedAt:     now.Add(-2 * time.Hour),
		},
	}
	for _, log := range logs {
		r.auditLogs[log.LogID] = cloneAuditEntry(log)
		r.auditOrder = append(r.auditOrder, log.LogID)
	}
}
