<template>
  <div class="w-full">
    <UCard variant="subtle">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <UIcon name="i-lucide-user-plus" :size="18" />
            <span class="text-sm font-semibold">关注列表</span>
          </div>
        </div>
      </template>

      <div v-if="loading" class="space-y-3">
        <USkeleton v-for="i in 3" :key="i" class="h-20 w-full" />
      </div>

      <UEmpty
        v-else-if="items.length === 0"
        icon="i-lucide-user-plus"
        title="暂无关注"
        description="去搜索并关注一些你感兴趣的人吧"
        class="py-12"
      >
        <template #actions>
          <UButton color="primary" @click="navigateTo('/search')">去搜索</UButton>
        </template>
      </UEmpty>

      <div v-else class="space-y-3">
        <UserCard
          v-for="it in items"
          :key="it.user.user_id"
          :user="toUserCardData(it)"
          @click="openUser(it.user.user_id)"
          @message="openChat(it.user.user_id)"
        />

        <div v-if="pagination.has_more" class="pt-2">
          <UButton
            variant="soft"
            color="neutral"
            :loading="loadingMore"
            :disabled="loadingMore"
            block
            @click="loadMore"
          >
            加载更多
          </UButton>
        </div>
      </div>
    </UCard>
  </div>
</template>

<script setup lang="ts">
import type { FollowingItem } from '~/types/api'
import { useUsersService } from '~/services/users.service'
import { useDashboardHeader } from '~/composables/useDashboardHeader'

definePageMeta({
  title: '我的关注',
  description: '我关注的用户',
  requiresAuth: true,
  layout: 'dashboard',
  dashboardHeader: true
})

const userService = useUsersService()
const dashboardHeader = useDashboardHeader()

const loading = ref(true)
const loadingMore = ref(false)
const items = ref<FollowingItem[]>([])
const cursor = ref<string | undefined>(undefined)
const pagination = reactive({ has_more: false as boolean, next_cursor: undefined as string | undefined })

watchEffect(() => {
  dashboardHeader.setActions([
    { key: 'refresh', label: '刷新', icon: 'i-lucide-refresh-cw', variant: 'outline', loading: loading.value, action: () => reload() }
  ])
})

function toUserCardData(it: FollowingItem) {
  return {
    ...it.user,
    relationship: {
      is_following: true,
      is_mutual_follow: it.is_mutual
    }
  }
}

async function reload() {
  if (loading.value) return
  cursor.value = undefined
  await load(true)
}

async function load(reset = false) {
  if (reset) items.value = []
  loading.value = true
  try {
    const res = await userService.getFollowing({ limit: 20, cursor: cursor.value })
    items.value = res.items
    pagination.has_more = res.pagination.has_more
    pagination.next_cursor = res.pagination.next_cursor
    cursor.value = res.pagination.next_cursor
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  if (loadingMore.value) return
  if (!pagination.has_more || !pagination.next_cursor) return
  loadingMore.value = true
  try {
    const res = await userService.getFollowing({ limit: 20, cursor: pagination.next_cursor })
    items.value.push(...res.items)
    pagination.has_more = res.pagination.has_more
    pagination.next_cursor = res.pagination.next_cursor
  } finally {
    loadingMore.value = false
  }
}

function openUser(userId: number) {
  navigateTo(`/users/${userId}`)
}

function openChat(userId: number) {
  navigateTo(`/chat/users/${userId}`)
}

onMounted(() => {
  void load(true)
})
</script>

