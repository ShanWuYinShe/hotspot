/**
 * 认证状态存储 - 纯持久化状态管理
 * 
 * ✅ 职责：仅管理持久化状态
 * ❌ 不包含：API 调用、业务逻辑、副作用
 * 
 * 使用 localStorage 存储：
 * - 数据由 store-hydration 插件在客户端启动时同步恢复
 * - 插件执行早于 Vue onMounted，所以非 Vue 代码可直接使用
 * 
 * 使用约定：
 * - 非 Vue 代码（中间件、服务等）：直接使用 store getter
 * - Vue 组件：使用 useHydratedValue() composable 获取值，避免 SSR mismatch
 */

import { defineStore } from 'pinia'
import type { UserResponse } from '~/types/client'

export interface AuthSession {
  user: UserResponse
  accessToken: string              // access_token（用于 API 请求）
  refreshToken: string             // refresh_token（用于刷新 access_token）
  accessTokenExpiresAt: number     // access_token 过期时间（毫秒时间戳）
  refreshTokenExpiresAt: number    // refresh_token 过期时间（毫秒时间戳）
  loginTime: number
  lastActiveTime: number
}

export interface AuthState {
  currentUserId: string | null
  accounts: Record<string, AuthSession>
  accountOrder: string[]
}

const DEFAULT_AUTH_STATE: AuthState = {
  currentUserId: null,
  accounts: {},
  accountOrder: []
}

export const useAuthStore = defineStore('auth', () => {
  // ============ 内部状态 ============
  const _authState = ref<AuthState>({ ...DEFAULT_AUTH_STATE })

  // ============ Getter ============

  const authState = computed((): AuthState => _authState.value)

  const currentSession = computed(() => {
    if (!_authState.value.currentUserId) return null
    return _authState.value.accounts[_authState.value.currentUserId] || null
  })

  // access_token 是否有效（未过期）
  const isAccessTokenValid = computed(() => {
    const session = currentSession.value
    if (!session) return false
    return Date.now() < session.accessTokenExpiresAt
  })

  // refresh_token 是否有效（未过期）
  const isRefreshTokenValid = computed(() => {
    const session = currentSession.value
    if (!session) return false
    return Date.now() < session.refreshTokenExpiresAt
  })

  // 是否已认证：有 access_token 且用户信息存在
  const isAuthenticated = computed(() => {
    const session = currentSession.value
    if (!session) return false
    return !!(session.accessToken && session.user)
  })

  // access_token 是否过期（用于判断是否需要刷新）
  const isTokenExpired = computed(() => {
    const session = currentSession.value
    if (!session) return true
    return Date.now() > session.accessTokenExpiresAt
  })

  const user = computed(() => currentSession.value?.user || null)

  const token = computed(() => currentSession.value?.accessToken || null)

  const refreshToken = computed(() => currentSession.value?.refreshToken || null)

  const accountsList = computed(() => {
    return _authState.value.accountOrder
      .map(userId => _authState.value.accounts[userId])
      .filter(Boolean)
  })

  const accountsCount = computed(() => _authState.value.accountOrder.length)

  // ============ Setter ============

  const setAccountSession = (
    userOrId: UserResponse | { user_id: number; nickname: string },
    accessToken: string,
    refreshToken: string,
    accessTokenExpiresAt: number,
    refreshTokenExpiresAt: number
  ) => {
    // 支持从 AuthData（只有 user_id 和 nickname）或完整的 UserResponse 创建 session
    const user: UserResponse = 'geo' in userOrId
      ? userOrId as UserResponse
      : {
          user_id: userOrId.user_id,
          nickname: userOrId.nickname,
          geo: { latitude: 0, longitude: 0, location_name: '', distance: 0, degrees: 0 }
        }
    // 内部存储使用 string key，因为 JavaScript 对象 key 必须是 string
    const userId = String(user.user_id)
    const now = Date.now()

    const session: AuthSession = {
      user,
      accessToken,
      refreshToken,
      accessTokenExpiresAt,
      refreshTokenExpiresAt,
      loginTime: _authState.value.accounts[userId]?.loginTime || now,
      lastActiveTime: now
    }

    _authState.value.accounts[userId] = session

    const orderIndex = _authState.value.accountOrder.indexOf(userId)
    if (orderIndex > -1) {
      _authState.value.accountOrder.splice(orderIndex, 1)
    }
    _authState.value.accountOrder.unshift(userId)

    _authState.value.currentUserId = userId
  }

  const switchToAccount = (userId: string) => {
    const account = _authState.value.accounts[userId]
    if (!account) {
      throw new Error(`账户 ${userId} 不存在`)
    }

    _authState.value.currentUserId = userId
    _authState.value.accounts[userId].lastActiveTime = Date.now()

    const orderIndex = _authState.value.accountOrder.indexOf(userId)
    if (orderIndex > 0) {
      _authState.value.accountOrder.splice(orderIndex, 1)
      _authState.value.accountOrder.unshift(userId)
    }

    return account.user
  }

  const updateCurrentTokens = (
    accessToken: string,
    refreshToken: string,
    accessTokenExpiresAt: number,
    refreshTokenExpiresAt: number
  ) => {
    if (!_authState.value.currentUserId) return

    const session = _authState.value.accounts[_authState.value.currentUserId]
    if (session) {
      session.accessToken = accessToken
      session.refreshToken = refreshToken
      session.accessTokenExpiresAt = accessTokenExpiresAt
      session.refreshTokenExpiresAt = refreshTokenExpiresAt
      session.lastActiveTime = Date.now()
    }
  }

  const updateCurrentUser = (userData: Partial<UserResponse>) => {
    if (!_authState.value.currentUserId) return

    const session = _authState.value.accounts[_authState.value.currentUserId]
    if (session) {
      session.user = { ...session.user, ...userData }
      session.lastActiveTime = Date.now()
    }
  }

  const removeAccount = (userId: string) => {
    const account = _authState.value.accounts[userId]
    if (!account) return null

    delete _authState.value.accounts[userId]

    const orderIndex = _authState.value.accountOrder.indexOf(userId)
    if (orderIndex > -1) {
      _authState.value.accountOrder.splice(orderIndex, 1)
    }

    if (_authState.value.currentUserId === userId) {
      _authState.value.currentUserId = _authState.value.accountOrder[0] || null
    }

    return account
  }

  const logoutCurrent = () => {
    if (_authState.value.currentUserId) {
      return removeAccount(_authState.value.currentUserId)
    }
    return null
  }

  const logoutAll = () => {
    const accounts = Object.values(_authState.value.accounts)
    _authState.value = { ...DEFAULT_AUTH_STATE }
    return accounts
  }

  // 清理 refresh_token 过期的账户（不是 access_token 过期）
  const cleanExpiredAccounts = () => {
    const now = Date.now()
    const expiredUserIds: string[] = []

    for (const [userId, session] of Object.entries(_authState.value.accounts)) {
      // 只有 refresh_token 过期才清理账户
      if (now > session.refreshTokenExpiresAt) {
        expiredUserIds.push(userId)
      }
    }

    expiredUserIds.forEach(userId => removeAccount(userId))
    return expiredUserIds.length
  }

  const getAccount = (userId: string) => {
    return _authState.value.accounts[userId] || null
  }

  return {
    // 内部状态（用于持久化）
    _authState,

    // Getter
    authState,
    currentSession,
    isAuthenticated,
    isAccessTokenValid,
    isRefreshTokenValid,
    isTokenExpired,
    user,
    token,
    refreshToken,
    accountsList,
    accountsCount,

    // Setter
    setAccountSession,
    switchToAccount,
    updateCurrentTokens,
    updateCurrentUser,
    removeAccount,
    logoutCurrent,
    logoutAll,
    cleanExpiredAccounts,
    getAccount
  }
}, {
  persist: {
    key: 'auth-multi-accounts',
    pick: ['_authState']
  }
})
