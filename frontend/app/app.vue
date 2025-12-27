<template>
    <!-- 禁用 Nuxt UI 内置 toaster：它内部使用 portal/ClientOnly，容易触发 SSR 水合不匹配。 -->
    <UApp :toaster="null">
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

// 客户端副作用统一延后到 mounted：
// 目标：SSR 首屏 HTML 与 CSR 首次 VDOM 完全一致，避免 hydration mismatch
onMounted(() => {
    const authStore = useAuthStore()
    const { initializeSubscriptions, cleanup } = useNotifications()

    // 位置同步服务 - 自动监听登录状态，登录后自动同步位置
    useLocationSync({
        syncInterval: 5 * 60 * 1000, // 每5分钟同步一次
        enableWatch: false
    })

    // 监听登录状态，初始化或清理通知订阅
    watch(() => authStore.isAuthenticated, (isAuthenticated) => {
        if (isAuthenticated) {
            initializeSubscriptions()
        } else {
            cleanup()
        }
    }, { immediate: true })
})
</script>
