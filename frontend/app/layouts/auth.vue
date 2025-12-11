<template>
    <!-- 专用于认证页的简洁布局（不包含全局导航） -->
    <div
        class="min-h-screen antialiased font-sans text-primary-content bg-gradient-neutral-page overflow-x-hidden"
    >
        <UContainer class="min-h-screen flex flex-col">
            <div class="flex-1">
                <slot />
            </div>
            <USeparator />
            <footer class="py-4 text-center text-sm text-secondary-content flex items-center justify-center gap-3">
                <UColorModeSwitch/>
                <span>© {{ new Date().getFullYear() }} HotSpot</span>
            </footer>
        </UContainer>
    </div>
</template>

<script setup>
// 认证页专用布局，无导航
</script>
