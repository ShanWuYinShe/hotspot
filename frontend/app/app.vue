<template>
    <UApp>
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
    const authStore = useAuthStore();
    const adminStore = useAdminStore();
    const { initializeSubscriptions, cleanup } = useNotifications();

    // 位置同步服务 - 自动监听登录状态，登录后自动同步位置
    useLocationSync({
        syncInterval: 5 * 60 * 1000, // 每5分钟同步一次
        enableWatch: false,
    });

    // 检查管理员权限（登录后主动检查，避免只在访问 /admin 路由时才检查）
    const checkAdminPermission = async () => {
        if (!authStore.isAuthenticated) {
            adminStore.setAccess(false);
            return;
        }
        // 如果已经检查过且权限有效，跳过
        if (adminStore.hasAccess) return;

        try {
            const { useAdminRbacService } = await import(
                "~/services/admin_rbac.service"
            );
            const rbac = useAdminRbacService();
            const me = await rbac.me();
            const permissions = me.permissions || [];
            adminStore.setAccess(true, { permissions });
        } catch {
            // 不是管理员或 API 调用失败，设置为 false
            adminStore.setAccess(false, { permissions: [] });
        }
    };

    // 监听登录状态，初始化或清理通知订阅
    watch(
        () => authStore.isAuthenticated,
        (isAuthenticated) => {
            if (isAuthenticated) {
                initializeSubscriptions();
                // 登录后主动检查管理员权限
                void checkAdminPermission();
            } else {
                cleanup();
                adminStore.setAccess(false);
            }
        },
        { immediate: true },
    );
});
</script>
