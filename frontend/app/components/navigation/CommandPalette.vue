<template>
  <!-- 全局命令面板 - 必须用 UModal 包装以符合 useOverlay 管理要求 -->
  <UModal>
    <template #body>
      <UCommandPalette
        v-model:search-term="searchTerm"
        :groups="commandGroups"
        :loading="isLoading"
        placeholder="搜索用户、群组、动态... (输入 '/' 查看高级选项)"
        icon="i-heroicons-magnifying-glass"
        :close="{
          color: 'neutral',
          variant: 'ghost',
          size: 'sm'
        }"
        class="h-[500px] max-h-[70vh] border-none shadow-none"
        @update:model-value="handleSelect"
      >
        <template #footer>
          <div class="flex items-center justify-between gap-2 p-2">
            <div class="flex items-center gap-1 text-xs text-neutral-500">
              <UIcon name="i-heroicons-fire" class="size-4 text-primary" />
              <span>HotSpot 全局搜索</span>
            </div>
            <div class="flex items-center gap-1">
              <UButton 
                color="neutral" 
                variant="ghost" 
                size="xs"
                class="text-neutral-500"
              >
                <template #trailing>
                  <span class="text-xs text-neutral-500 mr-1">选择</span>
                  <UKbd value="enter" />
                </template>
              </UButton>
              <USeparator orientation="vertical" class="h-3" />
              <UButton 
                color="neutral" 
                variant="ghost" 
                size="xs" 
                class="text-neutral-500"
                @click="handleClose"
              >
                <template #trailing>
                  <span class="text-xs text-neutral-500 mr-1">关闭</span>
                  <UKbd value="esc" />
                </template>
              </UButton>
            </div>
          </div>
        </template>
      </UCommandPalette>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useDebounceFn } from "@vueuse/core";
import { useUserService } from "~/services/users.service";
import { useGroupService } from "~/services/groups.service";
import { useMomentService } from "~/services/moments.service";

// 使用兼容的类型定义
type CustomCommandPaletteItem = {
  id?: string
  label?: string
  suffix?: string
  icon?: string
  avatar?: {
    src?: string
    fallback?: string
  }
  chip?: {
    text?: string | number
    color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'
  }
  onSelect?: (e?: Event) => void
  to?: string | any // 允许任何路由类型
}

interface CommandPaletteGroup {
  id: string
  label: string
  items: CustomCommandPaletteItem[]
}

interface Props {
  onCreateMoment?: () => void
}

const props = withDefaults(defineProps<Props>(), {})
const { formatTime } = useFormatters()

// 服务实例  
const userService = useUserService()
const groupService = useGroupService()
const momentService = useMomentService()

// 响应式状态

const searchTerm = ref('')
const isLoading = ref(false)

// 默认快速操作组
const quickActionsGroup: CommandPaletteGroup = {
  id: 'quick-actions',
  label: '快速操作',
  items: [
    {
      id: 'create-group',
      label: '创建群组',
      suffix: '创建一个新的群组',
      icon: 'i-heroicons-user-group',
      onSelect: () => {
        handleClose()
        navigateTo('/groups/create')
      }
    },
    {
      id: 'create-moment',
      label: '发布动态',
      suffix: '分享你的精彩瞬间',
      icon: 'i-heroicons-camera',
      onSelect: () => {
        handleClose()
        emit('create-moment')
        props.onCreateMoment?.()
      }
    },
    {
      id: 'users-nearby', 
      label: '附近用户',
      suffix: '发现周围的朋友',
      icon: 'i-heroicons-users',
      onSelect: () => {
        handleClose()
        navigateTo('/users/nearby')
      }
    },
    {
      id: 'search-users',
      label: '搜索用户',
      suffix: '按用户类型、在线状态筛选',
      icon: 'i-heroicons-user-circle',
      onSelect: () => {
        handleClose()
        navigateTo('/search')
      }
    },
    {
      id: 'map-explore',
      label: '地图探索', 
      suffix: '在地图上发现附近内容',
      icon: 'i-heroicons-map',
      onSelect: () => {
        handleClose()
        navigateTo('/map')
      }
    }
  ]
}

// 高级命令组
const advancedCommandsGroup: CommandPaletteGroup = {
  id: 'advanced-commands',
  label: '高级命令',
  items: [
    {
      id: 'users-online',
      label: '在线用户',
      suffix: '查看当前在线的用户',
      icon: 'i-heroicons-signal',
      onSelect: () => {
        handleClose()
        navigateTo('/search?status=online')
      }
    },
    {
      id: 'groups-public',
      label: '公开群组',
      suffix: '浏览所有公开群组',
      icon: 'i-heroicons-globe-alt',
      onSelect: () => {
        handleClose()
        navigateTo('/groups?filter=public')
      }
    },
    {
      id: 'moments-trending',
      label: '热门动态',
      suffix: '查看最受欢迎的动态',
      icon: 'i-heroicons-fire',
      onSelect: () => {
        handleClose()
        navigateTo('/moments?sort=trending')
      }
    },
    {
      id: 'settings-profile',
      label: '个人资料',
      suffix: '编辑个人信息和设置',
      icon: 'i-heroicons-cog-6-tooth',
      onSelect: () => {
        handleClose()
        navigateTo('/profile')
      }
    }
  ]
}

// 动态搜索结果组
const searchResultGroups = reactive<Record<string, CommandPaletteGroup>>({
  users: {
    id: 'users',
    label: '用户',
    items: []
  },
  groups: {
    id: 'groups', 
    label: '群组',
    items: []
  },
  moments: {
    id: 'moments',
    label: '动态',
    items: []
  }
})

// 合并命令组
const commandGroups = computed(() => {
  const groups: CommandPaletteGroup[] = []
  
  // 显示高级命令（当输入 '/' 时）
  if (searchTerm.value.startsWith('/')) {
    const query = searchTerm.value.toLowerCase()
    const filteredAdvancedItems = advancedCommandsGroup.items.filter(item => {
      if (query === '/') return true
      return item.label?.toLowerCase().includes(query.slice(1)) || 
             item.suffix?.toLowerCase().includes(query.slice(1))
    })
    
    if (filteredAdvancedItems.length > 0) {
      groups.push({
        ...advancedCommandsGroup,
        label: query === '/' ? '所有命令' : '匹配的命令',
        items: filteredAdvancedItems
      })
    }
  } else {
    // 默认显示快速操作
    groups.push(quickActionsGroup)
    
    // 添加搜索结果（仅当有搜索词且有结果时）
    if (searchTerm.value.trim()) {
      Object.values(searchResultGroups).forEach(group => {
        if (group.items.length > 0) {
          groups.push(group)
        }
      })
    }
  }
  
  return groups
})

// 防抖搜索函数
const debouncedSearch = useDebounceFn(async (query: string) => {
  if (!query.trim() || query.startsWith('/')) {
    // 清空搜索结果
    Object.values(searchResultGroups).forEach(group => {
      group.items = []
    })
    return
  }
  
  isLoading.value = true
  
  try {
    // 并行搜索所有类型
    const [usersResponse, groupsResponse, momentsResponse] = await Promise.allSettled([
      userService.searchUsers({ query }),
      groupService.searchGroups({ query }),
      momentService.searchMoments({ tags: [query], limit: 5 })
    ])
    
    // 处理用户搜索结果
    if (usersResponse.status === 'fulfilled' && usersResponse.value.items) {
      searchResultGroups.users.items = usersResponse.value.items.map((user: any): CustomCommandPaletteItem => ({
        id: `user-${user.user_id}`,
        label: user.nickname,
        suffix: user.distance ? `附近 ${Math.round(user.distance)}m` : '位置未知',
        avatar: {
          src: user.avatar_url,
          fallback: user.nickname?.charAt(0) || 'U'
        },
        chip: user.last_active_at ? {
          text: '活跃',
          color: 'success' as const
        } : undefined,
        onSelect: () => {
          handleClose()
          navigateTo(`/users/${user.user_id}`)
        }
      }))
    }
    
    // 处理群组搜索结果
    if (groupsResponse.status === 'fulfilled' && groupsResponse.value.items) {
      searchResultGroups.groups.items = groupsResponse.value.items.map((group: any): CustomCommandPaletteItem => ({
        id: `group-${group.group_id}`,
        label: group.name,
        suffix: `${group.member_count} 名成员`,
        icon: 'i-heroicons-user-group',
        chip: group.is_member ? {
          text: '已加入',
          color: 'primary' as const
        } : {
          text: '可加入', 
          color: 'neutral' as const
        },
        onSelect: () => {
          handleClose()
          navigateTo(`/groups/${group.group_id}`)
        }
      }))
    }
    
    // 处理动态搜索结果
    if (momentsResponse.status === 'fulfilled' && momentsResponse.value.items) {
      searchResultGroups.moments.items = momentsResponse.value.items.map((moment: any): CustomCommandPaletteItem => ({
        id: `moment-${moment.moment_id}`,
        label: moment.description || '无标题',
        suffix: `${moment.nickname} · ${formatTime(moment.created_at)}`,
        icon: 'i-heroicons-photo',
        chip: moment.likes_count > 10 ? {
          text: '热门',
          color: 'warning' as const
        } : undefined,
        onSelect: () => {
          handleClose()
          navigateTo(`/moments/${moment.moment_id}`)
        }
      }))
    }
  } catch (error) {
    console.error('搜索失败:', error)
  } finally {
    isLoading.value = false
  }
}, 300)

// 监听搜索词变化
watch(searchTerm, debouncedSearch)

// 处理选择项目
const handleSelect = (item: any) => {
  if (item && typeof item === 'object' && typeof item.onSelect === 'function') {
    item.onSelect()
  }
}

// 处理关闭
const handleClose = () => {
  searchTerm.value = ''
  Object.values(searchResultGroups).forEach(group => {
    group.items = []
  })
  // 通知 useOverlay 关闭
  emit('close')
}

// 添加 close emit
const emit = defineEmits<{
  'create-moment': []
  'close': []
}>()

// CommandPalette 会自动聚焦输入框

// formatTime 函数已从统一工具函数导入

// 暴露控制方法
defineExpose({
  close: handleClose
})
</script>
