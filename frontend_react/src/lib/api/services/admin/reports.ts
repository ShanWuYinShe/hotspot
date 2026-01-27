import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { PaginatedData, ReportHandleAction, ReportItem, ReportStatus, ReportTargetType } from '@/types/api';

export const AdminReportService = {
  list: (params?: { cursor?: string; limit?: number; status?: ReportStatus; target_type?: ReportTargetType }) =>
    apiClient.post<PaginatedData<ReportItem>>(ENDPOINTS.ADMIN.REPORTS.LIST, params ?? {}),

  handle: (data: { report_id: string; action: ReportHandleAction; note?: string }) =>
    apiClient.post(ENDPOINTS.ADMIN.REPORTS.HANDLE, data),
};
