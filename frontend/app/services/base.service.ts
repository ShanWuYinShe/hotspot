/**
 * 基础服务类 - 提供统一的 API 接口
 * 遵循现代化服务设计模式
 */

import type { FetchOptions } from 'ofetch'
import type { PaginatedResponse, ApiResponse } from '~/types/api'

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
   * POST 请求（需要认证）
   * 根据API规范，所有API操作统一使用POST方法
   */
  protected async post<T>(endpoint: string, body?: any, options?: FetchOptions): Promise<T> {
    const builder = this.http.post(this.buildUrl(endpoint), body).auth()
    
    // 应用额外选项
    if (options?.timeout) {
      builder.timeout(options.timeout)
    }
    if (options?.retry) {
      builder.retry(options.retry)
    }
    if (options?.headers) {
      builder.headers(options.headers)
    }
    
    const response = await builder.send<T>()
    return response.data
  }

  /**
   * 公开 POST 请求（无需认证）
   * 根据API规范，所有API操作统一使用POST方法
   */
  protected async publicPost<T>(endpoint: string, body?: any, options?: FetchOptions): Promise<T> {
    const builder = this.http.post(this.buildUrl(endpoint), body)
    
    // 应用额外选项（公开接口不需要auth）
    if (options?.timeout) {
      builder.timeout(options.timeout)
    }
    if (options?.retry) {
      builder.retry(options.retry)
    }
    if (options?.headers) {
      builder.headers(options.headers)
    }
    
    const response = await builder.send<T>()
    return response.data
  }

  /**
   * 分页查询
   * 根据API规范，分页参数通过POST请求体传递
   */
  protected async paginate<T>(
    endpoint: string,
    params: {
      limit?: number
      cursor?: string
      [key: string]: any
    } = {}
  ): Promise<PaginatedResponse<T>> {
    return this.post<PaginatedResponse<T>>(endpoint, params)
  }

  /**
   * 上传单个文件
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

    const builder = this.http.post(this.buildUrl(endpoint), formData).auth()
    // 让浏览器自动设置 Content-Type（不手动设置headers）
    
    const response = await builder.send<T>()
    return response.data
  }

  /**
   * 批量上传文件
   */
  protected async uploadMultiple<T>(
    endpoint: string,
    files: File[],
    fieldName: string = 'files',
    additionalData?: Record<string, any>
  ): Promise<T> {
    const formData = new FormData()
    
    // 添加多个文件
    files.forEach(file => {
      formData.append(fieldName, file)
    })
    
    // 添加额外数据
    if (additionalData) {
      Object.entries(additionalData).forEach(([key, value]) => {
        formData.append(key, String(value))
      })
    }

    const builder = this.http.post(this.buildUrl(endpoint), formData).auth()
    // 让浏览器自动设置 Content-Type（不手动设置headers）
    
    const response = await builder.send<T>()
    return response.data
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
