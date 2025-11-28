<template>
  <UDashboardNavbar>
    <template #left>
      <div class="flex items-center gap-3">
        <UButton
          variant="ghost"
          size="sm"
          icon="i-heroicons-arrow-left"
          aria-label="返回"
          @click="$emit('back')"
          class="lg:hidden"
        />

        <UAvatar :alt="userInfo?.nickname" size="md">
          {{ userInfo?.nickname?.charAt(0) || "U" }}
        </UAvatar>

        <div>
          <h1 class="text-lg font-semibold text-neutral-900 dark:text-white">
            {{ userInfo?.nickname || "私聊" }}
          </h1>
          <p class="text-sm text-neutral-500 dark:text-neutral-400">
            {{ onlineStatus }}
          </p>
        </div>
      </div>
    </template>

    <template #right>
      <div class="flex items-center gap-1">
        <!-- 搜索按钮 -->
        <UButton
          variant="ghost"
          size="sm"
          icon="i-heroicons-magnifying-glass"
          aria-label="搜索聊天记录"
          @click="$emit('toggle-search')"
        />
        
        <!-- 语音通话 -->
        <UButton
          variant="ghost"
          size="sm"
          icon="i-heroicons-phone"
          aria-label="语音通话"
          @click="$emit('voice-call')"
        />
        
        <!-- 视频通话 -->
        <UButton
          variant="ghost"
          size="sm"
          icon="i-heroicons-video-camera"
          aria-label="视频通话"
          @click="$emit('video-call')"
        />
        
        <!-- 更多操作 -->
        <UDropdownMenu :items="menuItems">
          <UButton
            variant="ghost"
            size="sm"
            icon="i-heroicons-ellipsis-horizontal"
            aria-label="更多操作"
          />
        </UDropdownMenu>
      </div>
    </template>
  </UDashboardNavbar>
</template>

<script setup lang="ts">
import type { UserResponse } from '~/types/api'

interface Props {
  userInfo?: UserResponse
  isOnline?: boolean
}

interface Emits {
  back: []
  'toggle-search': []
  'voice-call': []
  'video-call': []
  'view-profile': []
  'clear-history': []
  'block-user': []
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 在线状态
const onlineStatus = computed(() => {
  if (!props.userInfo) return '未知'
  return props.isOnline ? '在线' : '离线'
})

// 菜单项
const menuItems = computed(() => [
  [{
    label: '查看资料',
    icon: 'i-heroicons-user',
    click: () => emit('view-profile')
  }],
  [{
    label: '清空聊天记录',
    icon: 'i-heroicons-trash',
    click: () => emit('clear-history')
  }, {
    label: '屏蔽用户',
    icon: 'i-heroicons-no-symbol',
    click: () => emit('block-user'),
    class: 'text-error-500'
  }]
])
</script>

