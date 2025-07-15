/**
 * 统一API服务 - 简化版本
 * 对应后端的简单响应结构，只需要根据code判断成功失败
 */

// 后端API响应格式
export interface ApiResponse<T = any> {
  code: number         // 状态码，0表示成功，非0表示失败
  msg: string          // 状态消息
  data: T | null       // 成功时为数据对象，失败时为null
}

/**
 * 统一API服务 - 简化版本
 * 对应后端的简单响应结构，只需要根据code判断成功失败
 */

// 后端API响应格式
export interface ApiResponse<T = any> {
  code: number         // 状态码，0表示成功，非0表示失败
  msg: string          // 状态消息
  data: T | null       // 成功时为数据对象，失败时为null
}

export interface RequestOptions {
  headers?: Record<string, string>
  timeout?: number
  authRequired?: boolean // 默认为true，需要认证
}

/**
 * 获取API基础URL
 */
export function useGetApiBaseURL() {
  return (): string => {
    return process.env.NUXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1'
  }
}

/**
 * 获取认证token
 */
export function useGetAuthToken() {
  return (): string | null => {
    try {
      if (typeof window !== 'undefined' && typeof localStorage !== 'undefined') {
        const authStore = localStorage.getItem('auth')
        if (authStore) {
          const parsedStore = JSON.parse(authStore)
          return parsedStore.token || null
        }
      }
    } catch (error) {
      console.warn('Failed to get auth token:', error)
    }
    return null
  }
}

/**
 * 核心API请求方法
 */
export function useApiRequest() {
  const getBaseURL = useGetApiBaseURL()
  const getAuthToken = useGetAuthToken()
  
  return async <T = any>(
    endpoint: string,
    data?: any,
    options: RequestOptions = {}
  ): Promise<T | null> => {
    const baseURL = getBaseURL()
    const token = getAuthToken()

    const requiresAuth = options.authRequired !== false
    
    if (requiresAuth && !token) {
      console.error(`Authentication required for ${endpoint}, but no token found.`)
      throw new Error('请先登录再执行此操作')
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...options.headers,
    }
    
    if (requiresAuth && token) {
      headers['Authorization'] = `Bearer ${token}`
    }

    try {
      const response: ApiResponse<T> = await $fetch(endpoint, {
        baseURL: baseURL,
        method: 'POST',
        headers,
        body: data,
        timeout: options.timeout || 10000,
      })
      
      // 根据code判断成功失败
      if (response.code === 0) {
        return response.data
      } else {
        console.warn(`API请求失败: ${endpoint}`, response.msg)
        throw new Error(response.msg || '操作失败')
      }
    } catch (error: any) {
      console.error(`API请求异常: ${endpoint}`, error)
      
      if (error.message) {
        throw error
      } else {
        throw new Error('网络请求失败，请稍后重试')
      }
    }
  }
}

/**
 * 安全的API请求方法（不抛出异常，返回null表示失败）
 */
export function useSafeApiRequest() {
  const apiRequest = useApiRequest()
  
  return async <T>(
    endpoint: string,
    data?: any,
    options?: RequestOptions
  ): Promise<T | null> => {
    try {
      return await apiRequest<T>(endpoint, data, options)
    } catch (error) {
      console.warn(`API请求失败: ${endpoint}`, error)
      return null
    }
  }
}
