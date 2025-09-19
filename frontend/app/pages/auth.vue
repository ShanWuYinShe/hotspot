<template>
    <!-- 专用于认证页的简洁布局（不包含全局导航） -->
    <div
        class="min-h-screen antialiased bg-gradient-to-br from-neutral-50 to-neutral-100 dark:from-neutral-900 dark:to-neutral-950 bg-radial-hero flex items-center py-8">
        <div class="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
            <UCard class="mx-auto max-w-4xl">
                <div class="grid grid-cols-1 lg:grid-cols-12">
                    <!-- 左：品牌与亮点（桌面显示） -->
                    <div class="hidden lg:flex flex-col justify-between p-8 lg:col-span-5">
                        <div>
                            <div class="flex items-center gap-4 mb-8">
                                <div
                                    class="w-14 h-14 rounded-2xl bg-primary-500 flex items-center justify-center shadow-lg">
                                    <UIcon name="i-heroicons-fire" class="w-8 h-8 text-white" />
                                </div>
                                <div>
                                    <h1 class="text-3xl font-bold text-neutral-900 dark:text-white">
                                        HotSpot
                                    </h1>
                                    <p class="text-lg text-neutral-700 dark:text-neutral-300">
                                        基于位置的社交平台
                                    </p>
                                </div>
                            </div>

                            <div class="space-y-6">
                                <div class="flex items-start gap-4">
                                    <div
                                        class="w-12 h-12 rounded-xl bg-primary-100 dark:bg-primary-900/40 flex items-center justify-center shadow-sm flex-shrink-0">
                                        <UIcon name="i-heroicons-map-pin"
                                            class="w-6 h-6 text-primary-600 dark:text-primary-300" />
                                    </div>
                                    <div>
                                        <div class="font-semibold text-lg text-neutral-900 dark:text-neutral-100">
                                            基于位置的发现
                                        </div>
                                        <div class="text-neutral-700 dark:text-neutral-300 mt-1">
                                            发现附近用户与群组，快速开始交流
                                        </div>
                                    </div>
                                </div>

                                <div class="flex items-start gap-4">
                                    <div
                                        class="w-12 h-12 rounded-xl bg-success-100 dark:bg-success-900/40 flex items-center justify-center shadow-sm flex-shrink-0">
                                        <UIcon name="i-heroicons-chat-bubble-left-right"
                                            class="w-6 h-6 text-success-600 dark:text-success-300" />
                                    </div>
                                    <div>
                                        <div class="font-semibold text-lg text-neutral-900 dark:text-neutral-100">
                                            实时消息
                                        </div>
                                        <div class="text-neutral-700 dark:text-neutral-300 mt-1">
                                            与朋友和群组实时沟通，支持多种消息类型
                                        </div>
                                    </div>
                                </div>

                                <div class="flex items-start gap-4">
                                    <div
                                        class="w-12 h-12 rounded-xl bg-info-100 dark:bg-info-900/40 flex items-center justify-center shadow-sm flex-shrink-0">
                                        <UIcon name="i-heroicons-photo"
                                            class="w-6 h-6 text-info-600 dark:text-info-300" />
                                    </div>
                                    <div>
                                        <div class="font-semibold text-lg text-neutral-900 dark:text-neutral-100">
                                            动态与互动
                                        </div>
                                        <div class="text-neutral-700 dark:text-neutral-300 mt-1">
                                            发布动态，点赞互动，记录精彩
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>


                    </div>

                    <!-- 右：登录/注册表单 -->
                    <div class="p-8 sm:p-10 lg:col-span-7">
                        <!-- 顶部移动端品牌 -->
                        <div class="lg:hidden text-center mb-8">
                            <div class="flex items-center justify-center gap-3 mb-4">
                                <div
                                    class="w-12 h-12 rounded-xl bg-primary-500 flex items-center justify-center shadow-lg">
                                    <UIcon name="i-heroicons-fire" class="w-7 h-7 text-white" />
                                </div>
                                <h1 class="text-3xl font-bold text-neutral-900 dark:text-white">
                                    HotSpot
                                </h1>
                            </div>
                            <p class="text-lg text-neutral-600 dark:text-neutral-300">
                                基于位置的社交平台
                            </p>
                        </div>

                        <!-- 使用 UTabs 切换 登录 / 注册 -->
                        <UTabs :items="authTabs" v-model="mode" :default-index="0" class="mb-8" />

                        <!-- 登录表单 -->
                        <div v-if="mode === 'login'" class="space-y-6">
                            <!-- 正式账号登录表单 -->
                            <UForm :state="loginForm" :validate="validateLogin" @submit="handleLogin" class="space-y-5"
                                aria-label="登录表单">
                                <UFormField label="用户名" name="loginId" required class="space-y-2">
                                    <UTooltip text="输入您注册时使用的用户名或邮箱">
                                        <UInput v-model="loginForm.loginId" placeholder="请输入用户名" :disabled="isLoading"
                                            size="lg" class="h-12 w-full" />
                                    </UTooltip>
                                </UFormField>

                                <UFormField label="密码" name="password" required class="space-y-2">
                                    <UTooltip text="输入您的账户密码">
                                        <UInput v-model="loginForm.password" type="password" placeholder="请输入密码"
                                            :disabled="isLoading" autocomplete="new-password" size="lg"
                                            class="h-12 w-full" />
                                    </UTooltip>
                                </UFormField>



                                <div class="flex items-center justify-end text-sm">
                                    <UButton variant="ghost" size="sm" class="px-2" @click="showForgotPassword">
                                        <span>忘记密码？</span>
                                    </UButton>
                                </div>

                                <UTooltip text="使用用户名和密码登录" :kbds="['enter']">
                                    <UButton type="submit" color="primary" size="lg" block class="h-12 mt-6"
                                        :loading="isLoading">
                                        <span class="font-semibold">登录</span>
                                    </UButton>
                                </UTooltip>
                            </UForm>



                            <div class="text-center">
                                <UTooltip text="切换到注册页面" :kbds="['alt', 'r']">
                                    <UButton type="button" variant="ghost" size="md" class="py-2" @click="mode = 'register'"
                                        :disabled="isLoading">
                                        <span class="text-sm">还没有账号？点击注册</span>
                                    </UButton>
                                </UTooltip>
                            </div>
                        </div>

                        <!-- 升级表单 -->
                        <div v-else-if="mode === 'upgrade'" class="space-y-6">
                            <div class="text-center mb-6">
                                <h3 class="text-xl font-semibold text-neutral-900 dark:text-white mb-2">
                                    升级为正式账户
                                </h3>
                                <p class="text-neutral-600 dark:text-neutral-300">
                                    将您的临时账户升级为正式账户，保存所有数据并解锁全部功能
                                </p>
                            </div>

                            <UForm :state="upgradeForm" @submit="handleUpgrade" class="space-y-5"
                                aria-label="升级账户表单">
                                <UFormField label="登录ID" name="loginId" required class="space-y-2">
                                    <UInput v-model="upgradeForm.loginId" placeholder="请输入邮箱或手机号" :disabled="isLoading"
                                        size="lg" class="h-12 w-full" />
                                </UFormField>

                                <UFormField label="密码" name="password" required class="space-y-2">
                                    <UInput v-model="upgradeForm.password" type="password" placeholder="请设置密码"
                                        :disabled="isLoading" autocomplete="new-password" size="lg"
                                        class="h-12 w-full" />
                                </UFormField>

                                <UTooltip text="升级为正式账户" :kbds="['enter']">
                                    <UButton type="submit" color="primary" size="lg" block class="h-12 mt-6"
                                        :loading="isLoading">
                                        <UIcon name="i-heroicons-arrow-up-circle" class="w-5 h-5 mr-2" />
                                        <span class="font-semibold">升级账户</span>
                                    </UButton>
                                </UTooltip>
                            </UForm>

                            <div class="text-center">
                                <UButton type="button" variant="ghost" size="md" class="py-2" @click="mode = 'login'"
                                    :disabled="isLoading">
                                    <span class="text-sm">返回登录</span>
                                </UButton>
                            </div>
                        </div>

                        <!-- 注册表单 -->
                        <div v-else class="space-y-6">
                            <!-- 正式账号注册表单 -->
                            <UForm :state="registerForm" :validate="validateRegister" @submit="handleRegister"
                                class="space-y-5" aria-label="注册表单">
                                <UFormField label="用户名" name="loginId" required class="space-y-2">
                                    <UTooltip text="3-32个字符，支持字母、数字和下划线">
                                        <UInput v-model="registerForm.loginId" placeholder="请输入用户名" :disabled="isLoading"
                                            size="lg" class="h-12 w-full" />
                                    </UTooltip>
                                </UFormField>

                                <UFormField label="昵称" name="nickname" class="space-y-2">
                                    <UTooltip text="2-32个字符，用于显示的友好名称">
                                        <UInput v-model="registerForm.nickname" placeholder="请输入昵称" :disabled="isLoading"
                                            size="lg" class="h-12 w-full" />
                                    </UTooltip>
                                </UFormField>

                                <UFormField label="密码" name="password" required class="space-y-2">
                                    <UTooltip text="请设置一个安全的密码">
                                        <UInput v-model="registerForm.password" type="password" placeholder="请输入密码"
                                            :disabled="isLoading" autocomplete="new-password" size="lg"
                                            class="h-12 w-full" />
                                    </UTooltip>
                                </UFormField>



                                <UTooltip text="创建新账户" :kbds="['enter']">
                                    <UButton type="submit" color="primary" size="lg" block class="h-12 mt-6"
                                        :loading="isLoading">
                                        <span class="font-semibold">注册</span>
                                    </UButton>
                                </UTooltip>
                            </UForm>



                            <div class="text-center">
                                <UTooltip text="切换到登录页面" :kbds="['alt', 'l']">
                                    <UButton type="button" variant="ghost" size="md" class="py-2" @click="mode = 'login'"
                                        :disabled="isLoading">
                                        <span class="text-sm">已有账号？点击登录</span>
                                    </UButton>
                                </UTooltip>
                            </div>
                        </div>
                    </div>
                </div>
            </UCard>

        </div>
    </div>
</template>

<script setup lang="ts">

definePageMeta({ layout: "auth" });

const authStore = useAuthStore();
const userService = useUserService();
const isLoading = ref(false);
const toast = useSmartToast();
const route = useRoute();
const { openConfirmModal } = useModals();

// 表单模式 - 检查是否是升级模式
const isUpgradeMode = computed(() => route.query.upgrade === '1');
const mode = ref<"login" | "register" | "upgrade">(
  isUpgradeMode.value ? "upgrade" : "login"
);

// UTabs items（键值需与 mode 匹配）
const authTabs = computed(() => {
  const tabs = [
    { value: 'login', label: '登录', icon: 'i-heroicons-arrow-right-on-rectangle' },
    { value: 'register', label: '注册', icon: 'i-heroicons-user-plus' }
  ];
  
  if (isUpgradeMode.value) {
    tabs.unshift({ value: 'upgrade', label: '升级账户', icon: 'i-heroicons-arrow-up-circle' });
  }
  
  return tabs;
});


// defineShortcuts：快速切换登录/注册（统一在 onMounted 中初始化，避免 SSR/CSR 分叉）
// 移除了可能与系统快捷键冲突的快捷键定义

// 登录表单
const loginForm = ref({
    loginId: "",
    password: "",
});

// 注册表单
const registerForm = ref({
    loginId: "",
    nickname: "",
    password: "",
});

// 升级表单
const upgradeForm = ref({
    loginId: "",
    password: "",
});

// 临时体验入口已迁移至首页 CTA

// 前端生成昵称（未提供时）
const genNickname = () => `用户${Math.floor(Date.now() % 100000)}`;

// 显示忘记密码说明
const showForgotPassword = () => {
    openConfirmModal({
        title: "重置密码",
        message: "当前版本暂未开放密码找回服务，请联系管理员或使用临时用户体验。",
        confirmText: "我知道了",
        cancelText: "",
        confirmColor: "primary",
        onConfirm: () => {}
    });
};

// 如果已经登录，重定向到首页

// 使用 useModals 询问是否立即登录
function askLoginNow(): Promise<boolean> {
    return new Promise((resolve) => {
        openConfirmModal({
            title: "注册成功",
            message: "现在登录以开始使用你的账户。",
            confirmText: "登录",
            cancelText: "稍后",
            confirmColor: "primary",
            onConfirm: () => resolve(true),
            onCancel: () => resolve(false)
        });
    });
}

// 处理登录
const handleLogin = async () => {
    try {
        isLoading.value = true;
        await authStore.login(
            loginForm.value.loginId,
            loginForm.value.password,
        );
        toast.success("登录成功");
        await navigateTo("/");
    } catch (error: any) {
        toast.error({
            title: "登录失败",
            description: error?.message || "请检查账号与密码后重试"
        });
    } finally {
        isLoading.value = false;
    }
};

// 处理注册
const handleRegister = async () => {
    try {
        const nickname =
            (registerForm.value.nickname || "").trim() || genNickname();
        isLoading.value = true;
        
        // 使用userService直接注册，不自动保存认证状态
        const authData = await userService.register(
            registerForm.value.loginId,
            nickname,
            registerForm.value.password,
        );

        // 询问是否立即登录（Nuxt UI 模态）
        const wantLogin = await askLoginNow();
        if (wantLogin) {
            // 立即登录：设置状态到 store
            await authStore.setAuthData(authData);
            toast.add({
                title: "注册并已登录",
                description: "已保存本机加密凭证",
                color: "success"
            });
        } else {
            // 选择暂不登录：仅提示注册成功，不设置认证状态
            toast.add({
                title: "注册成功",
                description: "你可以稍后在登录页登录",
                color: "success"
            });
        }
        await navigateTo("/");
    } catch (error: any) {
        toast.error({
            title: "注册失败",
            description: error?.message || "请稍后重试"
        });
    } finally {
        isLoading.value = false;
    }
};

// 处理账户升级
const handleUpgrade = async () => {
    try {
        if (!upgradeForm.value.loginId.trim()) {
            toast.warning("请输入登录ID");
            return;
        }
        if (!upgradeForm.value.password.trim()) {
            toast.warning("请输入密码");
            return;
        }
        
        isLoading.value = true;
        await authStore.upgradeAccount(
            upgradeForm.value.loginId,
            upgradeForm.value.password
        );
        
        toast.add({
            title: "账户升级成功",
            description: "您现在是正式用户了！",
            color: "success"
        });
        upgradeForm.value.loginId = "";
        upgradeForm.value.password = "";
        await navigateTo("/");
    } catch (error: any) {
        toast.error({
            title: "升级失败",
            description: error?.message || "请检查登录信息后重试"
        });
    } finally {
        isLoading.value = false;
    }
};

// 临时体验已从认证页移除

// 登录后重定向
watch(
    () => authStore.isAuthenticated,
    (isAuth) => {
        if (isAuth) {
            navigateTo("/");
        }
    },
    { immediate: true },
);

// 使用通用表单验证
const { validateRequired, createFormValidator } = useFormValidation();

// 创建验证器
const validateLogin = createFormValidator({
    loginId: [(value) => validateRequired(value, "用户名")],
    password: [(value) => validateRequired(value, "密码")]
});

const validateRegister = createFormValidator({
    loginId: [(value) => validateRequired(value, "用户名")],
    password: [(value) => validateRequired(value, "密码")],
    nickname: [(value) => validateRequired(value, "昵称")]
});

</script>
