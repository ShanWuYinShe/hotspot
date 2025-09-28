/**
 * 认证状态存储 - 支持多账户切换
 * 职责：纯粹的状态管理和持久化，不包含 API 调用
 */

import type { UserResponse } from '~/types/api'

export interface AuthSession {
  user: UserResponse
  token: string
  refreshToken: string
  expiresAt: number
  loginTime: number // 登录时间戳
  lastActiveTime: number // 最后活跃时间
}

export interface AuthState {
  currentUserId: string | null // 当前活跃用户ID
  accounts: Record<string, AuthSession> // 用户ID -> 会话信息
  accountOrder: string[] // 账户切换顺序（最近使用的在前）
}

export const useAuthStore = defineStore('auth', () => {
  // ============ 状态 ============
  const authState = ref<AuthState>({
    currentUserId: null,
    accounts: {},
    accountOrder: []
  })

  // ============ 计算属性 ============

  /**
   * 当前活跃会话
   */
  const currentSession = computed(() => {
    if (!authState.value.currentUserId) return null
    return authState.value.accounts[authState.value.currentUserId] || null
  })

  /**
   * 是否已认证
   */
  const isAuthenticated = computed(() => {
    const session = currentSession.value
    if (!session) return false

    // 检查令牌是否过期
    if (Date.now() > session.expiresAt) return false

    return !!(session.token && session.user)
  })

  /**
   * 当前令牌是否过期
   */
  const isTokenExpired = computed(() => {
    const session = currentSession.value
    if (!session) return true
    return Date.now() > session.expiresAt
  })

  /**
   * 当前用户信息
   */
  const user = computed(() => currentSession.value?.user || null)

  /**
   * 当前访问令牌
   */
  const token = computed(() => currentSession.value?.token || null)

  /**
   * 当前刷新令牌
   */
  const refreshToken = computed(() => currentSession.value?.refreshToken || null)

  /**
   * 所有已登录账户列表（按最近使用排序）
   */
  const accountsList = computed(() => {
    return authState.value.accountOrder
      .map(userId => authState.value.accounts[userId])
      .filter(Boolean)
  })

  /**
   * 账户数量
   */
  const accountsCount = computed(() => authState.value.accountOrder.length)

  // ============ 状态操作方法 ============

  /**
   * 添加或更新账户会话
   */
  const setAccountSession = (user: UserResponse, token: string, refreshToken: string, expiresAt: number) => {
    const userId = user.user_id
    const now = Date.now()

    // 创建会话信息
    const session: AuthSession = {
      user,
      token,
      refreshToken,
      expiresAt,
      loginTime: authState.value.accounts[userId]?.loginTime || now,
      lastActiveTime: now
    }

    // 保存会话
    authState.value.accounts[userId] = session

    // 更新账户顺序（最近使用的在前）
    const orderIndex = authState.value.accountOrder.indexOf(userId)
    if (orderIndex > -1) {
      authState.value.accountOrder.splice(orderIndex, 1)
    }
    authState.value.accountOrder.unshift(userId)

    // 设置为当前活跃用户
    authState.value.currentUserId = userId
  }

  /**
   * 切换到指定账户
   */
  const switchToAccount = (userId: string) => {
    const account = authState.value.accounts[userId]
    if (!account) {
      throw new Error(`账户 ${userId} 不存在`)
    }

    // 更新当前用户
    authState.value.currentUserId = userId

    // 更新最后活跃时间
    authState.value.accounts[userId].lastActiveTime = Date.now()

    // 更新账户顺序
    const orderIndex = authState.value.accountOrder.indexOf(userId)
    if (orderIndex > 0) {
      authState.value.accountOrder.splice(orderIndex, 1)
      authState.value.accountOrder.unshift(userId)
    }

    return account.user
  }


  /**
   * 更新当前会话的令牌信息
   */
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

  /**
   * 更新当前用户信息
   */
  const updateCurrentUser = (userData: Partial<UserResponse>) => {
    if (!authState.value.currentUserId) return

    const session = authState.value.accounts[authState.value.currentUserId]
    if (session) {
      session.user = { ...session.user, ...userData }
      session.lastActiveTime = Date.now()
    }
  }

  /**
   * 移除指定账户
   */
  const removeAccount = (userId: string) => {
    const account = authState.value.accounts[userId]
    if (!account) return

    // 删除账户数据
    delete authState.value.accounts[userId]

    // 从顺序中移除
    const orderIndex = authState.value.accountOrder.indexOf(userId)
    if (orderIndex > -1) {
      authState.value.accountOrder.splice(orderIndex, 1)
    }

    // 如果删除的是当前用户，切换到下一个用户或清空
    if (authState.value.currentUserId === userId) {
      if (authState.value.accountOrder.length > 0) {
        authState.value.currentUserId = authState.value.accountOrder[0]
      } else {
        authState.value.currentUserId = null
      }
    }

    return account // 返回被删除的账户信息，供调用者处理服务器登出
  }

  /**
   * 登出当前账户
   */
  const logoutCurrent = () => {
    if (authState.value.currentUserId) {
      return removeAccount(authState.value.currentUserId)
    }
    return null
  }

  /**
   * 登出所有账户
   */
  const logoutAll = () => {
    const accounts = Object.values(authState.value.accounts)

    // 清空本地所有账户
    authState.value = {
      currentUserId: null,
      accounts: {},
      accountOrder: []
    }

    // 返回所有账户信息，供调用者处理服务器登出
    return accounts
  }

  /**
   * 清理过期的账户
   */
  const cleanExpiredAccounts = () => {
    const now = Date.now()
    const expiredUserIds: string[] = []

    // 找出过期的账户
    for (const [userId, session] of Object.entries(authState.value.accounts)) {
      if (now > session.expiresAt) {
        expiredUserIds.push(userId)
      }
    }

    // 移除过期账户
    expiredUserIds.forEach(userId => {
      removeAccount(userId)
    })

    return expiredUserIds.length
  }

  /**
   * 获取指定账户信息
   */
  const getAccount = (userId: string) => {
    return authState.value.accounts[userId] || null
  }

  // ============ 兼容性方法（保持向后兼容） ============

  /**
   * 设置会话（兼容旧接口）
   */
  const setSession = (sessionData: { user: UserResponse, token: string, refreshToken: string, expiresAt: number }) => {
    setAccountSession(sessionData.user, sessionData.token, sessionData.refreshToken, sessionData.expiresAt)
  }

  /**
   * 清除会话（兼容旧接口）
   */
  const clearSession = () => {
    logoutCurrent()
  }

  // ============ 初始化 ============
  // Pinia 插件会自动恢复持久化的状态
  // 应用启动时自动清理过期账户
  if (typeof window !== 'undefined') {
    nextTick(() => {
      cleanExpiredAccounts()
    })
  }

  return {
    // 状态
    authState,

    // 计算属性
    currentSession,
    isAuthenticated,
    isTokenExpired,
    user,
    token,
    refreshToken,
    accountsList,
    accountsCount,

    // 多账户管理方法
    setAccountSession,
    switchToAccount,
    updateCurrentTokens,
    updateCurrentUser,
    removeAccount,
    logoutCurrent,
    logoutAll,
    cleanExpiredAccounts,
    getAccount,

    // 兼容性方法
    setSession,
    clearSession
  }
}, {
  persist: {
    key: 'auth-multi-accounts',
    pick: ['authState']
    // 现在持久化完整的多账户状态，包括用户信息
  }
})
