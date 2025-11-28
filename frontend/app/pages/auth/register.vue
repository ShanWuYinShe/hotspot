<template>
  <div class="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-neutral-50 to-neutral-100 dark:from-neutral-900 dark:to-neutral-950">
    <div class="w-full max-w-4xl">
      <!-- 注册步骤指示器 -->
      <div class="mb-8">
        <UStepper 
          :items="stepperItems" 
          v-model="currentStep"
          :disabled="isLoading"
          class="w-full"
        />
      </div>

      <!-- 注册表单内容 -->
      <UPageCard class="w-full">
        <UAuthForm
          v-if="currentStep === 0"
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
              <p class="text-xs text-neutral-500 dark:text-neutral-400">
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

        <!-- 个人信息完善步骤 -->
        <div v-else-if="currentStep === 1" class="p-6">
          <div class="text-center mb-6">
            <UIcon name="i-heroicons-user-circle" class="w-16 h-16 text-neutral-600 dark:text-neutral-300 mx-auto mb-4" />
            <h2 class="text-2xl font-bold text-neutral-900 dark:text-white mb-2">完善个人信息</h2>
            <p class="text-neutral-600 dark:text-neutral-400">让我们更好地了解您</p>
          </div>
          
          <UForm :schema="profileSchema" :state="profileState" @submit="handleProfileComplete">
            <UFormField label="昵称" name="nickname" required>
              <UInput v-model="profileState.nickname" placeholder="请输入您的昵称" />
            </UFormField>
            
            <UFormField label="个人简介" name="bio">
              <UTextarea v-model="profileState.bio" placeholder="介绍一下自己..." :rows="3" />
            </UFormField>
            
            <UFormField label="兴趣爱好" name="interests">
              <UInputTags v-model="profileState.interests" placeholder="添加兴趣爱好标签" />
            </UFormField>
            
            <div class="flex gap-3 mt-6">
              <UButton 
                variant="outline" 
                @click="currentStep = 0"
                :disabled="isLoading"
                class="flex-1"
              >
                上一步
              </UButton>
              <UButton 
                type="submit" 
                :loading="isLoading"
                class="flex-1"
              >
                完成注册
              </UButton>
            </div>
          </UForm>
        </div>

        <!-- 注册成功步骤 -->
        <div v-else-if="currentStep === 2" class="p-6 text-center">
          <UIcon name="i-heroicons-check-circle" class="w-20 h-20 text-success-500 mx-auto mb-6" />
          <h2 class="text-2xl font-bold text-neutral-900 dark:text-white mb-4">注册成功！</h2>
          <p class="text-neutral-600 dark:text-neutral-400 mb-8">欢迎加入 HotSpot，开始您的社交之旅</p>
          
          <div class="space-y-3">
            <UButton 
              to="/" 
              color="primary" 
              size="lg" 
              class="w-full"
            >
              开始探索
            </UButton>
            <UButton 
              to="/settings" 
              variant="outline" 
              size="lg" 
              class="w-full"
            >
              完善资料
            </UButton>
          </div>
        </div>
      </UPageCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useUsersService } from '~/services/users.service';
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'
import type { StepperItem } from '@nuxt/ui'

definePageMeta({ 
  layout: "auth",
  title: "注册"
});

const authStore = useAuthStore();
const userService = useUsersService();
const toast = useToast();
const modal = useModal();

// 表单状态
const isLoading = ref(false);
const registerError = ref<string | null>(null);
const registerErrorDescription = ref<string | null>(null);
const currentStep = ref(0);

// 步骤指示器配置
const stepperItems: StepperItem[] = [
  {
    title: '账户信息',
    description: '创建您的账户',
    icon: 'i-heroicons-user-plus',
    value: 0
  },
  {
    title: '个人信息',
    description: '完善个人资料',
    icon: 'i-heroicons-user-circle',
    value: 1
  },
  {
    title: '完成注册',
    description: '开始使用',
    icon: 'i-heroicons-check-circle',
    value: 2
  }
];

// 个人信息表单状态
const profileState = reactive({
  nickname: '',
  bio: '',
  interests: [] as string[]
});

// 个人信息验证模式
const profileSchema = z.object({
  nickname: z.string().min(1, '请输入昵称'),
  bio: z.string().optional(),
  interests: z.array(z.string()).optional()
});

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
    modal.showConfirmModal({
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
    
    // 计算密码哈希 SHA256(password)
    const { hashPassword } = await import('~/utils/password')
    const password_hash = await hashPassword(password)
    
    // 使用 authStore 注册用户
    await authStore.register({
      login_id: loginId,
      nickname: finalNickname,
      password_hash
    });

    // 设置个人信息到状态
    profileState.nickname = finalNickname;
    
    // 进入下一步
    currentStep.value = 1;
    
    toast.add({
      title: "账户创建成功",
      description: "请完善您的个人信息",
      color: "success"
    });
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

// 处理个人信息完善
const handleProfileComplete = async (event: FormSubmitEvent<z.infer<typeof profileSchema>>) => {
  try {
    isLoading.value = true;
    
    // 更新用户资料
    const userService = useUsersService();
    await userService.updateProfile(profileState);
    
    // 进入完成步骤
    currentStep.value = 2;
    
    toast.add({
      title: "注册完成",
      description: "欢迎加入 HotSpot！",
      color: "success"
    });
  } catch (error: any) {
    console.error('完善信息错误:', error);
    
    toast.add({
      title: "保存失败",
      description: error?.message || "请重试",
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

// 快捷键管理 - 使用 Nuxt UI v4 原生支持
defineShortcuts({
  'l': {
    usingInput: false,
    handler: () => navigateTo('/auth/login'),
  },
});
</script>
