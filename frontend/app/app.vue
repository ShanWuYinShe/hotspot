<template>
    <UApp 
        :toaster="{
            position: 'bottom-right',
            expand: true,
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

// 客户端初始化
if (import.meta.client) {
    const authStore = useAuthStore()
    const { initializeSubscriptions, cleanup } = useNotifications()
    
    // 位置同步服务 - 自动监听登录状态，登录后自动同步位置
    useLocationSync({
        syncInterval: 5 * 60 * 1000, // 每5分钟同步一次
        distanceThreshold: 100, // 位置变化超过100米才上传
        enableWatch: true // 启用实时位置监听
    })
    
    // 监听登录状态，初始化或清理通知订阅
    watch(() => authStore.isAuthenticated, (isAuthenticated) => {
        if (isAuthenticated) {
            // 延迟初始化，确保 WebSocket 连接已建立
            setTimeout(() => {
                initializeSubscriptions()
            }, 1000)
        } else {
            cleanup()
        }
    }, { immediate: true })
}
</script>

<style>
html { scroll-behavior: smooth; }
</style>
