<template>
  <div class="flex flex-col gap-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">群组管理</h1>
        <p class="text-neutral-500 dark:text-neutral-400 mt-1">管理平台群组</p>
      </div>
    </div>

    <!-- Search -->
    <UCard variant="outline">
      <div class="flex flex-col md:flex-row gap-4">
        <UInput
          v-model="searchKeyword"
          placeholder="搜索群组名称..."
          icon="i-lucide-search"
          class="flex-1"
          @keyup.enter="handleSearch"
        />
        <UButton icon="i-lucide-search" @click="handleSearch">搜索</UButton>
      </div>
    </UCard>

    <!-- Groups Table -->
    <UCard variant="outline">
      <UTable
        :data="groups"
        :columns="columns"
        :loading="loading"
        :empty-state="{ icon: 'i-lucide-users-2', label: '暂无群组数据' }"
      >
        <template #group_chat_id-cell="{ row }">
          <span class="font-mono text-sm truncate max-w-[120px]" :title="row.original.group_chat_id">
            {{ row.original.group_chat_id.slice(0, 8) }}...
          </span>
        </template>

        <template #name-cell="{ row }">
          <div class="flex items-center gap-2">
            <UAvatar
              :src="row.original.avatar_url"
              :alt="row.original.name"
              size="sm"
            />
            <span class="font-medium">{{ row.original.name }}</span>
          </div>
        </template>

        <template #owner-cell="{ row }">
          <div v-if="row.original.owner" class="flex items-center gap-2">
            <UAvatar
              :src="row.original.owner.avatar_url"
              :alt="row.original.owner.nickname"
              size="xs"
            />
            <span class="text-sm">{{ row.original.owner.nickname }}</span>
          </div>
          <span v-else class="text-neutral-500">-</span>
        </template>

        <template #members_count-cell="{ row }">
          <UBadge variant="soft" color="neutral">
            {{ row.original.members_count }} 人
          </UBadge>
        </template>

        <template #created_at-cell="{ row }">
          <span class="text-sm text-neutral-500">{{ formatTime(row.original.created_at) }}</span>
        </template>

        <template #actions-cell="{ row }">
          <div class="flex items-center gap-2">
            <UButton
              variant="ghost"
              color="neutral"
              icon="i-lucide-eye"
              size="sm"
              @click="viewGroup(row.original)"
            />
            <UButton
              variant="ghost"
              color="error"
              icon="i-lucide-trash-2"
              size="sm"
              @click="confirmDismiss(row.original)"
            />
          </div>
        </template>
      </UTable>

      <!-- Pagination -->
      <div v-if="hasMore" class="flex justify-center pt-4">
        <UButton variant="ghost" :loading="loadingMore" @click="loadMore">
          加载更多
        </UButton>
      </div>
    </UCard>

    <!-- Group Detail Modal -->
    <UModal v-model:open="showDetailModal">
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-users-2" />
          <span>群组详情</span>
        </div>
      </template>
      <template #body>
        <div v-if="selectedGroup" class="space-y-4">
          <div class="flex items-center gap-4">
            <UAvatar
              :src="selectedGroup.avatar_url"
              :alt="selectedGroup.name"
              size="xl"
            />
            <div>
              <h3 class="text-lg font-semibold">{{ selectedGroup.name }}</h3>
              <p class="text-sm text-neutral-500">ID: {{ selectedGroup.group_chat_id }}</p>
            </div>
          </div>

          <div v-if="selectedGroup.description" class="text-sm text-neutral-600 dark:text-neutral-400">
            {{ selectedGroup.description }}
          </div>

          <div class="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span class="text-neutral-500">群主</span>
              <div v-if="selectedGroup.owner" class="flex items-center gap-2 mt-1">
                <UAvatar :src="selectedGroup.owner.avatar_url" size="xs" />
                <span>{{ selectedGroup.owner.nickname }}</span>
              </div>
            </div>
            <div>
              <span class="text-neutral-500">成员数</span>
              <p>{{ selectedGroup.members_count }} 人</p>
            </div>
            <div>
              <span class="text-neutral-500">创建时间</span>
              <p>{{ formatTime(selectedGroup.created_at) }}</p>
            </div>
            <div>
              <span class="text-neutral-500">加入方式</span>
              <p>{{ getJoinMethodLabel(selectedGroup.settings?.join_method) }}</p>
            </div>
          </div>

          <div v-if="selectedGroup.tags?.length" class="pt-2 border-t border-neutral-200 dark:border-neutral-700">
            <span class="text-neutral-500 text-sm">标签</span>
            <div class="flex flex-wrap gap-2 mt-2">
              <UBadge v-for="tag in selectedGroup.tags" :key="tag" variant="soft" color="neutral">
                {{ tag }}
              </UBadge>
            </div>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Dismiss Confirm Modal -->
    <UModal v-model:open="showDismissModal">
      <template #header>
        <div class="flex items-center gap-2 text-red-600">
          <UIcon name="i-lucide-alert-triangle" />
          <span>解散群组</span>
        </div>
      </template>
      <template #body>
        <div class="space-y-4">
          <p>确定要解散群组 <strong>{{ groupToDismiss?.name }}</strong> 吗？</p>
          <p class="text-sm text-neutral-500">此操作不可撤销，群组内所有成员将被移除。</p>
          <UTextarea
            v-model="dismissReason"
            placeholder="请输入解散原因（可选）"
            :rows="3"
          />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="showDismissModal = false">取消</UButton>
          <UButton color="error" :loading="dismissing" @click="doDismiss">确认解散</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { useAdminGroupsService } from '~/services/admin_groups.service'
import type { AdminGroupItem, AdminGroupDetailData } from '~/types/api'

definePageMeta({
  layout: 'dashboard',
  requiresAuth: true,
  middleware: ['admin']
})

const adminGroupsService = useAdminGroupsService()
const { formatTime } = useFormatters()
const toast = useToast()

// State
const groups = ref<AdminGroupItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const cursor = ref<string | undefined>()

const searchKeyword = ref('')

const showDetailModal = ref(false)
const selectedGroup = ref<AdminGroupDetailData | null>(null)

const showDismissModal = ref(false)
const groupToDismiss = ref<AdminGroupItem | null>(null)
const dismissReason = ref('')
const dismissing = ref(false)

// Table columns
const columns = [
  { accessorKey: 'group_chat_id', header: 'ID' },
  { accessorKey: 'name', header: '群组名' },
  { accessorKey: 'owner', header: '群主' },
  { accessorKey: 'members_count', header: '成员' },
  { accessorKey: 'created_at', header: '创建时间' },
  { accessorKey: 'actions', header: '操作' }
]

// Methods
function getJoinMethodLabel(method?: string) {
  switch (method) {
    case 'open': return '开放加入'
    case 'approval': return '需要审批'
    case 'invite_only': return '仅邀请'
    default: return '开放加入'
  }
}

async function fetchGroups(reset = false) {
  if (reset) {
    cursor.value = undefined
    groups.value = []
  }

  loading.value = reset
  loadingMore.value = !reset

  try {
    const res = await adminGroupsService.search({
      kw: searchKeyword.value || undefined,
      cursor: cursor.value,
      limit: 20
    })
    groups.value = reset ? res.items : [...groups.value, ...res.items]
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
  fetchGroups(true)
}

function loadMore() {
  fetchGroups(false)
}

async function viewGroup(group: AdminGroupItem) {
  try {
    selectedGroup.value = await adminGroupsService.detail({ group_chat_id: group.group_chat_id })
    showDetailModal.value = true
  } catch (e) {
    toast.add({ title: '获取详情失败', color: 'error' })
  }
}

function confirmDismiss(group: AdminGroupItem) {
  groupToDismiss.value = group
  dismissReason.value = ''
  showDismissModal.value = true
}

async function doDismiss() {
  if (!groupToDismiss.value) return
  dismissing.value = true
  try {
    await adminGroupsService.dismiss({
      group_chat_id: groupToDismiss.value.group_chat_id,
      reason: dismissReason.value || undefined
    })
    toast.add({ title: '解散成功', color: 'success' })
    showDismissModal.value = false
    fetchGroups(true)
  } catch (e) {
    toast.add({ title: '解散失败', color: 'error' })
  } finally {
    dismissing.value = false
  }
}

onMounted(() => {
  fetchGroups(true)
})
</script>
