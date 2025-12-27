import { BaseService } from './base.service'
import { REPORTS } from '~/config/api'
import type {
  PaginatedData,
  CreateReportRequest,
  CreateReportData,
  MyReportsRequest,
  ReportItem,
  ReportsRequest,
  HandleReportRequest
} from '~/types/api'

class ReportsService extends BaseService {
  async create(request: CreateReportRequest) {
    return this.post<CreateReportData>(REPORTS.CREATE, request)
  }

  async getMyReports(request: MyReportsRequest = {}) {
    return this.post<PaginatedData<ReportItem>>(REPORTS.ME, request)
  }

  // ============ Admin ============
  async adminList(request: ReportsRequest = {}) {
    return this.post<PaginatedData<ReportItem>>(REPORTS.ADMIN_LIST, request)
  }

  async adminHandle(request: HandleReportRequest) {
    return this.post<void>(REPORTS.ADMIN_HANDLE, request)
  }
}

export const reportsService = new ReportsService()
export function useReportsService() { return reportsService }
