package admin

import (
	"time"

	"hotspot/backend-go/internal/platform/types"
)

// auditLogItemFromEntry 把审计日志行转换为对外 item（PG 与测试内存实现共用；
// 内存实现已迁入测试构建，转换口径必须留在生产代码）。
func auditLogItemFromEntry(entry AuditLogEntry, includeIPAddress bool) AuditLogItem {
	item := AuditLogItem{
		LogID:         entry.LogID,
		Action:        entry.Action,
		Operator:      entry.Operator,
		TargetType:    entry.TargetType,
		TargetID:      entry.TargetID,
		TargetName:    types.StrPtr(entry.TargetName),
		Details:       cloneAuditDetails(entry.Details),
		IPLocation:    types.StrPtr(entry.IPLocation),
		DeviceSummary: types.StrPtr(entry.DeviceSummary),
		CreatedAt:     entry.CreatedAt.UTC().Format(time.RFC3339),
	}
	if includeIPAddress && entry.IPAddress != "" {
		ip := entry.IPAddress
		item.IPAddress = &ip
	}
	return item
}

// cloneAuditDetails 深拷贝审计详情 map（避免调用方/存储共享可变引用）。
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
