<template>
  <div class="min-h-screen flex">
    <!-- 左侧装饰区域（仅桌面端） -->
    <div class="hidden lg:flex lg:w-1/2 relative bg-linear-to-br from-accent-600 via-accent-500 to-primary-500 overflow-hidden">
      <!-- 背景装饰 -->
      <div class="absolute inset-0">
        <div class="absolute top-20 right-20 w-72 h-72 bg-white/10 rounded-full blur-3xl" />
        <div class="absolute bottom-20 left-20 w-96 h-96 bg-white/10 rounded-full blur-3xl" />
        <div class="absolute top-1/3 right-1/3 w-48 h-48 bg-white/5 rounded-full blur-2xl" />
      </div>
      
      <!-- 内容 -->
      <div class="relative z-10 flex flex-col justify-center px-12 xl:px-20">
        <div class="max-w-lg">
          <!-- Logo -->
          <div class="flex items-center gap-3 mb-12">
            <div class="w-12 h-12 rounded-2xl bg-white/20 backdrop-blur flex items-center justify-center">
              <UIcon name="i-heroicons-map-pin" class="w-7 h-7 text-white" />
            </div>
            <span class="text-3xl font-bold text-white">HotSpot</span>
          </div>
          
          <!-- 标语 -->
          <h1 class="text-4xl xl:text-5xl font-bold text-white mb-6 leading-tight">
            加入我们<br />
            <span class="text-white/80">开启新旅程</span>
          </h1>
          <p class="text-lg text-white/70 mb-12">
            创建账户，与附近的朋友分享生活中的美好时刻
          </p>
          
          <!-- 统计数据 -->
          <div class="grid grid-cols-3 gap-6">
            <div>
              <p class="text-3xl font-bold text-white">10K+</p>
              <p class="text-sm text-white/60">活跃用户</p>
            </div>
            <div>
              <p class="text-3xl font-bold text-white">50K+</p>
              <p class="text-sm text-white/60">精彩瞬间</p>
            </div>
            <div>
              <p class="text-3xl font-bold text-white">100+</p>
              <p class="text-sm text-white/60">热门地点</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧注册区域 -->
    <div class="flex-1 flex items-center justify-center p-6 sm:p-12 bg-surface">
      <div class="w-full max-w-md">
        <!-- 移动端 Logo -->
        <div class="lg:hidden flex items-center justify-center gap-2 mb-8">
          <div class="w-10 h-10 rounded-xl bg-icon-primary flex items-center justify-center">
            <UIcon name="i-heroicons-map-pin" class="w-6 h-6 text-white" />
          </div>
          <span class="text-2xl font-bold text-primary-content">HotSpot</span>
        </div>

        <!-- 注册卡片 -->
        <UCard :ui="{ root: 'shadow-xl border-0', body: 'p-8' }">
          <!-- 标题 -->
          <div class="text-center mb-8">
            <h2 class="text-2xl font-bold text-primary-content mb-2">
              创建账户
            </h2>
            <p class="text-secondary-content">
              加入 HotSpot，开始探索
            </p>
          </div>

          <!-- 错误提示 -->
          <UAlert 
            v-if="registerError"
            color="error" 
            icon="i-heroicons-exclamation-triangle" 
            :title="registerError"
            :description="registerErrorDescription || undefined"
            class="mb-6"
          />

          <!-- 注册表单 -->
          <UForm :schema="registerSchema" :state="formState" @submit="handleRegister" class="space-y-4">
            <UFormField label="用户名" name="loginId" required help="3-32个字符，支持字母、数字和下划线">
              <UInput 
                v-model="formState.loginId"
                placeholder="请输入用户名"
                size="lg"
                icon="i-heroicons-at-symbol"
                :disabled="isLoading"
              />
            </UFormField>

            <UFormField label="昵称" name="nickname" help="用于显示的友好名称（可选）">
              <UInput 
                v-model="formState.nickname"
                placeholder="请输入昵称"
                size="lg"
                icon="i-heroicons-user"
                :disabled="isLoading"
              />
            </UFormField>
            
            <UFormField label="密码" name="password" required>
              <UInput 
                v-model="formState.password"
                type="password"
                placeholder="请设置密码（至少6位）"
                size="lg"
                icon="i-heroicons-lock-closed"
                :disabled="isLoading"
              />
            </UFormField>

            <UFormField label="确认密码" name="confirmPassword" required>
              <UInput 
                v-model="formState.confirmPassword"
                type="password"
                placeholder="请再次输入密码"
                size="lg"
                icon="i-heroicons-lock-closed"
                :disabled="isLoading"
              />
            </UFormField>

            <UCheckbox 
              v-model="formState.terms" 
              name="terms"
              required
            >
              <template #label>
                <span class="text-sm text-secondary-content">
                  我同意
                  <ULink to="/terms" class="text-accent">服务条款</ULink>
                  和
                  <ULink to="/privacy" class="text-accent">隐私政策</ULink>
                </span>
              </template>
            </UCheckbox>

            <UButton
              type="submit"
              color="primary"
              size="lg"
              block
              :loading="isLoading"
              class="mt-6"
            >
              注册
            </UButton>
          </UForm>

          <!-- 分隔线 -->
          <USeparator label="或者" class="my-8" />

          <!-- 快速体验按钮 - 突出显示 -->
          <div class="space-y-3">
            <UButton
              variant="soft"
              color="accent"
              size="lg"
              block
              :loading="isCreatingTemp"
              @click="createTempUser"
            >
              <UIcon name="i-heroicons-bolt" class="w-5 h-5 mr-2" />
              免注册快速体验
            </UButton>
            <p class="text-center text-xs text-secondary-content">
              创建临时账户，随时可升级为正式账户
            </p>
          </div>

          <!-- 第三方注册 -->
          <div class="mt-6">
            <div class="grid grid-cols-2 gap-3">
              <UButton
                variant="outline"
                color="neutral"
                size="lg"
                @click="handleWechatRegister"
              >
                <UIcon name="i-simple-icons-wechat" class="w-5 h-5 mr-2 text-green" />
                微信
              </UButton>
              <UButton
                variant="outline"
                color="neutral"
                size="lg"
                @click="handleQQRegister"
              >
                <UIcon name="i-simple-icons-tencentqq" class="w-5 h-5 mr-2 text-blue" />
                QQ
              </UButton>
            </div>
          </div>

          <!-- 登录链接 -->
          <p class="text-center text-sm text-secondary-content mt-8">
            已有账号？
            <ULink to="/auth/login" class="text-accent hover:text-accent font-medium">
              立即登录
            </ULink>
          </p>
        </UCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'
import { getErrorMessage } from '~/utils/error'

definePageMeta({ 
  layout: 'auth',
  title: '注册'
})

const authStore = useAuthStore()
const toast = useToast()

// 表单状态
const formState = reactive({
  loginId: '',
  nickname: '',
  password: '',
  confirmPassword: '',
  terms: false
})

const isLoading = ref(false)
const isCreatingTemp = ref(false)
const registerError = ref<string | null>(null)
const registerErrorDescription = ref<string | null>(null)

// 验证模式
const registerSchema = z.object({
  loginId: z.string()
    .min(3, '用户名至少3个字符')
    .max(32, '用户名最多32个字符')
    .regex(/^[a-zA-Z0-9_]+$/, '用户名只能包含字母、数字和下划线'),
  nickname: z.string()
    .min(2, '昵称至少2个字符')
    .max(32, '昵称最多32个字符')
    .optional()
    .or(z.literal('')),
  password: z.string()
    .min(6, '密码至少6个字符')
    .max(128, '密码最多128个字符'),
  confirmPassword: z.string(),
  terms: z.boolean().refine(val => val === true, '请同意服务条款')
}).refine((data) => data.password === data.confirmPassword, {
  message: '两次输入的密码不一致',
  path: ['confirmPassword'],
})

type RegisterSchema = z.output<typeof registerSchema>

// 生成默认昵称
const generateNickname = () => `用户${Math.floor(Date.now() % 100000)}`

// 处理注册
const handleRegister = async (event: FormSubmitEvent<RegisterSchema>) => {
  try {
    isLoading.value = true
    registerError.value = null
    registerErrorDescription.value = null

    const { loginId, nickname, password } = event.data
    const finalNickname = nickname?.trim() || generateNickname()
    
    const { hashPassword } = await import('~/utils/password')
    const password_hash = await hashPassword(password)
    
    const { useAuthService } = await import('~/services/auth.service')
    const authService = useAuthService()
    const authData = await authService.register({
      login_id: loginId,
      nickname: finalNickname,
      password_hash
    })
    authStore.setAccountSession(
      { user_id: authData.user_id, nickname: authData.nickname },
      authData.access_token,
      authData.refresh_token,
      new Date(authData.access_token_expires_at).getTime(),
      new Date(authData.refresh_token_expires_at).getTime()
    )
    
    toast.add({
      title: '注册成功',
      description: '欢迎加入 HotSpot！',
      color: 'success'
    })
    
    await navigateTo('/')
  } catch (error: unknown) {
    registerError.value = '注册失败'
    registerErrorDescription.value = getErrorMessage(error, '请稍后重试')
  } finally {
    isLoading.value = false
  }
}

// 创建临时用户
const createTempUser = async () => {
  try {
    isCreatingTemp.value = true
    const { useAuthService } = await import('~/services/auth.service')
    const authService = useAuthService()
    const authData = await authService.createTempUser({ nickname: `访客_${Date.now().toString(36).slice(-4)}` })
    authStore.setAccountSession(
      { user_id: authData.user_id, nickname: authData.nickname },
      authData.access_token,
      authData.refresh_token,
      new Date(authData.access_token_expires_at).getTime(),
      new Date(authData.refresh_token_expires_at).getTime()
    )
    toast.add({
      title: '欢迎体验 HotSpot',
      description: '临时账户已创建，开始探索吧！',
      color: 'success'
    })
    await navigateTo('/')
  } catch (error: unknown) {
    toast.add({
      title: '创建失败',
      description: getErrorMessage(error, '请稍后重试'),
      color: 'error'
    })
  } finally {
    isCreatingTemp.value = false
  }
}

// 第三方注册（暂未开放）
const handleWechatRegister = () => {
  toast.add({ 
    title: '微信注册', 
    description: '功能开发中，敬请期待',
    color: 'info'
  })
}

const handleQQRegister = () => {
  toast.add({ 
    title: 'QQ注册', 
    description: '功能开发中，敬请期待',
    color: 'info'
  })
}

// 已登录则重定向
watch(
  () => authStore.isAuthenticated,
  (isAuth) => {
    if (isAuth) navigateTo('/')
  },
  { immediate: true }
)

// 快捷键
defineShortcuts({
  'l': {
    usingInput: false,
    handler: () => navigateTo('/auth/login'),
  },
})
</script>
