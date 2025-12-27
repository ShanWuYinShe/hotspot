<template>
  <div class="min-h-screen flex items-center justify-center px-4 py-10">
    <UPageCard class="w-full max-w-md">
      <div class="flex flex-col gap-6">
        <UTabs v-model="activeTab" :items="tabs" />

        <UAuthForm
          :schema="activeSchema"
          :fields="activeFields"
          :providers="providers"
          :loading="isLoading"
          :title="activeTitle"
          :description="activeDescription"
          icon="i-lucide-map-pin"
          separator="或者"
          :submit="{ label: activeSubmitLabel, block: true }"
          @submit="handleSubmit"
        >
          <template v-if="activeTab === 'login'" #password-hint>
            <UButton variant="link" size="xs" color="primary" @click="showForgotPassword">
              忘记密码？
            </UButton>
          </template>

          <template #footer>
            <div class="text-sm">
              <template v-if="activeTab === 'login'">
                还没有账号？
                <UButton variant="link" size="xs" color="primary" @click="switchTo('register')">
                  立即注册
                </UButton>
              </template>
              <template v-else>
                已有账号？
                <UButton variant="link" size="xs" color="primary" @click="switchTo('login')">
                  立即登录
                </UButton>
              </template>
            </div>
          </template>
        </UAuthForm>
      </div>
    </UPageCard>
  </div>
</template>

<script setup lang="ts">
import * as z from 'zod'
import type { AuthFormField, FormSubmitEvent } from '@nuxt/ui'
import { useAuthService } from '~/services/auth.service'
import { useUsersService } from '~/services/users.service'
import { useFeedbackStore } from '~/stores/feedback.store'

definePageMeta({
  layout: 'auth',
  title: '认证'
})

type AuthTab = 'login' | 'register'

const authStore = useAuthStore()
const feedback = useFeedbackStore()
const modal = useModal()
const route = useRoute()
const { withErrorHandling } = useErrorHandler()
const authService = useAuthService()
const usersService = useUsersService()

const isLoading = ref(false)
const isCreatingTemp = ref(false)

const isUpgrade = computed(() => route.query.upgrade === '1')

const tabs = computed(() => {
  const registerLabel = isUpgrade.value ? '升级账户' : '注册'
  return [
    { label: '登录', value: 'login' },
    { label: registerLabel, value: 'register' }
  ]
})

const activeTab = ref<AuthTab>((route.query.mode === 'register' ? 'register' : 'login') as AuthTab)

watch(
  () => route.query.mode,
  (mode) => {
    if (mode === 'register') activeTab.value = 'register'
    if (mode === 'login') activeTab.value = 'login'
  }
)

watch(
  isUpgrade,
  (v) => {
    if (v) activeTab.value = 'register'
  },
  { immediate: true }
)

const switchTo = async (tab: AuthTab) => {
  activeTab.value = tab
  const query = { ...route.query, mode: tab }
  await navigateTo({ path: '/auth', query })
}

// ============ schema ============

const loginSchema = z.object({
  loginId: z.string().min(1, '请输入用户名'),
  password: z.string().min(1, '请输入密码'),
  remember: z.boolean().optional()
})
type LoginSchema = z.output<typeof loginSchema>

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
  path: ['confirmPassword']
})
type RegisterSchema = z.output<typeof registerSchema>

const upgradeSchema = z.object({
  loginId: z.string()
    .min(3, '用户名至少3个字符')
    .max(32, '用户名最多32个字符')
    .regex(/^[a-zA-Z0-9_]+$/, '用户名只能包含字母、数字和下划线'),
  password: z.string()
    .min(6, '密码至少6个字符')
    .max(128, '密码最多128个字符'),
  confirmPassword: z.string()
}).refine((data) => data.password === data.confirmPassword, {
  message: '两次输入的密码不一致',
  path: ['confirmPassword']
})
type UpgradeSchema = z.output<typeof upgradeSchema>

const loginFields: AuthFormField[] = [
  { name: 'loginId', label: '用户名 / 邮箱', placeholder: '请输入用户名或邮箱', type: 'text', required: true, icon: 'i-lucide-user' },
  { name: 'password', label: '密码', placeholder: '请输入密码', type: 'password', required: true, icon: 'i-lucide-lock' },
  { name: 'remember', label: '记住我', type: 'checkbox' }
]

const registerFields: AuthFormField[] = [
  { name: 'loginId', label: '用户名', placeholder: '请输入用户名', type: 'text', required: true, help: '3-32个字符，支持字母、数字和下划线', icon: 'i-lucide-at-sign' },
  { name: 'nickname', label: '昵称', placeholder: '可选，不填将自动生成', type: 'text', icon: 'i-lucide-user' },
  { name: 'password', label: '密码', placeholder: '请输入密码（至少6位）', type: 'password', required: true, icon: 'i-lucide-lock' },
  { name: 'confirmPassword', label: '确认密码', placeholder: '请再次输入密码', type: 'password', required: true, icon: 'i-lucide-lock' },
  { name: 'terms', label: '我已阅读并同意服务条款', type: 'checkbox', required: true }
]

const upgradeFields: AuthFormField[] = [
  { name: 'loginId', label: '用户名', placeholder: '请输入用户名', type: 'text', required: true, help: '3-32个字符，支持字母、数字和下划线', icon: 'i-lucide-at-sign' },
  { name: 'password', label: '设置密码', placeholder: '请输入密码（至少6位）', type: 'password', required: true, icon: 'i-lucide-lock' },
  { name: 'confirmPassword', label: '确认密码', placeholder: '请再次输入密码', type: 'password', required: true, icon: 'i-lucide-lock' }
]

const activeSchema = computed(() => {
  if (activeTab.value === 'login') return loginSchema
  return isUpgrade.value ? upgradeSchema : registerSchema
})

const activeFields = computed(() => {
  if (activeTab.value === 'login') return loginFields
  return isUpgrade.value ? upgradeFields : registerFields
})

const activeTitle = computed(() => {
  if (activeTab.value === 'login') return '欢迎回来'
  return isUpgrade.value ? '升级账户' : '创建账户'
})

const activeDescription = computed(() => {
  if (activeTab.value === 'login') return '登录您的账户以继续'
  return isUpgrade.value ? '为临时账户设置登录凭据' : '加入 HotSpot，开始探索'
})

const activeSubmitLabel = computed(() => {
  if (activeTab.value === 'login') return '登录'
  return isUpgrade.value ? '升级' : '注册'
})

const providers = computed(() => [
  {
    label: '免注册快速体验',
    icon: 'i-lucide-zap',
    color: 'neutral',
    variant: 'soft',
    block: true,
    loading: isCreatingTemp.value,
    onClick: createTempUser
  }
])

const generateNickname = () => `用户${Math.floor(Date.now() % 100000)}`

const setSessionFromAuth = (authData: { user_id: number; nickname: string; access_token: string; refresh_token?: string; access_token_expires_at: string; refresh_token_expires_at: string }) => {
  authStore.setAccountSession(
    { user_id: authData.user_id, nickname: authData.nickname },
    authData.access_token,
    authData.refresh_token ?? null,
    new Date(authData.access_token_expires_at).getTime(),
    new Date(authData.refresh_token_expires_at).getTime()
  )
}

const handleSubmit = async (event: FormSubmitEvent<LoginSchema | RegisterSchema | UpgradeSchema>) => {
  if (activeTab.value === 'login') {
    return handleLogin(event as FormSubmitEvent<LoginSchema>)
  }
  if (isUpgrade.value) {
    return handleUpgrade(event as FormSubmitEvent<UpgradeSchema>)
  }
  return handleRegister(event as FormSubmitEvent<RegisterSchema>)
}

const handleLogin = async (event: FormSubmitEvent<LoginSchema>) => {
  await withErrorHandling(async () => {
    const { loginId, password } = event.data
    const authData = await authService.login({ login_id: loginId, password, client_type: 'app' })
    setSessionFromAuth(authData)
    feedback.success('登录成功', '欢迎回来！')
    const redirect = (route.query.redirect as string) || '/'
    await navigateTo(redirect)
  }, {
    title: '登录失败',
    fallbackMessage: '请检查用户名和密码后重试',
    loading: isLoading
  })
}

const handleRegister = async (event: FormSubmitEvent<RegisterSchema>) => {
  await withErrorHandling(async () => {
    const { loginId, nickname, password } = event.data
    const finalNickname = nickname?.trim() || generateNickname()
    const authData = await authService.register({ login_id: loginId, nickname: finalNickname, password, client_type: 'app' })
    setSessionFromAuth(authData)
    feedback.success('注册成功', '欢迎加入 HotSpot！')
    const redirect = (route.query.redirect as string) || '/'
    await navigateTo(redirect)
  }, {
    title: '注册失败',
    fallbackMessage: '请稍后重试',
    loading: isLoading
  })
}

const handleUpgrade = async (event: FormSubmitEvent<UpgradeSchema>) => {
  await withErrorHandling(async () => {
    if (!authStore.isAuthenticated) {
      return await switchTo('login')
    }
    const { loginId, password } = event.data
    const authData = await usersService.upgradeAccount({ login_id: loginId, password })
    setSessionFromAuth(authData)
    feedback.success('升级成功', '已设置登录凭据')
    const redirect = (route.query.redirect as string) || '/profile'
    await navigateTo(redirect)
  }, {
    title: '升级失败',
    fallbackMessage: '请稍后重试',
    loading: isLoading
  })
}

const createTempUser = async () => {
  await withErrorHandling(async () => {
    const authData = await authService.guest({ nickname: `访客_${Date.now().toString(36).slice(-4)}`, client_type: 'app' })
    setSessionFromAuth(authData)
    feedback.success('欢迎体验 HotSpot', '临时账户已创建，开始探索吧！')
    const redirect = (route.query.redirect as string) || '/'
    await navigateTo(redirect)
  }, {
    title: '创建失败',
    fallbackMessage: '请稍后重试',
    loading: isCreatingTemp
  })
}

const showForgotPassword = async () => {
  await modal.showResetPassword()
}

watch(
  () => authStore.isAuthenticated,
  (isAuth) => {
    if (isAuth && activeTab.value === 'login' && !route.query.add) {
      const redirect = (route.query.redirect as string) || '/'
      navigateTo(redirect)
    }
  },
  { immediate: true }
)

defineShortcuts({
  l: { usingInput: false, handler: () => switchTo('login') },
  r: { usingInput: false, handler: () => switchTo('register') }
})
</script>

