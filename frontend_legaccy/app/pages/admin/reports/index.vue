<template>
    <div class="flex flex-col gap-6">
        <!-- Sticky Header & Controls -->
        <div class="sticky top-[64px] z-20 bg-white/95 dark:bg-neutral-900/95 backdrop-blur-sm border-b border-neutral-200 dark:border-neutral-800 -mx-3 sm:-mx-4 lg:-mx-6 px-3 sm:px-4 lg:px-6 py-4 space-y-4">
            <!-- Header -->
            <div class="flex items-center justify-between">
                <div>
                    <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">举报审核</h1>
                    <p class="text-neutral-500 dark:text-neutral-400 mt-1">处理用户举报内容</p>
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
            <div class="flex flex-wrap gap-4 items-center">
                <USelect
                    v-model="statusFilter"
                    :items="statusOptions"
                    placeholder="状态筛选"
                    class="w-40"
                />
                <USelect
                    v-model="typeFilter"
                    :items="typeOptions"
                    placeholder="类型筛选"
                    class="w-40"
                />
                <UButton
                    variant="ghost"
                    color="neutral"
                    icon="i-lucide-refresh-cw"
                    :loading="loading"
                    @click="refresh"
                >
                    刷新
                </UButton>
            </div>
        </div>

        <div class="space-y-6 pb-20">

            <!-- 列表 -->
            <div class="min-h-[50vh]">
                <div v-if="loading && reports.length === 0" class="space-y-4">
                    <USkeleton v-for="i in 5" :key="i" shape="rectangle" class="h-32 w-full" />
                </div>

                <div v-else-if="reports.length > 0" class="space-y-4">
                    <UCard
                        v-for="report in reports"
                        :key="report.report_id"
                        variant="subtle"
                        class="group"
                    >
                        <div class="flex items-start gap-4">
                            <!-- 图标 -->
                            <div class="mt-1">
                                <UAvatar
                                    :icon="getTypeIcon(report.target_type)"
                                    size="md"
                                />
                            </div>

                            <div class="flex-1 min-w-0 space-y-2">
                                <!-- 头部信息 -->
                                <div class="flex items-center justify-between gap-2">
                                    <div class="flex items-center gap-2">
                                        <UBadge variant="soft" :color="getTypeColor(report.target_type)" size="xs">
                                            {{ getTypeText(report.target_type) }}
                                        </UBadge>
                                        <span class="text-xs font-mono text-neutral-500">{{ report.target_id }}</span>
                                    </div>
                                    <UBadge :color="getStatusColor(report.status)" variant="soft" size="xs">
                                        {{ getStatusText(report.status) }}
                                    </UBadge>
                                </div>

                                <!-- 举报原因 -->
                                <div>
                                    <div class="font-medium text-neutral-900 dark:text-white">{{ report.reason }}</div>
                                    <p class="text-sm text-neutral-600 dark:text-neutral-400 mt-1">
                                        {{ report.description || '无详细描述' }}
                                    </p>
                                </div>

                                <!-- 底部信息 -->
                                <div class="flex items-center gap-4 text-xs text-neutral-500 pt-2">
                                    <span class="flex items-center gap-1">
                                        <UIcon name="i-lucide-clock" class="size-3" />
                                        {{ formatTime(report.created_at) }}
                                    </span>
                                    <span v-if="report.handled_at" class="flex items-center gap-1">
                                        <UIcon name="i-lucide-check-circle" class="size-3" />
                                        处理于 {{ formatTime(report.handled_at) }}
                                    </span>
                                </div>
                            </div>

                            <!-- 操作按钮 -->
                            <div v-if="report.status === 'pending' || report.status === 'reviewing'" class="flex flex-col gap-2">
                                <UButton size="sm" color="primary" @click="handleReport(report, 'resolve')">
                                    处理
                                </UButton>
                                <UButton size="sm" color="neutral" variant="ghost" @click="handleReport(report, 'dismiss')">
                                    忽略
                                </UButton>
                                <UButton size="sm" color="neutral" variant="ghost" icon="i-lucide-external-link" @click="viewTarget(report)">
                                    查看
                                </UButton>
                            </div>
                        </div>
                    </UCard>
                </div>

                <UEmpty
                    v-else
                    icon="i-lucide-check-circle"
                    title="暂无举报"
                    description="当前没有符合条件的举报记录"
                    class="py-20"
                />

                <!-- 加载更多 -->
                <div v-if="hasMore && reports.length > 0" class="py-8 flex justify-center">
                     <UButton
                        :loading="loadingMore"
                        variant="ghost"
                        color="neutral"
                        @click="loadMore"
                    >
                        {{ loadingMore ? '加载中...' : '加载更多' }}
                    </UButton>
                </div>
            </div>
        </div>
        
        <!-- 处理模态框 -->
        <ReportHandleModal
            v-model:open="showHandleModal"
            :report="currentReport"
            :action="currentAction"
            @success="refresh"
        />
    </div>
</template>

<script setup lang="ts">
import { useReportsService } from '~/services/reports.service'
import type { ReportItem, ReportStatus, ReportTargetType } from '~/types/api'
import ReportHandleModal from '~/components/modals/admin/ReportHandleModal.vue'

definePageMeta({
    layout: 'dashboard',
    requiresAuth: true,
    middleware: ['admin']
})

const reportsService = useReportsService()
const { formatTime } = useFormatters()
const router = useRouter()

const loading = ref(true)
const loadingMore = ref(false)
const reports = ref<ReportItem[]>([])
const cursor = ref<string | undefined>(undefined)
const hasMore = ref(true)

const statusFilter = ref<ReportStatus | 'pending'>('pending')
const typeFilter = ref<ReportTargetType | undefined>(undefined)

const showHandleModal = ref(false)
const currentReport = ref<ReportItem | undefined>(undefined)
const currentAction = ref<'resolve' | 'dismiss'>('resolve')

const statusOptions = [
    { label: '全部状态', value: undefined },
    { label: '待处理', value: 'pending' },
    { label: '审核中', value: 'reviewing' },
    { label: '已解决', value: 'resolved' },
    { label: '已忽略', value: 'dismissed' }
]

const typeOptions = [
    { label: '全部类型', value: undefined },
    { label: '用户', value: 'user' },
    { label: '动态', value: 'moment' },
    { label: '评论', value: 'comment' },
    { label: '群组', value: 'group' },
    { label: '消息', value: 'message' }
]

onMounted(() => {
    refresh()
})

watch([statusFilter, typeFilter], () => {
    refresh()
})

async function refresh() {
    loading.value = true
    cursor.value = undefined
    hasMore.value = true
    try {
        const res = await reportsService.adminList({
            status: statusFilter.value,
            target_type: typeFilter.value,
            limit: 20
        })
        reports.value = res.items
        cursor.value = res.pagination.next_cursor
        hasMore.value = res.pagination.has_more
    } catch (e) {
        // error
    } finally {
        loading.value = false
    }
}

async function loadMore() {
    if (loadingMore.value || !hasMore.value) return
    loadingMore.value = true
    try {
        const res = await reportsService.adminList({
            status: statusFilter.value,
            target_type: typeFilter.value,
            limit: 20,
            cursor: cursor.value
        })
        reports.value.push(...res.items)
        cursor.value = res.pagination.next_cursor
        hasMore.value = res.pagination.has_more
    } catch (e) {
        // error
    } finally {
        loadingMore.value = false
    }
}

function handleReport(report: ReportItem, action: 'resolve' | 'dismiss') {
    currentReport.value = report
    currentAction.value = action
    showHandleModal.value = true
}

function viewTarget(report: ReportItem) {
    switch (report.target_type) {
        case 'user': router.push(`/users/${report.target_id}`); break
        case 'moment': router.push(`/moments/${report.target_id}`); break
        case 'group': router.push(`/groups/${report.target_id}`); break
        // 其他类型暂无详情页或需特殊处理
    }
}

function getTypeIcon(type: string) {
    switch (type) {
        case 'user': return 'i-lucide-user'
        case 'moment': return 'i-lucide-camera'
        case 'comment': return 'i-lucide-message-square'
        case 'group': return 'i-lucide-users'
        case 'message': return 'i-lucide-message-circle'
        default: return 'i-lucide-alert-circle'
    }
}

function getTypeColor(type: string) {
    switch (type) {
        case 'user': return 'primary'
        case 'moment': return 'secondary' // Nuxt UI default palette doesn't have secondary, maybe orange/purple?
        case 'comment': return 'neutral'
        case 'group': return 'success'
        case 'message': return 'info'
        default: return 'neutral'
    }
}

function getTypeText(type: string) {
    const map: Record<string, string> = {
        user: '用户',
        moment: '动态',
        comment: '评论',
        group: '群组',
        message: '消息'
    }
    return map[type] || type
}

function getStatusColor(status: string) {
    switch (status) {
        case 'pending': return 'warning'
        case 'reviewing': return 'info'
        case 'resolved': return 'success'
        case 'dismissed': return 'neutral'
        default: return 'neutral'
    }
}

function getStatusText(status: string) {
    const map: Record<string, string> = {
        pending: '待处理',
        reviewing: '审核中',
        resolved: '已解决',
        dismissed: '已忽略'
    }
    return map[status] || status
}
</script>
