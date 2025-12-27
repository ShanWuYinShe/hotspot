<template>
  <div class="w-full">
    <div v-if="canRenderAdmin" class="grid grid-cols-1 lg:grid-cols-3 gap-3">
      <!-- 左：配置列表 -->
      <UCard class="lg:col-span-1">
        <template #header>
          <div class="flex items-center gap-2">
            <UInput
              v-model="query"
              icon="i-lucide-search"
              placeholder="搜索 key/category"
              class="w-full"
            />
          </div>
        </template>

        <div v-if="loading" class="space-y-2">
          <USkeleton class="h-10 w-full" />
          <USkeleton class="h-10 w-full" />
          <USkeleton class="h-10 w-full" />
        </div>

        <UEmpty
          v-else-if="filtered.length === 0"
          icon="i-lucide-settings"
          title="暂无配置"
          description="没有匹配的配置项"
        />

        <div v-else>
          <UPageList divide>
            <UPageCard
              v-for="item in filtered"
              :key="item.key"
              variant="ghost"
              :title="item.key"
              :description="item.category"
              :highlight="selectedKey === item.key"
              highlight-color="primary"
              :on-click="() => select(item.key)"
            >
              <template #footer>
                <div class="flex justify-end">
                  <UBadge variant="soft" size="xs" color="neutral">{{ item.type }}</UBadge>
                </div>
              </template>
            </UPageCard>
          </UPageList>
        </div>
      </UCard>

      <!-- 右：详情/编辑 -->
      <UCard class="lg:col-span-2">
        <template #header>
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <div class="font-semibold truncate">
                {{ mode === 'create' ? '新建配置' : (detail?.key || '配置详情') }}
              </div>
              <UBadge color="neutral" variant="soft" size="xs" class="mt-1 max-w-full truncate">
                {{ mode === 'create' ? '创建新的系统配置项' : (detail?.description || '选择一条配置查看详情') }}
              </UBadge>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <UButton
                v-if="mode !== 'create' && selectedKey"
                variant="ghost"
                color="neutral"
                icon="i-lucide-clock"
                size="sm"
                :square="true"
                @click="toggleHistory"
              />
              <UButton
                v-if="mode !== 'create' && selectedKey"
                variant="ghost"
                color="error"
                icon="i-lucide-trash-2"
                size="sm"
                :square="true"
                @click="removeSelected"
              />
            </div>
          </div>
        </template>

        <UEmpty
          v-if="mode === 'view' && !detail"
          icon="i-lucide-settings"
          title="请选择配置项"
          description="从左侧列表选择一条配置进行查看和编辑"
        />

        <div v-else class="space-y-4">
          <UForm :state="form" class="space-y-4">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <UFormField label="Key" required>
                <UInput v-model="form.key" :disabled="mode !== 'create'" placeholder="例如：access_token_ttl" />
              </UFormField>
              <UFormField label="类型" required>
                <USelectMenu
                  v-model="form.type"
                  :items="typeOptions"
                  value-key="value"
                  :disabled="mode !== 'create'"
                />
              </UFormField>
            </div>

            <UFormField label="分类" required>
              <UInput v-model="form.category" :disabled="mode !== 'create'" placeholder="例如：auth / geo / limits" />
            </UFormField>

            <UFormField label="描述" required>
              <UTextarea v-model="form.description" :disabled="mode !== 'create'" :rows="3" placeholder="用于后台展示与审计说明" />
            </UFormField>

            <UFormField label="值" required>
              <UTextarea v-model="form.value" :rows="3" placeholder="注意：存储为字符串，type 用于解释/校验" />
              <template #hint>
                <UBadge color="neutral" variant="soft" size="xs">
                  updated_at: {{ detail?.updated_at ? formatTime(detail.updated_at) : '-' }}
                </UBadge>
              </template>
            </UFormField>

            <div class="flex items-center justify-end gap-2 pt-1">
              <UButton
                v-if="mode === 'create'"
                variant="outline"
                color="neutral"
                @click="cancelCreate"
              >
                取消
              </UButton>
              <UButton
                color="primary"
                :loading="saving"
                @click="save"
              >
                {{ mode === 'create' ? '创建' : '保存' }}
              </UButton>
            </div>
          </UForm>

          <!-- 历史记录 -->
          <div v-if="showHistory && selectedKey" class="pt-4">
            <USeparator class="mb-4" />
            <div class="flex items-center justify-between mb-3">
              <div class="font-semibold">变更历史</div>
              <UButton variant="ghost" color="neutral" size="sm" icon="i-lucide-refresh-cw" :square="true" :loading="historyLoading" @click="loadHistory(true)" />
            </div>

            <div v-if="historyLoading" class="space-y-2">
              <USkeleton class="h-10 w-full" />
              <USkeleton class="h-10 w-full" />
            </div>

            <UEmpty
              v-else-if="history.items.length === 0"
              icon="i-lucide-clock"
              title="暂无历史"
              description="该配置还没有变更记录"
            />

            <UTable
              v-else
              :data="history.items"
              :columns="historyColumns"
            >
              <template #changed_at-cell="{ row }">
                <span class="tabular-nums">{{ formatTime(row.original.changed_at) }}</span>
              </template>
            </UTable>

            <div class="flex items-center justify-end mt-3">
              <UButton
                v-if="history.pagination.has_more"
                variant="outline"
                color="neutral"
                size="sm"
                :loading="historyMoreLoading"
                @click="loadHistoryMore"
              >
                加载更多
              </UButton>
            </div>
          </div>
        </div>
      </UCard>
    </div>

    <div v-else class="w-full space-y-3">
      <USkeleton class="h-10 w-full" />
      <USkeleton class="h-10 w-full" />
      <USkeleton class="h-10 w-full" />
      <USkeleton class="h-64 w-full" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ConfigItem, ConfigHistoryItem } from '~/types/api'
import type { TableColumn } from '@nuxt/ui'
import { useDashboardHeader } from '~/composables/useDashboardHeader'
import { useAdminStore } from '~/stores/admin.store'
import { useHydratedStore } from '~/composables/useHydratedStore'

definePageMeta({
  title: '系统配置',
  requiresAuth: true,
  description: '管理系统配置项（创建/更新/删除/历史）',
  layout: 'dashboard',
  middleware: 'admin',
  dashboardHeader: true
})

const router = useRouter()
import { useAdminConfigService } from '~/services/admin_config.service'
const service = useAdminConfigService()
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
      key: 'create',
      label: '新建',
      icon: 'i-lucide-plus',
      variant: 'outline',
      disabled: loading.value,
      action: () => startCreate()
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
const saving = ref(false)
const query = ref('')

const selectedKey = ref<string | null>(null)
const detail = ref<ConfigItem | null>(null)

const mode = ref<'view' | 'create'>('view')

const typeOptions = [
  { label: 'string', value: 'string' },
  { label: 'number', value: 'number' },
  { label: 'boolean', value: 'boolean' }
]

const list = ref<ConfigItem[]>([])

const form = reactive<{ key: string; type: 'string' | 'number' | 'boolean'; category: string; description: string; value: string }>({
  key: '',
  type: 'string',
  category: '',
  description: '',
  value: ''
})

const reload = async (manual = false) => {
  await withErrorHandling(async () => {
    loading.value = true
    const res = await service.list({ limit: 100 })
    list.value = res.items
  }, { title: '加载失败', loading, showToast: manual })
  loading.value = false
}

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return list.value
  return list.value.filter(i =>
    i.key.toLowerCase().includes(q) || i.category.toLowerCase().includes(q)
  )
})

const select = async (key: string) => {
  selectedKey.value = key
  mode.value = 'view'
  showHistory.value = false
  await withErrorHandling(async () => {
    const res = await service.detail({ key })
    detail.value = res
    form.key = res.key
    form.type = res.type
    form.category = res.category
    form.description = res.description
    form.value = res.value
  }, { title: '加载详情失败' })
}

const startCreate = () => {
  mode.value = 'create'
  selectedKey.value = null
  detail.value = null
  showHistory.value = false
  form.key = ''
  form.type = 'string'
  form.category = ''
  form.description = ''
  form.value = ''
}

const cancelCreate = () => {
  mode.value = 'view'
}

const save = async () => {
  if (!form.key.trim() || !form.category.trim() || !form.description.trim()) {
    feedback.warning('请完整填写', 'key/category/description 为必填')
    return
  }

  await withErrorHandling(async () => {
    saving.value = true
    if (mode.value === 'create') {
      await service.create({
        key: form.key.trim(),
        value: form.value,
        type: form.type,
        category: form.category.trim(),
        description: form.description.trim()
      })
      feedback.success('创建成功')
      mode.value = 'view'
      await reload(false)
      await select(form.key.trim())
    } else {
      await service.update({ key: form.key, value: form.value })
      feedback.success('已更新')
      await select(form.key)
    }
  }, { title: '保存失败', loading: saving })
  saving.value = false
}

const removeSelected = async () => {
  const key = selectedKey.value
  if (!key) return
  const confirmed = await modal.showConfirm({
    title: '删除配置',
    message: `确定删除配置 ${key} 吗？此操作不可恢复。`,
    confirmText: '删除',
    confirmColor: 'error'
  })
  if (!confirmed) return

  await withErrorHandling(async () => {
    await service.delete({ key })
    feedback.success('已删除')
    selectedKey.value = null
    detail.value = null
    await reload(false)
  }, { title: '删除失败' })
}

// history
const showHistory = ref(false)
const historyLoading = ref(false)
const historyMoreLoading = ref(false)
const history = reactive<{ items: ConfigHistoryItem[]; pagination: { has_more: boolean; next_cursor?: string } }>({
  items: [],
  pagination: { has_more: false }
})

const historyColumns: TableColumn<ConfigHistoryItem>[] = [
  { id: 'changed_at', accessorKey: 'changed_at', header: '时间' },
  { id: 'changed_by', accessorKey: 'changed_by', header: '操作者' },
  { id: 'old_value', accessorKey: 'old_value', header: '旧值' },
  { id: 'new_value', accessorKey: 'new_value', header: '新值' }
]

const toggleHistory = () => {
  showHistory.value = !showHistory.value
  if (showHistory.value) {
    void loadHistory(true)
  }
}

const loadHistory = async (manual = false) => {
  const key = selectedKey.value
  if (!key) return
  await withErrorHandling(async () => {
    historyLoading.value = true
    const res = await service.history({ key, limit: 20 })
    history.items = res.items
    history.pagination = res.pagination
  }, { title: '加载历史失败', loading: historyLoading, showToast: manual })
  historyLoading.value = false
}

const loadHistoryMore = async () => {
  const key = selectedKey.value
  if (!key || !history.pagination.has_more || !history.pagination.next_cursor) return
  await withErrorHandling(async () => {
    historyMoreLoading.value = true
    const res = await service.history({ key, limit: 20, cursor: history.pagination.next_cursor })
    history.items = history.items.concat(res.items)
    history.pagination = res.pagination
  }, { title: '加载失败', loading: historyMoreLoading })
  historyMoreLoading.value = false
}

onMounted(() => {
  void reload(true)
})
</script>

