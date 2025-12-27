<template>
  <div class="sticky top-0 z-30">
    <UDashboardNavbar :toggle="false">
      <template #left>
        <div class="flex items-center gap-3 min-w-0">
          <UButton
            variant="ghost"
            color="neutral"
            size="md"
            icon="i-lucide-arrow-left"
            :square="true"
            class="shrink-0"
            @click="$emit('go-back')"
          />

          <UUser
            class="min-w-0 cursor-pointer"
            :name="userInfo?.nickname || '私聊'"
            :description="isOnline ? '在线' : lastOnlineText"
            :avatar="{
              src: userInfo?.avatar_url,
              alt: userInfo?.nickname,
              size: 'md',
              chip: isOnline ? { color: 'success', position: 'bottom-right', size: '2xs' } : false
            }"
            @click="$emit('view-profile')"
          />
        </div>
      </template>

      <template #right>
        <UTooltip text="搜索聊天记录" :portal="false">
          <UButton
            variant="ghost"
            color="neutral"
            size="md"
            icon="i-lucide-search"
            aria-label="搜索聊天记录"
            :square="true"
            @click="$emit('toggle-search')"
          />
        </UTooltip>

        <UDropdownMenu :items="props.menuItems?.length ? props.menuItems : defaultMenuItems">
          <UButton
            variant="ghost"
            color="neutral"
            size="md"
            icon="i-lucide-ellipsis"
            aria-label="更多操作"
            :square="true"
          />
        </UDropdownMenu>
      </template>
    </UDashboardNavbar>
  </div>
</template>

<script setup lang="ts">
import type { UserDetailData } from '~/types/client'
import type { DropdownMenuItem } from '@nuxt/ui'

interface Props {
  userInfo?: UserDetailData
  isOnline?: boolean
  lastOnline?: string  // ISO 8601 格式的最后在线时间
  menuItems?: DropdownMenuItem[][]
}

interface Emits {
  'go-back': []
  'toggle-search': []
  'view-profile': []
  'clear-history': []
  'block-user': []
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 离线时显示的最后在线时间文字
const lastOnlineText = computed(() => {
  if (!props.userInfo) return '未知';
  if (props.isOnline) return '';
  
  // 如果有最后在线时间，计算友好的显示文字
  if (props.lastOnline) {
    const lastActive = new Date(props.lastOnline);
    const now = new Date();
    const diff = now.getTime() - lastActive.getTime();
    
    if (diff < 60000) return '刚刚在线';  // 1分钟内
    if (diff < 300000) return '5分钟前在线';  // 5分钟内
    if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前在线`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前在线`;
    return `${Math.floor(diff / 86400000)}天前在线`;
  }
  
  return '离线';
})

// 菜单项（默认）
const defaultMenuItems = computed((): DropdownMenuItem[][] => [
  [{
    label: '查看资料',
    icon: 'i-lucide-user',
    onSelect: () => emit('view-profile')
  }],
  [{
    label: '清空聊天记录',
    icon: 'i-lucide-trash-2',
    onSelect: () => emit('clear-history')
  }, {
    label: '屏蔽用户',
    icon: 'i-lucide-ban',
    color: 'error',
    onSelect: () => emit('block-user')
  }]
])
</script>