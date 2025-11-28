/**
 * 举报服务
 * 提供举报创建、查询、管理等功能
 */

import { BaseService } from './base.service'
import type {
  CreateReportRequest,
  CreateReportItem,
  ReportItem,
  PaginatedData
} from '~/types/api'

// 请求类型
interface ListUserReportsRequest {
  limit?: number
  cursor?: string
}

/**
 * 举报服务类
 */
export class ReportsService extends BaseService {
  protected readonly baseUrl = '/protected/reports'

  /**
   * 创建举报
   */
  async create(request: CreateReportRequest): Promise<CreateReportItem> {
    return this.post<CreateReportItem>('/create', request)
  }

  /**
   * 获取我的举报记录
   */
  async getMyReports(request: ListUserReportsRequest = {}): Promise<PaginatedData<ReportItem>> {
    return this.post<PaginatedData<ReportItem>>('/get_my_reports', request)
  }
}

/**
 * 使用举报服务的 Composable
 */
export function useReportsService() {
  return new ReportsService()
}
