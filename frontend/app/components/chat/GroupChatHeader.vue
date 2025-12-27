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
            :name="groupInfo?.name || '群聊'"
            :description="`${groupInfo?.members_count || 0} 名成员`"
            :avatar="{
              src: groupInfo?.avatar_url,
              alt: groupInfo?.name,
              size: 'md'
            }"
            @click="$emit('view-info')"
          />
        </div>
      </template>

      <template #right>
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
      </template>
    </UDashboardNavbar>
  </div>
</template>

<script setup lang="ts">
import type { GroupSearchItem } from "~/types/api"
import type { DropdownMenuItem } from '@nuxt/ui'

interface Props {
  groupInfo?: GroupSearchItem | null
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


