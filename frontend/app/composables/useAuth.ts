/**
 * 认证业务逻辑 Composable
 * 职责：处理认证相关的业务逻辑和 API 调用，操作 Auth Store
 */

import type { UserResponse } from '~/types/api'

export function useAuth() {
  const authStore = useAuthStore()
  const { success, error } = useNotifications()
  
  /**
   * 登录
   */
  const login = async (credentials: {
    email?: string
    phone?: string
    password: string
  }) => {
    try {
      const http = useHttp()
      const response = await http.post('/public/auth/login', credentials)
      
      // 添加到多账户系统
      authStore.setAccountSession(
        response.data.user,
        response.data.access_token,
        response.data.refresh_token,
        Date.now() + (response.data.expires_in * 1000)
      )
      
      success('登录成功')
      return response.data.user
    } catch (err: any) {
      error(err.message || '登录失败')
      throw err
    }
  }
  
  /**
   * 注册
   */
  const register = async (userData: {
    email?: string
    phone?: string
    password: string
    nickname: string
    verification_code: string
  }) => {
    try {
      const http = useHttp()
      const response = await http.post('/public/auth/register', userData)
      
      // 注册成功后自动添加到多账户系统
      authStore.setAccountSession(
        response.data.user,
        response.data.access_token,
        response.data.refresh_token,
        Date.now() + (response.data.expires_in * 1000)
      )
      
      success('注册成功')
      return response.data.user
    } catch (err: any) {
      error(err.message || '注册失败')
      throw err
    }
  }
  
  /**
   * 刷新令牌
   */
  const refreshToken = async (): Promise<boolean> => {
    if (!authStore.refreshToken) {
      return false
    }
    
    try {
      const http = useHttp()
      const response = await http.post('/public/auth/refresh', {
        refresh_token: authStore.refreshToken
      })
      
      // 更新当前账户的令牌信息
      authStore.updateCurrentTokens(
        response.data.access_token,
        response.data.refresh_token,
        Date.now() + (response.data.expires_in * 1000)
      )
      
      return true
    } catch (error) {
      // 刷新失败，登出当前账户
      authStore.logoutCurrent()
      return false
    }
  }
  
  /**
   * 登出当前账户
   */
  const logout = async () => {
    try {
      if (authStore.token) {
        const http = useHttp()
        await http.post('/protected/auth/logout', {}, { auth: true })
      }
    } catch (err) {
      console.warn('登出请求失败:', err)
    } finally {
      authStore.logoutCurrent()
      success('已退出登录')
    }
  }
  
  /**
   * 切换账户
   */
  const switchAccount = async (userId: string) => {
    try {
      const account = authStore.getAccount(userId)
      if (!account) {
        throw new Error('账户不存在')
      }
      
      // 检查令牌是否过期
      if (Date.now() > account.expiresAt) {
        // 尝试刷新令牌
        const refreshed = await refreshAccountToken(userId)
        if (!refreshed) {
          throw new Error('账户已过期，请重新登录')
        }
      }
      
      // 切换账户
      const user = authStore.switchToAccount(userId)
      success(`已切换到 ${user.nickname}`)
      return user
    } catch (err: any) {
      error(err.message || '切换账户失败')
      throw err
    }
  }
  
  /**
   * 刷新指定账户的令牌
   */
  const refreshAccountToken = async (userId: string): Promise<boolean> => {
    const account = authStore.getAccount(userId)
    if (!account) return false
    
    try {
      const http = useHttp()
      const response = await http.post('/public/auth/refresh', {
        refresh_token: account.refreshToken
      })
      
      // 更新令牌信息
      authStore.updateCurrentTokens(
        response.data.access_token,
        response.data.refresh_token,
        Date.now() + (response.data.expires_in * 1000)
      )
      
      return true
    } catch (err) {
      // 刷新失败，移除该账户
      authStore.removeAccount(userId)
      return false
    }
  }
  
  /**
   * 移除账户
   */
  const removeAccount = async (userId: string) => {
    try {
      const removedAccount = authStore.removeAccount(userId)
      
      // 如果有有效令牌，通知服务器登出
      if (removedAccount && removedAccount.token && Date.now() < removedAccount.expiresAt) {
        try {
          const http = useHttp()
          await http.post('/protected/auth/logout', {}, { 
            headers: { Authorization: `Bearer ${removedAccount.token}` }
          })
        } catch (err) {
          console.warn('服务器登出失败:', err)
        }
      }
      
      success(`已移除账户 ${removedAccount?.user.nickname}`)
    } catch (err: any) {
      error(err.message || '移除账户失败')
      throw err
    }
  }
  
  /**
   * 登出所有账户
   */
  const logoutAll = async () => {
    try {
      const accounts = authStore.logoutAll()
      
      // 通知服务器登出所有有效会话
      const logoutPromises = accounts
        .filter(account => account.token && Date.now() < account.expiresAt)
        .map(account => {
          const http = useHttp()
          return http.post('/protected/auth/logout', {}, {
            headers: { Authorization: `Bearer ${account.token}` }
          }).catch(err => {
            console.warn(`账户 ${account.user.nickname} 服务器登出失败:`, err)
          })
        })
      
      await Promise.allSettled(logoutPromises)
      success('已登出所有账户')
    } catch (err: any) {
      error(err.message || '登出失败')
      throw err
    }
  }
  
  /**
   * 获取用户信息
   */
  const fetchUserProfile = async () => {
    try {
      const http = useHttp()
      const response = await http.get('/protected/user/profile', { auth: true })
      
      // 更新当前用户信息
      authStore.updateCurrentUser(response.data.user)
      
      return response.data.user
    } catch (err: any) {
      error(err.message || '获取用户信息失败')
      throw err
    }
  }
  
  /**
   * 更新用户信息
   */
  const updateProfile = async (updates: Partial<UserResponse>) => {
    try {
      const http = useHttp()
      const response = await http.put('/protected/user/profile', updates, { auth: true })
      
      // 更新当前用户信息
      authStore.updateCurrentUser(response.data.user)
      
      success('个人信息更新成功')
      return response.data.user
    } catch (err: any) {
      error(err.message || '更新失败')
      throw err
    }
  }
  
  /**
   * 修改密码
   */
  const changePassword = async (data: {
    current_password: string
    new_password: string
  }) => {
    try {
      const http = useHttp()
      await http.put('/protected/user/password', data, { auth: true })
      success('密码修改成功')
    } catch (err: any) {
      error(err.message || '密码修改失败')
      throw err
    }
  }
  
  /**
   * 验证并自动刷新令牌
   */
  const validateAndRefresh = async (): Promise<boolean> => {
    if (!authStore.isAuthenticated) {
      authStore.restoreSession()
    }
    
    if (!authStore.isAuthenticated) {
      return false
    }
    
    // 如果令牌即将过期（5分钟内），尝试刷新
    if (authStore.session.expiresAt && Date.now() > authStore.session.expiresAt - 5 * 60 * 1000) {
      return await refreshToken()
    }
    
    return true
  }
  
  /**
   * 初始化认证状态
   */
  const initialize = async () => {
    // 验证并刷新令牌
    const isValid = await validateAndRefresh()
    
    // 如果有有效令牌但没有用户信息，获取用户信息
    if (isValid && authStore.token && !authStore.user) {
      await fetchUserProfile()
    }
    
    return isValid
  }
  
  return {
    // 状态（来自 store）
    isAuthenticated: authStore.isAuthenticated,
    isTokenExpired: authStore.isTokenExpired,
    user: authStore.user,
    token: authStore.token,
    accountsList: authStore.accountsList,
    accountsCount: authStore.accountsCount,
    
    // 业务方法
    login,
    register,
    logout,
    refreshToken,
    fetchUserProfile,
    updateProfile,
    changePassword,
    validateAndRefresh,
    initialize,
    
    // 多账户管理
    switchAccount,
    removeAccount,
    logoutAll
  }
}
