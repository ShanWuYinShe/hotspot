<template>
  <div class="flex flex-col gap-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">审计日志</h1>
        <p class="text-neutral-500 dark:text-neutral-400 mt-1">查看系统操作记录</p>
      </div>
      <UButton variant="outline" icon="i-lucide-download" @click="exportLogs">
        导出日志
      </UButton>
    </div>

    <!-- Filters -->
    <UCard variant="outline">
      <div class="flex flex-col md:flex-row gap-4">
        <USelectMenu
          v-model="actionFilter"
          :items="actionOptions"
          placeholder="操作类型"
          class="w-40"
        />
        <USelectMenu
          v-model="targetTypeFilter"
          :items="targetTypeOptions"
          placeholder="目标类型"
          class="w-40"
        />
        <UInput
          v-model="operatorIdFilter"
          type="number"
          placeholder="操作者 ID"
          class="w-40"
        />
        <UButton icon="i-lucide-search" @click="handleSearch">筛选</UButton>
        <UButton variant="ghost" color="neutral" @click="resetFilters">重置</UButton>
      </div>
    </UCard>

    <!-- Audit Logs Table -->
    <UCard variant="outline">
      <UTable
        :data="logs"
        :columns="columns"
        :loading="loading"
        :empty-state="{ icon: 'i-lucide-scroll-text', label: '暂无审计日志' }"
      >
        <template #audit_id-cell="{ row }">
          <span class="font-mono text-sm truncate max-w-[100px]" :title="row.original.audit_id">
            {{ row.original.audit_id.slice(0, 8) }}...
          </span>
        </template>

        <template #action-cell="{ row }">
          <UBadge :color="getActionColor(row.original.action)" variant="soft">
            {{ getActionLabel(row.original.action) }}
          </UBadge>
        </template>

        <template #operator-cell="{ row }">
          <div class="flex items-center gap-2">
            <UAvatar
              :src="row.original.operator.avatar_url"
              :alt="row.original.operator.nickname"
              size="xs"
            />
            <span class="text-sm">{{ row.original.operator.nickname }}</span>
          </div>
        </template>

        <template #target-cell="{ row }">
          <div v-if="row.original.target_type" class="text-sm">
            <span class="text-neutral-500">{{ getTargetTypeLabel(row.original.target_type) }}:</span>
            <span class="font-mono ml-1">{{ row.original.target_id?.slice(0, 8) }}...</span>
          </div>
          <span v-else class="text-neutral-400">-</span>
        </template>

        <template #created_at-cell="{ row }">
          <span class="text-sm text-neutral-500">{{ formatTime(row.original.created_at) }}</span>
        </template>

        <template #ip_address-cell="{ row }">
          <span class="text-sm font-mono text-neutral-500">{{ row.original.ip_address || '-' }}</span>
        </template>

        <template #actions-cell="{ row }">
          <UButton
            variant="ghost"
            color="neutral"
            icon="i-lucide-eye"
            size="sm"
            @click="viewDetail(row.original)"
          />
        </template>
      </UTable>

      <!-- Pagination -->
      <div v-if="hasMore" class="flex justify-center pt-4">
        <UButton variant="ghost" :loading="loadingMore" @click="loadMore">
          加载更多
        </UButton>
      </div>
    </UCard>

    <!-- Detail Modal -->
    <UModal v-model:open="showDetailModal">
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-scroll-text" />
          <span>日志详情</span>
        </div>
      </template>
      <template #body>
        <div v-if="selectedLog" class="space-y-4">
          <div class="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span class="text-neutral-500">日志 ID</span>
              <p class="font-mono">{{ selectedLog.audit_id }}</p>
            </div>
            <div>
              <span class="text-neutral-500">操作类型</span>
              <p>
                <UBadge :color="getActionColor(selectedLog.action)" variant="soft">
                  {{ getActionLabel(selectedLog.action) }}
                </UBadge>
              </p>
            </div>
            <div>
              <span class="text-neutral-500">操作者</span>
              <div class="flex items-center gap-2 mt-1">
                <UAvatar :src="selectedLog.operator.avatar_url" size="xs" />
                <span>{{ selectedLog.operator.nickname }}</span>
              </div>
            </div>
            <div>
              <span class="text-neutral-500">操作时间</span>
              <p>{{ formatTime(selectedLog.created_at) }}</p>
            </div>
            <div>
              <span class="text-neutral-500">目标类型</span>
              <p>{{ selectedLog.target_type ? getTargetTypeLabel(selectedLog.target_type) : '-' }}</p>
            </div>
            <div>
              <span class="text-neutral-500">目标 ID</span>
              <p class="font-mono">{{ selectedLog.target_id || '-' }}</p>
            </div>
            <div>
              <span class="text-neutral-500">IP 地址</span>
              <p class="font-mono">{{ selectedLog.ip_address || '-' }}</p>
            </div>
          </div>

          <div v-if="selectedLog.details" class="pt-2 border-t border-neutral-200 dark:border-neutral-700">
            <span class="text-neutral-500 text-sm">详细信息</span>
            <pre class="mt-2 p-3 bg-neutral-100 dark:bg-neutral-800 rounded-lg text-sm overflow-auto max-h-60">{{ JSON.stringify(selectedLog.details, null, 2) }}</pre>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Export Modal -->
    <UModal v-model:open="showExportModal">
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-download" />
          <span>导出审计日志</span>
        </div>
      </template>
      <template #body>
        <div class="space-y-4">
          <UFormField label="导出格式">
            <USelect
              v-model="exportFormat"
              :items="[{ label: 'CSV', value: 'csv' }, { label: 'JSON', value: 'json' }]"
            />
          </UFormField>
          <p class="text-sm text-neutral-500">
            将导出符合当前筛选条件的所有日志记录。大量数据可能需要较长时间处理。
          </p>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="showExportModal = false">取消</UButton>
          <UButton :loading="exporting" @click="doExport">开始导出</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { useAdminAuditService } from '~/services/admin_audit.service'
import type { AuditLogItem, AuditAction } from '~/types/api'

definePageMeta({
  layout: 'dashboard',
  requiresAuth: true,
  middleware: ['admin']
})

const adminAuditService = useAdminAuditService()
const { formatTime } = useFormatters()
const toast = useToast()

// State
const logs = ref<AuditLogItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const cursor = ref<string | undefined>()

const actionFilter = ref('')
const targetTypeFilter = ref('')
const operatorIdFilter = ref('')

const showDetailModal = ref(false)
const selectedLog = ref<AuditLogItem | null>(null)

const showExportModal = ref(false)
const exporting = ref(false)
const exportFormat = ref<'csv' | 'json'>('csv')

// Options
const actionOptions = [
  { label: '全部操作', value: '' },
  { label: '封禁用户', value: 'user_ban' },
  { label: '解封用户', value: 'user_unban' },
  { label: '解散群组', value: 'group_dismiss' },
  { label: '处理举报', value: 'report_handle' },
  { label: '发布公告', value: 'announcement_create' },
  { label: '删除公告', value: 'announcement_delete' },
  { label: '授予管理员', value: 'admin_grant' },
  { label: '撤销管理员', value: 'admin_revoke' },
  { label: '创建角色', value: 'role_create' },
  { label: '更新角色', value: 'role_update' },
  { label: '删除角色', value: 'role_delete' },
  { label: '更新配置', value: 'config_update' }
]

const targetTypeOptions = [
  { label: '全部类型', value: '' },
  { label: '用户', value: 'user' },
  { label: '群组', value: 'group' },
  { label: '公告', value: 'announcement' },
  { label: '举报', value: 'report' },
  { label: '角色', value: 'role' },
  { label: '配置', value: 'config' }
]

// Table columns
const columns = [
  { accessorKey: 'audit_id', header: 'ID' },
  { accessorKey: 'action', header: '操作' },
  { accessorKey: 'operator', header: '操作者' },
  { accessorKey: 'target', header: '目标' },
  { accessorKey: 'created_at', header: '时间' },
  { accessorKey: 'ip_address', header: 'IP' },
  { accessorKey: 'actions', header: '详情' }
]

// Methods
function getActionColor(action: AuditAction) {
  if (action.includes('ban') || action.includes('dismiss') || action.includes('delete') || action.includes('revoke')) {
    return 'error'
  }
  if (action.includes('create') || action.includes('grant')) {
    return 'success'
  }
  return 'primary'
}

function getActionLabel(action: AuditAction) {
  const labels: Record<AuditAction, string> = {
    user_ban: '封禁用户',
    user_unban: '解封用户',
    group_dismiss: '解散群组',
    report_handle: '处理举报',
    announcement_create: '发布公告',
    announcement_delete: '删除公告',
    admin_grant: '授予管理员',
    admin_revoke: '撤销管理员',
    role_create: '创建角色',
    role_update: '更新角色',
    role_delete: '删除角色',
    config_update: '更新配置'
  }
  return labels[action] || action
}

function getTargetTypeLabel(type: string) {
  const labels: Record<string, string> = {
    user: '用户',
    group: '群组',
    announcement: '公告',
    report: '举报',
    role: '角色',
    config: '配置'
  }
  return labels[type] || type
}

async function fetchLogs(reset = false) {
  if (reset) {
    cursor.value = undefined
    logs.value = []
  }

  loading.value = reset
  loadingMore.value = !reset

  try {
    const res = await adminAuditService.list({
      action: actionFilter.value as AuditAction || undefined,
      target_type: targetTypeFilter.value as 'user' | 'group' | 'announcement' | 'report' | 'role' | 'config' || undefined,
      operator_id: operatorIdFilter.value ? Number(operatorIdFilter.value) : undefined,
      cursor: cursor.value,
      limit: 20
    })
    logs.value = reset ? res.items : [...logs.value, ...res.items]
    hasMore.value = res.pagination.has_more
    cursor.value = res.pagination.next_cursor
  } catch (e) {
    toast.add({ title: '加载失败', color: 'error' })
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function handleSearch() {
  fetchLogs(true)
}

function resetFilters() {
  actionFilter.value = undefined
  targetTypeFilter.value = undefined
  operatorIdFilter.value = ''
  fetchLogs(true)
}

function loadMore() {
  fetchLogs(false)
}

async function viewDetail(log: AuditLogItem) {
  try {
    selectedLog.value = await adminAuditService.detail({ audit_id: log.audit_id })
    showDetailModal.value = true
  } catch (e) {
    toast.add({ title: '获取详情失败', color: 'error' })
  }
}

function exportLogs() {
  showExportModal.value = true
}

async function doExport() {
  exporting.value = true
  try {
    const res = await adminAuditService.export({
      action: actionFilter.value as AuditAction || undefined,
      target_type: targetTypeFilter.value as 'user' | 'group' | 'announcement' | 'report' | 'role' | 'config' || undefined,
      operator_id: operatorIdFilter.value ? Number(operatorIdFilter.value) : undefined,
      format: exportFormat.value
    })
    toast.add({ title: '导出任务已创建', description: `任务 ID: ${res.task_id}`, color: 'success' })
    showExportModal.value = false
  } catch (e) {
    toast.add({ title: '导出失败', color: 'error' })
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  fetchLogs(true)
})
</script>
