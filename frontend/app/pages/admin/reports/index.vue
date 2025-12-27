<template>
  <div class="w-full">
    <UCard v-if="canRenderAdmin">
        <template #header>
          <div class="flex flex-col sm:flex-row gap-3 sm:items-center sm:justify-between">
            <div class="flex items-center gap-2 min-w-0">
              <USelectMenu
                v-model="filters.status"
                :items="statusOptions"
                value-key="value"
                class="w-40"
              />
              <USelectMenu
                v-model="filters.target_type"
                :items="targetTypeOptions"
                value-key="value"
                class="w-40"
              />
            </div>
            <div class="flex items-center gap-2">
              <UInput
                v-model="filters.query"
                placeholder="搜索原因/描述/target_id"
                icon="i-lucide-search"
                class="w-full sm:w-72"
              />
            </div>
          </div>
        </template>

      <div v-if="loading" class="space-y-3">
        <USkeleton class="h-10 w-full" />
        <USkeleton class="h-10 w-full" />
        <USkeleton class="h-10 w-full" />
      </div>

      <UEmpty
        v-else-if="filtered.length === 0"
        icon="i-lucide-flag"
        title="暂无举报"
        description="当前筛选条件下没有数据"
      />

      <UTable
        v-else
        :data="filtered"
        :columns="columns"
      >
        <template #status-cell="{ row }">
          <UBadge :color="statusColor(row.original.status)" variant="soft" size="sm">
            {{ statusLabel(row.original.status) }}
          </UBadge>
        </template>

        <template #target-cell="{ row }">
          <div class="flex flex-col min-w-0">
            <div class="font-medium truncate">{{ row.original.target_type }}</div>
            <UBadge color="neutral" variant="soft" size="xs" class="mt-1 max-w-full truncate">
              {{ row.original.target_id }}
            </UBadge>
          </div>
        </template>

        <template #reason-cell="{ row }">
          <div class="flex flex-col min-w-0">
            <div class="truncate">{{ row.original.reason }}</div>
            <UBadge
              v-if="row.original.description"
              color="neutral"
              variant="soft"
              size="xs"
              class="mt-1 max-w-full"
            >
              {{ row.original.description }}
            </UBadge>
          </div>
        </template>

        <template #time-cell="{ row }">
          <div class="flex flex-col items-end tabular-nums">
            <span class="text-sm">{{ formatTime(row.original.created_at) }}</span>
            <UBadge v-if="row.original.handled_at" color="neutral" variant="soft" size="xs" class="mt-1">
              {{ formatTime(row.original.handled_at) }}
            </UBadge>
          </div>
        </template>

        <template #actions-cell="{ row }">
          <div class="flex items-center justify-end gap-1">
            <UButton
              variant="ghost"
              color="success"
              icon="i-lucide-check"
              size="sm"
              :square="true"
              :disabled="row.original.status === 'resolved'"
              @click="handle(row.original.id, 'resolve')"
            />
            <UButton
              variant="ghost"
              color="neutral"
              icon="i-lucide-refresh-cw"
              size="sm"
              :square="true"
              :disabled="row.original.status === 'reviewing'"
              @click="handle(row.original.id, 'escalate')"
            />
            <UButton
              variant="ghost"
              color="error"
              icon="i-lucide-x"
              size="sm"
              :square="true"
              :disabled="row.original.status === 'dismissed'"
              @click="handle(row.original.id, 'dismiss')"
            />
          </div>
        </template>
      </UTable>

      <div class="flex items-center justify-between mt-3">
        <UBadge color="neutral" variant="soft" size="xs">
          显示 {{ filtered.length }} 条
        </UBadge>
        <UButton
          v-if="page.pagination.has_more"
          variant="outline"
          color="neutral"
          size="sm"
          :loading="loadingMore"
          @click="loadMore"
        >
          加载更多
        </UButton>
      </div>
    </UCard>

    <div v-else class="w-full space-y-3">
      <USkeleton class="h-10 w-full" />
      <USkeleton class="h-10 w-full" />
      <USkeleton class="h-10 w-full" />
      <USkeleton class="h-48 w-full" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { HandleReportRequest, ReportItem, ReportsRequest } from '~/types/api'
import { useDashboardHeader } from '~/composables/useDashboardHeader'
import { useAdminStore } from '~/stores/admin.store'
import { useHydratedStore } from '~/composables/useHydratedStore'

definePageMeta({
  title: '举报处理',
  requiresAuth: true,
  description: '审核用户举报并处理违规内容',
  layout: 'dashboard',
  middleware: 'admin',
  dashboardHeader: true
})

const router = useRouter()
import { useReportsService } from '~/services/reports.service'
const reportsService = useReportsService()
const modal = useModal()
const feedback = useFeedbackStore()
const { withErrorHandling } = useErrorHandler()
const { formatTime } = useFormatters()
const dashboardHeader = useDashboardHeader()
const authStore = useAuthStore()
const adminStore = useAdminStore()
const isAuthenticated = useHydratedStore(authStore, 'isAuthenticated', false)
const canRenderAdmin = computed(() => isAuthenticated.value && adminStore.hasAccess)

watchEffect(() => {
  dashboardHeader.setActions([
    {
      key: 'back',
      label: '返回',
      icon: 'i-lucide-arrow-left',
      color: 'neutral',
      variant: 'outline',
      action: () => router.back()
    },
    {
      key: 'refresh',
      label: '刷新',
      icon: 'i-lucide-refresh-cw',
      variant: 'outline',
      loading: loading.value,
      action: () => reload(true)
    }
  ])
})

const loading = ref(false)
const loadingMore = ref(false)

const filters = reactive<{ status?: ReportsRequest['status'] | 'all'; target_type?: ReportsRequest['target_type'] | 'all'; query: string }>({
  status: 'pending',
  target_type: 'all',
  query: ''
})

const statusOptions = [
  { label: '待处理', value: 'pending' },
  { label: '处理中', value: 'reviewing' },
  { label: '已解决', value: 'resolved' },
  { label: '已驳回', value: 'dismissed' },
  { label: '全部', value: 'all' }
]

const targetTypeOptions = [
  { label: '全部', value: 'all' },
  { label: '用户', value: 'user' },
  { label: '动态', value: 'moment' },
  { label: '评论', value: 'comment' },
  { label: '消息', value: 'message' },
  { label: '群组', value: 'group' }
]

import type { TableColumn } from '@nuxt/ui'

const columns: TableColumn<ReportItem>[] = [
  { id: 'status', accessorKey: 'status', header: '状态' },
  { id: 'target', accessorKey: 'target_id', header: '目标' },
  { id: 'reason', accessorKey: 'reason', header: '原因' },
  { id: 'time', accessorKey: 'created_at', header: '时间' },
  { id: 'actions', accessorKey: 'id', header: '操作' }
]

const page = reactive<{ items: ReportItem[]; pagination: { has_more: boolean; next_cursor?: string } }>({
  items: [],
  pagination: { has_more: false, next_cursor: undefined }
})

const buildRequest = (cursor?: string): ReportsRequest => {
  const req: ReportsRequest = {
    limit: 20,
    cursor
  }
  if (filters.status && filters.status !== 'all') req.status = filters.status
  if (filters.target_type && filters.target_type !== 'all') req.target_type = filters.target_type
  return req
}

const reload = async (manual = false) => {
  await withErrorHandling(async () => {
    loading.value = true
    const res = await reportsService.adminList(buildRequest())
    page.items = res.items
    page.pagination = res.pagination
  }, { title: '加载失败', loading, showToast: manual })
  loading.value = false
}

const loadMore = async () => {
  if (!page.pagination.has_more || !page.pagination.next_cursor) return
  await withErrorHandling(async () => {
    loadingMore.value = true
    const res = await reportsService.adminList(buildRequest(page.pagination.next_cursor))
    page.items = page.items.concat(res.items)
    page.pagination = res.pagination
  }, { title: '加载失败', loading: loadingMore })
  loadingMore.value = false
}

const filtered = computed(() => {
  const q = filters.query.trim().toLowerCase()
  if (!q) return page.items
  return page.items.filter(r =>
    r.reason.toLowerCase().includes(q)
    || (r.description || '').toLowerCase().includes(q)
    || r.target_id.toLowerCase().includes(q)
  )
})

const statusColor = (s: ReportItem['status']) => {
  if (s === 'pending') return 'warning'
  if (s === 'reviewing') return 'info'
  if (s === 'resolved') return 'success'
  return 'neutral'
}
const statusLabel = (s: ReportItem['status']) => {
  if (s === 'pending') return '待处理'
  if (s === 'reviewing') return '处理中'
  if (s === 'resolved') return '已解决'
  return '已驳回'
}

const handle = async (id: string, action: HandleReportRequest['action']) => {
  const confirmed = await modal.showConfirm({
    title: '确认操作',
    message: action === 'resolve'
      ? '确认将该举报标记为已解决？'
      : action === 'dismiss'
        ? '确认驳回该举报？'
        : '确认将该举报升级为处理中？',
    confirmText: '确认',
    confirmColor: action === 'dismiss' ? 'error' : 'primary'
  })
  if (!confirmed) return

  await withErrorHandling(async () => {
    await reportsService.adminHandle({ id, action })
    feedback.success('操作成功')
    await reload(false)
  }, { title: '操作失败' })
}

watch(() => [filters.status, filters.target_type], () => {
  void reload(false)
})

onMounted(() => {
  void reload(true)
})
</script>

