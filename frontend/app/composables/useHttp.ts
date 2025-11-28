/**
 * 现代化 HTTP 客户端 - 基于建造者模式和装饰器模式
 * 提供链式调用和中间件支持的优雅API
 */

import type { ApiResponse } from '~/types/api'

// 使用 Nuxt 内置的 FetchOptions 类型
export interface HttpOptions {
  timeout?: number
  retry?: number
  headers?: Record<string, string>
  [key: string]: any
}

export interface HttpMiddleware {
  readonly name: string
  readonly priority: number
  process(request: HttpRequest): Promise<HttpRequest>
}

export interface HttpRequest {
  url: string
  method: string
  headers: Record<string, string>
  body?: any
  options: HttpOptions
}

export interface HttpResponse<T = any> {
  data: T
  status: number
  headers: Record<string, string>
  ok: boolean
}

/**
 * HTTP 请求建造者
 */
class HttpRequestBuilder {
  private request: HttpRequest
  private middlewares: HttpMiddleware[] = []

  constructor(url: string, method: string = 'GET') {
    this.request = {
      url,
      method: method.toUpperCase(),
      headers: {
        'Content-Type': 'application/json'
      },
      options: {}
    }
  }

  /**
   * 设置请求头
   */
  headers(headers: Record<string, string>): this {
    Object.assign(this.request.headers, headers)
    return this
  }

  /**
   * 设置请求体
   */
  body(body: any): this {
    this.request.body = body
    return this
  }

  /**
   * 设置认证
   */
  auth(token?: string): this {
    if (token) {
      this.request.headers.Authorization = `Bearer ${token}`
    } else {
      const authStore = useAuthStore()
      if (authStore.token) {
        this.request.headers.Authorization = `Bearer ${authStore.token}`
      }
    }
    return this
  }

  /**
   * 设置超时
   */
  timeout(ms: number): this {
    this.request.options.timeout = ms
    return this
  }

  /**
   * 设置重试
   */
  retry(attempts: number): this {
    this.request.options.retry = attempts
    return this
  }

  /**
   * 添加中间件
   */
  use(middleware: HttpMiddleware): this {
    this.middlewares.push(middleware)
    this.middlewares.sort((a, b) => b.priority - a.priority)
    return this
  }

  /**
   * 执行请求
   */
  async send<T = any>(): Promise<HttpResponse<T>> {
    try {
      // 应用中间件
      let processedRequest = { ...this.request }
      for (const middleware of this.middlewares) {
        processedRequest = await middleware.process(processedRequest)
      }

      // 执行请求
      const config = useRuntimeConfig()
      const response = await $fetch<ApiResponse<T>>(processedRequest.url, {
        method: processedRequest.method as any,
        baseURL: config.public.apiBaseUrl as string,
        headers: processedRequest.headers,
        body: processedRequest.body,
        ...processedRequest.options,
        onResponseError: ({ response }) => {
          throw createError({
            statusCode: response.status,
            statusMessage: response.statusText,
            data: response._data
          })
        }
      })

      // 处理业务响应
      return this.processResponse(response)
    } catch (error: any) {
      // ✅ 直接抛出错误，由调用方（Service 层或组件层）处理
      // 这样做的好处：
      // 1. 避免在非 setup 上下文中调用 useToast() 导致的 SSR 错误
      // 2. 让业务层决定如何处理和显示错误
      // 3. 符合单一职责原则：HTTP 客户端只负责通信，不负责 UI 交互
      throw error
    }
  }

  /**
   * 处理 API 响应
   */
  private processResponse<T>(response: ApiResponse<T>): HttpResponse<T> {
    // 标准 API 响应格式处理
    if (typeof response === 'object' && response !== null && 'code' in response) {
      if (response.code === 0) {
        return {
          data: response.data as T,
          status: 200,
          headers: {},
          ok: true
        }
      } else {
        const error = new Error(response.message || '操作失败')
        ;(error as any).isBusinessError = true
        ;(error as any).businessCode = response.code
        ;(error as any).data = response
        throw error
      }
    }

    // 直接返回数据
    return {
      data: response as T,
      status: 200,
      headers: {},
      ok: true
    }
  }
}

/**
 * 认证中间件
 */
export class AuthMiddleware implements HttpMiddleware {
  readonly name = 'AuthMiddleware'
  readonly priority = 100

  async process(request: HttpRequest): Promise<HttpRequest> {
    // 如果已经有认证头，跳过
    if (request.headers.Authorization) {
      return request
    }

    const authStore = useAuthStore()
    if (authStore.token) {
      return {
        ...request,
        headers: {
          ...request.headers,
          Authorization: `Bearer ${authStore.token}`
        }
      }
    }

    return request
  }
}

/**
 * 日志中间件
 */
export class LoggingMiddleware implements HttpMiddleware {
  readonly name = 'LoggingMiddleware'
  readonly priority = 10

  async process(request: HttpRequest): Promise<HttpRequest> {
    if (import.meta.dev) {
      console.log(`🌐 HTTP ${request.method} ${request.url}`, {
        headers: request.headers,
        body: request.body
      })
    }
    return request
  }
}

/**
 * 缓存中间件
 */
export class CacheMiddleware implements HttpMiddleware {
  readonly name = 'CacheMiddleware'
  readonly priority = 90
  private cache = new Map<string, { data: any; timestamp: number; ttl: number }>()

  constructor(private defaultTtl: number = 5 * 60 * 1000) {} // 5分钟默认缓存

  async process(request: HttpRequest): Promise<HttpRequest> {
    // 只缓存 GET 请求
    if (request.method !== 'GET') {
      return request
    }

    const cacheKey = this.getCacheKey(request)
    const cached = this.cache.get(cacheKey)
    
    if (cached && Date.now() - cached.timestamp < cached.ttl) {
      // 返回缓存数据（通过抛出特殊错误来中断请求）
      throw { __cached: true, data: cached.data }
    }

    return request
  }

  private getCacheKey(request: HttpRequest): string {
    return `${request.method}:${request.url}:${JSON.stringify(request.body)}`
  }
}

/**
 * HTTP 客户端工厂
 */
class HttpClientFactory {
  private globalMiddlewares: HttpMiddleware[] = []

  /**
   * 添加全局中间件
   */
  addGlobalMiddleware(middleware: HttpMiddleware): void {
    this.globalMiddlewares.push(middleware)
    this.globalMiddlewares.sort((a, b) => b.priority - a.priority)
  }

  /**
   * 创建请求建造者
   */
  create(url: string, method: string = 'GET'): HttpRequestBuilder {
    const builder = new HttpRequestBuilder(url, method)
    
    // 应用全局中间件
    this.globalMiddlewares.forEach(middleware => {
      builder.use(middleware)
    })
    
    return builder
  }

  /**
   * GET 请求
   */
  get(url: string): HttpRequestBuilder {
    return this.create(url, 'GET')
  }

  /**
   * POST 请求
   */
  post(url: string, body?: any): HttpRequestBuilder {
    return this.create(url, 'POST').body(body)
  }

  /**
   * PUT 请求
   */
  put(url: string, body?: any): HttpRequestBuilder {
    return this.create(url, 'PUT').body(body)
  }

  /**
   * DELETE 请求
   */
  delete(url: string): HttpRequestBuilder {
    return this.create(url, 'DELETE')
  }

  /**
   * PATCH 请求
   */
  patch(url: string, body?: any): HttpRequestBuilder {
    return this.create(url, 'PATCH').body(body)
  }
}

// 全局 HTTP 客户端实例
let httpClient: HttpClientFactory | null = null

/**
 * 现代化 HTTP 客户端
 */
export function useHttp(): HttpClientFactory {
  if (!httpClient) {
    httpClient = new HttpClientFactory()
    
    // 添加默认中间件
    httpClient.addGlobalMiddleware(new AuthMiddleware())
    httpClient.addGlobalMiddleware(new LoggingMiddleware())
  }
  
  return httpClient
}

/**
 * 简化的 API 调用
 */
export function useApi() {
  const http = useHttp()
  
  return {
    // 只支持 POST（根据后端约束）
    async post<T = any>(url: string, body?: any, options: { auth?: boolean } = {}) {
      const builder = http.post(url, body)
      
      if (options.auth !== false) {
        builder.auth()
      }
      
      const response = await builder.send<T>()
      return response.data
    },

    async postForm<T = any>(url: string, formData: FormData, options: { auth?: boolean } = {}) {
      const builder = http.post(url, formData)
        .headers({ 'Content-Type': 'multipart/form-data' })
      
      if (options.auth !== false) {
        builder.auth()
      }
      
      const response = await builder.send<T>()
      return response.data
    },

    async postPublic<T = any>(url: string, body?: any) {
      const response = await http.post(url, body).send<T>()
      return response.data
    }
  }
}