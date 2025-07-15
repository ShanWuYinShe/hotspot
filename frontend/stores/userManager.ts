import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useAuthStore } from './auth'
import { login, register, createTempUser, refreshToken } from '~/services/auth/auth.service'
import { extractErrorMessage } from '~/utils/error-handler'
import type { UserProfile } from '~/types/models/user'
import type { LoginRequest, RegisterRequest } from '~/types/api/auth'

// 用户账户类型
export interface UserAccount {
  user_id: number
  login_id?: string
  nickname: string
  avatar_url?: string
  email?: string
  is_temporary: boolean
  last_used: string
  created_at: string
  // 存储的认证信息
  stored_auth: {
    access_token: string
    refresh_token: string
    expires_at: string
  }
}

// 用户管理Store
export const useUserManagerStore = defineStore('userManager', () => {
  // 状态
  const accounts = ref<UserAccount[]>([])
  const currentUserId = ref<number | null>(null)
  const isLoading = ref<boolean>(false)

  // 计算属性
  const currentUser = computed(() => {
    if (!currentUserId.value) return null
    return accounts.value.find(account => account.user_id === currentUserId.value) || null
  })

  const hasUsers = computed(() => accounts.value.length > 0)

  const recentUsers = computed(() => {
    return [...accounts.value]
      .sort((a, b) => new Date(b.last_used).getTime() - new Date(a.last_used).getTime())
      .slice(0, 5)
  })

  // 添加用户账户
  function addAccount(authData: any, isTemporary = false) {
    const account: UserAccount = {
      user_id: authData.user_id,
      login_id: authData.login_id || '',
      nickname: authData.nickname,
      avatar_url: authData.avatar_url || '',
      email: authData.email || '',
      is_temporary: isTemporary,
      last_used: new Date().toISOString(),
      created_at: authData.created_at || new Date().toISOString(),
      stored_auth: {
        access_token: authData.access_token,
        refresh_token: authData.refresh_token,
        expires_at: authData.expires_at
      }
    }

    // 检查是否已存在
    const existingIndex = accounts.value.findIndex(account => account.user_id === account.user_id)
    if (existingIndex >= 0) {
      // 更新现有账户
      accounts.value[existingIndex] = account
    } else {
      // 添加新账户
      accounts.value.push(account)
    }

    currentUserId.value = account.user_id

    return account
  }

  // 切换到指定用户
  async function switchToUser(userId: number) {
    const account = accounts.value.find(account => account.user_id === userId)
    if (!account) {
      throw new Error('用户账户不存在')
    }

    isLoading.value = true
    try {
      const authStore = useAuthStore()

      // 检查token是否过期
      const isExpired = new Date(account.stored_auth.expires_at).getTime() <= Date.now()

      if (isExpired) {
        // 尝试刷新token
        try {
          console.log('Token已过期，尝试刷新...')
          const refreshResult = await refreshToken(account.stored_auth.refresh_token)
          
          // 更新账户中的token信息
          account.stored_auth = {
            access_token: refreshResult.access_token,
            refresh_token: refreshResult.refresh_token,
            expires_at: refreshResult.expires_at
          }
          
          console.log('Token刷新成功')
        } catch (refreshError) {
          console.error('Token刷新失败:', refreshError)
          // 刷新失败，移除过期账户
          await removeAccount(userId)
          throw new Error('账户已过期且刷新失败，请重新登录')
        }
      }

      // 设置到AuthStore
      await authStore.setAuthData({
        user_id: account.user_id,
        nickname: account.nickname,
        access_token: account.stored_auth.access_token,
        refresh_token: account.stored_auth.refresh_token,
        expires_at: account.stored_auth.expires_at
      })

      // 更新最后使用时间
      account.last_used = new Date().toISOString()
      currentUserId.value = userId

      return account
    } finally {
      isLoading.value = false
    }
  }

  // 移除用户账户
  async function removeAccount(userId: number) {
    const index = accounts.value.findIndex(account => account.user_id === userId)
    if (index >= 0) {
      accounts.value.splice(index, 1)

      if (currentUserId.value === userId) {
        // 如果移除的是当前用户，切换到最近使用的用户或清空
        if (accounts.value.length > 0) {
          const mostRecent = recentUsers.value[0]
          await switchToUser(mostRecent.user_id)
        } else {
          currentUserId.value = null
          const authStore = useAuthStore()
          authStore.logout()
        }
      }
    }
  }

  // 添加新用户（登录）
  async function addUserByLogin(credentials: LoginRequest) {
    isLoading.value = true
    try {
      const authData = await login(credentials)

      const account = addAccount(authData, false)

      // 同时设置到AuthStore
      const authStore = useAuthStore()
      await authStore.setAuthData(authData)

      return account
    } catch (error) {
      throw error
    } finally {
      isLoading.value = false
    }
  }

  // 添加新用户（注册）
  async function addUserByRegister(userData: RegisterRequest) {
    isLoading.value = true
    try {
      const authData = await register(userData)

      const account = addAccount(authData, false)

      // 同时设置到AuthStore
      const authStore = useAuthStore()
      await authStore.setAuthData(authData)

      return account
    } catch (error) {
      throw error
    } finally {
      isLoading.value = false
    }
  }

  // 创建临时用户
  async function createTempUserAccount(nickname?: string) {
    isLoading.value = true
    try {
      const authData = await createTempUser(nickname)

      const account = addAccount(authData, true)

      // 同时设置到AuthStore
      const authStore = useAuthStore()
      await authStore.setAuthData(authData)

      return account
    } catch (error) {
      throw error
    } finally {
      isLoading.value = false
    }
  }

  // 自动选择用户
  async function autoSelectUser() {
    // 1. 如果已有当前用户，直接切换到该用户
    if (currentUserId.value && accounts.value.find(account => account.user_id === currentUserId.value)) {
      try {
        await switchToUser(currentUserId.value)
        return currentUser.value
      } catch (error) {
        console.warn('切换到上次用户失败:', error)
      }
    }

    // 2. 如果有其他用户，切换到最近使用的
    if (accounts.value.length > 0) {
      const mostRecent = recentUsers.value[0]
      try {
        await switchToUser(mostRecent.user_id)
        return currentUser.value
      } catch (error) {
        console.warn('切换到最近用户失败:', error)
      }
    }

    // 3. 没有任何用户，创建临时用户
    try {
      const tempUser = await createTempUserAccount()
      return tempUser
    } catch (error) {
      console.error('创建临时用户失败:', error)
      return null
    }
  }

  // 清理过期账户
  function cleanupExpiredAccounts() {
    const now = new Date().getTime()
    const validAccounts = accounts.value.filter(user => {
      // 检查是否过期（超过30天未使用的临时用户）
      if (user.is_temporary) {
        const lastUsed = new Date(user.last_used).getTime()
        const daysSinceLastUse = (now - lastUsed) / (1000 * 60 * 60 * 24)
        return daysSinceLastUse <= 30
      }
      return true
    })

    if (validAccounts.length !== accounts.value.length) {
      accounts.value = validAccounts
    }
  }

  // 初始化
  function initialize() {
    cleanupExpiredAccounts()
  }

  return {
    // 状态
    accounts,
    currentUserId,
    recentUsers,
    isLoading,

    // 计算属性
    currentUser,
    hasUsers,

    // 方法
    initialize,
    switchToUser,
    removeAccount,
    addUserByLogin,
    addUserByRegister,
    createTempUserAccount,
    autoSelectUser,
    cleanupExpiredAccounts,
    addAccount
  }
}, {
  persist: {
    key: 'hotspot-user-manager',
    storage: typeof window !== 'undefined' ? localStorage : undefined,
    debug: process.dev
  }
}) 