<template>
    <div
        class="min-h-screen antialiased flex items-center justify-center"
    >
        <UCard class="px-6 py-8">
            <!-- 404 图标 -->
            <div class="mb-8">
                <UAvatar
                    size="3xl"
                    icon="i-lucide-circle-question-mark"
                    color="neutral"
                    class="mx-auto mb-6"
                />
            </div>

            <!-- 错误信息 -->
            <div class="text-center">
                <h1 class="text-7xl font-extrabold tracking-tight mb-4">404</h1>
                <h2 class="text-2xl sm:text-3xl font-bold tracking-tight text-neutral-900 dark:text-white mb-4">{{ $t('notFound.title') }}</h2>
                <p class="text-base text-neutral-600 dark:text-neutral-300 mb-8 max-w-md mx-auto">
                    {{ $t('notFound.description') }}
                </p>
            </div>

            <!-- 操作按钮 -->
            <div
                class="space-y-4 sm:space-y-0 sm:space-x-4 sm:flex sm:justify-center"
            >
                <UButton
                    size="lg"
                    leading-icon="i-lucide-house"
                    @click="navigateTo('/')"
                    class="w-full sm:w-auto"
                >
                    {{ $t('notFound.actions.goHome') }}
                </UButton>
                <UButton
                    variant="outline"
                    size="lg"
                    leading-icon="i-lucide-arrow-left"
                    @click="router.back()"
                    class="w-full sm:w-auto"
                >
                    {{ $t('notFound.actions.goBack') }}
                </UButton>
            </div>

            <!-- 帮助链接 -->
            <div class="mt-12 text-base text-neutral-600 dark:text-neutral-300 text-sm">
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

const router = useRouter()
</script>
