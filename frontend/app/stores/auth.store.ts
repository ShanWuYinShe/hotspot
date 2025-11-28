/**
 * 认证状态存储 - 支持多账户切换
 * 职责：状态管理、持久化 + API 调用
 * 
 * SSR 水合策略：
 * - pinia-plugin-persistedstate 使用 localStorage 自动恢复数据
 * - 但组件通过 displayXxx 计算属性获取值，这些属性在 hydrate() 调用前返回 null
 * - app.vue 的 onMounted 中调用 hydrate()，之后组件才能看到真实值
 * - 这确保了 SSR 和客户端初始渲染一致（都是 null），避免水合不匹配
 */

import { defineStore } from 'pinia'
import type { 
  UserResponse,
  RegisterRequest, 
  LoginRequest, 
  CreateTempRequest, 
  SetPasswordRequest 
} from '~/types/api'
import { useAuthService } from '~/services/auth.service'

export interface AuthSession {
  user: UserResponse
  token: string
  refreshToken: string
  expiresAt: number
  loginTime: number
  lastActiveTime: number
}

export interface AuthState {
  currentUserId: string | null
  accounts: Record<string, AuthSession>
  accountOrder: string[]
}

export const useAuthStore = defineStore('auth', () => {
  // ============ 状态 ============
  const authState = ref<AuthState>({
    currentUserId: null,
    accounts: {},
    accountOrder: []
  })
  
  // SSR 水合标记：onMounted 后设为 true，组件才能看到真实数据
  const isReady = ref(false)
  
  // 在 app.vue 的 onMounted 中调用，启用显示
  const hydrate = () => {
    isReady.value = true
    // 同时清理过期账户
    cleanExpiredAccounts()
  }

  // ============ 计算属性 ============

  const currentSession = computed(() => {
    if (!authState.value.currentUserId) return null
    return authState.value.accounts[authState.value.currentUserId] || null
  })

  const isAuthenticated = computed(() => {
    const session = currentSession.value
    if (!session) return false
    if (Date.now() > session.expiresAt) return false
    return !!(session.token && session.user)
  })

  const isTokenExpired = computed(() => {
    const session = currentSession.value
    if (!session) return true
    return Date.now() > session.expiresAt
  })

  // 内部使用的原始值（API 调用等）
  const token = computed(() => currentSession.value?.token || null)
  const refreshToken = computed(() => currentSession.value?.refreshToken || null)

  const accountsList = computed(() => {
    return authState.value.accountOrder
      .map(userId => authState.value.accounts[userId])
      .filter(Boolean)
  })

  const accountsCount = computed(() => authState.value.accountOrder.length)
  
  // ============ 显示用计算属性（SSR 安全）============
  // 这些属性在 hydrate() 调用前返回 null/空值，避免水合不匹配
  
  const user = computed(() => isReady.value ? (currentSession.value?.user || null) : null)
  
  const displayAccountsList = computed(() => 
    isReady.value ? accountsList.value : []
  )
  
  const displayAccountsCount = computed(() => 
    isReady.value ? accountsCount.value : 0
  )
  
  const displayIsAuthenticated = computed(() => 
    isReady.value ? isAuthenticated.value : false
  )

  // ============ 状态操作方法 ============

  const setAccountSession = (user: UserResponse, token: string, refreshToken: string, expiresAt: number) => {
    const userId = user.user_id
    const now = Date.now()

    const session: AuthSession = {
      user,
      token,
      refreshToken,
      expiresAt,
      loginTime: authState.value.accounts[userId]?.loginTime || now,
      lastActiveTime: now
    }

    authState.value.accounts[userId] = session

    const orderIndex = authState.value.accountOrder.indexOf(userId)
    if (orderIndex > -1) {
      authState.value.accountOrder.splice(orderIndex, 1)
    }
    authState.value.accountOrder.unshift(userId)

    authState.value.currentUserId = userId
  }

  const switchToAccount = (userId: string) => {
    const account = authState.value.accounts[userId]
    if (!account) {
      throw new Error(`账户 ${userId} 不存在`)
    }

    authState.value.currentUserId = userId
    authState.value.accounts[userId].lastActiveTime = Date.now()

    const orderIndex = authState.value.accountOrder.indexOf(userId)
    if (orderIndex > 0) {
      authState.value.accountOrder.splice(orderIndex, 1)
      authState.value.accountOrder.unshift(userId)
    }

    return account.user
  }

  const updateCurrentTokens = (token: string, refreshToken: string, expiresAt: number) => {
    if (!authState.value.currentUserId) return

    const session = authState.value.accounts[authState.value.currentUserId]
    if (session) {
      session.token = token
      session.refreshToken = refreshToken
      session.expiresAt = expiresAt
      session.lastActiveTime = Date.now()
    }
  }

  const updateCurrentUser = (userData: Partial<UserResponse>) => {
    if (!authState.value.currentUserId) return

    const session = authState.value.accounts[authState.value.currentUserId]
    if (session) {
      session.user = { ...session.user, ...userData }
      session.lastActiveTime = Date.now()
    }
  }

  const removeAccount = (userId: string) => {
    const account = authState.value.accounts[userId]
    if (!account) return

    delete authState.value.accounts[userId]

    const orderIndex = authState.value.accountOrder.indexOf(userId)
    if (orderIndex > -1) {
      authState.value.accountOrder.splice(orderIndex, 1)
    }

    if (authState.value.currentUserId === userId) {
      if (authState.value.accountOrder.length > 0) {
        authState.value.currentUserId = authState.value.accountOrder[0]
      } else {
        authState.value.currentUserId = null
      }
    }

    return account
  }

  const logoutCurrent = () => {
    if (authState.value.currentUserId) {
      return removeAccount(authState.value.currentUserId)
    }
    return null
  }

  const logoutAll = () => {
    const accounts = Object.values(authState.value.accounts)
    authState.value = {
      currentUserId: null,
      accounts: {},
      accountOrder: []
    }
    return accounts
  }

  const cleanExpiredAccounts = () => {
    const now = Date.now()
    const expiredUserIds: string[] = []

    for (const [userId, session] of Object.entries(authState.value.accounts)) {
      if (now > session.expiresAt) {
        expiredUserIds.push(userId)
      }
    }

    expiredUserIds.forEach(userId => {
      removeAccount(userId)
    })

    return expiredUserIds.length
  }

  const getAccount = (userId: string) => {
    return authState.value.accounts[userId] || null
  }

  // ============ API 调用方法 ============

  const register = async (data: RegisterRequest) => {
    const authService = useAuthService()
    const response = await authService.register(data)
    const expiresAt = new Date(response.expires_at).getTime()
    
    setAccountSession(
      { user_id: response.user_id, nickname: response.nickname },
      response.access_token,
      response.refresh_token,
      expiresAt
    )
    
    return response
  }

  const login = async (data: LoginRequest) => {
    const authService = useAuthService()
    const response = await authService.login(data)
    const expiresAt = new Date(response.expires_at).getTime()
    
    setAccountSession(
      { user_id: response.user_id, nickname: response.nickname },
      response.access_token,
      response.refresh_token,
      expiresAt
    )
    
    return response
  }

  const createTempUser = async (data: CreateTempRequest) => {
    const authService = useAuthService()
    const response = await authService.createTempUser(data)
    const expiresAt = new Date(response.expires_at).getTime()
    
    setAccountSession(
      { user_id: response.user_id, nickname: response.nickname },
      response.access_token,
      response.refresh_token,
      expiresAt
    )
    
    return response
  }

  const refreshAccessToken = async () => {
    if (!refreshToken.value) {
      throw new Error('没有刷新令牌')
    }

    const authService = useAuthService()
    
    try {
      const response = await authService.refreshToken({ 
        refresh_token: refreshToken.value 
      })
      
      const expiresAt = new Date(response.expires_at).getTime()
      updateCurrentTokens(response.access_token, response.refresh_token, expiresAt)
      
      return response
    } catch (error) {
      logoutCurrent()
      throw error
    }
  }

  const logout = async () => {
    const authService = useAuthService()
    
    try {
      await authService.logout()
    } finally {
      logoutCurrent()
    }
  }

  const setPassword = async (data: SetPasswordRequest) => {
    const authService = useAuthService()
    return await authService.setPassword(data)
  }

  const checkStatus = async () => {
    const authService = useAuthService()
    
    try {
      const response = await authService.checkStatus()
      
      if (!response.is_valid) {
        logoutCurrent()
        return false
      }
      
      return true
    } catch (error) {
      logoutCurrent()
      return false
    }
  }

  return {
    // 状态
    authState,
    isReady,
    
    // SSR 水合
    hydrate,
    
    // 内部计算属性（API 调用等使用）
    currentSession,
    isAuthenticated,
    isTokenExpired,
    token,
    refreshToken,
    accountsList,
    accountsCount,
    
    // 显示用计算属性（组件 UI 使用，SSR 安全）
    user,  // 已经是 SSR 安全的
    displayAccountsList,
    displayAccountsCount,
    displayIsAuthenticated,
    
    // 状态操作方法
    setAccountSession,
    switchToAccount,
    updateCurrentTokens,
    updateCurrentUser,
    removeAccount,
    logoutCurrent,
    logoutAll,
    cleanExpiredAccounts,
    getAccount,
    
    // API 调用方法
    register,
    login,
    createTempUser,
    refreshAccessToken,
    logout,
    setPassword,
    checkStatus
  }
}, {
  persist: {
    key: 'auth-multi-accounts',
    pick: ['authState']
  }
})
