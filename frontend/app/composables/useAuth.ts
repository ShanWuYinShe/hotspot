/**
 * 认证相关的 Composable
 */

export const useAuth = () => {
  const api = useApi()
  const authStore = useAuthStore()

  // 登录
  const login = async (loginId: string, password: string) => {
    try {
      authStore.isLoading = true
      authStore.error = null

      const response: any = await api.post('/auth/login', {
        login_id: loginId,
        password
      })

      if (response?.success) {
        authStore.setAuthData(response.data)
        return response.data
      } else {
        throw new Error(response?.message || '登录失败')
      }
    } catch (error: any) {
      authStore.error = error.message || '登录失败'
      throw error
    } finally {
      authStore.isLoading = false
    }
  }

  // 注册
  const register = async (loginId: string, nickname: string, password: string) => {
    try {
      authStore.isLoading = true
      authStore.error = null

      const response: any = await api.post('/auth/register', {
        login_id: loginId,
        nickname,
        password
      })

      if (response?.success) {
        authStore.setAuthData(response.data)
        return response.data
      } else {
        throw new Error(response?.message || '注册失败')
      }
    } catch (error: any) {
      authStore.error = error.message || '注册失败'
      throw error
    } finally {
      authStore.isLoading = false
    }
  }

  // 登出
  const logout = async () => {
    try {
      if (authStore.token) {
        await api.post('/auth/logout')
      }
    } catch (error) {
      console.warn('登出请求失败:', error)
    } finally {
      authStore.clearAuth()
    }
  }

  return {
    login,
    register,
    logout,
    // 响应式状态
    isAuthenticated: computed(() => authStore.isAuthenticated),
    user: computed(() => authStore.currentUser),
    isLoading: computed(() => authStore.isLoading),
    error: computed(() => authStore.error)
  }
}
