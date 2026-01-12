<template>
  <UPageCard
    variant="solid"
  >
    <div class="flex flex-col gap-6">
      <!-- Header Section -->
      <div class="flex flex-col gap-2 text-center mb-2">
        <h1 class="text-xl font-bold text-gray-900 dark:text-white">
          {{ activeTitle }}
        </h1>
        <p class="text-sm font-medium text-gray-600 dark:text-gray-400">
          {{ activeDescription }}
        </p>
      </div>

      <!-- Tabs -->
      <UTabs
        v-if="!isUpgrade"
        v-model="activeTab"
        :items="tabs"
        :content="false"
        size="sm"
        variant="pill"
        color="primary"
      />

      <!-- Form -->
      <UForm
        :schema="activeSchema"
        :state="formState"
        @submit="handleSubmit"
      >
        <div class="flex flex-col gap-5">
          <!-- Login Fields -->
          <template v-if="activeTab === 'login'">
            <UFormField
              label="用户名 / 邮箱"
              name="loginId"
              required
            >
              <UInput
                v-model="formState.loginId"
                placeholder="请输入用户名或邮箱"
                icon="i-lucide-user"
                size="sm"
                variant="outline"
              />
            </UFormField>

            <UFormField
              label="密码"
              name="password"
              required
            >
              <UInput
                v-model="formState.password"
                type="password"
                placeholder="请输入密码"
                icon="i-lucide-lock"
                size="sm"
                variant="outline"
              />
            </UFormField>

            <div class="flex items-center justify-between">
              <UCheckbox
                v-model="formState.remember"
                label="记住我"
                name="remember"
              />
              <UButton
                variant="link"
                size="xs"
                color="primary"
                @click="showForgotPassword"
              >
                忘记密码？
              </UButton>
            </div>
          </template>

          <!-- Register/Upgrade Fields -->
          <template v-else>
            <UFormField
              label="用户名"
              name="loginId"
              required
              hint="3-32个字符，支持字母、数字和下划线"
            >
              <UInput
                v-model="formState.loginId"
                placeholder="设置用户名"
                icon="i-lucide-at-sign"
                size="sm"
                variant="outline"
              />
            </UFormField>

            <UFormField
              v-if="!isUpgrade"
              label="昵称"
              name="nickname"
              hint="可选，不填将自动生成"
            >
              <UInput
                v-model="formState.nickname"
                placeholder="设置昵称"
                icon="i-lucide-user"
                size="sm"
                variant="outline"
              />
            </UFormField>

            <UFormField
              label="密码"
              name="password"
              required
              hint="至少6位"
            >
              <UInput
                v-model="formState.password"
                type="password"
                placeholder="设置密码"
                icon="i-lucide-lock"
                size="sm"
                variant="outline"
              />
            </UFormField>

            <UFormField
              label="确认密码"
              name="confirmPassword"
              required
            >
              <UInput
                v-model="formState.confirmPassword"
                type="password"
                placeholder="再次输入密码"
                icon="i-lucide-lock"
                size="sm"
                variant="outline"
              />
            </UFormField>

            <UFormField
              v-if="!isUpgrade"
              name="terms"
            >
              <UCheckbox
                v-model="formState.terms"
                required
              >
                <template #label>
                  我已阅读并同意
                  <UButton
                    variant="link"
                    size="xs"
                    color="primary"
                    to="/terms"
                    target="_blank"
                  >
                    服务条款
                  </UButton>
                </template>
              </UCheckbox>
            </UFormField>
          </template>

          <!-- Submit Button -->
          <UButton
            type="submit"
            size="md"
            block
            :loading="isLoading"
            :label="activeSubmitLabel"
          />
        </div>
      </UForm>

      <!-- Divider & Alternative Actions -->
      <template v-if="!isUpgrade">
        <USeparator 
          label="或者"
          color="neutral"
          size="xs"
          class="my-2"
        />

        <div class="flex flex-col gap-3">
          <UButton
            v-for="provider in providers"
            :key="provider.label"
            block
            variant="soft"
            color="neutral"
            size="md"
            :icon="provider.icon"
            :loading="provider.loading"
            @click="provider.onClick"
          >
            {{ provider.label }}
          </UButton>
        </div>
      </template>
    </div>
  </UPageCard>
</template>

<script setup lang="ts">
import * as z from "zod";
import type { FormSubmitEvent } from "@nuxt/ui";
import { useAuthService } from "~/services/auth.service";
import { useUsersService } from "~/services/users.service";
import { useFeedbackStore } from "~/stores/feedback.store";
import { useAuthStore } from "~/stores/auth.store";
import type { AuthTokenPairApp } from "~/types/api";

definePageMeta({
  layout: "auth",
  title: "认证",
});

const authStore = useAuthStore();
const feedback = useFeedbackStore();
const modal = useModal();
const route = useRoute();
const { withErrorHandling } = useErrorHandler();
const authService = useAuthService();
const usersService = useUsersService();

const isLoading = ref(false);
const isCreatingTemp = ref(false);

const isUpgrade = computed(() => route.query.upgrade === "1");

const tabs = [
  { label: "登录", value: "login", icon: "i-lucide-log-in" },
  { label: "注册", value: "register", icon: "i-lucide-user-plus" },
];

const activeTab = ref(
  route.query.mode === "register" ? "register" : "login",
);

// Sync tab with query
watch(
  () => route.query.mode,
  (mode) => {
    if (mode === "register" || mode === "login") {
      activeTab.value = mode;
    }
  },
);

// Handle upgrade mode
watch(
  isUpgrade,
  (v) => {
    if (v) activeTab.value = "register";
  },
  { immediate: true },
);

const formState = reactive({
  loginId: "",
  password: "",
  confirmPassword: "",
  nickname: "",
  remember: false,
  terms: false,
});

// Validation Schemas
const loginSchema = z.object({
  loginId: z.string().min(1, "请输入用户名"),
  password: z.string().min(1, "请输入密码"),
  remember: z.boolean().optional(),
});

const registerSchema = z
  .object({
    loginId: z
      .string()
      .min(3, "用户名至少3个字符")
      .max(32, "用户名最多32个字符")
      .regex(
        /^[a-zA-Z0-9_]+$/,
        "用户名只能包含字母、数字和下划线",
      ),
    nickname: z.string().max(32, "昵称最多32个字符").optional(),
    password: z
      .string()
      .min(6, "密码至少6个字符")
      .max(128, "密码最多128个字符"),
    confirmPassword: z.string(),
    terms: z.boolean().refine((val) => val === true, "请同意服务条款"),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "两次输入的密码不一致",
    path: ["confirmPassword"],
  });

const upgradeSchema = z
  .object({
    loginId: z
      .string()
      .min(3, "用户名至少3个字符")
      .max(32, "用户名最多32个字符")
      .regex(
        /^[a-zA-Z0-9_]+$/,
        "用户名只能包含字母、数字和下划线",
      ),
    password: z
      .string()
      .min(6, "密码至少6个字符")
      .max(128, "密码最多128个字符"),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "两次输入的密码不一致",
    path: ["confirmPassword"],
  });

const activeSchema = computed(() => {
  if (activeTab.value === "login") return loginSchema;
  return isUpgrade.value ? upgradeSchema : registerSchema;
});

const activeTitle = computed(() => {
  if (activeTab.value === "login") return "欢迎回来";
  return isUpgrade.value ? "升级账户" : "创建账户";
});

const activeDescription = computed(() => {
  if (activeTab.value === "login") return "登录您的账户以继续";
  return isUpgrade.value
    ? "为临时账户设置登录凭据"
    : "加入 HotSpot，开始探索";
});

const activeSubmitLabel = computed(() => {
  if (activeTab.value === "login") return "登录";
  return isUpgrade.value ? "升级" : "注册";
});

const providers = computed(() => [
  {
    label: "免注册快速体验",
    icon: "i-lucide-zap",
    loading: isCreatingTemp.value,
    onClick: createTempUser,
  },
]);

const generateNickname = () =>
  `用户${Math.floor(Date.now() % 100000)}`;

const setSessionFromAuth = (authData: AuthTokenPairApp) => {
  authStore.setAccountSession(
    { user_id: authData.user_id, nickname: authData.nickname },
    authData.access_token,
    authData.refresh_token ?? null,
    new Date(authData.access_token_expires_at).getTime(),
    new Date(authData.refresh_token_expires_at).getTime(),
  );
};

const handleSubmit = async (event: FormSubmitEvent<any>) => {
  if (activeTab.value === "login") {
    return handleLogin(event);
  }
  if (isUpgrade.value) {
    return handleUpgrade(event);
  }
  return handleRegister(event);
};

const handleLogin = async (event: FormSubmitEvent<any>) => {
  await withErrorHandling(
    async () => {
      const { loginId, password } = event.data;
      const authData = await authService.login<AuthTokenPairApp>({
        login_id: loginId,
        password,
        client_type: "app",
      });
      // 显式断言为 AppAuthData，因为我们传入了 client_type: 'app'
      const appAuthData = authData;
      setSessionFromAuth(appAuthData);
      feedback.success("登录成功", "欢迎回来！");
      const redirect = (route.query.redirect as string) || "/";
      await navigateTo(redirect);
    },
    {
      title: "登录失败",
      fallbackMessage: "请检查用户名和密码后重试",
      loading: isLoading,
    },
  );
};

const handleRegister = async (event: FormSubmitEvent<any>) => {
  await withErrorHandling(
    async () => {
      const { loginId, nickname, password } = event.data;
      const finalNickname = nickname?.trim() || generateNickname();
      const authData = await authService.register<AuthTokenPairApp>({
        login_id: loginId,
        nickname: finalNickname,
        password,
        client_type: "app",
      });
      const appAuthData = authData;
      setSessionFromAuth(appAuthData);
      feedback.success("注册成功", "欢迎加入 HotSpot！");
      const redirect = (route.query.redirect as string) || "/";
      await navigateTo(redirect);
    },
    {
      title: "注册失败",
      fallbackMessage: "请稍后重试",
      loading: isLoading,
    },
  );
};

const handleUpgrade = async (event: FormSubmitEvent<any>) => {
  await withErrorHandling(
    async () => {
      if (!authStore.isAuthenticated) {
        activeTab.value = "login";
        return;
      }
      const { loginId, password } = event.data;
      const authData = await usersService.upgradeAccount({
        login_id: loginId,
        password,
      });
      // Note: upgrade usually doesn't return auth data, but we might want to refresh profile
      // Assuming backend might require re-login or just success
      feedback.success("升级成功", "已设置登录凭据");
      const redirect = (route.query.redirect as string) || "/profile";
      await navigateTo(redirect);
    },
    {
      title: "升级失败",
      fallbackMessage: "请稍后重试",
      loading: isLoading,
    },
  );
};

const createTempUser = async () => {
  await withErrorHandling(
    async () => {
      const authData = await authService.guest<AuthTokenPairApp>({
        nickname: `访客_${Date.now().toString(36).slice(-4)}`,
        client_type: "app",
      });
      const appAuthData = authData;
      setSessionFromAuth(appAuthData);
      feedback.success(
        "欢迎体验 HotSpot",
        "临时账户已创建，开始探索吧！",
      );
      const redirect = (route.query.redirect as string) || "/";
      await navigateTo(redirect);
    },
    {
      title: "创建失败",
      fallbackMessage: "请稍后重试",
      loading: isCreatingTemp,
    },
  );
};

const showForgotPassword = async () => {
  await modal.showResetPassword();
};

// Redirect if already logged in and not upgrading/adding
watch(
  () => authStore.isAuthenticated,
  (isAuth) => {
    if (
      isAuth &&
      activeTab.value === "login" &&
      !route.query.add &&
      !isUpgrade.value
    ) {
      const redirect = (route.query.redirect as string) || "/";
      navigateTo(redirect);
    }
  },
  { immediate: true },
);

watch(activeTab, (val) => {
  // Clear form on tab switch
  formState.loginId = "";
  formState.password = "";
  formState.confirmPassword = "";
  formState.nickname = "";

  // Update query param
  if (!isUpgrade.value) {
    navigateTo({
      path: "/auth",
      query: { ...route.query, mode: val },
    });
  }
});
</script>
