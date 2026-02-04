<template>
  <header class="flex items-center justify-between px-4 py-3 border-b border-neutral-200 dark:border-neutral-800">
      <div class="flex items-center gap-3 min-w-0">
        <UButton
          variant="ghost"
          color="neutral"
          size="md"
          icon="i-lucide-arrow-left"
          :square="true"
          @click="$emit('go-back')"
        />

        <div class="flex items-center gap-3 min-w-0 cursor-pointer" @click="$emit('view-profile')">
             <UAvatar 
                :src="userInfo?.avatar_url" 
                :alt="userInfo?.nickname"
                size="md"
                :chip="isOnline ? { color: 'success', position: 'bottom-right', size: '2xs' } : undefined"
             />
             <div class="flex flex-col min-w-0">
                 <span class="text-sm font-medium truncate text-neutral-900 dark:text-white">{{ userInfo?.nickname || '私聊' }}</span>
                 <span class="text-xs text-neutral-500 dark:text-neutral-400 truncate">{{ isOnline ? '在线' : lastOnlineText }}</span>
             </div>
        </div>
      </div>

      <div class="flex items-center gap-1">
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
      </div>
  </header>
</template>

<script setup lang="ts">
import type { PrivateChatDetailData } from '~/types/api'
import type { DropdownMenuItem } from '@nuxt/ui'

interface Props {
  chat?: PrivateChatDetailData
  loading?: boolean
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

const userInfo = computed(() => props.chat?.other_user)
const isOnline = computed(() => false) // 暂未接入实时在线状态
const lastOnlineText = computed(() => '离线') // 暂未接入

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
