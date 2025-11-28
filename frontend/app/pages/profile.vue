<template>
    <PageLayout
        title="个人资料"
        description="管理您的个人信息和账户设置"
        :stats="profileStats"
        :header-actions="headerActions"
    >
        <UserProfileEditor
            :profile="userProfile"
            @save="handleSave"
            @avatar-change="handleAvatarChange"
        />
    </PageLayout>
</template>

<script setup lang="ts">
import type { StatItem } from '~/components/data/StatsGrid.vue'

// 需要登录 - 个人资料需要认证
definePageMeta({
    title: '个人资料',
    description: '管理您的个人信息和账户设置',
    requiresAuth: true,
    layout: 'dashboard'
})

// 导入服务
import { useUsersService } from '~/services/users.service';

// 服务和状态
const authStore = useAuthStore()
const userService = useUsersService();
const toast = useToast()

// 用户资料数据（根据 API 文档 3.2.2 MyProfileData）
const userProfile = ref({
  user_id: '',
  nickname: '',
  bio: '',
  avatar_url: '',
  email: '',
  phone: '',
  location_name: '',
  is_online: false,
  last_seen: '',
  created_at: ''
})

// 统计数据
const profileStats = computed<StatItem[]>(() => [
    {
        key: 'profile-completion',
        label: '资料完整度',
        value: calculateProfileCompletion(),
        icon: 'i-heroicons-user',
        color: 'blue',
        description: '完善资料可获得更多关注',
        onClick: () => toast.add({ color: 'info', title: '完善更多资料信息' })
    },
    {
        key: 'privacy-score',
        label: '隐私安全',
        value: calculatePrivacyScore(),
        icon: 'i-heroicons-shield-check',
        color: 'green',
        description: '您的隐私设置安全等级',
        onClick: () => toast.add({ color: 'info', title: '查看隐私设置详情' })
    },
    {
        key: 'last-updated',
        label: '最后更新',
        value: '2天前',
        icon: 'i-heroicons-clock',
        color: 'orange',
        description: '资料最后更新时间',
        onClick: () => toast.add({ color: 'info', title: '查看更新历史' })
    },
    {
        key: 'profile-views',
        label: '资料浏览',
        value: 156,
        icon: 'i-heroicons-eye',
        color: 'purple',
        description: '最近30天的资料浏览次数',
        onClick: () => toast.add({ color: 'info', title: '查看浏览详情' })
    }
])

// 头部操作
const headerActions = computed(() => [
    {
        key: 'preview',
        label: '预览资料',
        icon: 'i-heroicons-eye',
        variant: 'outline' as const,
        action: previewProfile
    },
    {
        key: 'export',
        label: '导出数据',
        icon: 'i-heroicons-document-arrow-down',
        variant: 'outline' as const,
        action: exportProfileData
    }
])

// 方法
const calculateProfileCompletion = () => {
    const fields = [
        'nickname', 'bio', 'avatar_url', 'email', 'phone', 'location_name'
    ]
    
    const completedFields = fields.filter(field => {
        const value = userProfile.value[field as keyof typeof userProfile.value]
        return value && value.toString().trim() !== ''
    })
    
    return Math.round((completedFields.length / fields.length) * 100)
}

const calculatePrivacyScore = () => {
    // 隐私设置应该从 /user/get_settings API 获取
    // 这里返回一个固定值或从 settingsStore 获取
    return 85 // 默认值
}

const handleSave = async (profile: any) => {
    try {
        // 调用 API 保存用户资料（API 3.2.7 update_profile）
        await userService.updateProfile({
            nickname: profile.nickname,
            avatar_url: profile.avatar_url,
            bio: profile.bio
        })
        
        userProfile.value = { ...userProfile.value, ...profile }
        toast.add({ color: 'success', title: '个人资料保存成功' })
        
        // 更新认证状态中的用户信息
        if (authStore.user) {
            authStore.user.nickname = profile.nickname
            authStore.user.avatar_url = profile.avatar_url
        }
    } catch (error) {
        toast.add({ color: 'error', title: '保存失败，请重试' })
    }
}

const handleAvatarChange = (url: string) => {
    userProfile.value.avatar_url = url
    
    // 实时更新认证状态中的头像
    if (authStore.user) {
        authStore.user.avatar_url = url
    }
}

const previewProfile = () => {
    // 在新窗口中预览用户资料
    const previewUrl = `/users/${userProfile.value.user_id}`
    window.open(previewUrl, '_blank')
}

const exportProfileData = () => {
    // 创建导出数据
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
    
    // 创建下载链接
    const dataStr = JSON.stringify(exportData, null, 2)
    const dataBlob = new Blob([dataStr], { type: 'application/json' })
    const url = URL.createObjectURL(dataBlob)
    
    // 下载文件
    const link = document.createElement('a')
    link.href = url
    link.download = `my-profile-${new Date().toISOString().split('T')[0]}.json`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
    
    toast.add({ color: 'success', title: '个人资料已导出' })
}

// 页面加载时获取用户资料
onMounted(async () => {
    try {
        // 调用 API 获取完整的用户资料（API 3.2.2 me）
        const profile = await userService.getMe();
        userProfile.value = {
            user_id: profile.user_id,
            nickname: profile.nickname,
            bio: profile.bio || '',
            avatar_url: profile.avatar_url || '',
            email: profile.email || '',
            phone: profile.phone || '',
            location_name: profile.location_name || '',
            is_online: profile.is_online,
            last_seen: profile.last_seen || '',
            created_at: profile.created_at
        };
    } catch (error) {
        toast.add({ color: 'error', title: '加载用户资料失败' })
    }
})
</script>