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

        <div class="flex items-center gap-3 min-w-0 cursor-pointer" @click="$emit('view-info')">
             <UAvatar 
                :src="groupInfo?.avatar_url" 
                :alt="groupInfo?.name"
                size="md"
             />
             <div class="flex flex-col min-w-0">
                 <span class="text-sm font-medium truncate text-neutral-900 dark:text-white">{{ groupInfo?.name || '群聊' }}</span>
                 <span class="text-xs text-neutral-500 dark:text-neutral-400 truncate">{{ groupInfo?.members_count || 0 }} 名成员</span>
             </div>
        </div>
      </div>

      <div class="flex items-center gap-1">
        <UTooltip text="搜索聊天记录">
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

        <UTooltip text="群成员">
          <UButton
            variant="ghost"
            color="neutral"
            size="md"
            icon="i-lucide-users"
            aria-label="群成员"
            :square="true"
            @click="$emit('view-members')"
          />
        </UTooltip>

        <UDropdownMenu :items="menuItems">
          <UButton
            variant="ghost"
            color="neutral"
            size="md"
            icon="i-lucide-ellipsis-vertical"
            aria-label="更多操作"
            :square="true"
          />
        </UDropdownMenu>
      </div>
  </header>
</template>

<script setup lang="ts">
import type { GroupItem } from "~/types/api"
import type { DropdownMenuItem } from '@nuxt/ui'

interface Props {
  groupInfo?: GroupItem | null
}

interface Emits {
  'go-back': []
  'toggle-search': []
  'view-members': []
  'view-info': []
  'view-settings': []
  'leave-group': []
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 菜单项（Nuxt UI v4：使用 onSelect）
const menuItems = computed((): DropdownMenuItem[][] => [
  [{
    label: '群聊信息',
    icon: 'i-lucide-info',
    onSelect: () => emit('view-info')
  }, {
    label: '群组设置',
    icon: 'i-lucide-settings',
    onSelect: () => emit('view-settings')
  }],
  [{
    label: '退出群聊',
    icon: 'i-lucide-log-out',
    color: 'error',
    onSelect: () => emit('leave-group')
  }]
])
</script>
