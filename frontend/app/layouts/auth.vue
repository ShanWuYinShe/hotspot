<template>
    <!-- 专用于认证页的简洁布局（不包含全局导航） -->
    <div
        class="min-h-screen antialiased font-sans text-neutral-900 dark:text-neutral-100 bg-gradient-to-br from-neutral-50 to-neutral-100 dark:from-neutral-900 dark:to-neutral-950 overflow-x-hidden"
    >
        <UContainer class="min-h-screen flex flex-col">
            <div class="flex-1">
                <slot />
            </div>
            <USeparator />
            <footer class="py-4 text-center text-sm text-neutral-600 dark:text-neutral-300 flex items-center justify-center gap-3">
                <UColorModeSwitch/>
                <span>© {{ new Date().getFullYear() }} HotSpot</span>
            </footer>
        </UContainer>
    </div>
</template>

<script setup>
// 认证页专用布局，无导航
</script>
