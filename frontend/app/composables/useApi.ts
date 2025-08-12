/**
 * API 客户端 Composable
 * 封装统一的 HTTP 请求逻辑
 */

export const useApi = () => {
  const config = useRuntimeConfig()
  
  // 获取认证 token
  const getAuthHeaders = () => {
    const authStore = useAuthStore()
    return authStore.token ? {
      'Authorization': `Bearer ${authStore.token}`
    } : {}
  }
  
  // 错误处理
  const handleError = (error: any) => {
    const toast = useToast()
    
    if (error.status === 401) {
      const authStore = useAuthStore()
      authStore.clearAuth()
      navigateTo('/auth')
      toast.add({
        title: '认证过期',
        description: '请重新登录',
        color: 'error'
      })
    } else {
      toast.add({
        title: '请求失败',
        description: error.data?.message || '未知错误',
        color: 'error'
      })
    }
    
    throw error
  }

  return {
    // GET 请求
    async get(url: string, options: any = {}) {
      try {
        return await $fetch(url, {
          baseURL: config.public.apiBaseUrl,
          method: 'GET',
          headers: {
            ...getAuthHeaders(),
            ...options.headers
          },
          ...options
        })
      } catch (error) {
        handleError(error)
      }
    },

    // POST 请求
    async post(url: string, body?: any, options: any = {}) {
      try {
        return await $fetch(url, {
          baseURL: config.public.apiBaseUrl,
          method: 'POST',
          body,
          headers: {
            'Content-Type': 'application/json',
            ...getAuthHeaders(),
            ...options.headers
          },
          ...options
        })
      } catch (error) {
        handleError(error)
      }
    },

    // PUT 请求
    async put(url: string, body?: any, options: any = {}) {
      try {
        return await $fetch(url, {
          baseURL: config.public.apiBaseUrl,
          method: 'PUT',
          body,
          headers: {
            'Content-Type': 'application/json',
            ...getAuthHeaders(),
            ...options.headers
          },
          ...options
        })
      } catch (error) {
        handleError(error)
      }
    },

    // DELETE 请求
    async delete(url: string, options: any = {}) {
      try {
        return await $fetch(url, {
          baseURL: config.public.apiBaseUrl,
          method: 'DELETE',
          headers: {
            ...getAuthHeaders(),
            ...options.headers
          },
          ...options
        })
      } catch (error) {
        handleError(error)
      }
    }
  }
}
