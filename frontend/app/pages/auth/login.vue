<template>
  <div class="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-neutral-50 to-neutral-100 dark:from-neutral-900 dark:to-neutral-950">
    <UPageCard class="w-full max-w-md">
      <UAuthForm
        :schema="loginSchema"
        :fields="loginFields"
        :providers="authProviders"
        title="欢迎回来"
        description="登录您的 HotSpot 账户"
        icon="i-heroicons-user-circle"
        :loading="isLoading"
        @submit="handleLogin"
      >
        <template #description>
          还没有账号？<ULink to="/auth/register" class="text-primary font-medium">立即注册</ULink>
        </template>
        
        <template #password-hint>
          <ULink @click="showForgotPassword" class="text-primary font-medium cursor-pointer" tabindex="-1">
            忘记密码？
          </ULink>
        </template>

        <template #validation v-if="loginError">
          <UAlert 
            color="error" 
            icon="i-heroicons-exclamation-triangle" 
            :title="loginError || undefined"
            :description="loginErrorDescription || undefined"
          />
        </template>

        <template #footer>
          <div class="text-center space-y-2">
            <p class="text-sm text-neutral-600 dark:text-neutral-400">
              或者
            </p>
            <UButton
              to="/"
              variant="ghost"
              size="sm"
              icon="i-heroicons-eye"
              class="text-primary"
            >
              先体验一下
            </UButton>
          </div>
        </template>
      </UAuthForm>
    </UPageCard>
  </div>
</template>

<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig('auth-login')

definePageMeta({ 
  layout: "auth",
  title: seoConfig.title,
  description: seoConfig.description
});

// 设置 SEO meta
useSeoMeta(buildSeoMeta('auth-login'));

const authStore = useAuthStore();
const toast = useToast();

// 表单状态
const isLoading = ref(false);
const loginError = ref<string | null>(null);
const loginErrorDescription = ref<string | null>(null);

// Zod 验证模式
const loginSchema = z.object({
  loginId: z.string().min(1, '请输入用户名'),
  password: z.string().min(1, '请输入密码'),
  remember: z.boolean().optional()
});

type LoginSchema = z.output<typeof loginSchema>;

// 表单字段定义
const loginFields = [
  {
    name: 'loginId',
    type: 'text' as const,
    label: '用户名',
    placeholder: '请输入用户名或邮箱',
    required: true,
    size: 'lg' as const
  },
  {
    name: 'password',
    type: 'password' as const,
    label: '密码',
    placeholder: '请输入密码',
    required: true,
    size: 'lg' as const
  },
  {
    name: 'remember',
    type: 'checkbox' as const,
    label: '记住我',
  }
];

// 第三方登录提供商（示例）
const authProviders = [
  {
    label: '微信登录',
    icon: 'i-simple-icons-wechat',
    color: 'success' as const,
    onClick: () => {
      toast.add({ 
        title: '微信登录', 
        description: '功能开发中，敬请期待',
        color: 'info'
      });
    }
  },
  {
    label: 'QQ登录',
    icon: 'i-simple-icons-tencentqq',
    color: 'primary' as const,
    onClick: () => {
      toast.add({ 
        title: 'QQ登录', 
        description: '功能开发中，敬请期待',
        color: 'info'
      });
    }
  }
];

// 处理登录
const handleLogin = async (event: FormSubmitEvent<LoginSchema>) => {
  try {
    isLoading.value = true;
    loginError.value = null;
    loginErrorDescription.value = null;

    const { loginId, password } = event.data;
    
    await authStore.login(loginId, password);
    
    toast.add({
      title: "登录成功",
      description: "欢迎回来！",
      color: "success"
    });
    
    await navigateTo("/");
  } catch (error: any) {
    console.error('登录错误:', error);
    
    loginError.value = "登录失败";
    loginErrorDescription.value = error?.message || "请检查用户名和密码后重试";
    
    toast.add({
      title: "登录失败",
      description: error?.message || "请检查用户名和密码后重试",
      color: "error"
    });
  } finally {
    isLoading.value = false;
  }
};

// 忘记密码处理
const showForgotPassword = () => {
  const { openConfirmModal } = useModals();
  openConfirmModal({
    title: "重置密码",
    message: "当前版本暂未开放密码找回服务，请联系管理员或使用临时用户体验。",
    confirmText: "我知道了",
    cancelText: "",
    confirmColor: "primary",
    onConfirm: () => {}
  });
};

// 如果已登录，重定向到首页
watch(
  () => authStore.isAuthenticated,
  (isAuth) => {
    if (isAuth) {
      navigateTo("/");
    }
  },
  { immediate: true }
);

// 快捷键
onMounted(() => {
  const shortcuts = useShortcuts();
  shortcuts.register('alt+r', () => navigateTo('/auth/register'), '切换到注册', '认证');
});
</script>
