/**
 * 举报服务 - 内容安全和用户举报管理
 * 提供举报创建、查询、管理等功能
 */

import { REPORT_ENDPOINTS, buildReportUrl } from "~/config/api"
import { withApiErrorHandling } from '~/utils/api-error'
import type {
  CreateReportRequest,
  CreateReportResponse,
  GetReportsRequest,
  PaginatedResponse,
  ReportResponse,
  ReportStatus,
  ReportType
} from "~/types/api"

export const useReportsService = () => {
  const api = useApi()

  /**
   * 创建举报
   */
  const createReport = async (request: {
    target_type: ReportType
    target_id: string
    reason: string
    description?: string
  }): Promise<CreateReportResponse> => {
    return withApiErrorHandling(
      async () => {
        const reportRequest: CreateReportRequest = {
          target_type: request.target_type,
          target_id: request.target_id,
          reason: request.reason,
          description: request.description
        }

        const response = await api.post(
          buildReportUrl(REPORT_ENDPOINTS.PROTECTED.CREATE),
          reportRequest,
          { auth: true }
        )

        return response as CreateReportResponse
      },
      {
        operation: 'createReport',
        service: 'ReportsService',
        fallbackMessage: '提交举报失败'
      }
    )
  }

  /**
   * 获取我的举报记录
   */
  const getMyReports = async (params: {
    cursor?: string
    limit?: number
    status?: ReportStatus
  } = {}): Promise<PaginatedResponse<ReportResponse>> => {
    return withApiErrorHandling(
      async () => {
        const request: GetReportsRequest = {
          cursor: params.cursor,
          limit: params.limit || 20,
          status: params.status
        }

        const response = await api.post(
          buildReportUrl(REPORT_ENDPOINTS.PROTECTED.MY_REPORTS),
          request,
          { auth: true }
        )

        return response as PaginatedResponse<ReportResponse>
      },
      {
        operation: 'getMyReports',
        service: 'ReportsService',
        fallbackMessage: '获取举报记录失败'
      }
    )
  }

  /**
   * 撤销举报
   */
  const cancelReport = async (reportId: string): Promise<void> => {
    return withApiErrorHandling(
      async () => {
        await api.post(
          buildReportUrl(REPORT_ENDPOINTS.PROTECTED.CANCEL),
          { report_id: reportId },
          { auth: true }
        )
      },
      {
        operation: 'cancelReport',
        service: 'ReportsService',
        fallbackMessage: '撤销举报失败'
      }
    )
  }

  /**
   * 获取举报详情
   */
  const getReportDetails = async (reportId: string): Promise<ReportResponse> => {
    return withApiErrorHandling(
      async () => {
        const response = await api.post(
          buildReportUrl(REPORT_ENDPOINTS.PROTECTED.GET_DETAILS),
          { report_id: reportId },
          { auth: true }
        )

        return response as ReportResponse
      },
      {
        operation: 'getReportDetails',
        service: 'ReportsService',
        fallbackMessage: '获取举报详情失败'
      }
    )
  }

  return {
    createReport,
    getMyReports,
    cancelReport,
    getReportDetails
  }
}
