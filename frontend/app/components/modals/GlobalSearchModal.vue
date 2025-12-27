<template>
  <UModal
    :title="t('globalSearch.title')"
    :description="t('globalSearch.description')"
    scrollable
  >
    <template #body="{ close }">
      <UCommandPalette
        v-model:search-term="searchTerm"
        :loading="isSearching"
        :groups="groups"
        :placeholder="t('globalSearch.placeholder')"
        close
        :class="paletteSizeClass"
        @update:open="(open) => { if (!open) close() }"
        @update:model-value="(item) => handleSelect(item, close)"
      >
        <template #empty>
          <UEmpty
            v-if="!searchTerm.trim()"
            icon="i-lucide-search"
            title="全局搜索"
            description="输入关键词，搜索用户、群组与瞬间"
          >
            <template #actions>
              <div class="hidden sm:flex flex-wrap items-center justify-center gap-2">
                <div class="flex items-center gap-1">
                  <UKbd value="enter" />
                  <span>搜索</span>
                </div>
                <div class="flex items-center gap-1">
                  <UKbd value="esc" />
                  <span>关闭</span>
                </div>
              </div>
            </template>
          </UEmpty>
          <UEmpty
            v-else
            icon="i-lucide-frown"
            :title="hasSearched ? `未找到「${searchTerm.trim()}」相关结果` : '正在准备搜索…'"
            :description="hasSearched ? '尝试更换关键词' : '继续输入关键词以开始搜索'"
          />
        </template>

        <template #footer>
          <div v-if="hasSearched" class="flex items-center justify-between gap-2 text-xs">
            <span>{{ resultCountText }}</span>
            <div class="hidden sm:flex items-center gap-2">
              <div class="flex items-center gap-1">
                <UKbd value="up" />
                <UKbd value="down" />
                <span>选择</span>
              </div>
              <div class="flex items-center gap-1">
                <UKbd value="enter" />
                <span>打开</span>
              </div>
            </div>
          </div>
        </template>
      </UCommandPalette>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { AvatarProps, ChipProps, CommandPaletteGroup, CommandPaletteItem } from '@nuxt/ui'
import type { UserSearchItem, GroupSearchItem, MomentSearchItem } from '~/types/api'
import { useUsersService } from '~/services/users.service'
import { useGroupsService } from '~/services/groups.service'
import { useMomentsService } from '~/services/moments.service'
import { useFeedbackStore } from '~/stores/feedback.store'

const { t } = useI18n()
const feedback = useFeedbackStore()

type GlobalSearchEntity = 'user' | 'group' | 'moment'
type GlobalSearchPaletteItem = CommandPaletteItem & { _type: GlobalSearchEntity; _id: string }

const searchTerm = ref('')
const isSearching = ref(false)
const hasSearched = ref(false)

// 不使用固定高度：用 max-height 控制上限，让内容少时自然收缩，内容多时再滚动（避免大块空白）
const paletteSizeClass = computed(() => {
  return hasSearched.value
    ? 'max-h-[min(70vh,32rem)]'
    : 'max-h-[min(60vh,28rem)]'
})

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

const searchSeq = ref(0)
const coords = shallowRef<{ latitude: number; longitude: number } | null>(null)

const doSearch = async (keyword: string, opts?: { silent?: boolean }) => {
  const kw = keyword.trim()
  if (!kw) {
    hasSearched.value = false
    userResults.value = []
    groupResults.value = []
    momentResults.value = []
    return
  }

  const seq = ++searchSeq.value
  isSearching.value = true
  hasSearched.value = true

  try {
    if (!coords.value) coords.value = await geolocation.getCoordinates()
    const location = coords.value
    if (!location) return

    const [users, groups, moments] = await Promise.all([
      usersService.searchByKeyword({ kw, latitude: location.latitude, longitude: location.longitude, limit: 20 }).catch(() => ({ items: [], pagination: { has_more: false } })),
      groupsService.searchByKeyword({ kw, latitude: location.latitude, longitude: location.longitude, limit: 20 }).catch(() => ({ items: [], pagination: { has_more: false } })),
      momentsService.searchByKeyword({ kw, latitude: location.latitude, longitude: location.longitude, limit: 20 }).catch(() => ({ items: [], pagination: { has_more: false } }))
    ])

    if (seq !== searchSeq.value) return
    userResults.value = users.items || []
    groupResults.value = groups.items || []
    momentResults.value = moments.items || []
  } catch (err: unknown) {
    if (!opts?.silent) {
      console.error('搜索失败:', err)
      feedback.error('搜索失败', getErrorMessage(err, '请稍后重试'))
    }
    if (seq === searchSeq.value) {
      userResults.value = []
      groupResults.value = []
      momentResults.value = []
    }
  } finally {
    if (seq === searchSeq.value) isSearching.value = false
  }
}

const debouncedSearch = useDebounceFn(() => {
  void doSearch(searchTerm.value, { silent: true })
}, 250)

watch(searchTerm, () => {
  debouncedSearch()
})

const memberChip: ChipProps = { color: 'success', size: 'sm' }

const groups = computed<CommandPaletteGroup<CommandPaletteItem>[]>(() => {
  const kw = searchTerm.value.trim()
  const users: CommandPaletteItem[] = userResults.value.map((u) => {
    const avatar: AvatarProps = { src: u.avatar_url || undefined, alt: u.nickname || undefined }
    return {
      _type: 'user',
      _id: String(u.user_id),
      label: u.nickname || `用户 ${u.user_id}`,
      suffix: u.geo?.location_name || '',
      avatar
    } satisfies GlobalSearchPaletteItem
  })

  const groupsItems: CommandPaletteItem[] = groupResults.value.map((g) => {
    const avatar: AvatarProps = { src: g.avatar_url || undefined, alt: g.name || undefined }
    return {
      _type: 'group',
      _id: g.id,
      label: g.name || '群组',
      suffix: `${g.members_count ?? 0} 成员`,
      avatar,
      chip: isGroupMember(g.id) ? memberChip : undefined
    } satisfies GlobalSearchPaletteItem
  })

  const moments: CommandPaletteItem[] = momentResults.value.map((m) => {
    const avatar: AvatarProps = { src: m.user?.avatar_url || undefined, alt: m.user?.nickname || undefined }
    return {
      _type: 'moment',
      _id: m.id,
      label: m.description || '瞬间',
      prefix: m.user?.nickname || '',
      suffix: m.geo?.location_name || '',
      avatar
    } satisfies GlobalSearchPaletteItem
  })

  return [
    { id: 'users', label: kw ? `用户（${users.length}）` : '用户', items: users, ignoreFilter: true },
    { id: 'groups', label: kw ? `群组（${groupsItems.length}）` : '群组', items: groupsItems, ignoreFilter: true },
    { id: 'moments', label: kw ? `瞬间（${moments.length}）` : '瞬间', items: moments, ignoreFilter: true }
  ]
})

const handleSelect = (item: CommandPaletteItem, close: () => void) => {
  const it = item as Partial<GlobalSearchPaletteItem>
  if (!it._type || !it._id) return
  close()

  if (it._type === 'user') {
    const uid = Number(it._id)
    if (!Number.isNaN(uid) && uid === authStore.user?.user_id) return void navigateTo('/profile')
    return void navigateTo(`/users/${it._id}`)
  }
  if (it._type === 'group') return void navigateTo(`/groups/${it._id}`)
  return void navigateTo(`/moments/${it._id}`)
}
</script>
