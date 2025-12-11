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
   * 获取 API 基础 URL
   */
  private get baseURL(): string {
    const config = useRuntimeConfig()
    return config.public.apiBaseUrl as string
  }

  /**
   * 获取认证 Token
   */
  private get token(): string | null {
    const authStore = useAuthStore()
    return authStore.token
  }

  /**
   * 获取刷新 Token
   */
  private get refreshToken(): string | null {
    const authStore = useAuthStore()
    return authStore.refreshToken
  }

  /**
   * 处理 API 响应
   */
  private processResponse<T>(response: unknown): T {
    if (typeof response === 'object' && response !== null && 'code' in response) {
      const apiResponse = response as ApiResponse<T>
      if (apiResponse.code === 0) {
        return apiResponse.data as T
      }
      // 创建带有业务错误信息的 Error
      throw Object.assign(new Error(apiResponse.message || '操作失败'), {
        isBusinessError: true,
        businessCode: apiResponse.code,
        data: apiResponse
      })
    }
    return response as T
  }

  /**
   * POST 请求（需要认证，支持自动刷新 token）
   */
  protected async post<T>(url: string, body?: object): Promise<T> {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' }
    if (this.token) {
      headers.Authorization = `Bearer ${this.token}`
    }

    try {
      const response = await $fetch<ApiResponse<T>>(url, {
        method: 'POST',
        baseURL: this.baseURL,
        headers,
        body
      })
      return this.processResponse<T>(response)
    } catch (error) {
      // Token 过期，尝试刷新后重试
      if (isTokenExpiredError(error) && this.refreshToken) {
        const refreshed = await ensureRefreshToken()
        if (refreshed) {
          // 使用新 token 重试请求
          const newHeaders: Record<string, string> = { 'Content-Type': 'application/json' }
          const newToken = this.token
          if (newToken) {
            newHeaders.Authorization = `Bearer ${newToken}`
          }
          const response = await $fetch<ApiResponse<T>>(url, {
            method: 'POST',
            baseURL: this.baseURL,
            headers: newHeaders,
            body
          })
          return this.processResponse<T>(response)
        }
      }
      throw error
    }
  }

  /**
   * 公开 POST 请求（无需认证）
   */
  protected async publicPost<T>(url: string, body?: object): Promise<T> {
    const response = await $fetch<ApiResponse<T>>(url, {
      method: 'POST',
      baseURL: this.baseURL,
      headers: { 'Content-Type': 'application/json' },
      body
    })
    return this.processResponse<T>(response)
  }

  /**
   * 分页查询
   */
  protected async paginate<T, P extends Record<string, unknown> = Record<string, unknown>>(
    url: string,
    params: { limit?: number; cursor?: string } & P = {} as { limit?: number; cursor?: string } & P
  ): Promise<PaginatedData<T>> {
    return this.post<PaginatedData<T>>(url, params)
  }

  /**
   * 上传文件
   */
  protected async upload<T>(
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

    const headers: Record<string, string> = {}
    if (this.token) {
      headers.Authorization = `Bearer ${this.token}`
    }

    const response = await $fetch<ApiResponse<T>>(url, {
      method: 'POST',
      baseURL: this.baseURL,
      headers,
      body: formData
    })

    return this.processResponse<T>(response)
  }
}
