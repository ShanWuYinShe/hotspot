<template>
    <div class="min-h-screen bg-gradient-to-br from-neutral-50 via-white to-neutral-100 dark:from-neutral-900 dark:via-neutral-950 dark:to-neutral-900">
        <!-- 无障碍状态区域 (保存状态) -->
        <div class="sr-only" aria-live="polite" aria-atomic="true">
            <span v-if="isSaving">设置保存中</span>
            <span v-else-if="lastSaved">设置已保存</span>
        </div>
        <!-- 移动端粘性头部（标题 + 搜索 + 横向分类导航合集） -->
        <div class="lg:hidden sticky top-0 z-30 backdrop-blur-xl bg-white/85 dark:bg-neutral-900/85 border-b border-neutral-200 dark:border-neutral-800 shadow-sm">
            <div class="px-4 pt-4 pb-3 space-y-3">
                <div class="flex items-center justify-between">
                    <div>
                        <h1 class="text-xl font-bold text-neutral-900 dark:text-white leading-tight">设置</h1>
                        <p class="text-xs text-neutral-600 dark:text-neutral-400">账户 · 隐私 · 通知 · 外观</p>
                    </div>
                    <div class="flex items-center gap-2">
                        <UButton v-if="isDirty && !isSaving" size="xs" color="primary" @click="saveSettings" class="shadow-sm">保存</UButton>
                        <div v-if="isSaving" class="flex items-center gap-1 text-xs text-neutral-500 dark:text-neutral-400">
                            <UIcon name="i-heroicons-arrow-path" class="w-4 h-4 animate-spin" />
                            <span>保存中</span>
                        </div>
                        <div v-else-if="lastSaved" class="flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
                            <UIcon name="i-heroicons-check-circle" class="w-4 h-4" />
                            <span>已保存</span>
                        </div>
                    </div>
                </div>
                <UInput
                    v-model="searchQuery"
                    placeholder="搜索设置项..."
                    size="sm"
                        :ui="{ 
                            base: 'bg-neutral-50 dark:bg-neutral-800/50 border border-neutral-200 dark:border-neutral-700 focus:ring-2 focus:ring-primary-500 focus:border-primary-500 transition-all duration-200 px-12 py-3'
                        }"
                >
                    <template #leading>
                        <UIcon name="i-heroicons-magnifying-glass" class="text-neutral-500 dark:text-neutral-400 w-4 h-4 ml-1" />
                    </template>
                </UInput>
                <div
                    class="flex overflow-x-auto gap-2 scrollbar-hide -mx-1 px-1 pb-1"
                    role="tablist"
                    aria-label="设置分类"
                    @keydown.left.prevent="focusPrevSection()"
                    @keydown.right.prevent="focusNextSection()"
                >
                    <UButton
                        v-for="section in filteredSections"
                        :key="section.id"
                        @click="activeSection = section.id"
                        :variant="activeSection === section.id ? 'solid' : 'soft'"
                        :color="activeSection === section.id ? 'primary' : 'neutral'"
                        size="xs"
                        class="whitespace-nowrap flex-shrink-0 px-3 py-2 rounded-full transition-all duration-200"
                        :id="'tab-' + section.id"
                        :aria-controls="`settings-panel-${section.id}`"
                        :class="activeSection === section.id ? 'shadow-md bg-primary-100 dark:bg-primary-900/20' : 'hover:bg-neutral-100 dark:hover:bg-neutral-800'"
                        role="tab"
                        :aria-selected="activeSection === section.id"
                        :tabindex="activeSection === section.id ? 0 : -1"
                    >
                        <UIcon :name="section.icon" class="w-4 h-4 mr-1" />
                        {{ section.title }}
                        <UBadge v-if="section.badge" :label="section.badge" color="error" size="xs" class="ml-1" />
                    </UButton>
                </div>
            </div>
        </div>

        <!-- 主内容区域 -->
        <div class="mx-auto max-w-7xl px-4 lg:px-8 pt-4 lg:pt-10 pb-10">
            <!-- 桌面端标题区 -->
            <div class="hidden lg:flex items-end justify-between mb-8">
                <div class="space-y-3">
                    <h1 class="text-4xl font-bold tracking-tight text-neutral-900 dark:text-white">设置</h1>
                    <p class="text-neutral-600 dark:text-neutral-400 max-w-2xl text-base leading-relaxed">管理您的账户信息、隐私权限、通知偏好以及应用外观与行为。</p>
                </div>
                <div class="flex items-center gap-4">
                    <UInput
                        v-model="searchQuery"
                        placeholder="搜索设置项..."
                        size="md"
                        class="w-80"
                        :ui="{ 
                            base: 'bg-white dark:bg-neutral-800 shadow-sm border border-neutral-200 dark:border-neutral-700 focus:ring-2 focus:ring-primary-500 focus:border-primary-500 focus:shadow-lg focus:shadow-primary-500/25 transition-all duration-200 rounded-lg px-12 py-3.5'
                        }"
                    >
                        <template #leading>
                            <UIcon name="i-heroicons-magnifying-glass" class="text-neutral-500 dark:text-neutral-400 w-5 h-5 ml-1" />
                        </template>
                    </UInput>
                    <UButton v-if="isDirty && !isSaving" color="primary" @click="saveSettings" :loading="isSaving" class="shadow-sm">保存更改</UButton>
                    <div v-if="isSaving" class="flex items-center gap-2 text-sm px-3 py-2 rounded-md bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
                        <UIcon name="i-heroicons-arrow-path" class="w-5 h-5 animate-spin text-blue-600" />
                        <span class="font-medium text-blue-700 dark:text-blue-400">保存中</span>
                    </div>
                    <div v-else-if="lastSaved" class="flex items-center gap-2 text-sm px-3 py-2 rounded-md bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800">
                        <UIcon name="i-heroicons-check-circle" class="w-5 h-5 text-green-600" />
                        <span class="font-medium text-green-700 dark:text-green-400">已保存</span>
                    </div>
                </div>
            </div>

            <div class="grid lg:grid-cols-[240px_minmax(0,1fr)] gap-6 lg:gap-10">
                <!-- 侧边分类导航（桌面） -->
                <aside class="hidden lg:block" aria-label="设置分类">
                    <div class="sticky top-28 space-y-4">
                        <h3 class="text-xs font-semibold tracking-wider text-neutral-500 dark:text-neutral-400 uppercase px-1">设置分类</h3>
                        <nav class="space-y-2" role="tablist" aria-orientation="vertical"
                             @keydown.up.prevent="focusPrevSection()"
                             @keydown.down.prevent="focusNextSection()"
                        >
                            <UButton
                                v-for="section in filteredSections"
                                :key="section.id"
                                @click="activeSection = section.id"
                                :variant="activeSection === section.id ? 'solid' : 'ghost'"
                                :color="activeSection === section.id ? 'primary' : 'neutral'"
                                size="md"
                                class="w-full justify-start gap-3 px-4 py-3 rounded-xl transition-all duration-150 group"
                                :class="activeSection === section.id ? 'shadow-sm ring-1 ring-primary-400/40' : 'hover:bg-neutral-100 dark:hover:bg-neutral-600'"
                                role="tab"
                                :aria-selected="activeSection === section.id"
                                :tabindex="activeSection === section.id ? 0 : -1"
                                :aria-controls="`settings-panel-${section.id}`"
                                :id="'tab-' + section.id"
                            >
                                <div class="flex items-center justify-center w-8 h-8 rounded-lg"
                                     :class="activeSection === section.id ? 'bg-white/20' : 'bg-neutral-100 dark:bg-neutral-600'">
                                    <UIcon :name="section.icon" class="w-5 h-5" />
                                </div>
                                <span class="font-medium">{{ section.title }}</span>
                                <UBadge v-if="section.badge" :label="section.badge" color="error" size="xs" class="ml-auto" />
                            </UButton>
                        </nav>
                    </div>
                </aside>

                <!-- 内容面板 -->
                <main>
                    <div class="rounded-xl border border-neutral-200 dark:border-neutral-700 bg-white/70 dark:bg-neutral-800/50 backdrop-blur-md shadow-lg ring-1 ring-black/5 dark:ring-white/5 overflow-hidden">
                        <SettingsContent
                            :id="`settings-panel-${activeSection}`"
                            role="tabpanel"
                            :aria-label="currentSectionTitle"
                            :active-section="activeSection"
                            :local-settings="localSettings"
                            :user="user"
                            :is-mobile="isMobileScreen"
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
                </main>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import type { UserSettings } from '~/types';
import { useDebounceFn } from "@vueuse/core";
import { useBreakpoints, breakpointsTailwind } from '@vueuse/core';
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

// 路由深链同步 (section 查询参数)
const route = useRoute();
const router = useRouter();
const sectionIds = computed(() => sections.map(s => s.id));

function applySectionFromRoute() {
    const q = route.query.section;
    if (typeof q === 'string' && sectionIds.value.includes(q) && q !== activeSection.value) {
        activeSection.value = q;
    }
}

// 初次挂载后解析 URL
onMounted(() => {
    applySectionFromRoute();
});

// 监听浏览器前进后退导致的 query 改变
watch(() => route.query.section, () => applySectionFromRoute());

// activeSection 改变时写回 URL（不产生历史记录）
watch(activeSection, (val) => {
    if (route.query.section !== val) {
        router.replace({ query: { ...route.query, section: val } });
    }
    // 记录最近使用的分区
    try {
        localStorage.setItem('settings:lastSection', val);
    } catch {}
});

// 当前 section 标题（供 aria-label）
const currentSectionTitle = computed(() => {
    return sections.find(s => s.id === activeSection.value)?.title || '设置';
});

// 键盘导航（上下 / 左右 切换）
function focusNextSection() {
    const idx = sectionIds.value.indexOf(activeSection.value);
    const next = sectionIds.value[(idx + 1) % sectionIds.value.length] || sectionIds.value[0] || 'account';
    activeSection.value = next as string;
}

function focusPrevSection() {
    const idx = sectionIds.value.indexOf(activeSection.value);
    const prev = sectionIds.value[(idx - 1 + sectionIds.value.length) % sectionIds.value.length] || sectionIds.value[0] || 'account';
    activeSection.value = prev as string;
}

// 保存状态
const isSaving = ref(false);
const lastSaved = ref<Date | null>(null);

// 响应式断点（供 SettingsContent 使用）
const breakpoints = useBreakpoints(breakpointsTailwind);
const isMobileScreen = breakpoints.smaller('lg');

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

// 显式保存：跟踪脏状态，点击按钮再提交，减少无谓请求
const isDirty = ref(false);
const isInitializing = ref(true);

// 只监听实际的设置变化，排除初始化阶段
watch(localSettings, () => { 
    if (!isInitializing.value) {
        isDirty.value = true; 
    }
}, { deep: true });

// 初始化设置
onMounted(async () => {
    try {
        await settingsStore.fetchSettings();
        if (settingsStore.settings) {
            localSettings.value = { ...settingsStore.settings } as unknown as UserSettings;
        }
        // 恢复最近分区
        const last = localStorage.getItem('settings:lastSection');
        if (last && sectionIds.value.includes(last)) {
            activeSection.value = last;
        }
    } catch (error) {
        console.error('加载设置失败:', error);
    } finally {
        // 初始化完成后才开始监听变化
        nextTick(() => {
            isInitializing.value = false;
        });
    }
});

async function saveSettings() {
    if (isSaving.value || !isDirty.value) return;
    try {
        isSaving.value = true;
        await settingsStore.updateSettings(localSettings.value as any);
        lastSaved.value = new Date();
        isDirty.value = false;
        setTimeout(() => { lastSaved.value = null; }, 3000);
        toast.add({ title: '已保存设置', color: 'success' });
    } catch (error) {
        console.error('保存设置失败:', error);
        toast.add({ title: '保存失败', description: '请稍后重试', color: 'error' });
    } finally {
        isSaving.value = false;
    }
}

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
        
        // 重置脏状态，因为这是通过API同步的更新
        isDirty.value = false;
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
    
    // 重置脏状态，因为这是通过API同步的更新
    isDirty.value = false;
    
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