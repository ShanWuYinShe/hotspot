/**
 * 基础服务类 - 提供统一的 HTTP 请求方法
 * 
 * 特性：
 * - 自动刷新过期的 access_token
 * - 统一响应处理
 * - 统一错误处理
 */

import type { PaginatedData, ApiResponse } from '~/types/api'
import { ensureRefreshToken, isTokenExpiredError } from '~/utils/token'

export abstract class BaseService {
  /**
   * 获取认证 Token
   */
  private get token(): string | null {
    return useAuthStore().token
  }

  /**
   * 核心请求方法
   * 封装了 Token 注入、错误处理和自动重试逻辑
   */
  private async request<T>(url: string, options: any = {}): Promise<T> {
    const performRequest = async (forceToken?: string) => {
      const token = forceToken ?? this.token
      const headers = { ...options.headers }
      
      if (token) {
        headers.Authorization = `Bearer ${token}`
      }

      const response = await $fetch<ApiResponse<T>>(url, {
        // Web: 需要携带 HttpOnly refresh cookie（用于 refresh_token、logout 等）
        credentials: 'include',
        ...options,
        headers
      })

      // 统一响应处理
      if (typeof response === 'object' && response !== null && 'code' in response) {
        if (response.code === 0) {
          return response.data as T
        }
        // 业务错误
        throw Object.assign(new Error(response.message || '操作失败'), {
          isBusinessError: true,
          businessCode: response.code,
          data: response
        })
      }
      return response as T
    }

    try {
      return await performRequest()
    } catch (error) {
      // Token 过期处理
      // 前置条件：本次请求确实带了 access_token（未登录/缺 token 的 401 不应触发 refresh）
      if (this.token && isTokenExpiredError(error)) {
        const refreshed = await ensureRefreshToken()
        if (refreshed) {
          // 使用新 Token 重试
          return await performRequest(useAuthStore().token!)
        }
      }
      throw error
    }
  }

  /**
   * POST 请求
   */
  protected post<T>(url: string, body?: object): Promise<T> {
    return this.request<T>(url, {
      method: 'POST',
      body
    })
  }

  /**
   * 公开 POST 请求（无需认证）
   * 
   * 虽然 request 方法会自动处理 token，但显式分开可以提高代码可读性
   * 且此处不强制 token 存在，request 内部 logic 兼容 token 为空的情况
   */
  protected publicPost<T>(url: string, body?: object): Promise<T> {
    return this.request<T>(url, {
      method: 'POST',
      body
    })
  }

  /**
   * 分页查询
   */
  protected paginate<T>(
    url: string,
    params: { limit?: number; cursor?: string } & Record<string, any> = {}
  ): Promise<PaginatedData<T>> {
    return this.post<PaginatedData<T>>(url, params)
  }

  /**
   * 上传文件
   */
  protected upload<T>(
    url: string,
    file: File,
    additionalData?: Record<string, string | number | boolean>
  ): Promise<T> {
    const formData = new FormData()
    formData.append('file', file)

    if (additionalData) {
      Object.entries(additionalData).forEach(([key, value]) => {
        formData.append(key, String(value))
      })
    }

    return this.request<T>(url, {
      method: 'POST',
      body: formData
    })
  }
}
