/**
 * 举报服务
 */

import { get, post, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PaginatedData,
  CreateReportRequest,
  ReportItem,
} from '@/types/api';

export const reportsService = {
  /**
   * 创建举报
   */
  async create(data: CreateReportRequest): Promise<ApiResponse<{ report_id: string }>> {
    return post<{ report_id: string }>(ENDPOINTS.REPORTS.CREATE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 获取我的举报列表
   */
  async getMyReports(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<ReportItem>>> {
    return get<PaginatedData<ReportItem>>(ENDPOINTS.REPORTS.MY_REPORTS, params);
  },
};
