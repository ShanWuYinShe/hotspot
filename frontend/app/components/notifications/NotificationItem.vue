<template>
  <div
    class="group relative p-4 rounded-lg border transition-all duration-200 cursor-pointer"
    :class="[
      notification.is_read 
        ? 'bg-card-elevated border-default' 
        : 'bg-blue-light border-blue',
      'interactive-card'
    ]"
    @click="$emit('click')"
  >
    <!-- 选择框 -->
    <div class="absolute top-3 left-3 opacity-0 group-hover:opacity-100 transition-opacity">
      <UCheckbox
        :model-value="selected"
        @update:model-value="$emit('select')"
        @click.stop
      />
    </div>
    
    <!-- 主要内容 -->
    <div class="flex items-start gap-4" :class="{ 'ml-6': true }">
      <!-- 头像 -->
      <div class="shrink-0">
        <UAvatar
          :src="notification.avatar"
          :alt="notification.sender_name || notification.title"
          size="md"
          :class="typeIconClass"
        />
        
        <!-- 类型图标 -->
        <div
          class="absolute -bottom-1 -right-1 w-5 h-5 rounded-full flex items-center justify-center"
          :class="typeIconBgClass"
        >
          <UIcon :name="typeIcon" class="w-3 h-3 text-white" />
        </div>
      </div>
      
      <!-- 通知内容 -->
      <div class="flex-1 min-w-0">
        <div class="flex items-start justify-between gap-2">
          <div class="flex-1 min-w-0">
            <h4 
              class="font-medium truncate"
              :class="notification.is_read ? 'text-black-70' : 'text-primary-content'"
            >
              {{ notification.title }}
            </h4>
            <p 
              class="text-sm mt-1 line-clamp-2"
              :class="notification.is_read ? 'text-muted' : 'text-secondary-content'"
            >
              {{ notification.content }}
            </p>
          </div>
          
          <!-- 时间和状态 -->
          <div class="flex flex-col items-end gap-1 shrink-0">
            <time 
              class="text-xs"
              :class="notification.is_read ? 'text-muted' : 'text-slate'"
            >
              {{ formatTime(notification.created_at) }}
            </time>
            <div v-if="!notification.is_read" class="w-2 h-2 bg-icon-primary rounded-full"></div>
          </div>
        </div>
        
        <!-- 发送者信息 -->
        <div v-if="notification.sender_name" class="flex items-center gap-1 mt-2">
          <UIcon name="i-heroicons-user" class="w-4 h-4 text-black-40" />
          <span class="text-xs text-muted">{{ notification.sender_name }}</span>
        </div>
      </div>
    </div>
    
    <!-- 操作按钮 -->
    <div class="absolute top-3 right-3 opacity-0 group-hover:opacity-100 transition-opacity">
      <div class="flex items-center gap-1">
        <UButton
          v-if="!notification.is_read"
          icon="i-heroicons-check"
          variant="ghost"
          size="sm"
          @click.stop="$emit('mark-read')"
        />
        <UDropdownMenu :items="actionItems">
          <UButton
            icon="i-heroicons-ellipsis-vertical"
            variant="ghost"
            size="sm"
            @click.stop
          />
        </UDropdownMenu>
      </div>
    </div>
    
    <!-- 悬浮效果 -->
    <div class="absolute inset-0 rounded-lg bg-linear-to-r from-transparent via-primary-500/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"></div>
  </div>
</template>

<script setup lang="ts">

interface NotificationItem {
  notification_id: string
  type: 'system' | 'friend' | 'group' | 'moment' | 'message'
  title: string
  content: string
  is_read: boolean
  created_at: string
  data?: Record<string, unknown>
  avatar?: string
  sender_name?: string
}

interface Props {
  notification: NotificationItem
  selected?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  selected: false
})
const { formatTime } = useFormatters()

const emit = defineEmits<{
  click: []
  select: []
  'mark-read': []
  delete: []
}>()

// 计算属性
const typeIcon = computed(() => {
  switch (props.notification.type) {
    case 'friend':
      return 'i-heroicons-users'
    case 'group':
      return 'i-heroicons-user-group'
    case 'moment':
      return 'i-heroicons-photo'
    case 'message':
      return 'i-heroicons-chat-bubble-left-right'
    case 'system':
    default:
      return 'i-heroicons-cog-6-tooth'
  }
})

const typeIconClass = computed(() => {
  switch (props.notification.type) {
    case 'friend':
      return 'ring-2 ring-blue'
    case 'group':
      return 'ring-2 ring-green'
    case 'moment':
      return 'ring-2 ring-orange'
    case 'message':
      return 'ring-2 ring-purple'
    case 'system':
    default:
      return 'ring-2 ring-slate'
  }
})

const typeIconBgClass = computed(() => {
  switch (props.notification.type) {
    case 'friend':
      return 'bg-icon-primary'
    case 'group':
      return 'bg-icon-success'
    case 'moment':
      return 'bg-icon-orange'
    case 'message':
      return 'bg-icon-purple'
    case 'system':
    default:
      return 'bg-icon-slate'
  }
})

const actionItems = computed(() => [
  [{
    label: props.notification.is_read ? '标记未读' : '标记已读',
    icon: props.notification.is_read ? 'i-heroicons-bell' : 'i-heroicons-check',
    click: () => emit('mark-read')
  }],
  [{
    label: '删除',
    icon: 'i-heroicons-trash',
    click: () => emit('delete')
  }]
])
</script>
