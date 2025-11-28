<template>
    <UApp 
        :toaster="{
            position: 'bottom-right',
            expand: true,
            progress: true,
            duration: 5000
        }"
    >
        <NuxtLayout>
            <NuxtPage />
        </NuxtLayout>
    </UApp>
</template>

<script setup lang="ts">
// 设置应用头部信息
useSeoMeta({
    title: "HotSpot - 基于位置的社交平台",
    description: "发现身边有趣的人和事",
});

// 客户端挂载后启用 auth store 的显示
// 使用 nextTick 确保在水合完成后再改变状态
// 这确保 SSR 和客户端初始渲染一致（都是 null），避免水合不匹配
onMounted(() => {
    nextTick(() => {
        const authStore = useAuthStore()
        authStore.hydrate()
    })
})
</script>

<style>
html { scroll-behavior: smooth; }
</style>
