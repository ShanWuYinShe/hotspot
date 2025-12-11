<template>
  <div class="px-3 py-2.5 flex items-center justify-between glass-strong border-b border-glass shrink-0">
    <!-- 左侧：返回按钮、头像、用户信息 -->
    <div class="flex items-center gap-2">
      <!-- 返回按钮 -->
      <UButton
        variant="ghost"
        size="sm"
        icon="i-heroicons-arrow-left"
        square
        class="shrink-0"
        @click="$emit('go-back')"
      />
      
      <!-- 头像带在线状态 badge -->
      <div class="relative cursor-pointer" @click="$emit('view-profile')">
        <UAvatar 
          :alt="userInfo?.nickname" 
          :src="userInfo?.avatar_url"
          size="md"
        >
          {{ userInfo?.nickname?.charAt(0) || "U" }}
        </UAvatar>
        <!-- 在线状态指示器 (仅在线时显示绿色小圆点) -->
        <span 
          v-if="isOnline" 
          class="badge-position-br status-dot-online"
        />
      </div>

      <!-- 用户信息 -->
      <div class="cursor-pointer min-w-0" @click="$emit('view-profile')">
        <h1 class="text-sm font-semibold text-primary-content hover-text-accent transition-colors truncate">
          {{ userInfo?.nickname || "私聊" }}
        </h1>
        <!-- 离线时显示离线状态文字，在线时仅靠 badge 指示 -->
        <p v-if="!isOnline" class="text-xs text-secondary-content">
          {{ lastOnlineText }}
        </p>
      </div>
    </div>

    <!-- 右侧：操作按钮 -->
    <div class="flex items-center gap-0.5">
      <!-- 搜索按钮 -->
      <UTooltip text="搜索聊天记录">
        <UButton
          variant="ghost"
          size="sm"
          icon="i-heroicons-magnifying-glass"
          aria-label="搜索聊天记录"
          @click="$emit('toggle-search')"
        />
      </UTooltip>
      
      <!-- 语音通话 -->
      <UTooltip text="语音通话">
        <UButton
          variant="ghost"
          size="sm"
          icon="i-heroicons-phone"
          aria-label="语音通话"
          @click="$emit('voice-call')"
        />
      </UTooltip>
      
      <!-- 视频通话 -->
      <UTooltip text="视频通话">
        <UButton
          variant="ghost"
          size="sm"
          icon="i-heroicons-video-camera"
          aria-label="视频通话"
          @click="$emit('video-call')"
        />
      </UTooltip>
      
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
  </div>
</template>

<script setup lang="ts">
import type { UserResponse } from '~/types/client'

interface Props {
  userInfo?: UserResponse
  isOnline?: boolean
  lastOnline?: string  // ISO 8601 格式的最后在线时间
}

interface Emits {
  'go-back': []
  'toggle-search': []
  'voice-call': []
  'video-call': []
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
    class: 'text-error'
  }]
])
</script>

