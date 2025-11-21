/**
 * 举报服务 - 内容安全和用户举报管理
 * 提供举报创建、查询、管理等功能
 * 
 * 符合 API 规范 3.11 节
 */

import { BaseService } from './base.service'
import type {
  CreateReportRequest,
  CreateReportData,
  ListUserReportsRequest,
  ReportItem,
  PaginatedData
} from '~/types/api'

/**
 * 举报服务类
 */
export class ReportsService extends BaseService {
  constructor() {
    super('/api/v1/protected/reports')
  }

  /**
   * 创建举报
   * 3.11.1. POST /api/v1/protected/reports/create - 创建举报
   */
  async create(request: CreateReportRequest): Promise<CreateReportData> {
    return this.post<CreateReportData>('/create', request)
  }

  /**
   * 获取我的举报记录
   * 3.11.2. POST /api/v1/protected/reports/get_my_reports - 我的举报记录
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
