<template>
    <div
        class="min-h-screen antialiased bg-gradient-to-br from-neutral-50 to-neutral-100 dark:from-neutral-900 dark:to-neutral-950 flex items-center justify-center"
    >
        <UCard class="text-center px-6 py-8">
            <!-- 404 图标 -->
            <div class="mb-8">
                <div
                    class="w-32 h-32 rounded-full flex items-center justify-center mx-auto mb-6"
                >
                    <UIcon
                        name="i-heroicons-question-mark-circle"
                        class="w-16 h-16 text-neutral-700 dark:text-neutral-300"
                    />
                </div>
            </div>

            <!-- 错误信息 -->
            <h1 class="text-6xl font-bold mb-4">404</h1>
            <h2 class="text-2xl font-semibold mb-4">{{ $t('notFound.title') }}</h2>
            <p class="opacity-80 mb-8 max-w-md mx-auto">
                {{ $t('notFound.description') }}
            </p>

            <!-- 操作按钮 -->
            <div
                class="space-y-4 sm:space-y-0 sm:space-x-4 sm:flex sm:justify-center"
            >
                <UButton
                    size="lg"
                    @click="navigateTo('/')"
                    class="w-full sm:w-auto"
                >
                    <UIcon name="i-heroicons-home" class="w-5 h-5 mr-2" />
                    {{ $t('notFound.actions.goHome') }}
                </UButton>
                <UButton
                    variant="outline"
                    size="lg"
                    @click="goBack"
                    class="w-full sm:w-auto"
                >
                    <UIcon name="i-heroicons-arrow-left" class="w-5 h-5 mr-2" />
                    {{ $t('notFound.actions.goBack') }}
                </UButton>
            </div>

            <!-- 帮助链接 -->
            <div class="mt-12 text-sm opacity-80">
                <p class="mb-2">{{ $t('notFound.help.title') }}</p>
                <div class="space-x-4">
                    <UButton
                        variant="ghost"
                        size="sm"
                        @click="navigateTo('/settings')"
                    >
                        {{ $t('notFound.help.contactSupport') }}
                    </UButton>
                    <UButton
                        variant="ghost"
                        size="sm"
                        @click="navigateTo('/settings')"
                    >
                        {{ $t('notFound.help.feedback') }}
                    </UButton>
                </div>
            </div>
        </UCard>
    </div>
</template>

<script setup lang="ts">
// 页面元数据
definePageMeta({
    title: 'notFound.title',
    layout: false, // 使用无布局样式
});

// 返回上一页
const goBack = () => {
    if (history.length > 1) {
        history.back();
    } else {
        navigateTo("/");
    }
};
</script>
