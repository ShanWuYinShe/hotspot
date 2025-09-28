/**
 * 系统管理服务 - 管理员功能
 * 提供系统健康检查、统计数据、管理操作等功能
 */

import { ADMIN_ENDPOINTS, buildAdminUrl } from "~/config/api"
import { withApiErrorHandling } from '~/utils/api-error'
import type {
  SystemHealthResponse,
  SystemStatsResponse,
  UserManagementRequest,
  UserManagementResponse,
  ContentModerationRequest,
  ContentModerationResponse
} from "~/types/api"

export const useSystemService = () => {
  const api = useApi()

  /**
   * 获取系统健康状态
   */
  const getHealthStatus = async (): Promise<SystemHealthResponse> => {
    return withApiErrorHandling(
      async () => {
        const response = await api.post(
          buildAdminUrl(ADMIN_ENDPOINTS.PROTECTED.HEALTH_CHECK),
          {},
          { auth: true }
        )
        return response as SystemHealthResponse
      },
      {
        operation: 'getHealthStatus',
        service: 'SystemService',
        fallbackMessage: '获取系统状态失败'
      }
    )
  }

  /**
   * 获取系统统计数据
   */
  const getSystemStats = async (): Promise<SystemStatsResponse> => {
    return withApiErrorHandling(
      async () => {
        const response = await api.post(
          buildAdminUrl(ADMIN_ENDPOINTS.PROTECTED.STATS),
          {},
          { auth: true }
        )
        return response as SystemStatsResponse
      },
      {
        operation: 'getSystemStats',
        service: 'SystemService',
        fallbackMessage: '获取系统统计失败'
      }
    )
  }

  /**
   * 用户管理操作
   */
  const manageUser = async (request: UserManagementRequest): Promise<UserManagementResponse> => {
    return withApiErrorHandling(
      async () => {
        const response = await api.post(
          buildAdminUrl(ADMIN_ENDPOINTS.PROTECTED.USER_MANAGEMENT),
          request,
          { auth: true }
        )
        return response as UserManagementResponse
      },
      {
        operation: 'manageUser',
        service: 'SystemService',
        fallbackMessage: '用户管理操作失败'
      }
    )
  }

  /**
   * 内容审核操作
   */
  const moderateContent = async (request: ContentModerationRequest): Promise<ContentModerationResponse> => {
    return withApiErrorHandling(
      async () => {
        const response = await api.post(
          buildAdminUrl(ADMIN_ENDPOINTS.PROTECTED.CONTENT_MODERATION),
          request,
          { auth: true }
        )
        return response as ContentModerationResponse
      },
      {
        operation: 'moderateContent',
        service: 'SystemService',
        fallbackMessage: '内容审核操作失败'
      }
    )
  }

  /**
   * 导出系统报告
   */
  const exportSystemReport = async (format: 'json' | 'csv' | 'pdf' = 'json'): Promise<Blob> => {
    return withApiErrorHandling(
      async () => {
        const response = await api.post(
          buildAdminUrl(ADMIN_ENDPOINTS.PROTECTED.EXPORT_REPORT),
          { format },
          { 
            auth: true,
            responseType: 'blob'
          }
        )
        return response as Blob
      },
      {
        operation: 'exportSystemReport',
        service: 'SystemService',
        fallbackMessage: '导出系统报告失败'
      }
    )
  }

  /**
   * 清理系统缓存
   */
  const clearSystemCache = async (): Promise<void> => {
    return withApiErrorHandling(
      async () => {
        await api.post(
          buildAdminUrl(ADMIN_ENDPOINTS.PROTECTED.CLEAR_CACHE),
          {},
          { auth: true }
        )
      },
      {
        operation: 'clearSystemCache',
        service: 'SystemService',
        fallbackMessage: '清理系统缓存失败'
      }
    )
  }

  return {
    getHealthStatus,
    getSystemStats,
    manageUser,
    moderateContent,
    exportSystemReport,
    clearSystemCache
  }
}
