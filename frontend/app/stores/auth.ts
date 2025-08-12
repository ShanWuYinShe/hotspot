import { defineStore } from 'pinia'
import { ref, computed, readonly } from 'vue'
import type { User, AuthResponse } from '~/types'

export const useAuthStore = defineStore('auth', () => {
  // 状态
  const user = ref<User | null>(null)
  const token = ref<string | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // 计算属性
  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const currentUser = computed(() => user.value)

  // 设置认证数据
  function setAuthData(authData: AuthResponse) {
    user.value = {
      user_id: authData.user_id,
      login_id: authData.login_id,
      nickname: authData.nickname,
      avatar_url: authData.avatar_url,
      email: authData.email
    }
    token.value = authData.access_token
    error.value = null
  }

  // 清除认证数据
  function clearAuth() {
    user.value = null
    token.value = null
    error.value = null
  }

  // 登录
  async function login(loginId: string, password: string) {
    isLoading.value = true
    error.value = null
    
    try {
      const runtimeConfig = useRuntimeConfig()
      const { data } = await $fetch<{ data: AuthResponse }>('/auth/login', {
        method: 'POST',
        baseURL:  runtimeConfig.public.apiBaseUrl,
        body: { login_id: loginId, password }
      })
      
      setAuthData(data)
      return data
    } catch (err: any) {
      error.value = err.data?.message || '登录失败'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  // 注册
  async function register(loginId: string, nickname: string, password: string) {
    isLoading.value = true
    error.value = null
    
    try {
      const runtimeConfig = useRuntimeConfig()
      const { data } = await $fetch<{ data: AuthResponse }>('/auth/register', {
        method: 'POST',
        baseURL: runtimeConfig.public.apiBaseUrl,
        body: { login_id: loginId, nickname, password }
      })
      
      setAuthData(data)
      return data
    } catch (err: any) {
      error.value = err.data?.message || '注册失败'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  // 创建临时用户
  async function createTempUser(nickname?: string) {
    isLoading.value = true
    error.value = null
    
    try {
      const runtimeConfig = useRuntimeConfig()
      const { data } = await $fetch<{ data: AuthResponse }>('/auth/temp-user', {
        method: 'POST',
        baseURL: runtimeConfig.public.apiBaseUrl,
        body: { nickname }
      })
      
      setAuthData(data)
      return data
    } catch (err: any) {
      error.value = err.data?.message || '创建临时用户失败'
      throw err
    } finally {
      isLoading.value = false
    }
  }

  // 登出
  function logout() {
    clearAuth()
    navigateTo('/auth')
  }

  return {
    // 状态
    user: readonly(user),
    token: readonly(token),
    isLoading: readonly(isLoading),
    error: readonly(error),
    
    // 计算属性
    isAuthenticated,
    currentUser,
    
    // 方法
    setAuthData,
    clearAuth,
    login,
    register,
    createTempUser,
    logout
  }
}, {
  persist: true
})
