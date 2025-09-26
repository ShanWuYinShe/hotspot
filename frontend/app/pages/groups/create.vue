<template>
    <div class="min-h-screen bg-gradient-to-br from-neutral-50 via-white to-neutral-100 dark:from-neutral-900 dark:via-neutral-950 dark:to-neutral-900">
        <div class="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8 py-8 sm:py-12">
        <!-- 面包屑导航 -->
            <div class="mb-8">
            <UBreadcrumb :links="breadcrumbLinks" />
        </div>
        
            <!-- 页面标题 -->
            <div class="text-center mb-12">
                <div class="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-primary-500 to-primary-600 rounded-2xl shadow-lg mb-6">
                    <UIcon name="i-heroicons-user-group" class="w-8 h-8 text-white" />
                </div>
                <h1 class="text-4xl font-bold text-neutral-900 dark:text-white mb-4">创建群组</h1>
                <p class="text-lg text-neutral-600 dark:text-neutral-400 max-w-2xl mx-auto">
                    创建一个新的群组，与附近的朋友分享生活瞬间，开启精彩的社交体验
                </p>
            </div>
            
            <!-- 创建选项卡片 -->
            <div class="grid gap-6 md:grid-cols-2 mb-12">
                <!-- 快速创建 -->
                <UCard 
                    class="cursor-pointer transition-colors duration-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 border-2 border-transparent hover:border-primary-200 dark:hover:border-primary-800 bg-white/80 dark:bg-neutral-800/80 backdrop-blur-sm"
                    @click="quickCreate"
                >
                    <div class="text-center p-6">
                        <div class="inline-flex items-center justify-center w-12 h-12 bg-primary-100 dark:bg-primary-900/50 rounded-xl mb-4 border border-primary-200 dark:border-primary-700">
                            <UIcon name="i-heroicons-bolt" class="w-6 h-6 text-primary-600 dark:text-primary-300" />
                        </div>
                        <h3 class="text-xl font-semibold text-neutral-900 dark:text-white mb-2">快速创建</h3>
                        <p class="text-neutral-600 dark:text-neutral-400 mb-4">
                            使用当前位置和默认设置快速创建群组
                        </p>
                        <UBadge color="primary" variant="soft" size="sm">推荐</UBadge>
                    </div>
                </UCard>
                
                <!-- 自定义创建 -->
                <UCard 
                    class="cursor-pointer transition-colors duration-200 hover:bg-neutral-100 dark:hover:bg-neutral-800 border-2 border-transparent hover:border-primary-200 dark:hover:border-primary-800 bg-white/80 dark:bg-neutral-800/80 backdrop-blur-sm"
                    @click="customCreate"
                >
                    <div class="text-center p-6">
                        <div class="inline-flex items-center justify-center w-12 h-12 bg-purple-100 dark:bg-purple-900/50 rounded-xl mb-4 border border-purple-200 dark:border-purple-700">
                            <UIcon name="i-heroicons-cog-6-tooth" class="w-6 h-6 text-purple-600 dark:text-purple-300" />
                        </div>
                        <h3 class="text-xl font-semibold text-neutral-900 dark:text-white mb-2">自定义创建</h3>
                        <p class="text-neutral-600 dark:text-neutral-400 mb-4">
                            详细配置群组信息、位置和权限设置
                        </p>
                        <UBadge color="secondary" variant="soft" size="sm">高级</UBadge>
                    </div>
                </UCard>
            </div>
            
            <!-- 创建指南 -->
            <UCard class="bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-950/50 dark:to-indigo-950/50 border-blue-200 dark:border-blue-800">
                <div class="p-6">
                    <div class="flex items-start gap-4">
                        <div class="flex-shrink-0">
                            <UIcon name="i-heroicons-light-bulb" class="w-6 h-6 text-blue-600 dark:text-blue-300" />
                        </div>
                        <div>
                            <h3 class="text-lg font-semibold text-blue-900 dark:text-blue-100 mb-2">创建群组小贴士</h3>
                            <ul class="space-y-2 text-blue-800 dark:text-blue-200">
                                <li class="flex items-center gap-2">
                                    <UIcon name="i-heroicons-check-circle" class="w-4 h-4 text-blue-600 dark:text-blue-300" />
                                    <span>选择一个有吸引力的群组名称</span>
                                </li>
                                <li class="flex items-center gap-2">
                                    <UIcon name="i-heroicons-check-circle" class="w-4 h-4 text-blue-600 dark:text-blue-300" />
                                    <span>设置合适的群组范围，方便附近的人发现</span>
                                </li>
                                <li class="flex items-center gap-2">
                                    <UIcon name="i-heroicons-check-circle" class="w-4 h-4 text-blue-600 dark:text-blue-300" />
                                    <span>添加详细的群组描述，说明群组的目的和规则</span>
                                </li>
                                <li class="flex items-center gap-2">
                                    <UIcon name="i-heroicons-check-circle" class="w-4 h-4 text-blue-600 dark:text-blue-300" />
                                    <span>考虑是否需要设置加入密码来筛选成员</span>
                                </li>
                            </ul>
                        </div>
                    </div>
                </div>
            </UCard>
            
            <!-- 最近创建的群组 -->
            <div v-if="recentGroupResponses.length > 0" class="mt-12">
                <h2 class="text-2xl font-bold text-neutral-900 dark:text-white mb-6">您最近创建的群组</h2>
                <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    <UCard 
                        v-for="group in recentGroupResponses" 
                        :key="group.group_id"
                        class="group cursor-pointer transition-colors duration-200"
                        @click="navigateTo(`/groups/${group.group_id}`)"
                    >
                        <div class="p-4">
                            <div class="flex items-center gap-3 mb-3">
                                <UAvatar 
                                    :src="group.avatar_url" 
                                    :alt="group.name"
                                    size="sm"
                                    class="bg-primary-500 dark:bg-primary-400"
                                >
                                    <UIcon name="i-heroicons-user-group" class="w-4 h-4" />
                                </UAvatar>
                                <div class="flex-1 min-w-0">
                                    <h3 class="font-semibold text-neutral-900 dark:text-white truncate">{{ group.name }}</h3>
                                    <p class="text-sm text-neutral-500 dark:text-neutral-400">{{ group.member_count }} 成员</p>
                                </div>
                            </div>
                            <p class="text-sm text-neutral-600 dark:text-neutral-400 line-clamp-2">{{ group.description }}</p>
            </div>
        </UCard>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import type { GroupResponse } from '~/types/api';

definePageMeta({ title: "创建群组", requiresAuth: true, layout: 'dashboard' });

        // 移除已删除的 modal 函数
const groupsStore = useGroupsStore();
const toast = useSmartToast();
const router = useRouter();

// 面包屑导航链接
const breadcrumbLinks = [
    {
        label: '首页',
        icon: 'i-heroicons-home',
        to: '/'
    },
    {
        label: '群组',
        icon: 'i-heroicons-user-group',
        to: '/groups'
    },
    {
        label: '创建群组',
        icon: 'i-heroicons-plus'
    }
];

// 最近创建的群组
const recentGroupResponses = ref<GroupResponse[]>([]);

// 快速创建群组
const quickCreate = async () => {
    try {
        const coordinates = await useGeolocation().getCurrentPosition();
        
        // 暂时移除快速创建功能
        toast.add({
            title: '功能开发中',
            description: '快速创建功能正在开发中',
            color: 'info'
        });
    } catch (error) {
        toast.add({
            title: '无法获取位置',
            description: '请手动设置群组位置',
            color: 'warning'
        });
        customCreate();
    }
};

// 自定义创建群组
const customCreate = () => {
    // 暂时移除自定义创建功能
    toast.add({
        title: '功能开发中',
        description: '自定义创建功能正在开发中',
        color: 'info'
    });
};

// 加载最近创建的群组
const loadRecentGroupResponses = async () => {
    try {
        // 这里应该调用API获取用户最近创建的群组
        // const groups = await groupsStore.fetchOwnedGroupResponses({ limit: 6 });
        // recentGroupResponses.value = groups;
        
        // 临时使用空数组，等待后端API实现
        recentGroupResponses.value = [];
    } catch (error) {
        console.error('加载最近群组失败:', error);
    }
};

onMounted(() => {
    loadRecentGroupResponses();
});
</script>
