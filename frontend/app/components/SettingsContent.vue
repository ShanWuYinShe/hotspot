<template>
    <!-- 账户设置 -->
    <div v-show="activeSection === 'account'" :class="isMobile ? 'space-y-4' : 'space-y-6'">
        <UCard :class="isMobile ? 'shadow-sm' : ''">
            <template #header>
                <div class="flex items-center gap-3">
                    <UIcon name="i-heroicons-user-circle" class="w-6 h-6 text-primary-600" />
                    <div>
                        <h2 :class="isMobile ? 'text-lg font-semibold' : 'text-xl font-semibold'" class="text-neutral-900 dark:text-white">账户信息</h2>
                        <p class="text-sm text-neutral-600 dark:text-neutral-400">管理您的个人资料和账户设置</p>
                    </div>
                </div>
            </template>

            <div :class="isMobile ? 'space-y-4' : 'space-y-6'">
                <!-- 用户头像 -->
                <div :class="isMobile ? 'flex flex-col items-center text-center gap-4' : 'flex items-center gap-6'">
                    <div class="relative">
                        <UAvatar
                            :src="user?.avatar_url"
                            :alt="user?.nickname"
                            :size="isMobile ? 'lg' : 'xl'"
                            class="ring-4 ring-white dark:ring-neutral-800 shadow-lg"
                        />
                        <UButton
                            @click="$emit('changeAvatar')"
                            icon="i-heroicons-camera"
                            size="sm"
                            color="primary"
                            class="absolute -bottom-1 -right-1 rounded-full p-2 shadow-lg"
                            square
                        />
                    </div>
                    <div>
                        <h3 :class="isMobile ? 'text-lg' : 'text-lg'" class="font-medium text-neutral-900 dark:text-white">{{ user?.nickname || '未设置昵称' }}</h3>
                        <p class="text-sm text-neutral-600 dark:text-neutral-400">{{ (user as any)?.login_id || '临时用户' }}</p>
                        <UButton variant="soft" size="sm" @click="$emit('changeAvatar')" class="mt-2">
                            更换头像
                        </UButton>
                    </div>
                </div>

                <USeparator />

                <!-- 基本信息 -->
                <div :class="isMobile ? 'space-y-4' : 'grid grid-cols-1 md:grid-cols-2 gap-6'">
                    <UFormField label="昵称" class="space-y-2">
                        <div class="flex gap-2">
                            <UInput
                                :value="user?.nickname || ''"
                                readonly
                                class="flex-1"
                                placeholder="未设置昵称"
                            />
                            <UButton variant="outline" size="sm" @click="$emit('editNickname')">
                                编辑
                            </UButton>
                        </div>
                    </UFormField>

                    <UFormField label="账户类型" class="space-y-2">
                        <div class="flex items-center gap-2">
                            <UBadge
                                :label="(user as any)?.is_temp ? '临时用户' : '正式用户'"
                                :color="(user as any)?.is_temp ? 'warning' : 'success'"
                                variant="soft"
                            />
                            <UButton
                                v-if="(user as any)?.is_temp"
                                variant="soft"
                                size="xs"
                                @click="$emit('upgradeAccount')"
                            >
                                升级账户
                            </UButton>
                        </div>
                    </UFormField>
                </div>

                <!-- 账户安全 -->
                <div v-if="!(user as any)?.is_temp" class="space-y-4">
                    <h4 :class="isMobile ? 'text-base' : 'text-lg'" class="font-medium text-neutral-900 dark:text-white">账户安全</h4>
                    <div :class="isMobile ? 'space-y-3' : 'grid grid-cols-1 md:grid-cols-2 gap-4'">
                        <UCard :class="isMobile ? 'p-3' : 'p-4'">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <UIcon name="i-heroicons-key" class="w-5 h-5 text-neutral-600" />
                                    <div>
                                        <div class="font-medium">登录密码</div>
                                        <div class="text-sm text-neutral-600">上次修改：30天前</div>
                                    </div>
                                </div>
                                <UButton variant="ghost" size="sm" @click="$emit('changePassword')">
                                    修改
                                </UButton>
                            </div>
                        </UCard>

                        <UCard :class="isMobile ? 'p-3' : 'p-4'">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <UIcon name="i-heroicons-device-phone-mobile" class="w-5 h-5 text-neutral-600" />
                                    <div>
                                        <div class="font-medium">双重认证</div>
                                        <div class="text-sm text-neutral-600">未启用</div>
                                    </div>
                                </div>
                                <UButton variant="ghost" size="sm" @click="$emit('setup2FA')">
                                    设置
                                </UButton>
                            </div>
                        </UCard>
                    </div>
                </div>
            </div>
        </UCard>
    </div>

    <!-- 隐私设置 -->
    <div v-show="activeSection === 'privacy'" :class="isMobile ? 'space-y-4' : 'space-y-6'">
        <UCard :class="isMobile ? 'shadow-sm' : ''">
            <template #header>
                <div class="flex items-center gap-3">
                    <UIcon name="i-heroicons-shield-check" class="w-6 h-6 text-primary-600" />
                    <div>
                        <h2 :class="isMobile ? 'text-lg font-semibold' : 'text-xl font-semibold'" class="text-neutral-900 dark:text-white">隐私设置</h2>
                        <p class="text-sm text-neutral-600 dark:text-neutral-400">控制您的隐私和数据共享偏好</p>
                    </div>
                </div>
            </template>

            <div :class="isMobile ? 'space-y-4' : 'space-y-6'">
                <!-- 位置隐私 -->
                <div class="space-y-4">
                    <h4 :class="isMobile ? 'text-base' : 'text-lg'" class="font-medium text-neutral-900 dark:text-white">位置隐私</h4>
                    <div class="space-y-3">
                        <div :class="isMobile ? 'p-3' : 'p-4'" class="bg-neutral-50 dark:bg-neutral-800 rounded-lg">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <UIcon name="i-heroicons-map-pin" class="w-5 h-5 text-neutral-600" />
                                    <div>
                                        <div class="font-medium">位置共享</div>
                                        <div class="text-sm text-neutral-600">允许其他用户查看我的位置</div>
                                    </div>
                                </div>
                                <USwitch v-model="localSettings.privacy.location_sharing" />
                            </div>
                        </div>

                        <div :class="isMobile ? 'p-3' : 'p-4'" class="bg-neutral-50 dark:bg-neutral-800 rounded-lg">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <UIcon name="i-heroicons-eye" class="w-5 h-5 text-neutral-600" />
                                    <div>
                                        <div class="font-medium">在线状态</div>
                                        <div class="text-sm text-neutral-600">显示我的在线状态给其他用户</div>
                                    </div>
                                </div>
                                <USwitch v-model="localSettings.privacy.online_status" />
                            </div>
                        </div>

                        <div :class="isMobile ? 'p-3' : 'p-4'" class="bg-neutral-50 dark:bg-neutral-800 rounded-lg">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <UIcon name="i-heroicons-magnifying-glass" class="w-5 h-5 text-neutral-600" />
                                    <div>
                                        <div class="font-medium">搜索可见性</div>
                                        <div class="text-sm text-neutral-600">允许其他用户通过搜索找到我</div>
                                    </div>
                                </div>
                                <USwitch v-model="localSettings.privacy.show_in_search" />
                            </div>
                        </div>
                    </div>
                </div>

                <USeparator />

                <!-- 资料可见性 -->
                <div class="space-y-4">
                    <h4 :class="isMobile ? 'text-base' : 'text-lg'" class="font-medium text-neutral-900 dark:text-white">资料可见性</h4>
                    <UFormField label="谁可以查看我的资料" class="space-y-2">
                        <USelect
                            v-model="localSettings.privacy.profile_visibility"
                            :options="profileVisibilityOptions"
                            :class="isMobile ? 'w-full' : 'w-full max-w-xs'"
                        />
                    </UFormField>
                </div>
            </div>
        </UCard>
    </div>

    <!-- 通知设置 -->
    <div v-show="activeSection === 'notifications'" :class="isMobile ? 'space-y-4' : 'space-y-6'">
        <UCard :class="isMobile ? 'shadow-sm' : ''">
            <template #header>
                <div class="flex items-center gap-3">
                    <UIcon name="i-heroicons-bell" class="w-6 h-6 text-primary-600" />
                    <div>
                        <h2 :class="isMobile ? 'text-lg font-semibold' : 'text-xl font-semibold'" class="text-neutral-900 dark:text-white">通知设置</h2>
                        <p class="text-sm text-neutral-600 dark:text-neutral-400">管理您接收通知的方式和频率</p>
                    </div>
                </div>
            </template>

            <div :class="isMobile ? 'space-y-4' : 'space-y-6'">
                <!-- 推送通知 -->
                <div class="space-y-4">
                    <h4 :class="isMobile ? 'text-base' : 'text-lg'" class="font-medium text-neutral-900 dark:text-white">推送通知</h4>
                    <div class="space-y-3">
                        <div :class="isMobile ? 'p-3' : 'p-4'" class="bg-neutral-50 dark:bg-neutral-800 rounded-lg">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5 text-neutral-600" />
                                    <div>
                                        <div class="font-medium">消息通知</div>
                                        <div class="text-sm text-neutral-600">接收新消息的推送通知</div>
                                    </div>
                                </div>
                                <USwitch v-model="localSettings.notifications.message_notifications" />
                            </div>
                        </div>

                        <div :class="isMobile ? 'p-3' : 'p-4'" class="bg-neutral-50 dark:bg-neutral-800 rounded-lg">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <UIcon name="i-heroicons-user-group" class="w-5 h-5 text-neutral-600" />
                                    <div>
                                        <div class="font-medium">群组通知</div>
                                        <div class="text-sm text-neutral-600">接收群组活动的推送通知</div>
                                    </div>
                                </div>
                                <USwitch v-model="localSettings.notifications.group_notifications" />
                            </div>
                        </div>

                        <div :class="isMobile ? 'p-3' : 'p-4'" class="bg-neutral-50 dark:bg-neutral-800 rounded-lg">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <UIcon name="i-heroicons-user-plus" class="w-5 h-5 text-neutral-600" />
                                    <div>
                                        <div class="font-medium">好友请求</div>
                                        <div class="text-sm text-neutral-600">接收好友请求的推送通知</div>
                                    </div>
                                </div>
                                <USwitch v-model="localSettings.notifications.friend_requests" />
                            </div>
                        </div>
                    </div>
                </div>

                <USeparator />

                <!-- 应用内通知 -->
                <div class="space-y-4">
                    <h4 :class="isMobile ? 'text-base' : 'text-lg'" class="font-medium text-neutral-900 dark:text-white">应用内通知</h4>
                    <div class="space-y-3">
                        <div :class="isMobile ? 'p-3' : 'p-4'" class="bg-neutral-50 dark:bg-neutral-800 rounded-lg">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <UIcon name="i-heroicons-chat-bubble-oval-left" class="w-5 h-5 text-neutral-600" />
                                    <div>
                                        <div class="font-medium">私聊消息提示</div>
                                        <div class="text-sm text-neutral-600">在应用内显示私聊消息弹窗</div>
                                    </div>
                                </div>
                                <USwitch v-model="localSettings.notifications.direct_message_toast" />
                            </div>
                        </div>

                        <div :class="isMobile ? 'p-3' : 'p-4'" class="bg-neutral-50 dark:bg-neutral-800 rounded-lg">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <UIcon name="i-heroicons-users" class="w-5 h-5 text-neutral-600" />
                                    <div>
                                        <div class="font-medium">群组消息提示</div>
                                        <div class="text-sm text-neutral-600">在应用内显示群组消息弹窗</div>
                                    </div>
                                </div>
                                <USwitch v-model="localSettings.notifications.group_message_toast" />
                            </div>
                        </div>

                        <div :class="isMobile ? 'p-3' : 'p-4'" class="bg-neutral-50 dark:bg-neutral-800 rounded-lg">
                            <div class="flex items-center justify-between">
                                <div class="flex items-center gap-3">
                                    <UIcon name="i-heroicons-exclamation-triangle" class="w-5 h-5 text-neutral-600" />
                                    <div>
                                        <div class="font-medium">系统通知</div>
                                        <div class="text-sm text-neutral-600">显示系统重要通知</div>
                                    </div>
                                </div>
                                <USwitch v-model="localSettings.notifications.system_notification_toast" />
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </UCard>
    </div>

    <!-- 外观设置 -->
    <div v-show="activeSection === 'appearance'" :class="isMobile ? 'space-y-4' : 'space-y-6'">
        <UCard :class="isMobile ? 'shadow-sm' : ''">
            <template #header>
                <div class="flex items-center gap-3">
                    <UIcon name="i-heroicons-paint-brush" class="w-6 h-6 text-primary-600" />
                    <div>
                        <h2 :class="isMobile ? 'text-lg font-semibold' : 'text-xl font-semibold'" class="text-neutral-900 dark:text-white">外观设置</h2>
                        <p class="text-sm text-neutral-600 dark:text-neutral-400">自定义应用的外观和显示偏好</p>
                    </div>
                </div>
            </template>

            <div :class="isMobile ? 'space-y-4' : 'space-y-6'">
                <!-- 主题设置 -->
                <div class="space-y-4">
                    <h4 :class="isMobile ? 'text-base' : 'text-lg'" class="font-medium text-neutral-900 dark:text-white">主题</h4>
                    <div :class="isMobile ? 'space-y-3' : 'grid grid-cols-1 sm:grid-cols-3 gap-4'">
                        <UButton
                            v-for="theme in themeOptions"
                            :key="theme.value"
                            @click="localSettings.preferences.theme = theme.value as 'light' | 'dark' | 'auto'"
                            :variant="localSettings.preferences.theme === theme.value ? 'soft' : 'outline'"
                            :color="localSettings.preferences.theme === theme.value ? 'primary' : 'neutral'"
                            :class="[isMobile ? 'p-3' : 'p-4', 'w-full justify-start']"
                        >
                            <div class="flex items-center gap-3">
                                <UIcon :name="theme.icon" class="w-6 h-6" />
                                <div class="text-left">
                                    <div class="font-medium">{{ theme.label }}</div>
                                    <div class="text-sm text-neutral-600">{{ theme.description }}</div>
                                </div>
                            </div>
                        </UButton>
                    </div>
                </div>

                <USeparator />

                <!-- 语言设置 -->
                <div class="space-y-4">
                    <h4 :class="isMobile ? 'text-base' : 'text-lg'" class="font-medium text-neutral-900 dark:text-white">语言</h4>
                    <UFormField label="界面语言" class="space-y-2">
                        <USelect
                            v-model="localSettings.preferences.language"
                            :options="languageOptions"
                            :class="isMobile ? 'w-full' : 'w-full max-w-xs'"
                        />
                    </UFormField>
                </div>

                <USeparator />

                <!-- 其他偏好 -->
                <div class="space-y-4">
                    <h4 :class="isMobile ? 'text-base' : 'text-lg'" class="font-medium text-neutral-900 dark:text-white">其他偏好</h4>
                    <div :class="isMobile ? 'space-y-4' : 'grid grid-cols-1 md:grid-cols-2 gap-6'">
                        <UFormField label="距离单位" class="space-y-2">
                            <USelect
                                v-model="localSettings.preferences.distance_unit"
                                :options="distanceUnitOptions"
                                class="w-full"
                            />
                        </UFormField>
                        <UFormField label="时间格式" class="space-y-2">
                            <USelect
                                v-model="localSettings.preferences.time_format"
                                :options="timeFormatOptions"
                                class="w-full"
                            />
                        </UFormField>
                    </div>
                </div>
            </div>
        </UCard>
    </div>

    <!-- 关于应用 -->
    <div v-show="activeSection === 'about'" :class="isMobile ? 'space-y-4' : 'space-y-6'">
        <UCard :class="isMobile ? 'shadow-sm' : ''">
            <template #header>
                <div class="flex items-center gap-3">
                    <UIcon name="i-heroicons-information-circle" class="w-6 h-6 text-primary-600" />
                    <div>
                        <h2 :class="isMobile ? 'text-lg font-semibold' : 'text-xl font-semibold'" class="text-neutral-900 dark:text-white">关于应用</h2>
                        <p class="text-sm text-neutral-600 dark:text-neutral-400">应用信息和支持选项</p>
                    </div>
                </div>
            </template>

            <div :class="isMobile ? 'space-y-4' : 'space-y-6'">
                <!-- 应用信息 -->
                <div :class="isMobile ? 'text-center py-6' : 'text-center py-8'">
                    <div :class="isMobile ? 'w-16 h-16' : 'w-20 h-20'" class="bg-primary-100 dark:bg-primary-900/20 rounded-2xl flex items-center justify-center mx-auto mb-4">
                        <UIcon :name="'i-heroicons-map-pin'" :class="isMobile ? 'w-8 h-8' : 'w-10 h-10'" class="text-primary-600" />
                    </div>
                    <h3 :class="isMobile ? 'text-xl' : 'text-2xl'" class="font-bold text-neutral-900 dark:text-white mb-2">HotSpot</h3>
                    <p class="text-neutral-600 dark:text-neutral-400 mb-4">基于位置的社交平台</p>
                    <UBadge label="v1.0.0" color="primary" variant="soft" />
                </div>

                <USeparator />

                <!-- 快捷操作 -->
                <div class="space-y-3">
                    <UButton
                        variant="ghost"
                        block
                        justify="start"
                        @click="$emit('clearCache')"
                        :class="isMobile ? 'h-14' : 'h-12'"
                    >
                        <UIcon name="i-heroicons-trash" class="w-5 h-5 mr-3" />
                        <div class="text-left">
                            <div class="font-medium">清除缓存</div>
                            <div class="text-sm text-neutral-600">清除应用缓存数据</div>
                        </div>
                    </UButton>

                    <UButton
                        variant="ghost"
                        block
                        justify="start"
                        @click="$emit('showHelp')"
                        :class="isMobile ? 'h-14' : 'h-12'"
                    >
                        <UIcon name="i-heroicons-question-mark-circle" class="w-5 h-5 mr-3" />
                        <div class="text-left">
                            <div class="font-medium">帮助与支持</div>
                            <div class="text-sm text-neutral-600">获取使用帮助</div>
                        </div>
                    </UButton>

                    <UButton
                        variant="ghost"
                        block
                        justify="start"
                        @click="$emit('showPrivacyPolicy')"
                        :class="isMobile ? 'h-14' : 'h-12'"
                    >
                        <UIcon name="i-heroicons-document-text" class="w-5 h-5 mr-3" />
                        <div class="text-left">
                            <div class="font-medium">隐私政策</div>
                            <div class="text-sm text-neutral-600">查看隐私政策</div>
                        </div>
                    </UButton>

                    <UButton
                        variant="ghost"
                        block
                        justify="start"
                        @click="$emit('showTerms')"
                        :class="isMobile ? 'h-14' : 'h-12'"
                    >
                        <UIcon name="i-heroicons-document-text" class="w-5 h-5 mr-3" />
                        <div class="text-left">
                            <div class="font-medium">服务条款</div>
                            <div class="text-sm text-neutral-600">查看服务条款</div>
                        </div>
                    </UButton>
                </div>
            </div>
        </UCard>
    </div>

    <!-- 危险操作 -->
    <div v-show="activeSection === 'danger'" :class="isMobile ? 'space-y-4' : 'space-y-6'">
        <UCard :class="isMobile ? 'shadow-sm' : ''">
            <template #header>
                <div class="flex items-center gap-3">
                    <UIcon name="i-heroicons-exclamation-triangle" class="w-6 h-6 text-error-600" />
                    <div>
                        <h2 :class="isMobile ? 'text-lg font-semibold' : 'text-xl font-semibold'" class="text-error-700 dark:text-error-400">危险操作</h2>
                        <p class="text-sm text-error-600 dark:text-error-500">这些操作不可逆转，请谨慎操作</p>
                    </div>
                </div>
            </template>

            <div :class="isMobile ? 'space-y-4' : 'space-y-6'">
                <UAlert
                    color="error"
                    variant="soft"
                    title="警告"
                    description="以下操作将永久删除您的数据，且无法恢复。请确保您已备份重要信息。"
                />

                <div class="space-y-4">
                    <div :class="isMobile ? 'p-4' : 'p-6'" class="border-2 border-error-200 dark:border-error-800 rounded-lg bg-error-50 dark:bg-error-900/10">
                        <div class="flex items-start gap-4">
                            <UIcon name="i-heroicons-trash" class="w-6 h-6 text-error-600 mt-1" />
                            <div class="flex-1">
                                <h4 :class="isMobile ? 'text-base' : 'text-lg'" class="font-medium text-error-900 dark:text-error-100">删除账户</h4>
                                <p class="text-error-700 dark:text-error-300 mt-1">
                                    永久删除您的账户和所有相关数据，包括消息、群组、好友关系等。此操作不可撤销。
                                </p>
                                <UButton
                                    color="error"
                                    variant="solid"
                                    @click="$emit('deleteAccount')"
                                    class="mt-4"
                                    :size="isMobile ? 'md' : 'sm'"
                                >
                                    <UIcon name="i-heroicons-trash" class="w-4 h-4 mr-2" />
                                    删除我的账户
                                </UButton>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </UCard>
    </div>
</template>

<script setup lang="ts">
import type { UserSettings } from '~/types';

interface Props {
    activeSection: string;
    localSettings: UserSettings;
    user: any;
    isMobile?: boolean;
}

defineProps<Props>();

defineEmits<{
    saveNickname: [nickname: string];
    avatarUploaded: [avatarUrl: string];
    upgradeAccount: [];
    changePassword: [];
    setup2FA: [];
    clearCache: [];
    showHelp: [];
    showPrivacyPolicy: [];
    showTerms: [];
    deleteAccount: [];
    editNickname: [];
    changeAvatar: [];
}>();

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
</script>
