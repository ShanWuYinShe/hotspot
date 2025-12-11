<template>
  <div class="min-h-screen flex">
    <!-- 左侧装饰区域（仅桌面端） -->
    <div class="hidden lg:flex lg:w-1/2 relative bg-linear-to-br from-primary-600 via-primary-500 to-accent-500 overflow-hidden">
      <!-- 背景装饰 -->
      <div class="absolute inset-0">
        <div class="absolute top-20 left-20 w-72 h-72 bg-white/10 rounded-full blur-3xl" />
        <div class="absolute bottom-20 right-20 w-96 h-96 bg-white/10 rounded-full blur-3xl" />
        <div class="absolute top-1/2 left-1/3 w-48 h-48 bg-white/5 rounded-full blur-2xl" />
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
            发现身边的<br />
            <span class="text-white/80">精彩瞬间</span>
          </h1>
          <p class="text-lg text-white/70 mb-12">
            基于位置的社交平台，连接你与周围的人和事
          </p>
          
          <!-- 特性列表 -->
          <div class="space-y-4">
            <div class="flex items-center gap-4 text-white/90">
              <div class="w-10 h-10 rounded-xl bg-white/20 flex items-center justify-center">
                <UIcon name="i-heroicons-map" class="icon-md" />
              </div>
              <span>探索附近的热门地点和活动</span>
            </div>
            <div class="flex items-center gap-4 text-white/90">
              <div class="w-10 h-10 rounded-xl bg-white/20 flex items-center justify-center">
                <UIcon name="i-heroicons-users" class="icon-md" />
              </div>
              <span>结识志同道合的新朋友</span>
            </div>
            <div class="flex items-center gap-4 text-white/90">
              <div class="w-10 h-10 rounded-xl bg-white/20 flex items-center justify-center">
                <UIcon name="i-heroicons-chat-bubble-left-right" class="icon-md" />
              </div>
              <span>实时分享你的生活瞬间</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧登录区域 -->
    <div class="flex-1 flex items-center justify-center p-6 sm:p-12 bg-surface">
      <div class="w-full max-w-md">
        <!-- 移动端 Logo -->
        <div class="lg:hidden flex items-center justify-center gap-2 mb-8">
          <div class="w-10 h-10 rounded-xl bg-icon-primary flex items-center justify-center">
            <UIcon name="i-heroicons-map-pin" class="w-6 h-6 text-white" />
          </div>
          <span class="text-2xl font-bold text-primary-content">HotSpot</span>
        </div>

        <!-- 登录卡片 -->
        <UCard :ui="{ root: 'shadow-xl border-0', body: 'p-8' }">
          <!-- 标题 -->
          <div class="text-center mb-8">
            <h2 class="text-2xl font-bold text-primary-content mb-2">
              欢迎回来
            </h2>
            <p class="text-secondary-content">
              登录您的 HotSpot 账户
            </p>
          </div>

          <!-- 错误提示 -->
          <UAlert 
            v-if="loginError"
            color="error" 
            icon="i-heroicons-exclamation-triangle" 
            :title="loginError"
            :description="loginErrorDescription || undefined"
            class="mb-6"
          />

          <!-- 登录表单 -->
          <UForm :schema="loginSchema" :state="formState" @submit="handleLogin" class="space-y-5">
            <UFormField label="用户名 / 邮箱" name="loginId" required>
              <UInput 
                v-model="formState.loginId"
                placeholder="请输入用户名或邮箱"
                size="lg"
                icon="i-heroicons-user"
                :disabled="isLoading"
              />
            </UFormField>
            
            <UFormField label="密码" name="password" required>
              <template #hint>
                <UButton 
                  variant="link" 
                  size="xs" 
                  color="neutral"
                  @click="showForgotPassword"
                  class="text-xs"
                >
                  忘记密码？
                </UButton>
              </template>
              <UInput 
                v-model="formState.password"
                type="password"
                placeholder="请输入密码"
                size="lg"
                icon="i-heroicons-lock-closed"
                :disabled="isLoading"
              />
            </UFormField>

            <div class="flex items-center justify-between">
              <UCheckbox v-model="formState.remember" label="记住我" />
            </div>

            <UButton
              type="submit"
              color="primary"
              size="lg"
              block
              :loading="isLoading"
            >
              登录
            </UButton>
          </UForm>

          <!-- 分隔线 -->
          <USeparator label="或者" class="my-8" />

          <!-- 快速体验按钮 - 突出显示 -->
          <div class="space-y-3">
            <UButton
              variant="soft"
              color="primary"
              size="lg"
              block
              :loading="isCreatingTemp"
              @click="createTempUser"
            >
              <UIcon name="i-heroicons-bolt" class="w-5 h-5 mr-2" />
              免注册快速体验
            </UButton>
            <p class="text-center text-xs text-secondary-content">
              无需注册，即可体验 HotSpot 的全部功能
            </p>
          </div>

          <!-- 第三方登录 -->
          <div class="mt-6">
            <div class="grid grid-cols-2 gap-3">
              <UButton
                variant="outline"
                color="neutral"
                size="lg"
                @click="handleWechatLogin"
              >
                <UIcon name="i-simple-icons-wechat" class="w-5 h-5 mr-2 text-green" />
                微信
              </UButton>
              <UButton
                variant="outline"
                color="neutral"
                size="lg"
                @click="handleQQLogin"
              >
                <UIcon name="i-simple-icons-tencentqq" class="w-5 h-5 mr-2 text-blue" />
                QQ
              </UButton>
            </div>
          </div>

          <!-- 注册链接 -->
          <p class="text-center text-sm text-secondary-content mt-8">
            还没有账号？
            <ULink to="/auth/register" class="text-accent hover:text-accent font-medium">
              立即注册
            </ULink>
          </p>
        </UCard>

        <!-- 底部信息 -->
        <p class="text-center text-xs text-muted mt-8">
          登录即表示您同意我们的
          <ULink to="/terms" class="underline">服务条款</ULink>
          和
          <ULink to="/privacy" class="underline">隐私政策</ULink>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from '~/config/seo'
import { getErrorMessage } from '~/utils/error'
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

// SEO
const seoConfig = getPageSeoConfig('auth-login')
definePageMeta({ 
  layout: 'auth',
  title: seoConfig.title,
  description: seoConfig.description
})
useSeoMeta(buildSeoMeta('auth-login'))

const authStore = useAuthStore()
const toast = useToast()
const modal = useModal()

// 表单状态
const formState = reactive({
  loginId: '',
  password: '',
  remember: false
})

const isLoading = ref(false)
const isCreatingTemp = ref(false)
const loginError = ref<string | null>(null)
const loginErrorDescription = ref<string | null>(null)

// 验证模式
const loginSchema = z.object({
  loginId: z.string().min(1, '请输入用户名'),
  password: z.string().min(1, '请输入密码'),
  remember: z.boolean().optional()
})

type LoginSchema = z.output<typeof loginSchema>

// 处理登录
const handleLogin = async (event: FormSubmitEvent<LoginSchema>) => {
  try {
    isLoading.value = true
    loginError.value = null
    loginErrorDescription.value = null

    const { loginId, password } = event.data
    
    const { hashPassword } = await import('~/utils/password')
    const password_hash = await hashPassword(password)
    
    const { useAuthService } = await import('~/services/auth.service')
    const authService = useAuthService()
    const authData = await authService.login({
      login_id: loginId,
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
      title: '登录成功',
      description: '欢迎回来！',
      color: 'success'
    })
    
    await navigateTo('/')
  } catch (error: unknown) {
    loginError.value = '登录失败'
    loginErrorDescription.value = getErrorMessage(error, '请检查用户名和密码后重试')
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

// 忘记密码
const showForgotPassword = async () => {
  await modal.showConfirm({
    title: '重置密码',
    message: '当前版本暂未开放密码找回服务，请联系管理员。',
    confirmText: '我知道了',
    confirmColor: 'primary'
  })
}

// 第三方登录（暂未开放）
const handleWechatLogin = () => {
  toast.add({ 
    title: '微信登录', 
    description: '功能开发中，敬请期待',
    color: 'info'
  })
}

const handleQQLogin = () => {
  toast.add({ 
    title: 'QQ登录', 
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
  'r': {
    usingInput: false,
    handler: () => navigateTo('/auth/register'),
  },
})
</script>
