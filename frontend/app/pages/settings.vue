<template>
    <div class="min-h-screen bg-neutral-50 dark:bg-neutral-900">
        <!-- 移动端头部 -->
        <div class="lg:hidden sticky top-0 z-10">
            <UCard class="rounded-none border-x-0 border-t-0">
                <div class="flex items-center justify-between">
                    <h1 class="text-xl font-semibold">设置</h1>
                    <!-- 保存状态指示器 -->
                    <div v-if="isSaving" class="flex items-center gap-2 text-sm text-neutral-600">
                        <UIcon name="i-heroicons-arrow-path" class="w-4 h-4 animate-spin" />
                        <span class="hidden sm:inline">保存中...</span>
                    </div>
                    <div v-else-if="lastSaved" class="flex items-center gap-2 text-sm text-green-600">
                        <UIcon name="i-heroicons-check-circle" class="w-4 h-4" />
                        <span class="hidden sm:inline">已保存</span>
                    </div>
                </div>
                
                <!-- 移动端搜索 -->
                <div class="mt-3">
                    <UInput
                        v-model="searchQuery"
                        placeholder="搜索设置..."
                        icon="i-heroicons-magnifying-glass"
                        size="sm"
                        class="w-full"
                    />
                </div>
            </UCard>
        </div>
            
            <!-- 移动端标签导航 -->
            <div class="px-4 pb-3">
                <div class="flex overflow-x-auto gap-2 scrollbar-hide overscroll-x-contain">
                    <UButton
                        v-for="section in filteredSections"
                        :key="section.id"
                        @click="activeSection = section.id"
                        :variant="activeSection === section.id ? 'soft' : 'ghost'"
                        :color="activeSection === section.id ? 'primary' : 'neutral'"
                        size="sm"
                        class="whitespace-nowrap touch-manipulation min-h-[44px] flex-shrink-0"
                    >
                        <template #leading>
                            <UIcon :name="section.icon" class="w-4 h-4" />
                        </template>
                        {{ section.title }}
                        <template #trailing v-if="section.badge">
                            <UBadge
                                :label="section.badge"
                                color="error"
                                size="xs"
                            />
                        </template>
                    </UButton>
                </div>
            </div>

        <!-- 桌面端布局 -->
        <div class="hidden lg:block mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
            <div class="max-w-6xl mx-auto">
                <!-- 桌面端页面头部 -->
                <div class="mb-8">
                    <div class="flex items-center justify-between">
                        <div>
                            <h1 class="text-3xl font-bold text-neutral-900 dark:text-white">设置</h1>
                            <p class="mt-2 text-neutral-600 dark:text-neutral-400">管理您的账户和应用偏好设置</p>
                        </div>
                        <div class="flex items-center gap-3">
                            <!-- 搜索设置 -->
                            <UInput
                                v-model="searchQuery"
                                placeholder="搜索设置..."
                                icon="i-heroicons-magnifying-glass"
                                size="sm"
                                class="w-64"
                            />
                            <!-- 保存状态指示器 -->
                            <div v-if="isSaving" class="flex items-center gap-2 text-sm text-neutral-600">
                                <UIcon name="i-heroicons-arrow-path" class="w-4 h-4 animate-spin" />
                                <span>保存中...</span>
                            </div>
                            <div v-else-if="lastSaved" class="flex items-center gap-2 text-sm text-green-600">
                                <UIcon name="i-heroicons-check-circle" class="w-4 h-4" />
                                <span>已保存</span>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 桌面端设置内容 -->
                <div class="grid grid-cols-1 lg:grid-cols-4 gap-8">
                    <!-- 左侧导航 -->
                    <div class="lg:col-span-1">
                        <nav class="space-y-1 sticky top-6">
                            <UButton
                                v-for="section in filteredSections"
                                :key="section.id"
                                @click="activeSection = section.id"
                                :variant="activeSection === section.id ? 'soft' : 'ghost'"
                                :color="activeSection === section.id ? 'primary' : 'neutral'"
                                class="w-full justify-start gap-3 px-4 py-3"
                            >
                                <UIcon :name="section.icon" class="w-5 h-5 flex-shrink-0" />
                                <span class="font-medium">{{ section.title }}</span>
                                <UBadge
                                    v-if="section.badge"
                                    :label="section.badge"
                                    color="error"
                                    variant="soft"
                                    size="xs"
                                    class="ml-auto"
                                />
                            </UButton>
                        </nav>
                    </div>

                    <!-- 桌面端设置面板 -->
                    <div class="lg:col-span-3">
                        <SettingsContent 
                            :active-section="activeSection"
                            :local-settings="localSettings"
                            :user="user"
                            @save-nickname="onSaveNickname"
                            @avatar-uploaded="onAvatarUploaded"
                            @upgrade-account="upgradeAccount"
                            @change-password="changePassword"
                            @setup-2fa="setup2FA"
                            @clear-cache="clearCache"
                            @show-help="showHelp"
                            @show-privacy-policy="showPrivacyPolicy"
                            @show-terms="showTerms"
                            @delete-account="deleteAccount"
                            @edit-nickname="editNickname"
                            @change-avatar="changeAvatar"
                        />
                    </div>
                </div>
            </div>
        </div>

        <!-- 移动端内容 -->
        <div class="lg:hidden">
            <div class="px-4 py-4">
                <SettingsContent 
                    :active-section="activeSection"
                    :local-settings="localSettings"
                    :user="user"
                    :is-mobile="true"
                    @save-nickname="onSaveNickname"
                    @avatar-uploaded="onAvatarUploaded"
                    @upgrade-account="upgradeAccount"
                    @change-password="changePassword"
                    @setup-2fa="setup2FA"
                    @clear-cache="clearCache"
                    @show-help="showHelp"
                    @show-privacy-policy="showPrivacyPolicy"
                    @show-terms="showTerms"
                    @delete-account="deleteAccount"
                    @edit-nickname="editNickname"
                    @change-avatar="changeAvatar"
                />
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import type { UserSettings } from '~/types';
import { useDebounceFn } from "@vueuse/core";
import { useUserService } from '~/services/user.service';

// 页面元数据
definePageMeta({
    title: "设置",
    requiresAuth: true,
});

const authStore = useAuthStore();
const toast = useSmartToast();
const settingsStore = useSettingsStore();
const { openNicknameEditModal, openAvatarUploadModal, openConfirmModal } = useModals();

// 当前用户信息
const user = computed(() => authStore.user);

// 搜索功能
const searchQuery = ref('');

// 活动的设置分区
const activeSection = ref('account');

// 保存状态
const isSaving = ref(false);
const lastSaved = ref<Date | null>(null);

// 本地设置状态
const localSettings = ref<UserSettings>({
    notifications: {
        push_enabled: true,
        email_enabled: true,
        sms_enabled: false,
        group_notifications: true,
        message_notifications: true,
        friend_requests: true,
        direct_message_toast: true,
        group_message_toast: true,
        system_notification_toast: true,
        error_notifications: true,
        success_notifications: true,
    },
    privacy: {
        profile_visibility: "public",
        location_sharing: true,
        online_status: true,
        show_in_search: true,
    },
    preferences: {
        language: "zh-CN",
        theme: "auto",
        distance_unit: "km",
        time_format: "24h",
    },
    account: {
        nickname: user.value?.nickname || "",
        bio: "",
        email: user.value?.email || "",
        phone: "",
        avatar_url: user.value?.avatar_url || "",
    },
});

// 设置分区配置
const sections = [
    {
        id: 'account',
        title: '账户',
        icon: 'i-heroicons-user-circle',
        keywords: ['账户', '个人资料', '昵称', '头像', '安全', '密码']
    },
    {
        id: 'privacy',
        title: '隐私',
        icon: 'i-heroicons-shield-check',
        keywords: ['隐私', '位置', '在线状态', '可见性', '搜索']
    },
    {
        id: 'notifications',
        title: '通知',
        icon: 'i-heroicons-bell',
        keywords: ['通知', '消息', '推送', '提醒', '群组']
    },
    {
        id: 'appearance',
        title: '外观',
        icon: 'i-heroicons-paint-brush',
        keywords: ['外观', '主题', '语言', '界面', '显示']
    },
    {
        id: 'about',
        title: '关于',
        icon: 'i-heroicons-information-circle',
        keywords: ['关于', '帮助', '支持', '版本', '缓存']
    },
    {
        id: 'danger',
        title: '危险操作',
        icon: 'i-heroicons-exclamation-triangle',
        keywords: ['删除', '账户', '危险'],
        badge: '!'
    }
];

// 过滤的设置分区
const filteredSections = computed(() => {
    if (!searchQuery.value) return sections;
    
    const query = searchQuery.value.toLowerCase();
    return sections.filter(section => 
        section.title.toLowerCase().includes(query) ||
        section.keywords.some(keyword => keyword.toLowerCase().includes(query))
    );
});

// 选项数据
const themeOptions = [
    { 
        label: "跟随系统", 
        value: "auto",
        icon: "i-heroicons-computer-desktop",
        description: "根据系统设置自动切换"
    },
    { 
        label: "浅色模式", 
        value: "light",
        icon: "i-heroicons-sun",
        description: "始终使用浅色主题"
    },
    { 
        label: "深色模式", 
        value: "dark",
        icon: "i-heroicons-moon",
        description: "始终使用深色主题"
    },
];

const languageOptions = [
    { label: "简体中文", value: "zh-CN" },
    { label: "English", value: "en-US" },
];

const profileVisibilityOptions = [
    { label: "公开", value: "public" },
    { label: "仅好友", value: "friends" },
    { label: "私密", value: "private" },
];

const distanceUnitOptions = [
    { label: "公里 (km)", value: "km" },
    { label: "英里 (miles)", value: "miles" },
];

const timeFormatOptions = [
    { label: "24小时制", value: "24h" },
    { label: "12小时制", value: "12h" },
];

// 初始化设置
onMounted(async () => {
    try {
        await settingsStore.fetchSettings();
        if (settingsStore.settings) {
            localSettings.value = { ...settingsStore.settings } as unknown as UserSettings;
        }
    } catch (error) {
        console.error('加载设置失败:', error);
    }
});

// 监听设置变化并自动保存
const debouncedSave = useDebounceFn(async () => {
    if (isSaving.value) return;
    
    try {
        isSaving.value = true;
        await settingsStore.updateSettings(localSettings.value as any);
        lastSaved.value = new Date();
        
        // 3秒后清除保存状态
        setTimeout(() => {
            lastSaved.value = null;
        }, 3000);
    } catch (error) {
        console.error('保存设置失败:', error);
        toast.add({
            title: '保存失败',
            description: '设置保存失败，请稍后重试',
            color: 'error'
        });
    } finally {
        isSaving.value = false;
    }
}, 1000);

watch(localSettings, debouncedSave, { deep: true });

// 与全局主题设置联动
watch(
    () => localSettings.value.preferences.theme,
    (newTheme) => {
        settingsStore.setTheme(newTheme);
    }
);

// 昵称编辑
const editNickname = () => {
    openNicknameEditModal({
        currentNickname: user.value?.nickname || '',
        onSuccess: onSaveNickname
    });
};

// 更换头像
const changeAvatar = () => {
    openAvatarUploadModal({
        user: user.value,
        onUploaded: onAvatarUploaded
    });
};

// 保存昵称
const savingNickname = ref(false);
async function onSaveNickname(nickname: string) {
    if (savingNickname.value) return;
    savingNickname.value = true;
    try {
        const userService = useUserService();
        await userService.updateUserProfile({ 
            nickname: nickname.trim() 
        });
        
        toast.add({ title: '已更新昵称', color: 'success' });
        
        // 更新本地store中的用户信息
        if (authStore.user) {
            authStore.user.nickname = nickname.trim();
        }
        
        // 更新本地设置
        localSettings.value.account.nickname = nickname.trim();
    } catch (e: any) {
        toast.add({ title: '更新失败', description: e?.message, color: 'error' });
    } finally {
        savingNickname.value = false;
    }
}

// 头像上传成功处理
const onAvatarUploaded = (avatarUrl: string) => {
    // 更新本地store中的用户头像
    if (authStore.user) {
        authStore.user.avatar_url = avatarUrl;
    }
    
    // 更新本地设置
    localSettings.value.account.avatar_url = avatarUrl;
    
    toast.add({
        title: "头像更新成功",
        description: "您的头像已更新",
        color: "success",
    });
};

// 升级账户
const upgradeAccount = () => {
    toast.add({
        title: "功能开发中",
        description: "账户升级功能即将上线",
        color: "info",
    });
};

// 修改密码
const changePassword = () => {
    toast.add({
        title: "功能开发中",
        description: "密码修改功能即将上线",
        color: "info",
    });
};

// 设置双重认证
const setup2FA = () => {
    toast.add({
        title: "功能开发中",
        description: "双重认证功能即将上线",
        color: "info",
    });
};

// 清除缓存
const clearCache = () => {
    toast.add({
        title: "缓存已清除",
        description: "应用缓存数据已清除",
        color: "success",
    });
};

// 显示帮助
const showHelp = () => {
    toast.add({
        title: "帮助中心",
        description: "文档建设中，敬请期待",
        color: "info",
    });
};

// 显示隐私政策
const showPrivacyPolicy = () => {
    toast.add({
        title: "隐私政策",
        description: "隐私政策页面建设中",
        color: "info",
    });
};

// 显示服务条款
const showTerms = () => {
    toast.add({
        title: "服务条款",
        description: "服务条款页面建设中",
        color: "info",
    });
};

// 删除账户
const deletingAccount = ref(false);
const deleteAccount = () => {
    openConfirmModal({
        title: '确认删除账户',
        message: '此操作不可恢复，确定要删除账户吗？您的所有数据将被永久删除。',
        confirmText: '确认删除',
        confirmColor: 'error',
        onConfirm: confirmDeleteAccount
    });
};

async function confirmDeleteAccount() {
    if (deletingAccount.value) return;
    deletingAccount.value = true;
    try {
        const userService = useUserService();
        await userService.deleteAccount();
        
        toast.add({ 
            title: '账户已删除', 
            description: '您的账户和所有数据已被永久删除',
            color: 'success' 
        });
        
        // 清除本地认证状态并跳转
        authStore.logout();
        await navigateTo('/');
    } catch (e: any) {
        toast.add({ 
            title: '删除失败', 
            description: e?.message || '请稍后重试',
            color: 'error' 
        });
    } finally {
        deletingAccount.value = false;
    }
}
</script>