<template>
    <UApp 
        :toaster="{
            position: 'bottom-right',
            expand: true,
            progress: true,
            duration: 5000,
            class: 'toast-smart-position'
        }"
    >
        <NuxtLayout>
            <NuxtPage />
        </NuxtLayout>
        
        <!-- 全局 PIN 输入模态框 -->
        <GlobalPinModal />
    </UApp>
</template>

<script setup lang="ts">
// 设置应用头部信息
useSeoMeta({
    title: "HotSpot - 基于位置的社交平台",
    description: "发现身边有趣的人和事",
});
</script>
