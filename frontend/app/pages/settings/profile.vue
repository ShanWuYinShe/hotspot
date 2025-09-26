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
import type { StatItem } from '~/components/StatsGrid.vue'

definePageMeta({
    title: '个人资料',
    description: '管理您的个人信息和账户设置',
    layout: 'dashboard'
})

// 服务和状态
const authStore = useAuthStore()
const toast = useSmartToast()

// 用户资料数据
const userProfile = ref({
    user_id: '1',
    nickname: '张三',
    display_name: '张三同学',
    bio: '热爱技术，喜欢分享',
    email: 'zhangsan@example.com',
    phone: '13800138000',
    avatar_url: 'https://api.dicebear.com/7.x/avataaars/svg?seed=zhang',
    birthday: '1990-01-01',
    gender: 'male',
    location: '北京市',
    website: 'https://zhangsan.dev',
    allow_stranger_view: true,
    show_online_status: true,
    searchable_by_phone: false,
    location_sharing_enabled: true,
    precise_location: false,
    two_factor_enabled: false
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
        onClick: () => toast.info('完善更多资料信息')
    },
    {
        key: 'privacy-score',
        label: '隐私安全',
        value: calculatePrivacyScore(),
        icon: 'i-heroicons-shield-check',
        color: 'green',
        description: '您的隐私设置安全等级',
        onClick: () => toast.info('查看隐私设置详情')
    },
    {
        key: 'last-updated',
        label: '最后更新',
        value: '2天前',
        icon: 'i-heroicons-clock',
        color: 'orange',
        description: '资料最后更新时间',
        onClick: () => toast.info('查看更新历史')
    },
    {
        key: 'profile-views',
        label: '资料浏览',
        value: 156,
        icon: 'i-heroicons-eye',
        color: 'purple',
        description: '最近30天的资料浏览次数',
        onClick: () => toast.info('查看浏览详情')
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
        'nickname', 'display_name', 'bio', 'email', 'phone', 
        'avatar_url', 'birthday', 'gender', 'location', 'website'
    ]
    
    const completedFields = fields.filter(field => {
        const value = userProfile.value[field as keyof typeof userProfile.value]
        return value && value.toString().trim() !== ''
    })
    
    return Math.round((completedFields.length / fields.length) * 100)
}

const calculatePrivacyScore = () => {
    let score = 0
    
    // 基础隐私设置
    if (!userProfile.value.allow_stranger_view) score += 20
    if (!userProfile.value.show_online_status) score += 15
    if (!userProfile.value.searchable_by_phone) score += 20
    if (!userProfile.value.location_sharing_enabled) score += 25
    if (!userProfile.value.precise_location) score += 10
    if (userProfile.value.two_factor_enabled) score += 10
    
    return Math.min(score, 100)
}

const handleSave = async (profile: any) => {
    try {
        // 调用API保存用户资料
        await new Promise(resolve => setTimeout(resolve, 1000))
        
        userProfile.value = { ...profile }
        toast.success('个人资料保存成功')
        
        // 更新认证状态中的用户信息
        if (authStore.user) {
            authStore.user.nickname = profile.nickname
            authStore.user.avatar_url = profile.avatar_url
        }
    } catch (error) {
        toast.error('保存失败，请重试')
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
    const { exportData } = useDataExport()
    
    const exportColumns = [
        { key: 'nickname', label: '昵称' },
        { key: 'display_name', label: '显示名称' },
        { key: 'bio', label: '个人简介' },
        { key: 'email', label: '邮箱' },
        { key: 'phone', label: '手机号' },
        { key: 'birthday', label: '生日' },
        { key: 'gender', label: '性别', formatter: (v: string) => v === 'male' ? '男' : v === 'female' ? '女' : '保密' },
        { key: 'location', label: '所在地' },
        { key: 'website', label: '个人网站' }
    ]
    
    exportData([userProfile.value], exportColumns, {
        filename: 'my-profile',
        format: 'json'
    })
}

// 页面加载时获取用户资料
onMounted(async () => {
    try {
        // 从认证状态中获取基础信息
        if (authStore.user) {
            userProfile.value.user_id = authStore.user.user_id
            userProfile.value.nickname = authStore.user.nickname || userProfile.value.nickname
            userProfile.value.email = authStore.user.email || userProfile.value.email
            userProfile.value.avatar_url = authStore.user.avatar_url || userProfile.value.avatar_url
        }
        
        // 这里可以调用API获取完整的用户资料
        // const fullProfile = await userService.getProfile()
        // userProfile.value = { ...userProfile.value, ...fullProfile }
    } catch (error) {
        toast.error('加载用户资料失败')
    }
})
</script>