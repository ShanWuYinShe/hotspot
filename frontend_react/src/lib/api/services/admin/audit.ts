import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { AuditAction, AuditLogDetail, AuditLogExportResult, AuditLogItem, AuditTargetType, PaginatedData } from '@/types/api';

export const AdminAuditService = {
  list: (params?: {
    cursor?: string;
    limit?: number;
    action?: AuditAction;
    operator_id?: number;
    target_type?: AuditTargetType;
    target_id?: string;
    start_time?: string;
    end_time?: string;
  }) => apiClient.post<PaginatedData<AuditLogItem>>(ENDPOINTS.ADMIN.AUDIT.LIST, params ?? {}),

  detail: (logId: string) =>
    apiClient.post<AuditLogDetail>(ENDPOINTS.ADMIN.AUDIT.DETAIL, { log_id: logId }),

  export: (params?: {
    action?: AuditAction;
    operator_id?: number;
    target_type?: AuditTargetType;
    target_id?: string;
    start_time?: string;
    end_time?: string;
  }) => apiClient.post<AuditLogExportResult>(ENDPOINTS.ADMIN.AUDIT.EXPORT, params ?? {}),

  exportStatus: (exportId: string) =>
    apiClient.post<AuditLogExportResult>(ENDPOINTS.ADMIN.AUDIT.EXPORT_STATUS, { export_id: exportId }),
};
