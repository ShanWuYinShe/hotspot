<template>
    <div>
        <!-- 页面标题 -->
        <UPageHeader
            title="个人资料"
            description="管理您的个人信息"
        >
            <template #actions>
                <UButton
                    variant="ghost"
                    color="neutral"
                    icon="i-heroicons-eye"
                    @click="previewProfile"
                >
                    预览
                </UButton>
            </template>
        </UPageHeader>

        <!-- 加载状态 -->
        <div v-if="loading" class="grid grid-cols-1 lg:grid-cols-3 gap-8">
            <div class="lg:col-span-1 space-y-6">
                <div class="bg-card rounded-2xl shadow-sm border border-default p-6">
                    <div class="flex flex-col items-center">
                        <USkeleton class="w-24 h-24 rounded-full mb-4" />
                        <USkeleton class="h-6 w-32 mb-2" />
                        <USkeleton class="h-4 w-24 mb-3" />
                        <USkeleton class="h-6 w-16" />
                    </div>
                    <USeparator class="my-6" />
                    <div class="grid grid-cols-3 gap-4">
                        <USkeleton class="h-16" />
                        <USkeleton class="h-16" />
                        <USkeleton class="h-16" />
                    </div>
                </div>
                <USkeleton class="h-32 rounded-2xl" />
                <USkeleton class="h-28 rounded-2xl" />
            </div>
            <div class="lg:col-span-2 space-y-6">
                <USkeleton class="h-96 rounded-2xl" />
                <USkeleton class="h-32 rounded-2xl" />
            </div>
        </div>

        <!-- 主要内容区域 -->
        <div v-else class="grid grid-cols-1 lg:grid-cols-3 gap-8">
                <!-- 左侧：头像和预览卡片 -->
                <div class="lg:col-span-1 space-y-6">
                    <!-- 头像卡片 -->
                    <div class="bg-card rounded-2xl shadow-sm border border-default p-6">
                        <div class="flex flex-col items-center">
                            <!-- 头像 -->
                            <div class="relative group mb-4">
                                <UAvatar
                                    :src="userProfile.avatar_url"
                                    :alt="userProfile.nickname"
                                    size="3xl"
                                    class="ring-4 ring-default shadow-lg"
                                >
                                    {{ userProfile.nickname?.charAt(0) || 'U' }}
                                </UAvatar>
                                <UFileUpload
                                    v-slot="{ open }"
                                    v-model="selectedAvatarFile"
                                    accept="image/*"
                                    class="absolute inset-0"
                                    @update:model-value="handleAvatarSelected"
                                >
                                    <div
                                        class="absolute inset-0 flex items-center justify-center bg-black/50 rounded-full opacity-0 group-hover:opacity-100 cursor-pointer transition-opacity"
                                        @click="open()"
                                    >
                                        <UIcon name="i-heroicons-camera" class="w-8 h-8 text-white" />
                                    </div>
                                </UFileUpload>
                            </div>
                            
                            <!-- 用户名和ID -->
                            <h2 class="text-xl font-bold text-primary-content">
                                {{ userProfile.nickname || '未设置昵称' }}
                            </h2>
                            <p class="text-sm text-secondary-content mt-1">
                                ID: {{ userProfile.user_id }}
                            </p>
                            
                            <!-- 账户类型标签 -->
                            <UBadge 
                                :color="authStore.user?.user_type === 'normal' ? 'success' : 'warning'"
                                variant="soft"
                                class="mt-3"
                            >
                                {{ authStore.user?.user_type === 'normal' ? '正式用户' : '临时用户' }}
                            </UBadge>
                        </div>
                        
                        <!-- 分隔线 -->
                        <USeparator class="my-6" />
                        
                        <!-- 快速统计 -->
                        <div class="grid grid-cols-3 gap-4 text-center">
                            <div>
                                <p class="text-2xl font-bold text-primary-content">{{ stats.momentsCount }}</p>
                                <p class="text-xs text-secondary-content">动态</p>
                            </div>
                            <div>
                                <p class="text-2xl font-bold text-primary-content">{{ stats.friendsCount }}</p>
                                <p class="text-xs text-secondary-content">好友</p>
                            </div>
                            <div>
                                <p class="text-2xl font-bold text-primary-content">{{ stats.groupsCount }}</p>
                                <p class="text-xs text-secondary-content">群组</p>
                            </div>
                        </div>
                    </div>
                    
                    <!-- 资料完整度卡片 -->
                    <div class="bg-card rounded-2xl shadow-sm border border-default p-6">
                        <div class="flex items-center justify-between mb-4">
                            <h3 class="font-semibold text-primary-content">资料完整度</h3>
                            <span class="text-2xl font-bold text-primary">{{ profileCompletion }}%</span>
                        </div>
                        <UProgress :value="profileCompletion" color="primary" size="md" />
                        <p class="text-sm text-secondary-content mt-3">
                            完善资料可以让更多人发现你
                        </p>
                    </div>
                    
                    <!-- 账户信息卡片 -->
                    <div class="bg-card rounded-2xl shadow-sm border border-default p-6">
                        <h3 class="font-semibold text-primary-content mb-4">账户信息</h3>
                        <div class="space-y-4">
                            <div class="flex items-center gap-3">
                                <div class="p-2 bg-interactive rounded-lg">
                                    <UIcon name="i-heroicons-calendar" class="w-4 h-4 text-secondary-content" />
                                </div>
                                <div>
                                    <p class="text-xs text-secondary-content">注册时间</p>
                                    <p class="text-sm font-medium text-primary-content">
                                        {{ userProfile.created_at ? formatDate(userProfile.created_at) : '未知' }}
                                    </p>
                                </div>
                            </div>
                            <div class="flex items-center gap-3">
                                <div class="p-2 bg-interactive rounded-lg">
                                    <UIcon name="i-heroicons-map-pin" class="w-4 h-4 text-secondary-content" />
                                </div>
                                <div>
                                    <p class="text-xs text-secondary-content">当前位置</p>
                                    <p class="text-sm font-medium text-primary-content">
                                        {{ userProfile.location_name || '未设置' }}
                                    </p>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 右侧：编辑表单 -->
                <div class="lg:col-span-2 space-y-6">
                    <!-- 基本信息表单 -->
                    <div class="bg-card rounded-2xl shadow-sm border border-default p-6">
                        <div class="flex items-center gap-3 mb-6">
                            <div class="p-2 bg-primary-subtle rounded-lg">
                                <UIcon name="i-heroicons-user-circle" class="w-5 h-5 text-primary" />
                            </div>
                            <div>
                                <h3 class="font-semibold text-primary-content">基本信息</h3>
                                <p class="text-sm text-secondary-content">修改你的公开个人信息</p>
                            </div>
                        </div>
                        
                        <UForm :state="formState" class="space-y-6" @submit="handleSave">
                            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                                <UFormField label="昵称" name="nickname" required>
                                    <UInput 
                                        v-model="formState.nickname" 
                                        placeholder="请输入昵称"
                                        icon="i-heroicons-user"
                                        size="lg"
                                    />
                                </UFormField>

                                <UFormField label="邮箱" name="email">
                                    <UInput 
                                        v-model="formState.email" 
                                        type="email"
                                        placeholder="请输入邮箱"
                                        icon="i-heroicons-envelope"
                                        size="lg"
                                        disabled
                                    />
                                    <template #hint>
                                        <span class="text-muted">暂不支持修改</span>
                                    </template>
                                </UFormField>
                            </div>

                            <UFormField label="个人简介" name="bio">
                                <UTextarea 
                                    v-model="formState.bio" 
                                    placeholder="介绍一下自己，让别人更了解你..."
                                    :rows="4"
                                    size="lg"
                                />
                                <template #hint>
                                    <span class="text-muted">{{ formState.bio?.length || 0 }}/200</span>
                                </template>
                            </UFormField>

                            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                                <UFormField label="手机号" name="phone">
                                    <UInput 
                                        v-model="formState.phone" 
                                        placeholder="请输入手机号"
                                        icon="i-heroicons-phone"
                                        size="lg"
                                        disabled
                                    />
                                    <template #hint>
                                        <span class="text-muted">暂不支持修改</span>
                                    </template>
                                </UFormField>

                                <UFormField label="位置" name="location">
                                    <UInput 
                                        v-model="formState.location_name" 
                                        placeholder="当前位置"
                                        icon="i-heroicons-map-pin"
                                        size="lg"
                                        disabled
                                    />
                                    <template #hint>
                                        <span class="text-muted">位置会自动更新</span>
                                    </template>
                                </UFormField>
                            </div>

                            <!-- 操作按钮 -->
                            <USeparator class="mt-6 mb-4" />
                            <div class="flex items-center justify-end gap-3">
                                <UButton 
                                    type="button"
                                    variant="ghost" 
                                    color="neutral"
                                    size="lg"
                                    @click="resetForm"
                                >
                                    重置
                                </UButton>
                                <UButton 
                                    type="submit"
                                    size="lg"
                                    :loading="saving"
                                    :disabled="!hasChanges"
                                >
                                    保存更改
                                </UButton>
                            </div>
                        </UForm>
                    </div>
                    
                    <!-- 临时用户升级提示 -->
                    <div 
                        v-if="authStore.user?.user_type === 'temp'"
                        class="bg-gradient-primary-card rounded-2xl shadow-lg p-6 border border-primary-subtle"
                    >
                        <div class="flex items-start gap-4">
                            <div class="p-3 bg-primary-medium rounded-xl">
                                <UIcon name="i-heroicons-sparkles" class="w-6 h-6 text-accent-strong" />
                            </div>
                            <div class="flex-1">
                                <h3 class="text-lg font-bold mb-2 text-primary-content">升级为正式用户</h3>
                                <p class="text-strong text-sm mb-4">
                                    升级账户后可以享受更多功能，包括创建群组、发布更多动态等。
                                </p>
                                <UButton 
                                    color="white" 
                                    variant="solid"
                                    @click="navigateTo('/settings/upgrade')"
                                >
                                    立即升级
                                </UButton>
                            </div>
                        </div>
                    </div>
                    
                    <!-- 危险区域 -->
                    <div class="bg-card rounded-2xl shadow-sm border border-red p-6">
                        <div class="flex items-center gap-3 mb-4">
                            <div class="p-2 bg-red-subtle rounded-lg">
                                <UIcon name="i-heroicons-exclamation-triangle" class="w-5 h-5 text-error" />
                            </div>
                            <div>
                                <h3 class="font-semibold text-error">危险区域</h3>
                                <p class="text-sm text-secondary-content">这些操作不可逆，请谨慎操作</p>
                            </div>
                        </div>
                        
                        <div class="flex flex-wrap gap-3">
                            <UButton 
                                variant="outline" 
                                color="error"
                                icon="i-heroicons-document-arrow-down"
                                @click="exportProfileData"
                            >
                                导出数据
                            </UButton>
                            <UButton 
                                variant="outline" 
                                color="error"
                                icon="i-heroicons-trash"
                                @click="handleDeleteAccount"
                            >
                                删除账户
                            </UButton>
                        </div>
                    </div>
                </div>
            </div>
        </div>
</template>

<script setup lang="ts">
definePageMeta({
    title: '个人资料',
    description: '管理您的个人信息和账户设置',
    requiresAuth: true,
    layout: 'dashboard'
})

import { useUsersService } from '~/services/users.service'

const authStore = useAuthStore()
const userService = useUsersService()
const toast = useToast()
const modal = useModal()
const { formatDate } = useFormatters()

// 原始用户资料
const userProfile = ref({
    user_id: 0 as number,
    nickname: '',
    bio: '',
    avatar_url: '',
    email: '',
    phone: '',
    location_name: '',
    is_online: false,
    last_online: '',
    created_at: ''
})

// 用户统计
const stats = ref({
    momentsCount: 0,
    friendsCount: 0,
    groupsCount: 0
})

// 表单状态
const formState = ref({
    nickname: '',
    bio: '',
    email: '',
    phone: '',
    location_name: '',
    avatar_file_id: ''
})

const saving = ref(false)
const loading = ref(true)
// 头像文件 (UFileUpload)
const selectedAvatarFile = ref<File | null>(null)

// 检测是否有更改
const hasChanges = computed(() => {
    return (
        formState.value.nickname !== userProfile.value.nickname ||
        formState.value.bio !== (userProfile.value.bio || '') ||
        formState.value.avatar_file_id !== ''
    )
})

// 资料完整度
const profileCompletion = computed(() => {
    const fields = ['nickname', 'bio', 'avatar_url', 'email', 'phone', 'location_name']
    const completedFields = fields.filter(field => {
        const value = userProfile.value[field as keyof typeof userProfile.value]
        return value && value.toString().trim() !== ''
    })
    return Math.round((completedFields.length / fields.length) * 100)
})

// 头像更改处理 (UFileUpload 回调)
const handleAvatarSelected = async (file: File | null | undefined) => {
    if (!file) return

    // 简单预览
    const reader = new FileReader()
    reader.onload = (e) => {
        userProfile.value.avatar_url = e.target?.result as string
    }
    reader.readAsDataURL(file)

    // TODO: 上传头像到服务器
    toast.add({ title: '头像已更新', color: 'success' })
    
    // 重置选择
    selectedAvatarFile.value = null
}

// 保存表单
const handleSave = async () => {
    saving.value = true
    try {
        await userService.updateProfile({
            nickname: formState.value.nickname || undefined,
            avatar_file_id: formState.value.avatar_file_id || undefined,
            bio: formState.value.bio || undefined
        })
        
        userProfile.value.nickname = formState.value.nickname
        userProfile.value.bio = formState.value.bio
        formState.value.avatar_file_id = ''
        
        if (authStore.user) {
            authStore.user.nickname = formState.value.nickname
        }
        
        toast.add({ color: 'success', title: '个人资料保存成功' })
    } catch {
        toast.add({ color: 'error', title: '保存失败，请重试' })
    } finally {
        saving.value = false
    }
}

// 重置表单
const resetForm = () => {
    formState.value = {
        nickname: userProfile.value.nickname,
        bio: userProfile.value.bio || '',
        email: userProfile.value.email || '',
        phone: userProfile.value.phone || '',
        location_name: userProfile.value.location_name || '',
        avatar_file_id: ''
    }
}

// 预览资料
const previewProfile = () => {
    window.open(`/users/${userProfile.value.user_id}`, '_blank')
}

// 导出数据
const exportProfileData = () => {
    const exportData = {
        user_id: userProfile.value.user_id,
        nickname: userProfile.value.nickname,
        bio: userProfile.value.bio,
        email: userProfile.value.email,
        phone: userProfile.value.phone,
        avatar_url: userProfile.value.avatar_url,
        location_name: userProfile.value.location_name,
        created_at: userProfile.value.created_at,
        export_time: new Date().toISOString()
    }
    
    const dataStr = JSON.stringify(exportData, null, 2)
    const dataBlob = new Blob([dataStr], { type: 'application/json' })
    const url = URL.createObjectURL(dataBlob)
    
    const link = document.createElement('a')
    link.href = url
    link.download = `my-profile-${new Date().toISOString().split('T')[0]}.json`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
    
    toast.add({ color: 'success', title: '个人资料已导出' })
}

// 删除账户
const handleDeleteAccount = async () => {
    const confirmed = await modal.showConfirm({
        title: '删除账户',
        message: '此操作将永久删除您的账户和所有数据，且无法恢复。确定要继续吗？',
        confirmText: '确认删除',
        cancelText: '取消',
        confirmColor: 'error'
    })
    
    if (confirmed) {
        // TODO: 实现删除账户逻辑
        toast.add({ title: '账户删除功能开发中', color: 'warning' })
    }
}

// 加载用户统计
const loadStats = async () => {
    try {
        if (userProfile.value.user_id) {
            const data = await userService.getStats({ id: userProfile.value.user_id })
            stats.value = {
                momentsCount: data.moments_count,
                friendsCount: data.friends_count,
                groupsCount: data.groups_count
            }
        }
    } catch {
        // 忽略统计加载失败
    }
}

// 页面加载时获取用户资料
onMounted(async () => {
    loading.value = true
    try {
        const profile = await userService.getMe()
        userProfile.value = {
            user_id: profile.user_id,
            nickname: profile.nickname,
            bio: profile.bio || '',
            avatar_url: profile.avatar_url || '',
            email: profile.email || '',
            phone: profile.phone || '',
            location_name: profile.location_name || '',
            is_online: profile.is_online,
            last_online: profile.last_online || '',
            created_at: profile.created_at
        }
        
        resetForm()
        // 异步加载统计数据，不阻塞页面渲染
        loadStats()
    } catch {
        toast.add({ color: 'error', title: '加载用户资料失败' })
    } finally {
        loading.value = false
    }
})
</script>
