<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <div class="max-w-4xl mx-auto">
            <!-- 页面标题 -->
            <div class="mb-6">
                <h1 class="text-2xl font-bold text-neutral-900">设置</h1>
            </div>

            <!-- 设置选项 -->
            <UAccordion :items="settingsAccordionItems" multiple>
                <template #account="{ item, open }">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-user-circle" class="w-5 h-5" />
                        <span class="font-medium">账户设置</span>
                    </div>
                </template>
                
                <template #account-content>

                <div class="space-y-4">
                    <!-- 昵称设置 -->
                    <div class="flex items-center justify-between">
                        <div>
                            <div class="font-medium">昵称</div>
                            <div class="text-sm text-neutral-600">
                                {{ user?.nickname || "未设置" }}
                            </div>
                        </div>
                        <UButton variant="soft" size="sm" @click="showNicknameModal = true">
                            编辑
                        </UButton>
                    </div>

                    <!-- 头像设置 -->
                    <div class="flex items-center justify-between">
                        <div>
                            <div class="font-medium">头像</div>
                            <div class="text-sm text-neutral-600">
                                点击更换头像
                            </div>
                        </div>
                        <UButton variant="soft" size="sm" @click="showAvatarModal = true">
                            更换
                        </UButton>
                    </div>

                    <!-- 绑定状态 -->
                    <div v-if="false" class="border-t pt-4">
                        <UCard>
                            <EmptyState
                                icon="i-heroicons-envelope-open"
                                title="尚未绑定账号"
                                description="绑定手机号或邮箱以保存数据并提高安全性"
                                action-text="立即绑定"
                                :action-handler="bindAccount"
                                secondary-action-text="了解绑定"
                                :secondary-action-handler="
                                    () =>
                                        toast.add({
                                            title: '绑定说明',
                                            description:
                                                '绑定可提升找回与安全能力',
                                            color: 'info',
                                        })
                                "
                            >
                                <template #illustration>
                                    <EmptyAccount size="md" />
                                </template>
                            </EmptyState>
                        </UCard>
                    </div>
                </div>
                </template>
                
                <template #privacy="{ item, open }">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-shield-check" class="w-5 h-5" />
                        <span class="font-medium">隐私设置</span>
                    </div>
                </template>
                
                <template #privacy-content>

                <div class="space-y-4">
                    <!-- 位置共享 -->
                    <div class="flex items-center justify-between">
                        <div>
                            <div class="font-medium">位置共享</div>
                            <div class="text-sm text-neutral-600">
                                允许其他用户查看我的位置
                            </div>
                        </div>
                        <USwitch v-model="settings.shareLocation" />
                    </div>

                    <!-- 在线状态 -->
                    <div class="flex items-center justify-between">
                        <div>
                            <div class="font-medium">在线状态</div>
                            <div class="text-sm text-neutral-600">
                                显示我的在线状态
                            </div>
                        </div>
                        <USwitch v-model="settings.showOnlineStatus" />
                    </div>

                    <!-- 消息通知设置 -->
                    <div
                        class="space-y-4 pt-4 border-t border-neutral-200 dark:border-neutral-700"
                    >
                        <h4
                            class="font-medium text-neutral-900 dark:text-neutral-100"
                        >
                            消息通知设置
                        </h4>

                        <!-- 私聊消息提示 -->
                        <div class="flex items-center justify-between">
                            <div>
                                <div class="font-medium">私聊消息提示</div>
                                <div class="text-sm text-neutral-600">
                                    接收到私聊消息时显示弹窗提醒
                                </div>
                            </div>
                            <USwitch v-model="settings.directMessageToast" />
                        </div>

                        <!-- 群组消息提示 -->
                        <div class="flex items-center justify-between">
                            <div>
                                <div class="font-medium">群组消息提示</div>
                                <div class="text-sm text-neutral-600">
                                    接收到群组消息时显示弹窗提醒
                                </div>
                            </div>
                            <USwitch v-model="settings.groupMessageToast" />
                        </div>

                        <!-- 系统通知提示 -->
                        <div class="flex items-center justify-between">
                            <div>
                                <div class="font-medium">系统通知提示</div>
                                <div class="text-sm text-neutral-600">
                                    系统重要通知弹窗提醒
                                </div>
                            </div>
                            <USwitch
                                v-model="settings.systemNotificationToast"
                            />
                        </div>

                        <!-- 错误通知 -->
                        <div class="flex items-center justify-between">
                            <div>
                                <div class="font-medium">错误提示</div>
                                <div class="text-sm text-neutral-600">
                                    显示操作错误提示
                                </div>
                            </div>
                            <USwitch v-model="settings.errorNotifications" />
                        </div>

                        <!-- 成功通知 -->
                        <div class="flex items-center justify-between">
                            <div>
                                <div class="font-medium">成功提示</div>
                                <div class="text-sm text-neutral-600">
                                    显示操作成功提示
                                </div>
                            </div>
                            <USwitch v-model="settings.successNotifications" />
                        </div>
                    </div>
                </div>
                </template>
                
                <template #features="{ item, open }">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-cog-6-tooth" class="w-5 h-5" />
                        <span class="font-medium">功能设置</span>
                    </div>
                </template>
                
                <template #features-content>

                <div class="space-y-4">
                    <!-- 主题模式 -->
                    <div class="flex items-center justify-between">
                        <div>
                            <div class="font-medium">主题模式</div>
                            <div class="text-sm text-neutral-600">
                                切换明暗主题
                            </div>
                        </div>
                        <USelect
                            v-model="settings.theme"
                            :options="themeOptions"
                            size="sm"
                            class="w-32"
                        />
                    </div>

                    <!-- 语言设置 -->
                    <div class="flex items-center justify-between">
                        <div>
                            <div class="font-medium">语言</div>
                            <div class="text-sm text-neutral-600">
                                界面显示语言
                            </div>
                        </div>
                        <USelect
                            v-model="settings.language"
                            :options="languageOptions"
                            size="sm"
                            class="w-32"
                        />
                    </div>

                    <!-- 地图提供商 -->
                    <div class="flex items-center justify-between">
                        <div>
                            <div class="font-medium">地图提供商</div>
                            <div class="text-sm text-neutral-600">
                                选择地图服务
                            </div>
                        </div>
                        <USelect
                            v-model="settings.mapProvider"
                            :options="mapProviderOptions"
                            size="sm"
                            class="w-32"
                        />
                    </div>
                </div>
                </template>
                
                <template #other="{ item, open }">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-information-circle" class="w-5 h-5" />
                        <span class="font-medium">其他设置</span>
                    </div>
                </template>
                
                <template #other-content>
                    <div class="space-y-1">
                        <UButton
                            variant="ghost"
                            block
                            justify="start"
                            @click="clearCache"
                        >
                            <UIcon name="i-heroicons-trash" class="w-5 h-5 mr-3" />
                            清除缓存
                        </UButton>

                        <UButton
                            variant="ghost"
                            block
                            justify="start"
                            @click="showAbout"
                        >
                            <UIcon
                                name="i-heroicons-information-circle"
                                class="w-5 h-5 mr-3"
                            />
                            关于应用
                        </UButton>

                        <UButton
                            variant="ghost"
                            block
                            justify="start"
                            @click="showHelp"
                        >
                            <UIcon
                                name="i-heroicons-question-mark-circle"
                                class="w-5 h-5 mr-3"
                            />
                            帮助与支持
                        </UButton>
                    </div>
                </template>
                
                <template #danger="{ item, open }">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-exclamation-triangle" class="w-5 h-5 text-red-600" />
                        <span class="font-medium text-red-600">危险操作</span>
                    </div>
                </template>
                
                <template #danger-content>
                    <div class="space-y-1">
                        <UButton
                            variant="ghost"
                            color="error"
                            block
                            justify="start"
                            @click="showDeleteConfirm = true"
                        >
                            <UIcon name="i-heroicons-trash" class="w-5 h-5 mr-3" />
                            删除账户
                        </UButton>
                    </div>
                </template>
            </UAccordion>
        </div>
    </div>

    <!-- 昵称编辑 -->
    <UModal v-model:open="showNicknameModal" title="编辑昵称">
        <template #body>
            <UForm :state="nicknameForm" :validate="validateNickname" @submit="onSaveNickname" class="space-y-4">
                <UFormField label="昵称" name="nickname" required>
                    <UInput v-model="nicknameForm.nickname" placeholder="请输入新的昵称" :maxlength="20" />
                </UFormField>
                <div class="flex justify-end gap-2">
                    <UButton variant="ghost" @click="showNicknameModal = false">取消</UButton>
                    <UButton type="submit" :loading="savingNickname">保存</UButton>
                </div>
            </UForm>
        </template>
    </UModal>

    <!-- 头像上传 -->
    <UModal v-model:open="showAvatarModal" title="更换头像">
        <template #body>
            <AvatarUpload :user="user" @uploaded="onAvatarUploaded" />
        </template>
        <template #footer>
            <div class="flex justify-end">
                <UButton variant="ghost" @click="showAvatarModal = false">关闭</UButton>
            </div>
        </template>
    </UModal>

    <!-- 删除账户确认 -->
    <UModal v-model:open="showDeleteConfirm" title="确认删除账户">
        <template #body>
            <p class="text-neutral-600 dark:text-neutral-300">此操作不可恢复，确定要删除账户吗？</p>
        </template>
        <template #footer>
            <div class="flex justify-end gap-2">
                <UButton variant="ghost" @click="showDeleteConfirm = false">取消</UButton>
                <UButton color="error" :loading="deletingAccount" @click="confirmDeleteAccount">确认删除</UButton>
            </div>
        </template>
    </UModal>
</template>

<script setup lang="ts">
// 页面元数据
definePageMeta({
    title: "设置",
    requiresAuth: true,
});
import EmptyAccount from "~/components/illustrations/EmptyAccount.vue";

const authStore = useAuthStore();
const toast = useSmartToast();
const settingsStore = useSettingsStore();

// 当前用户信息
const user = computed(() => authStore.user);

// 设置数据
const settings = reactive({
    shareLocation: true,
    showOnlineStatus: true,
    messageNotifications: true,
    directMessageToast: true,
    groupMessageToast: true,
    systemNotificationToast: true,
    errorNotifications: true,
    successNotifications: false,
    theme: "auto",
    language: "zh-CN",
    mapProvider: "gaode",
});

// UAccordion 配置
const settingsAccordionItems = computed(() => [
    {
        label: '账户设置',
        icon: 'i-heroicons-user-circle',
        slot: 'account',
        defaultOpen: true,
        content: 'account-content'
    },
    {
        label: '隐私设置',
        icon: 'i-heroicons-shield-check',
        slot: 'privacy',
        defaultOpen: false,
        content: 'privacy-content'
    },
    {
        label: '功能设置',
        icon: 'i-heroicons-cog-6-tooth',
        slot: 'features',
        defaultOpen: false,
        content: 'features-content'
    },
    {
        label: '其他设置',
        icon: 'i-heroicons-information-circle',
        slot: 'other',
        defaultOpen: false,
        content: 'other-content'
    },
    {
        label: '危险操作',
        icon: 'i-heroicons-exclamation-triangle',
        slot: 'danger',
        defaultOpen: false,
        content: 'danger-content'
    }
]);

// 与全局主题设置联动（从全局 -> 本地选择器）
watch(
    () => settingsStore.theme,
    (val) => {
        settings.theme = val as any;
    },
    { immediate: true },
);

// 当本地主题选择变化时，更新到全局并持久化（从本地 -> 全局）
watch(
    () => settings.theme,
    (val) => {
        settingsStore.setTheme(val as "light" | "dark" | "auto");
    },
);

// 选项数据
const themeOptions = [
    { label: "跟随系统", value: "auto" },
    { label: "浅色模式", value: "light" },
    { label: "深色模式", value: "dark" },
];

const languageOptions = [
    { label: "简体中文", value: "zh-CN" },
    { label: "English", value: "en-US" },
];

const mapProviderOptions = [
    { label: "高德地图", value: "gaode" },
    { label: "百度地图", value: "baidu" },
    { label: "腾讯地图", value: "tencent" },
];

// 昵称编辑
const showNicknameModal = ref(false);
const savingNickname = ref(false);
const nicknameForm = ref({ nickname: '' });
watch(user, (u) => { nicknameForm.value.nickname = (u?.nickname || '').trim(); }, { immediate: true });

// 头像上传
const showAvatarModal = ref(false);

type ValidateResult = { path: string; message: string }[];
function validateNickname(): ValidateResult | void {
    const errors: ValidateResult = [];
    const n = (nicknameForm.value.nickname || '').trim();
    if (!n) errors.push({ path: 'nickname', message: '昵称为必填项' });
    if (n.length > 20) errors.push({ path: 'nickname', message: '昵称不能超过20个字符' });
    return errors.length ? errors : undefined;
}

async function onSaveNickname() {
    if (savingNickname.value) return;
    savingNickname.value = true;
    try {
        const userService = useUserService();
        await userService.updateUserProfile({ 
            nickname: nicknameForm.value.nickname.trim() 
        });
        
        toast.add({ title: '已更新昵称', color: 'success' });
        
        // 更新本地store中的用户信息
        if (authStore.user) {
            authStore.user.nickname = nicknameForm.value.nickname.trim();
        }
        
        showNicknameModal.value = false;
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
    
    toast.add({
        title: "头像更新成功",
        description: "您的头像已更新",
        color: "success",
    });
    
    showAvatarModal.value = false;
};

// 绑定账户
const bindAccount = () => {
    toast.add({
        title: "功能暂未开放",
        description: "绑定账号功能即将上线",
        color: "info",
    });
};

// 清除缓存
const clearCache = () => {
    toast.add({
        title: "缓存已清除",
        color: "success",
    });
};

// 显示关于
const showAbout = () => {
    toast.add({
        title: "HotSpot v1.0.0",
        description: "基于位置的社交平台",
        color: "info",
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

// 删除账户
const showDeleteConfirm = ref(false);
const deletingAccount = ref(false);
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
        
        showDeleteConfirm.value = false;
        
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

// 监听设置变化并保存
watch(
    settings,
    (newSettings) => {
        // 这里可以保存设置到本地存储或服务器
        // Settings updated successfully
    },
    { deep: true },
);
</script>
