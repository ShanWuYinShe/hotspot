<template>
  <div class="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-neutral-50 to-neutral-100 dark:from-neutral-900 dark:to-neutral-950">
    <UPageCard class="w-full max-w-md">
      <UAuthForm
        :schema="registerSchema"
        :fields="registerFields"
        :providers="authProviders"
        title="加入 HotSpot"
        description="创建您的账户，开始探索"
        icon="i-heroicons-user-plus"
        :loading="isLoading"
        @submit="handleRegister"
      >
        <template #description>
          已有账号？<ULink to="/auth/login" class="text-primary font-medium">立即登录</ULink>
        </template>

        <template #validation v-if="registerError">
          <UAlert 
            color="error" 
            icon="i-heroicons-exclamation-triangle" 
            :title="registerError || undefined"
            :description="registerErrorDescription || undefined"
          />
        </template>

        <template #footer>
          <div class="text-center space-y-2">
            <p class="text-xs text-neutral-500">
              注册即表示您同意我们的 
              <ULink to="#" class="text-primary font-medium">服务条款</ULink> 
              和 
              <ULink to="#" class="text-primary font-medium">隐私政策</ULink>
            </p>
            <div class="pt-2">
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
          </div>
        </template>
      </UAuthForm>
    </UPageCard>
  </div>
</template>

<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

definePageMeta({ 
  layout: "auth",
  title: "注册"
});

const authStore = useAuthStore();
const userService = useUserService();
const toast = useToast();
const { openConfirmModal } = useModals();

// 表单状态
const isLoading = ref(false);
const registerError = ref<string | null>(null);
const registerErrorDescription = ref<string | null>(null);

// Zod 验证模式
const registerSchema = z.object({
  loginId: z.string()
    .min(3, '用户名至少3个字符')
    .max(32, '用户名最多32个字符')
    .regex(/^[a-zA-Z0-9_]+$/, '用户名只能包含字母、数字和下划线'),
  nickname: z.string()
    .min(2, '昵称至少2个字符')
    .max(32, '昵称最多32个字符')
    .optional(),
  password: z.string()
    .min(6, '密码至少6个字符')
    .max(128, '密码最多128个字符'),
  confirmPassword: z.string(),
  terms: z.boolean().refine(val => val === true, '请同意服务条款')
}).refine((data) => data.password === data.confirmPassword, {
  message: "两次输入的密码不一致",
  path: ["confirmPassword"],
});

type RegisterSchema = z.output<typeof registerSchema>;

// 表单字段定义
const registerFields = [
  {
    name: 'loginId',
    type: 'text' as const,
    label: '用户名',
    placeholder: '请输入用户名',
    required: true,
    size: 'lg' as const,
    help: '3-32个字符，支持字母、数字和下划线'
  },
  {
    name: 'nickname',
    type: 'text' as const,
    label: '昵称',
    placeholder: '请输入昵称（可选）',
    size: 'lg' as const,
    help: '用于显示的友好名称'
  },
  {
    name: 'password',
    type: 'password' as const,
    label: '密码',
    placeholder: '请设置密码',
    required: true,
    size: 'lg' as const
  },
  {
    name: 'confirmPassword',
    type: 'password' as const,
    label: '确认密码',
    placeholder: '请再次输入密码',
    required: true,
    size: 'lg' as const
  },
  {
    name: 'terms',
    type: 'checkbox' as const,
    label: '我同意服务条款和隐私政策',
    required: true
  }
];

// 第三方登录提供商（示例）
const authProviders = [
  {
    label: '微信注册',
    icon: 'i-simple-icons-wechat',
    color: 'success' as const,
    onClick: () => {
      toast.add({ 
        title: '微信注册', 
        description: '功能开发中，敬请期待',
        color: 'info'
      });
    }
  },
  {
    label: 'QQ注册',
    icon: 'i-simple-icons-tencentqq',
    color: 'primary' as const,
    onClick: () => {
      toast.add({ 
        title: 'QQ注册', 
        description: '功能开发中，敬请期待',
        color: 'info'
      });
    }
  }
];

// 生成默认昵称
const generateNickname = () => `用户${Math.floor(Date.now() % 100000)}`;

// 询问是否立即登录
const askLoginNow = (): Promise<boolean> => {
  return new Promise((resolve) => {
    openConfirmModal({
      title: "注册成功",
      message: "现在登录以开始使用您的账户。",
      confirmText: "立即登录",
      cancelText: "稍后登录",
      confirmColor: "primary",
      onConfirm: () => resolve(true),
      onCancel: () => resolve(false)
    });
  });
};

// 处理注册
const handleRegister = async (event: FormSubmitEvent<RegisterSchema>) => {
  try {
    isLoading.value = true;
    registerError.value = null;
    registerErrorDescription.value = null;

    const { loginId, nickname, password } = event.data;
    const finalNickname = nickname?.trim() || generateNickname();
    
    // 使用 userService 直接注册
    const authData = await userService.register(loginId, finalNickname, password);

    // 询问是否立即登录
    const wantLogin = await askLoginNow();
    
    if (wantLogin) {
      // 立即登录：设置状态到 store
      await authStore.setAuthData(authData);
      toast.add({
        title: "注册并登录成功",
        description: "欢迎加入 HotSpot！",
        color: "success"
      });
    } else {
      // 选择暂不登录
      toast.add({
        title: "注册成功",
        description: "您可以稍后在登录页登录",
        color: "success"
      });
    }
    
    await navigateTo("/");
  } catch (error: any) {
    console.error('注册错误:', error);
    
    registerError.value = "注册失败";
    registerErrorDescription.value = error?.message || "请稍后重试";
    
    toast.add({
      title: "注册失败",
      description: error?.message || "请稍后重试",
      color: "error"
    });
  } finally {
    isLoading.value = false;
  }
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
  shortcuts.register('alt+l', () => navigateTo('/auth/login'), '切换到登录', '认证');
});
</script>
