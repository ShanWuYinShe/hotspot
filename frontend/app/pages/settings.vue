<template>
    <PageLayout
        title="设置中心"
        description="个性化定制您的体验，管理账户信息、隐私权限、通知偏好以及应用外观"
        :header-actions="headerActions"
        :searchable="true"
        :search-placeholder="'搜索设置项...'"
        @search="handleSearch"
    >
        <!-- 设置分类标签页 -->
        <div class="mb-6">
            <UTabs v-model="activeSection" :items="tabItems" />
        </div>

        <!-- 设置内容 -->
        <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
            <SettingsContent
                :id="`settings-panel-${activeSection}`"
                role="tabpanel"
                :aria-label="currentSectionTitle"
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
                @section-change="handleSectionChange"
            />
        </UCard>

        <!-- 保存状态提示 -->
        <UAlert
            v-if="isSaving"
            color="primary"
            icon="i-heroicons-arrow-path"
            title="保存中..."
            class="fixed bottom-4 right-4 z-50"
            :ui="{ icon: 'animate-spin' }"
        />
        <UAlert
            v-else-if="lastSaved"
            color="success"
            icon="i-heroicons-check-circle"
            title="设置已保存"
            class="fixed bottom-4 right-4 z-50"
            :timeout="3000"
        />
    </PageLayout>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from '~/config/seo';
import type { UserSettings } from '~/types/api';
import { useDebounceFn } from "@vueuse/core";

// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig('settings')

// 页面元数据
definePageMeta({
    title: seoConfig.title,
    description: seoConfig.description,
    requiresAuth: true,
    layout: 'dashboard'
});

// 设置 SEO meta
useSeoMeta(buildSeoMeta('settings'));

const authStore = useAuthStore();
const { success: toastSuccess, error: toastError, info: toastInfo } = useNotifications()
const toast = { 
    add: (options: { title: string; description?: string; color: string }) => {
        if (options.color === 'success') toastSuccess(options.title)
        else if (options.color === 'error') toastError(options.title)
        else toastInfo(options.title)
    }
}
const settingsStore = useSettingsStore();
// const { openConfirmModal } = useModals();

// 当前用户信息
const user = computed(() => authStore.user);

// 搜索功能
const searchQuery = ref('');

// 活动的设置分区
const activeSection = ref('account');

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
        keywords: ['外观', '语言', '界面', '显示']
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
        keywords: ['删除', '危险', '账户', '清除', '重置'],
        badge: '谨慎'
    }
];

// 标签页配置
const tabItems = computed(() => 
    sections.map(section => ({
        value: section.id,
        label: section.title,
        icon: section.icon,
        badge: section.badge
    }))
);

// 过滤后的分区
const filteredSections = computed(() => {
    if (!searchQuery.value.trim()) return sections;
    
    const query = searchQuery.value.toLowerCase();
    return sections.filter(section => 
        section.title.toLowerCase().includes(query) ||
        section.keywords.some(keyword => keyword.toLowerCase().includes(query))
    );
});

// 当前分区标题
const currentSectionTitle = computed(() => {
    return sections.find(s => s.id === activeSection.value)?.title || '设置';
});

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
        moment_notifications: true,
        comment_notifications: true,
    },
    privacy: {
        privacy_level: "public",
        location_sharing: true,
        online_status: true,
        show_in_search: true,
    },
    preferences: {
        language: "zh-CN",
        distance_unit: "km",
        time_format: "24h",
    },
    appearance: {
        language: "zh-CN",
        animations: true,
        compact_mode: false,
        high_contrast: false,
    },
    account: {
        auto_save: true,
        backup_enabled: false,
    },
});

// 头部操作按钮
const headerActions = computed(() => [
    {
        key: 'save',
        label: '保存设置',
        icon: 'i-heroicons-check',
        variant: 'solid' as const,
        color: 'primary' as const,
        loading: isSaving.value,
        disabled: !isDirty.value,
        action: saveSettings
    }
]);

// 检查是否有未保存的更改
const isDirty = computed(() => {
    // 这里可以添加更复杂的脏检查逻辑
    return false;
});

// 搜索处理
const handleSearch = useDebounceFn((query: string) => {
    searchQuery.value = query;
}, 300);

// 处理设置分类切换
const handleSectionChange = (section: string) => {
    activeSection.value = section;
};

// 保存设置
const saveSettings = async () => {
    if (isSaving.value) return;
    
    try {
        isSaving.value = true;
        
        // 调用设置存储的保存方法
        // await settingsStore.updateSettings(localSettings.value);
        
        lastSaved.value = new Date();
        toast.add({
            title: '设置已保存',
            description: '您的设置已成功保存',
            color: 'success'
        });
        
        // 3秒后清除保存状态
        setTimeout(() => {
            lastSaved.value = null;
        }, 3000);
        
    } catch (error) {
        console.error('保存设置失败:', error);
        toast.add({
            title: '保存失败',
            description: '设置保存时出现错误，请重试',
            color: 'error'
        });
    } finally {
        isSaving.value = false;
    }
};

// 事件处理函数
const onSaveNickname = (nickname: string) => {
    // 更新用户昵称逻辑
    toast.add({
        title: '昵称已更新',
        description: `昵称已更改为 ${nickname}`,
        color: 'success'
    });
};

const onAvatarUploaded = (avatarUrl: string) => {
    // 更新用户头像逻辑
    toast.add({
        title: '头像已更新',
        description: '头像已成功上传',
        color: 'success'
    });
};

const upgradeAccount = () => {
    navigateTo('/auth/register?upgrade=1');
};

const changePassword = () => {
    toast.add({
        title: '功能开发中',
        description: '密码修改功能即将上线',
        color: 'info'
    });
};

const setup2FA = () => {
    toast.add({
        title: '功能开发中',
        description: '双因素认证功能即将上线',
        color: 'info'
    });
};

const clearCache = () => {
    if (confirm('确定要清除应用缓存吗？这将清除所有本地数据。')) {
        // 清除缓存逻辑
        localStorage.clear();
        sessionStorage.clear();
        toast.add({
            title: '缓存已清除',
            description: '应用缓存已成功清除',
            color: 'success'
        });
    }
};

const showHelp = () => {
    toast.add({
        title: '帮助中心',
        description: '帮助文档即将上线',
        color: 'info'
    });
};

const showPrivacyPolicy = () => {
    toast.add({
        title: '隐私政策',
        description: '隐私政策页面即将上线',
        color: 'info'
    });
};

const showTerms = () => {
    toast.add({
        title: '服务条款',
        description: '服务条款页面即将上线',
        color: 'info'
    });
};

const deleteAccount = () => {
    if (confirm('确定要删除您的账户吗？此操作不可撤销，所有数据将被永久删除。')) {
        toast.add({
            title: '功能开发中',
            description: '账户删除功能即将上线',
            color: 'info'
        });
    }
};

const editNickname = () => {
    toast.add({
        title: '功能开发中',
        description: '昵称编辑功能即将上线',
        color: 'info'
    });
};

const changeAvatar = () => {
    toast.add({
        title: '功能开发中',
        description: '头像上传功能即将上线',
        color: 'info'
    });
};

// 页面加载时初始化设置
onMounted(async () => {
    try {
        // 从存储中加载设置
        // localSettings.value = { ...settingsStore.settings };
    } catch (error) {
        console.error('加载设置失败:', error);
    }
});
</script>
