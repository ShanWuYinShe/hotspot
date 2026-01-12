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
import type { UserDetailData } from '~/types/client'

export interface AuthSession {
  user: UserDetailData
  accessToken: string              // access_token（用于 API 请求）
  refreshToken: string | null      // refresh_token（app 场景）；web cookie 场景为 null（HttpOnly cookie 存储）
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

  // 是否存在当前会话（代表“曾经/正在登录”，但不代表 access_token 仍有效）
  const hasSession = computed(() => {
    return !!currentSession.value
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

  /**
   * 是否已认证：access_token 存在且未过期
   * - 该状态用于“当前能否访问 protected API”
   * - access_token 过期后即视为未认证（但仍可能具备 refresh 能力）
   */
  const isAuthenticated = computed(() => {
    const session = currentSession.value
    if (!session) return false
    return !!(session.accessToken && session.user && Date.now() < session.accessTokenExpiresAt)
  })

  /**
   * 是否允许尝试刷新（统一前置条件）
   * - 必须有会话（避免“完全未登录”还去打 refresh_token）
   * - refresh_token 未过期
   * - app 场景必须有 refreshToken 值；web 场景 refreshToken 可能为空（HttpOnly cookie）
   */
  const canAttemptRefresh = computed(() => {
    const session = currentSession.value
    if (!session) return false
    if (Date.now() >= session.refreshTokenExpiresAt) return false
    // web: refreshToken 可能为空（cookie），允许；app: refreshToken 有值也允许
    return true
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
    userOrId: UserDetailData | { user_id: number; nickname: string },
    accessToken: string,
    refreshToken: string | null | undefined,
    accessTokenExpiresAt: number,
    refreshTokenExpiresAt: number
  ) => {
    // 支持从 AuthData（只有 user_id 和 nickname）或完整的 UserDetailData 创建 session
    const user: UserDetailData = 'location' in userOrId
      ? userOrId as UserDetailData
      : {
          user_id: userOrId.user_id,
          nickname: userOrId.nickname,
          location: { latitude: 0, longitude: 0, location_name: '', distance: 0, bearing_degrees: 0 },
          // 补充 UserItem/UserInfo 必需字段的默认值
          user_type: 'normal',
          is_online: null,
          is_online_meta: { visibility: 'visible' },
          last_online: null,
          last_online_meta: { visibility: 'visible' },
          location_meta: { visibility: 'visible' }
        }
    // 内部存储使用 string key，因为 JavaScript 对象 key 必须是 string
    const userId = String(user.user_id)
    const now = Date.now()

    const session: AuthSession = {
      user,
      accessToken,
      refreshToken: typeof refreshToken === 'string' ? refreshToken : null,
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
    account.lastActiveTime = Date.now()

    const orderIndex = _authState.value.accountOrder.indexOf(userId)
    if (orderIndex > 0) {
      _authState.value.accountOrder.splice(orderIndex, 1)
      _authState.value.accountOrder.unshift(userId)
    }

    return account.user
  }

  const updateCurrentTokens = (
    accessToken: string,
    refreshToken: string | null | undefined,
    accessTokenExpiresAt: number,
    refreshTokenExpiresAt: number
  ) => {
    if (!_authState.value.currentUserId) return

    const session = _authState.value.accounts[_authState.value.currentUserId]
    if (session) {
      session.accessToken = accessToken
      // web cookie 模式刷新不会在 body 返回 refresh_token（cookie 已更新）：此时保持 null
      // app 模式：refresh_token 会更新
      if (typeof refreshToken === 'string') session.refreshToken = refreshToken
      session.accessTokenExpiresAt = accessTokenExpiresAt
      session.refreshTokenExpiresAt = refreshTokenExpiresAt
      session.lastActiveTime = Date.now()
    }
  }

  const updateCurrentUser = (userData: Partial<UserDetailData>) => {
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

  /**
   * 锁定当前会话（保留本地多账户列表，但进入“未登录态”）
   * - 用于多账户场景：用户希望暂时退出当前账号，但不删除本地已保存账户
   */
  const lock = () => {
    _authState.value.currentUserId = null
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

  /**
   * 兼容旧版本持久化结构（以及弱数据一致性修复）
   * - 旧版本 web cookie 模式会把 refreshToken 存成 ''，现在统一转为 null
   * - currentUserId 指向不存在账号时，重置为 null
   * - accountOrder 去重并移除不存在项
   */
  const normalizePersistedState = () => {
    const ids = Object.keys(_authState.value.accounts)

    // refreshToken: '' -> null
    for (const id of ids) {
      const s = _authState.value.accounts[id]
      if (s && s.refreshToken === '') {
        s.refreshToken = null
      }
    }

    // accountOrder: 去重 + 只保留存在的账号
    const nextOrder: string[] = []
    const seen = new Set<string>()
    for (const id of _authState.value.accountOrder) {
      if (seen.has(id)) continue
      if (!_authState.value.accounts[id]) continue
      seen.add(id)
      nextOrder.push(id)
    }
    // 若有账号但 order 为空，补齐
    if (nextOrder.length === 0 && ids.length > 0) {
      ids.forEach((id) => nextOrder.push(id))
    }
    _authState.value.accountOrder = nextOrder

    // currentUserId 修复
    if (_authState.value.currentUserId && !_authState.value.accounts[_authState.value.currentUserId]) {
      _authState.value.currentUserId = nextOrder[0] || null
    }
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
    hasSession,
    isAuthenticated,
    isAccessTokenValid,
    isRefreshTokenValid,
    canAttemptRefresh,
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
    lock,
    cleanExpiredAccounts,
    normalizePersistedState,
    getAccount
  }
}, {
  persist: {
    key: 'auth-multi-accounts',
    pick: ['_authState']
  }
})
