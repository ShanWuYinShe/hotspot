import { BaseService } from './base.service'
import { REPORTS } from '~/config/api'
import type {
  PaginatedData,
  CreateReportRequest,
  CreateReportData,
  GetMyReportsRequest,
  ReportItem
} from '~/types/api'

class ReportsService extends BaseService {
  async create(request: CreateReportRequest) {
    return this.post<CreateReportData>(REPORTS.CREATE, request)
  }

  async getMyReports(request: GetMyReportsRequest = {}) {
    return this.post<PaginatedData<ReportItem>>(REPORTS.ME_GET_LIST, request)
  }
}

export const reportsService = new ReportsService()
export function useReportsService() { return reportsService }
