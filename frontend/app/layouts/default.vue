<template>
    <div
        class="antialiased font-sans text-neutral-900 dark:text-neutral-100 overflow-x-hidden"
        :class="$ui.ui.pageLayouts.authBackground"
    >
        <!-- 统一导航栏组件（含桌面顶部与移动底部导航） -->
            <nav>
        <AppNavigation />
    </nav>

        <!-- 网络状态指示器 -->
        <UAlert
            v-if="showNetworkWarning"
            icon="i-heroicons-exclamation-triangle"
            color="warning"
            variant="solid"
            title="网络连接异常"
            description="部分功能可能不可用"
            class="fixed top-0 left-0 right-0 z-50 rounded-none border-0"
        />

        <!-- 主内容区域 -->
        <main
            class="flex-1 relative pb-20 md:pb-0"
            :class="{ 'pt-8': showNetworkWarning }"
        >
            <slot />
        </main>

        <!-- 快捷键帮助组件 -->
        <ShortcutHelp />
        
        <!-- 认证调试器（仅开发环境） -->
        <AuthDebugger v-if="isDev" />
    </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import AppNavigation from "~/components/AppNavigation.vue";

// 导入组件
const ShortcutHelp = defineAsyncComponent(() => import('~/components/ShortcutHelp.vue'));
const AuthDebugger = defineAsyncComponent(() => import('~/components/AuthDebugger.vue'));

// 开发环境检查
const isDev = process.env.NODE_ENV === 'development';

const systemStatus = useSystemStatus();

// 网络警告显示逻辑
const showNetworkWarning = computed(() => {
    const wsStatus = systemStatus.getWebSocketStatus();
    const apiStatus = systemStatus.getApiStatus();

    return (
        (wsStatus.status === "error" || apiStatus.status === "error") &&
        systemStatus.hasErrors.value
    );
});
</script>
