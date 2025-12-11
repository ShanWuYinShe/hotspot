<template>
  <div
    class="group relative p-4 rounded-lg border transition-all duration-200 cursor-pointer"
    :class="[
      notification.is_read 
        ? 'bg-white dark:bg-slate-800 border-slate-200 dark:border-slate-700' 
        : 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800',
      'hover:shadow-md hover:border-primary-300 dark:hover:border-primary-600'
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
      <div class="flex-shrink-0">
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
              :class="notification.is_read ? 'text-black/70 dark:text-white/70' : 'text-black dark:text-white'"
            >
              {{ notification.title }}
            </h4>
            <p 
              class="text-sm mt-1 line-clamp-2"
              :class="notification.is_read ? 'text-black/50 dark:text-white/50' : 'text-black/60 dark:text-white/60 dark:text-slate-300'"
            >
              {{ notification.content }}
            </p>
          </div>
          
          <!-- 时间和状态 -->
          <div class="flex flex-col items-end gap-1 flex-shrink-0">
            <time 
              class="text-xs"
              :class="notification.is_read ? 'text-slate-400' : 'text-slate-500'"
            >
              {{ formatTime(notification.created_at) }}
            </time>
            <div v-if="!notification.is_read" class="w-2 h-2 bg-primary-500 rounded-full"></div>
          </div>
        </div>
        
        <!-- 发送者信息 -->
        <div v-if="notification.sender_name" class="flex items-center gap-1 mt-2">
          <UIcon name="i-heroicons-user" class="w-3 h-3 text-black/40 dark:text-white/40" />
          <span class="text-xs text-black/50 dark:text-white/50">{{ notification.sender_name }}</span>
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
          size="xs"
          @click.stop="$emit('mark-read')"
        />
        <UDropdownMenu :items="actionItems">
          <UButton
            icon="i-heroicons-ellipsis-vertical"
            variant="ghost"
            size="xs"
            @click.stop
          />
        </UDropdownMenu>
      </div>
    </div>
    
    <!-- 悬浮效果 -->
    <div class="absolute inset-0 rounded-lg bg-gradient-to-r from-transparent via-primary-500/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"></div>
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
  data?: any
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
      return 'ring-2 ring-blue-200 dark:ring-blue-800'
    case 'group':
      return 'ring-2 ring-green-200 dark:ring-green-800'
    case 'moment':
      return 'ring-2 ring-orange-200 dark:ring-orange-800'
    case 'message':
      return 'ring-2 ring-purple-200 dark:ring-purple-800'
    case 'system':
    default:
      return 'ring-2 ring-slate-200 dark:ring-slate-700'
  }
})

const typeIconBgClass = computed(() => {
  switch (props.notification.type) {
    case 'friend':
      return 'bg-blue-500'
    case 'group':
      return 'bg-green-500'
    case 'moment':
      return 'bg-orange-500'
    case 'message':
      return 'bg-purple-500'
    case 'system':
    default:
      return 'bg-slate-500'
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

<style scoped>
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
