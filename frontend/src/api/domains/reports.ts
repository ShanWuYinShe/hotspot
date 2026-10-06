// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	ReportItem,
	ReportStatus,
	ReportTargetType,
	PaginatedData,
	ReportCreated,
	CreateReportRequest,
	PaginationParams,
} from '@/api/types';

// Reports
export async function createReport(params: CreateReportRequest)        { return request<ReportCreated>(Endpoints.Reports.Create, params); }
// spec: report.md → reports/fetch（我的举报列表，keyset 分页；含处理结果 resolution）
export async function listReports(params?: { status?: ReportStatus | null; target_type?: ReportTargetType | null; pagination?: PaginationParams }) { return request<PaginatedData<ReportItem>>(Endpoints.Reports.List, params ?? {}); }
