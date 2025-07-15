import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, register as registerApi, createTempUser as createTempUserApi } from '~/services/auth/auth.service'
import { extractErrorMessage } from '~/utils/error-handler'

export const useAuthStore = defineStore('auth', () => {
  // 基础状态
  const user = ref<any>(null)
  const token = ref<string | null>(null)
  const refreshToken = ref<string | null>(null)
  const expiresAt = ref<Date | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  
  // 计算属性
  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const isTempUser = computed(() => {
    if (!user.value) return false
    return user.value.nickname?.startsWith('游客') || !user.value.login_id
  })
  
  // 设置认证数据
  function setAuthData(authData: any) {
    user.value = {
      user_id: authData.user_id,
      login_id: authData.login_id || '',
      nickname: authData.nickname,
      avatar_url: authData.avatar_url || '',
      email: authData.email || ''
    }
    token.value = authData.access_token
    refreshToken.value = authData.refresh_token
    expiresAt.value = new Date(authData.expires_at)
    error.value = null
  }
  
  // 清除认证数据
  function clearAuthData() {
    user.value = null
    token.value = null
    refreshToken.value = null
    expiresAt.value = null
  }
  
  // 创建临时用户
  async function createTempUser(nickname?: string) {
    isLoading.value = true
    error.value = null
    try {
      const authData = await createTempUserApi(nickname)
      setAuthData(authData)
      return authData
    } catch (err: any) {
      const errorMessage = extractErrorMessage(err, '创建临时用户失败，请稍后重试')
      error.value = errorMessage
      throw new Error(errorMessage)
    } finally {
      isLoading.value = false
    }
  }
  
  // 登录
  async function login(credentials: any) {
    isLoading.value = true
    error.value = null
    try {
      const authData = await loginApi(credentials)
      setAuthData(authData)
      return authData
    } catch (err: any) {
      const errorMessage = extractErrorMessage(err, '登录失败，请稍后重试')
      error.value = errorMessage
      throw new Error(errorMessage)
    } finally {
      isLoading.value = false
    }
  }
  
  // 注册
  async function register(userData: any) {
    isLoading.value = true
    error.value = null
    try {
      const authData = await registerApi(userData)
      setAuthData(authData)
      return authData
    } catch (err: any) {
      const errorMessage = extractErrorMessage(err, '注册失败，请稍后重试')
      error.value = errorMessage
      throw new Error(errorMessage)
    } finally {
      isLoading.value = false
    }
  }
  
  // 登出
  function logout() {
    clearAuthData()
  }
  
  // 生成友好的临时用户昵称
  function generateTempNickname(): string {
    const adjectives = ['探索', '漫游', '发现', '寻找', '遇见', '追寻']
    const nouns = ['者', '客', '人', '员']
    const randomAdjective = adjectives[Math.floor(Math.random() * adjectives.length)]
    const randomNoun = nouns[Math.floor(Math.random() * nouns.length)]
    const randomNum = Math.floor(Math.random() * 9999) + 1
    return `${randomAdjective}${randomNoun}${randomNum}`
  }

  // 验证临时用户数据有效性
  function isValidTempUserData(userData: any): boolean {
    if (!userData || !userData.user_id || !userData.access_token) {
      return false
    }

    // 检查数据创建时间（超过7天认为无效）
    const createTime = userData.created_at ? new Date(userData.created_at).getTime() : 0
    const daysSinceCreate = (Date.now() - createTime) / (1000 * 60 * 60 * 24)

    if (daysSinceCreate > 7) {
      return false
    }

    // 检查token是否过期
    if (userData.expires_at) {
      const expiresAt = new Date(userData.expires_at).getTime()
      if (Date.now() >= expiresAt) {
        return false
      }
    }

    return true
  }

  // 检查本地临时用户数据
  async function checkLocalTempUser() {
    if (typeof window === 'undefined') return null
    
    const tempUserData = localStorage.getItem('temp_user_backup')
    if (tempUserData) {
      try {
        const userData = JSON.parse(tempUserData)

        if (isValidTempUserData(userData)) {
          setAuthData(userData)
          console.log('✅ 恢复本地临时用户身份')
          return user.value
        } else {
          localStorage.removeItem('temp_user_backup')
        }
      } catch (error) {
        console.error('解析临时用户数据失败:', error)
        localStorage.removeItem('temp_user_backup')
      }
    }

    return null
  }

  // 备份临时用户数据
  function saveTempUserBackup(userData: any) {
    if (typeof window === 'undefined') return
    
    try {
      const backupData = {
        ...userData,
        created_at: new Date().toISOString(),
        backup_timestamp: Date.now()
      }
      localStorage.setItem('temp_user_backup', JSON.stringify(backupData))
    } catch (error) {
      console.error('备份临时用户数据失败:', error)
    }
  }

  // 防重复创建的全局状态
  let tempUserCreationPromise: Promise<any> | null = null

  // 检测并创建临时用户
  async function checkAndCreateTempUser() {
    // 1. 如果已有认证用户，直接返回
    if (isAuthenticated.value) {
      return user.value
    }

    // 2. 检查本地是否有临时用户备份数据
    const localTempUser = await checkLocalTempUser()
    if (localTempUser) {
      return localTempUser
    }

    // 3. 防重复创建机制
    if (tempUserCreationPromise) {
      return tempUserCreationPromise
    }

    // 4. 创建新的临时用户
    try {
      tempUserCreationPromise = performTempUserCreation()
      const result = await tempUserCreationPromise
      return result
    } finally {
      tempUserCreationPromise = null
    }
  }

  // 执行临时用户创建
  async function performTempUserCreation() {
    try {
      const tempNickname = generateTempNickname()
      
      const tempUser = await createTempUser(tempNickname)

      if (tempUser) {
        saveTempUserBackup(tempUser)
        console.log('✅ 临时用户创建成功:', tempNickname)
        return user.value
      }

      throw new Error('临时用户创建返回空数据')
    } catch (error) {
      console.error('创建临时用户失败:', error)
      return null
    }
  }

  // 清理过期数据
  function cleanupExpiredData() {
    if (typeof window === 'undefined') return
    
    const tempUserData = localStorage.getItem('temp_user_backup')
    if (tempUserData) {
      try {
        const userData = JSON.parse(tempUserData)
        if (!isValidTempUserData(userData)) {
          localStorage.removeItem('temp_user_backup')
          console.log('✅ 已清理过期的临时用户数据')
        }
      } catch (error) {
        localStorage.removeItem('temp_user_backup')
      }
    }
  }

  return {
    // 状态
    user,
    token,
    refreshToken,
    expiresAt,
    isLoading,
    error,
    // 计算属性
    isAuthenticated,
    isTempUser,
    // 基础方法
    setAuthData,
    clearAuthData,
    login,
    register,
    createTempUser,
    logout,
    // 自动用户注册功能
    checkAndCreateTempUser,
    cleanupExpiredData
  }
}, {
  persist: {
    storage: typeof window !== 'undefined' ? localStorage : undefined
  }
})
