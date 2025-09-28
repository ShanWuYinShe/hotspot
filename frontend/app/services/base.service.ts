/**
 * 基础服务类 - 提供统一的 API 接口
 * 遵循现代化服务设计模式
 */

import type { FetchOptions } from 'ofetch'

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  limit: number
  has_more: boolean
  next_cursor?: string
}

export interface ApiResponse<T = any> {
  success: boolean
  data: T
  message?: string
  code?: string
}

export abstract class BaseService {
  protected http = useHttp()
  protected abstract readonly baseUrl: string

  /**
   * 构建完整 URL
   */
  protected buildUrl(endpoint: string): string {
    return `${this.baseUrl}${endpoint}`
  }

  /**
   * GET 请求
   */
  protected async get<T>(endpoint: string, options?: FetchOptions): Promise<T> {
    return this.http.get<T>(this.buildUrl(endpoint), {
      auth: true,
      ...options
    })
  }

  /**
   * POST 请求
   */
  protected async post<T>(endpoint: string, body?: any, options?: FetchOptions): Promise<T> {
    return this.http.post<T>(this.buildUrl(endpoint), body, {
      auth: true,
      ...options
    })
  }

  /**
   * PUT 请求
   */
  protected async put<T>(endpoint: string, body?: any, options?: FetchOptions): Promise<T> {
    return this.http.put<T>(this.buildUrl(endpoint), body, {
      auth: true,
      ...options
    })
  }

  /**
   * DELETE 请求
   */
  protected async delete<T>(endpoint: string, options?: FetchOptions): Promise<T> {
    return this.http.delete<T>(this.buildUrl(endpoint), {
      auth: true,
      ...options
    })
  }

  /**
   * PATCH 请求
   */
  protected async patch<T>(endpoint: string, body?: any, options?: FetchOptions): Promise<T> {
    return this.http.patch<T>(this.buildUrl(endpoint), body, {
      auth: true,
      ...options
    })
  }

  /**
   * 公开 GET 请求（无需认证）
   */
  protected async publicGet<T>(endpoint: string, options?: FetchOptions): Promise<T> {
    return this.http.get<T>(this.buildUrl(endpoint), {
      auth: false,
      ...options
    })
  }

  /**
   * 公开 POST 请求（无需认证）
   */
  protected async publicPost<T>(endpoint: string, body?: any, options?: FetchOptions): Promise<T> {
    return this.http.post<T>(this.buildUrl(endpoint), body, {
      auth: false,
      ...options
    })
  }

  /**
   * 分页查询
   */
  protected async paginate<T>(
    endpoint: string,
    params: {
      page?: number
      limit?: number
      cursor?: string
      [key: string]: any
    } = {}
  ): Promise<PaginatedResponse<T>> {
    const query = new URLSearchParams()
    
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        query.append(key, String(value))
      }
    })

    const url = query.toString() ? `${endpoint}?${query}` : endpoint
    return this.get<PaginatedResponse<T>>(url)
  }

  /**
   * 上传文件
   */
  protected async upload<T>(
    endpoint: string,
    file: File,
    additionalData?: Record<string, any>
  ): Promise<T> {
    const formData = new FormData()
    formData.append('file', file)
    
    if (additionalData) {
      Object.entries(additionalData).forEach(([key, value]) => {
        formData.append(key, String(value))
      })
    }

    return this.http.post<T>(this.buildUrl(endpoint), formData, {
      auth: true,
      headers: {
        // 让浏览器自动设置 Content-Type
      }
    })
  }

  /**
   * 批量操作
   */
  protected async batch<T>(
    endpoint: string,
    operations: Array<{
      method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
      url: string
      body?: any
    }>
  ): Promise<T[]> {
    return this.post<T[]>(endpoint, { operations })
  }
}
