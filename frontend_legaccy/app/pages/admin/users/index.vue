<template>
  <div class="flex flex-col gap-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">用户管理</h1>
        <p class="text-neutral-500 dark:text-neutral-400 mt-1">管理平台用户账户</p>
      </div>
    </div>

    <!-- Search & Filters -->
    <UCard variant="outline">
      <div class="flex flex-col md:flex-row gap-4">
        <UInput
          v-model="searchKeyword"
          placeholder="搜索用户昵称、邮箱、手机号..."
          icon="i-lucide-search"
          class="flex-1"
          @keyup.enter="handleSearch"
        />
        <USelectMenu
          v-model="statusFilter"
          :items="statusOptions"
          placeholder="账户状态"
          class="w-40"
        />
        <USelectMenu
          v-model="typeFilter"
          :items="typeOptions"
          placeholder="用户类型"
          class="w-40"
        />
        <UButton icon="i-lucide-search" @click="handleSearch">搜索</UButton>
      </div>
    </UCard>

    <!-- Users Table -->
    <UCard variant="outline">
      <UTable
        :data="users"
        :columns="columns"
        :loading="loading"
        :empty-state="{ icon: 'i-lucide-users', label: '暂无用户数据' }"
      >
        <template #user_id-cell="{ row }">
          <span class="font-mono text-sm">{{ row.original.user_id }}</span>
        </template>

        <template #nickname-cell="{ row }">
          <div class="flex items-center gap-2">
            <UAvatar
              :src="row.original.avatar_url"
              :alt="row.original.nickname"
              size="sm"
            />
            <span class="font-medium">{{ row.original.nickname }}</span>
          </div>
        </template>

        <template #user_type-cell="{ row }">
          <UBadge :color="getUserTypeColor(row.original.user_type)" variant="soft">
            {{ getUserTypeLabel(row.original.user_type) }}
          </UBadge>
        </template>

        <template #status-cell="{ row }">
          <UBadge :color="row.original.status === 'normal' ? 'success' : 'error'" variant="soft">
            {{ row.original.status === 'normal' ? '正常' : '已封禁' }}
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
              @click="viewUser(row.original)"
            />
            <UButton
              v-if="row.original.status === 'normal'"
              variant="ghost"
              color="error"
              icon="i-lucide-ban"
              size="sm"
              @click="confirmBan(row.original)"
            />
            <UButton
              v-else
              variant="ghost"
              color="success"
              icon="i-lucide-check-circle"
              size="sm"
              @click="confirmUnban(row.original)"
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

    <!-- User Detail Modal -->
    <UModal v-model:open="showDetailModal">
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-user" />
          <span>用户详情</span>
        </div>
      </template>
      <template #body>
        <div v-if="selectedUser" class="space-y-4">
          <div class="flex items-center gap-4">
            <UAvatar
              :src="selectedUser.avatar_url"
              :alt="selectedUser.nickname"
              size="xl"
            />
            <div>
              <h3 class="text-lg font-semibold">{{ selectedUser.nickname }}</h3>
              <p class="text-sm text-neutral-500">ID: {{ selectedUser.user_id }}</p>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span class="text-neutral-500">邮箱</span>
              <p>{{ selectedUser.email || '-' }}</p>
            </div>
            <div>
              <span class="text-neutral-500">手机</span>
              <p>{{ selectedUser.phone || '-' }}</p>
            </div>
            <div>
              <span class="text-neutral-500">用户类型</span>
              <p>{{ getUserTypeLabel(selectedUser.user_type) }}</p>
            </div>
            <div>
              <span class="text-neutral-500">账户状态</span>
              <p>{{ selectedUser.status === 'normal' ? '正常' : '已封禁' }}</p>
            </div>
            <div>
              <span class="text-neutral-500">动态数</span>
              <p>{{ selectedUser.moments_count }}</p>
            </div>
            <div>
              <span class="text-neutral-500">群组数</span>
              <p>{{ selectedUser.groups_count }}</p>
            </div>
            <div>
              <span class="text-neutral-500">注册时间</span>
              <p>{{ formatTime(selectedUser.created_at) }}</p>
            </div>
            <div>
              <span class="text-neutral-500">最后登录</span>
              <p>{{ selectedUser.last_login_at ? formatTime(selectedUser.last_login_at) : '-' }}</p>
            </div>
          </div>

          <div v-if="selectedUser.bio" class="pt-2 border-t border-neutral-200 dark:border-neutral-700">
            <span class="text-neutral-500 text-sm">个人简介</span>
            <p class="mt-1">{{ selectedUser.bio }}</p>
          </div>

          <div v-if="selectedUser.status === 'banned'" class="pt-2 border-t border-neutral-200 dark:border-neutral-700 bg-red-50 dark:bg-red-900/20 p-3 rounded-lg">
            <div class="flex items-center gap-2 text-red-600 dark:text-red-400">
              <UIcon name="i-lucide-alert-triangle" />
              <span class="font-medium">封禁信息</span>
            </div>
            <p class="text-sm mt-2">封禁时间: {{ selectedUser.banned_at ? formatTime(selectedUser.banned_at) : '-' }}</p>
            <p class="text-sm">封禁原因: {{ selectedUser.banned_reason || '无' }}</p>
          </div>
        </div>
      </template>
    </UModal>

    <!-- Ban Confirm Modal -->
    <UModal v-model:open="showBanModal">
      <template #header>
        <div class="flex items-center gap-2 text-red-600">
          <UIcon name="i-lucide-ban" />
          <span>封禁用户</span>
        </div>
      </template>
      <template #body>
        <div class="space-y-4">
          <p>确定要封禁用户 <strong>{{ userToBan?.nickname }}</strong> 吗？</p>
          <UTextarea
            v-model="banReason"
            placeholder="请输入封禁原因（可选）"
            :rows="3"
          />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="showBanModal = false">取消</UButton>
          <UButton color="error" :loading="banning" @click="doBan">确认封禁</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { useAdminUsersService } from '~/services/admin_users.service'
import type { AdminUserItem, AdminUserDetailData, UserType } from '~/types/api'

definePageMeta({
  layout: 'dashboard',
  requiresAuth: true,
  middleware: ['admin']
})

const adminUsersService = useAdminUsersService()
const { formatTime } = useFormatters()
const toast = useToast()

// State
const users = ref<AdminUserItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const cursor = ref<string | undefined>()

const searchKeyword = ref('')
const statusFilter = ref('')
const typeFilter = ref('')

const showDetailModal = ref(false)
const selectedUser = ref<AdminUserDetailData | null>(null)

const showBanModal = ref(false)
const userToBan = ref<AdminUserItem | null>(null)
const banReason = ref('')
const banning = ref(false)

// Options
const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '正常', value: 'normal' },
  { label: '已封禁', value: 'banned' }
]

const typeOptions = [
  { label: '全部类型', value: '' },
  { label: '普通用户', value: 'normal' },
  { label: '访客', value: 'temp' },
  { label: '管理员', value: 'admin' }
]

// Table columns
const columns = [
  { accessorKey: 'user_id', header: 'ID' },
  { accessorKey: 'nickname', header: '用户' },
  { accessorKey: 'user_type', header: '类型' },
  { accessorKey: 'status', header: '状态' },
  { accessorKey: 'created_at', header: '注册时间' },
  { accessorKey: 'actions', header: '操作' }
]

// Methods
function getUserTypeColor(type: UserType) {
  switch (type) {
    case 'admin': return 'primary'
    case 'normal': return 'success'
    case 'temp': return 'warning'
    case 'system': return 'neutral'
    default: return 'neutral'
  }
}

function getUserTypeLabel(type: UserType) {
  switch (type) {
    case 'admin': return '管理员'
    case 'normal': return '普通用户'
    case 'temp': return '访客'
    case 'system': return '系统'
    default: return type
  }
}

async function fetchUsers(reset = false) {
  if (reset) {
    cursor.value = undefined
    users.value = []
  }

  loading.value = reset
  loadingMore.value = !reset

  try {
    const res = await adminUsersService.search({
      kw: searchKeyword.value || undefined,
      status: statusFilter.value as 'normal' | 'banned' || undefined,
      user_type: typeFilter.value as UserType || undefined,
      cursor: cursor.value,
      limit: 20
    })
    users.value = reset ? res.items : [...users.value, ...res.items]
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
  fetchUsers(true)
}

function loadMore() {
  fetchUsers(false)
}

async function viewUser(user: AdminUserItem) {
  try {
    selectedUser.value = await adminUsersService.detail({ user_id: user.user_id })
    showDetailModal.value = true
  } catch (e) {
    toast.add({ title: '获取详情失败', color: 'error' })
  }
}

function confirmBan(user: AdminUserItem) {
  userToBan.value = user
  banReason.value = ''
  showBanModal.value = true
}

async function doBan() {
  if (!userToBan.value) return
  banning.value = true
  try {
    await adminUsersService.ban({
      user_id: userToBan.value.user_id,
      reason: banReason.value || undefined
    })
    toast.add({ title: '封禁成功', color: 'success' })
    showBanModal.value = false
    fetchUsers(true)
  } catch (e) {
    toast.add({ title: '封禁失败', color: 'error' })
  } finally {
    banning.value = false
  }
}

async function confirmUnban(user: AdminUserItem) {
  try {
    await adminUsersService.unban({ user_id: user.user_id })
    toast.add({ title: '解封成功', color: 'success' })
    fetchUsers(true)
  } catch (e) {
    toast.add({ title: '解封失败', color: 'error' })
  }
}

onMounted(() => {
  fetchUsers(true)
})
</script>
