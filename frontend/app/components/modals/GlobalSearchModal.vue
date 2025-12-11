<template>
  <UModal
    title="输入你想搜索的用户/群组/动态"
    :ui="{
      width: 'max-w-2xl',
      content: 'p-0'
    }"
  >
    <template #body="{ close }">
      <article class="flex flex-col max-h-[70vh]">
        <!-- 搜索输入框 -->
        <header class="p-4 border-b border-default">
          <UInput
            v-model="searchQuery"
            icon="i-heroicons-magnifying-glass"
            size="lg"
            :placeholder="t('search.placeholder')"
            autofocus
            :loading="isSearching"
            @keydown.enter="handleSearch"
            @keydown.escape="close"
          />
        </header>

        <!-- 搜索类型切换 -->
        <nav class="px-4 py-2 border-b border-default">
          <UTabs
            v-model="activeTab"
            :items="tabItems"
            size="sm"
            class="-mb-px"
          />
        </nav>

        <!-- 搜索结果区域 -->
        <section class="flex-1 overflow-y-auto min-h-64">
          <!-- 加载状态 -->
          <article
            v-if="isSearching"
            class="p-8 flex flex-col items-center gap-3 text-secondary-content"
          >
            <UIcon
              name="i-heroicons-arrow-path"
              class="icon-lg loading-spin"
            />
            <p class="text-sm">搜索中...</p>
          </article>

          <!-- 空状态 - 未输入 -->
          <article
            v-else-if="!searchQuery.trim()"
            class="p-8 text-center text-secondary-content"
          >
            <UIcon
              name="i-heroicons-magnifying-glass"
              class="icon-2xl mx-auto mb-3 opacity-30"
            />
            <p class="text-sm font-medium">全局搜索</p>
            <p class="text-xs mt-1 opacity-70">搜索所有范围的用户、群组或瞬间</p>
            <section class="mt-4 flex flex-wrap justify-center gap-2 text-xs">
              <span class="flex items-center gap-1">
                <UKbd>Enter</UKbd>
                搜索
              </span>
              <span class="flex items-center gap-1">
                <UKbd>Esc</UKbd>
                关闭
              </span>
            </section>
          </article>

          <!-- 无结果 -->
          <UAlert
            v-else-if="hasSearched && !hasResults"
            icon="i-heroicons-face-frown"
            color="neutral"
            variant="subtle"
            class="m-4"
            :title="`未找到${searchQuery}相关结果`"
            description="尝试更换关键词或搜索类型"
          />

          <!-- 搜索结果列表 -->
          <ul
            v-else
            class="divide-y divide-default"
          >
            <!-- 用户结果 -->
            <template v-if="activeTab === 'users' && userResults.length > 0">
              <li
                v-for="user in userResults"
                :key="user.user_id"
                class="interactive-list-item flex items-center gap-3 p-4 cursor-pointer"
                @click="goToUser(user, close)"
              >
                <UAvatar
                  :alt="user.nickname"
                  :src="user.avatar_url"
                  size="md"
                >
                  {{ user.nickname?.charAt(0) }}
                </UAvatar>
                <article class="flex-1 min-w-0">
                  <p class="font-medium truncate">{{ user.nickname }}</p>
                  <p
                    v-if="user.geo.location_name"
                    class="text-xs text-secondary-content truncate flex items-center gap-1"
                  >
                    <UIcon
                      name="i-heroicons-map-pin"
                      class="icon-xs"
                    />
                    {{ user.geo.location_name }}
                  </p>
                </article>
                <UIcon
                  name="i-heroicons-chevron-right"
                  class="icon-md text-secondary-content"
                />
              </li>
            </template>

            <!-- 群组结果 -->
            <template v-if="activeTab === 'groups' && groupResults.length > 0">
              <li
                v-for="group in groupResults"
                :key="group.id"
                class="interactive-list-item flex items-center gap-3 p-4 cursor-pointer"
                @click="goToGroup(group, close)"
              >
                <UAvatar
                  :alt="group.name"
                  :src="group.avatar_url"
                  size="md"
                >
                  {{ group.name?.charAt(0) }}
                </UAvatar>
                <article class="flex-1 min-w-0">
                  <p class="font-medium truncate">{{ group.name }}</p>
                  <p class="text-xs text-secondary-content flex items-center gap-1">
                    <UIcon
                      name="i-heroicons-users"
                      class="icon-xs"
                    />
                    {{ group.members_count }} 成员
                  </p>
                </article>
                <UChip
                  v-if="isGroupMember(group.id)"
                  size="sm"
                  color="success"
                >
                  已加入
                </UChip>
                <UIcon
                  name="i-heroicons-chevron-right"
                  class="icon-md text-secondary-content"
                />
              </li>
            </template>

            <!-- 瞬间结果 -->
            <template v-if="activeTab === 'moments' && momentResults.length > 0">
              <li
                v-for="moment in momentResults"
                :key="moment.id"
                class="interactive-list-item flex items-start gap-3 p-4 cursor-pointer"
                @click="goToMoment(moment, close)"
              >
                <UAvatar
                  :alt="moment.user?.nickname"
                  :src="moment.user?.avatar_url"
                  size="sm"
                >
                  {{ moment.user?.nickname?.charAt(0) }}
                </UAvatar>
                <article class="flex-1 min-w-0">
                  <p class="text-sm text-secondary-content mb-1">{{ moment.user?.nickname }}</p>
                  <p class="text-sm line-clamp-2">{{ moment.description }}</p>
                  <p v-if="moment.geo.location_name" class="text-xs text-muted mt-1 flex items-center gap-1">
                    <UIcon
                      name="i-heroicons-map-pin"
                      class="icon-xs"
                    />
                    {{ moment.geo.location_name }}
                  </p>
                </article>
                <NuxtImg
                  v-if="moment.images?.[0]"
                  :src="moment.images[0]"
                  class="w-16 h-16 rounded-lg object-cover"
                  :alt="moment.description"
                />
              </li>
            </template>
          </ul>
        </section>

        <!-- 底部提示 -->
        <footer class="px-4 py-2 border-t border-subtle text-xs text-muted flex items-center justify-between">
          <span>{{ resultCountText }}</span>
          <section class="flex items-center gap-2">
            <span class="flex items-center gap-1">
              <UKbd>↑↓</UKbd>
              导航
            </span>
            <span class="flex items-center gap-1">
              <UKbd>Enter</UKbd>
              打开
            </span>
          </section>
        </footer>
      </article>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { UserSearchItem, GroupSearchItem, MomentSearchItem } from '~/types/api'
import { getErrorMessage } from '~/utils/error'
import { useUsersService } from '~/services/users.service'
import { useGroupsService } from '~/services/groups.service'
import { useMomentsService } from '~/services/moments.service'

const { t } = useI18n()
const toast = useToast()

const searchQuery = ref('')
const activeTab = ref('users')
const isSearching = ref(false)
const hasSearched = ref(false)

const userResults = ref<UserSearchItem[]>([])
const groupResults = ref<GroupSearchItem[]>([])
const momentResults = ref<MomentSearchItem[]>([])

const usersService = useUsersService()
const groupsService = useGroupsService()
const momentsService = useMomentsService()
const geolocation = useGeolocation()
const authStore = useAuthStore()

// 缓存已加入的群组 ID 集合
const joinedGroupIds = ref<Set<string>>(new Set())

// 加载已加入的群组（用于本地缓存检查）
const loadJoinedGroups = async () => {
  if (!authStore.isAuthenticated) return
  try {
    const response = await groupsService.getJoined({ limit: 100 })
    joinedGroupIds.value = new Set(response.items.map(g => g.id))
  } catch {
    // 加载失败时保持空集合
  }
}

// 检查是否已加入群组（使用本地缓存）
const isGroupMember = (groupId: string): boolean => {
  return joinedGroupIds.value.has(groupId)
}

// 组件挂载时加载已加入的群组
onMounted(() => {
  loadJoinedGroups()
})

const tabItems = computed(() => [
  { label: `用户${userResults.value.length > 0 ? ` (${userResults.value.length})` : ''}`, value: 'users', icon: 'i-heroicons-user' },
  { label: `群组${groupResults.value.length > 0 ? ` (${groupResults.value.length})` : ''}`, value: 'groups', icon: 'i-heroicons-user-group' },
  { label: `瞬间${momentResults.value.length > 0 ? ` (${momentResults.value.length})` : ''}`, value: 'moments', icon: 'i-heroicons-photo' }
])

const hasResults = computed(() =>
  userResults.value.length > 0 ||
  groupResults.value.length > 0 ||
  momentResults.value.length > 0
)

const resultCountText = computed(() => {
  if (!hasSearched.value) return ''
  const total = userResults.value.length + groupResults.value.length + momentResults.value.length
  return `找到 ${total} 个结果`
})

const handleSearch = async () => {
  const keyword = searchQuery.value.trim()
  if (!keyword) return

  isSearching.value = true
  hasSearched.value = true

  try {
    const location = await geolocation.getCoordinates()

    const [users, groups, moments] = await Promise.all([
      usersService.searchByKeyword({ kw: keyword, latitude: location.latitude, longitude: location.longitude, limit: 20 }).catch(() => ({ items: [], pagination: { has_more: false } })),
      groupsService.searchByKeyword({ kw: keyword, latitude: location.latitude, longitude: location.longitude, limit: 20 }).catch(() => ({ items: [], pagination: { has_more: false } })),
      momentsService.searchByKeyword({ kw: keyword, latitude: location.latitude, longitude: location.longitude, limit: 20 }).catch(() => ({ items: [], pagination: { has_more: false } }))
    ])

    userResults.value = users.items || []
    groupResults.value = groups.items || []
    momentResults.value = moments.items || []

    if (userResults.value.length > 0) activeTab.value = 'users'
    else if (groupResults.value.length > 0) activeTab.value = 'groups'
    else if (momentResults.value.length > 0) activeTab.value = 'moments'
  } catch (err: unknown) {
    console.error('搜索失败:', err)
    toast.add({ title: '搜索失败', description: getErrorMessage(err, '请稍后重试'), color: 'error' })
  } finally {
    isSearching.value = false
  }
}

const goToUser = (user: UserSearchItem, close: () => void) => {
  close()
  // 如果是当前用户，直接跳转到个人主页
  if (user.user_id === authStore.user?.user_id) {
    navigateTo('/profile')
  } else {
    navigateTo(`/users/${user.user_id}`)
  }
}

const goToGroup = (group: GroupSearchItem, close: () => void) => {
  close()
  navigateTo(`/groups/${group.id}`)
}

const goToMoment = (moment: MomentSearchItem, close: () => void) => {
  close()
  navigateTo(`/moments/${moment.id}`)
}
</script>
