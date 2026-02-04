<template>
    <div class="flex flex-col gap-6">
        <!-- Sticky Header & Controls -->
        <div class="sticky top-[64px] z-20 bg-white/95 dark:bg-neutral-900/95 backdrop-blur-sm border-b border-neutral-200 dark:border-neutral-800 -mx-3 sm:-mx-4 lg:-mx-6 px-3 sm:px-4 lg:px-6 py-4 space-y-4">
            <!-- Header -->
            <div class="flex items-center justify-between">
                <div>
                    <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">系统配置</h1>
                    <p class="text-neutral-500 dark:text-neutral-400 mt-1">全局参数设置与管理</p>
                </div>
                <UButton
                    variant="ghost"
                    color="neutral"
                    icon="i-lucide-arrow-left"
                    to="/admin"
                >
                    返回控制台
                </UButton>
            </div>

            <!-- 筛选栏 -->
            <div class="flex gap-4 items-center">
                <USelect
                    v-model="selectedCategory"
                    :items="categories"
                    placeholder="选择分类"
                    class="w-48"
                />
                <UInput
                    v-model="searchQuery"
                    placeholder="搜索配置键..."
                    icon="i-lucide-search"
                    class="flex-1"
                />
                <UButton icon="i-lucide-plus" @click="openCreateModal">新增配置</UButton>
            </div>
        </div>

        <div class="space-y-6 pb-20">

            <!-- 列表 -->
            <div class="min-h-[50vh]">
                <div v-if="loading" class="space-y-4">
                    <USkeleton v-for="i in 5" :key="i" shape="rectangle" class="h-20 w-full" />
                </div>

                <div v-else-if="filteredConfigs.length > 0" class="space-y-4">
                    <UCard
                        v-for="config in filteredConfigs"
                        :key="config.key"
                        variant="subtle"
                    >
                        <div class="flex items-start justify-between gap-4">
                            <div class="space-y-1 flex-1">
                                <div class="flex items-center gap-2">
                                    <h3 class="text-sm font-mono font-bold text-neutral-900 dark:text-white">{{ config.key }}</h3>
                                    <UBadge color="neutral" variant="soft" size="xs">{{ config.type }}</UBadge>
                                    <UBadge color="neutral" variant="outline" size="xs">{{ config.category }}</UBadge>
                                </div>
                                <p class="text-xs text-neutral-500 dark:text-neutral-400">{{ config.description }}</p>
                            </div>
                            <div class="flex items-center gap-2">
                                <UButton variant="ghost" color="neutral" size="sm" icon="i-lucide-history" @click="viewHistory(config)" />
                                <UButton variant="ghost" color="neutral" size="sm" icon="i-lucide-edit" @click="editConfig(config)" />
                                <UButton variant="ghost" color="error" size="sm" icon="i-lucide-trash-2" @click="deleteConfig(config)" />
                            </div>
                        </div>

                        <div class="mt-4 p-3 bg-neutral-100 dark:bg-neutral-800 rounded font-mono text-sm break-all">
                            {{ config.value }}
                        </div>
                    </UCard>
                </div>

                <UEmpty
                    v-else
                    icon="i-lucide-settings-2"
                    title="暂无配置"
                    description="没有找到匹配的系统配置项"
                    class="py-20"
                />
            </div>
        </div>
        
        <!-- 模态框 (Create/Edit Config) 待实现 -->
    </div>
</template>

<script setup lang="ts">
import { useAdminConfigService } from '~/services/admin_config.service'
import type { ConfigItem } from '~/types/api'

definePageMeta({
    layout: 'dashboard',
    requiresAuth: true,
    middleware: ['admin']
})

const configService = useAdminConfigService()
const feedback = useFeedbackStore()

const loading = ref(true)
const configs = ref<ConfigItem[]>([])
const searchQuery = ref('')
const selectedCategory = ref<string | undefined>(undefined)

const categories = computed(() => {
    const cats = new Set(configs.value.map(c => c.category))
    return Array.from(cats).map(c => ({ label: c, value: c }))
})

const filteredConfigs = computed(() => {
    let res = configs.value
    if (selectedCategory.value) {
        res = res.filter(c => c.category === selectedCategory.value)
    }
    if (searchQuery.value) {
        const query = searchQuery.value.toLowerCase()
        res = res.filter(c => c.key.toLowerCase().includes(query) || c.description.toLowerCase().includes(query))
    }
    return res
})

onMounted(() => {
    refresh()
})

async function refresh() {
    loading.value = true
    try {
        const res = await configService.list({ limit: 100 })
        configs.value = res.items
    } catch (e) {
        // error
    } finally {
        loading.value = false
    }
}

function openCreateModal() {
    feedback.info('功能开发中')
}

function editConfig(config: ConfigItem) {
    feedback.info('功能开发中')
}

function viewHistory(config: ConfigItem) {
    feedback.info('功能开发中')
}

async function deleteConfig(config: ConfigItem) {
    if (!confirm(`确定要删除配置 ${config.key} 吗？`)) return
    try {
        await configService.delete({ key: config.key })
        feedback.success('删除成功')
        refresh()
    } catch (e) {
        feedback.error('删除失败')
    }
}
</script>
