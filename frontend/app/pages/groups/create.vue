<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-14 sm:py-16">
        <!-- 面包屑导航 -->
        <div class="mb-6">
            <UBreadcrumb :links="breadcrumbLinks" />
        </div>
        
        <UCard>
            <template #header>
                <h1 class="text-3xl font-bold text-neutral-900">创建群组</h1>
            </template>
            <div class="py-6">
                <CreateGroupModal v-model:open="open" @success="handleSuccess">
                    <UButton>打开创建弹窗</UButton>
                </CreateGroupModal>
            </div>
        </UCard>
    </div>
</template>

<script setup lang="ts">
definePageMeta({ title: "创建群组", requiresAuth: true });
import CreateGroupModal from "~/components/CreateGroupModal.vue";

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

const open = ref(true);
const handleSuccess = (g: any) => {
    navigateTo(`/chat/group/${g.group_id}`);
};
</script>
