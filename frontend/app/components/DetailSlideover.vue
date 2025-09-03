<template>
  <!-- 详情侧边面板 - 使用 USlideover -->
  <USlideover v-model:open="isOpen" :side="side" :overlay="true">
    <template #header>
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <UIcon :name="headerIcon" class="w-5 h-5 text-primary-500" />
          <h2 class="text-lg font-semibold">{{ title }}</h2>
        </div>
        <UButton variant="ghost" icon="i-heroicons-x-mark" size="sm" square @click="isOpen = false" />
      </div>
    </template>

    <template #body>
      <div class="space-y-6">
        <!-- 用户详情 -->
        <div v-if="type === 'user' && data" class="space-y-4">
          <!-- 用户头像和基本信息 -->
          <div class="text-center">
            <UAvatar 
              :alt="data.nickname" 
              size="2xl" 
              class="mx-auto mb-4 ring-4 ring-primary-100 dark:ring-primary-900"
            >
              {{ data.nickname?.charAt(0) || 'U' }}
            </UAvatar>
            <h3 class="text-xl font-bold text-neutral-900 dark:text-white">{{ data.nickname }}</h3>
            <p class="text-neutral-600 dark:text-neutral-400">{{ data.email || '临时用户' }}</p>
            <div class="flex justify-center mt-2">
              <UBadge :color="data.is_temporary ? 'warning' : 'success'" size="sm">
                {{ data.is_temporary ? '临时账户' : '正式用户' }}
              </UBadge>
            </div>
          </div>

          <!-- 用户统计 -->
          <UCard>
            <template #header>
              <h4 class="font-semibold">用户统计</h4>
            </template>
            <div class="grid grid-cols-2 gap-4 text-center">
              <div>
                <p class="text-2xl font-bold text-primary-600">{{ data.moments_count || 0 }}</p>
                <p class="text-sm text-neutral-500">动态</p>
              </div>
              <div>
                <p class="text-2xl font-bold text-blue-600">{{ data.friends_count || 0 }}</p>
                <p class="text-sm text-neutral-500">好友</p>
              </div>
            </div>
          </UCard>

          <!-- 操作按钮 -->
          <div class="space-y-2">
            <UButton block icon="i-heroicons-chat-bubble-left-right" @click="handleAction('sendMessage')">
              发送消息
            </UButton>
            <UButton block variant="outline" icon="i-heroicons-user-plus" @click="handleAction('addFriend')">
              添加好友
            </UButton>
          </div>
        </div>

        <!-- 群组详情 -->
        <div v-else-if="type === 'group' && data" class="space-y-4">
          <!-- 群组头像和基本信息 -->
          <div class="text-center">
            <div class="w-20 h-20 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl flex items-center justify-center mx-auto mb-4">
              <UIcon name="i-heroicons-user-group" class="w-10 h-10 text-white" />
            </div>
            <h3 class="text-xl font-bold text-neutral-900 dark:text-white">{{ data.name }}</h3>
            <p class="text-neutral-600 dark:text-neutral-400 mt-2">{{ data.description }}</p>
          </div>

          <!-- 群组统计 -->
          <UCard>
            <template #header>
              <h4 class="font-semibold">群组信息</h4>
            </template>
            <div class="space-y-3">
              <div class="flex justify-between">
                <span class="text-neutral-600 dark:text-neutral-400">成员数量</span>
                <span class="font-semibold">{{ data.member_count || 0 }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-neutral-600 dark:text-neutral-400">创建时间</span>
                <span class="font-semibold">{{ formatDate(data.created_at) }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-neutral-600 dark:text-neutral-400">群组类型</span>
                <UBadge :color="data.is_private ? 'warning' : 'success'" size="sm">
                  {{ data.is_private ? '私有' : '公开' }}
                </UBadge>
              </div>
            </div>
          </UCard>

          <!-- 操作按钮 -->
          <div class="space-y-2">
            <UButton block icon="i-heroicons-plus" @click="handleAction('joinGroup')">
              加入群组
            </UButton>
            <UButton block variant="outline" icon="i-heroicons-share" @click="handleAction('shareGroup')">
              分享群组
            </UButton>
          </div>
        </div>

        <!-- 动态详情 -->
        <div v-else-if="type === 'moment' && data" class="space-y-4">
          <!-- 作者信息 -->
          <div class="flex items-center gap-3">
            <UAvatar :alt="data.user?.nickname" size="md">
              {{ data.user?.nickname?.charAt(0) || 'U' }}
            </UAvatar>
            <div>
              <p class="font-semibold">{{ data.user?.nickname }}</p>
              <p class="text-sm text-neutral-500">{{ formatDate(data.created_at) }}</p>
            </div>
          </div>

          <!-- 动态内容 -->
          <div class="prose prose-sm dark:prose-invert max-w-none">
            <p>{{ data.content }}</p>
          </div>

          <!-- 动态图片轮播 -->
          <UCarousel 
            v-if="data.images && data.images.length > 0" 
            :items="carouselItems"
            :ui="{ item: 'basis-full' }"
            class="rounded-lg overflow-hidden"
          >
            <template #default="{ item }">
              <img 
                :src="item.src" 
                :alt="item.alt"
                class="w-full h-64 object-cover"
              />
            </template>
          </UCarousel>

          <!-- 动态统计 -->
          <div class="flex items-center gap-6 text-sm text-neutral-500">
            <div class="flex items-center gap-1">
              <UIcon name="i-heroicons-heart" class="w-4 h-4" />
              <span>{{ data.like_count || 0 }}</span>
            </div>
            <div class="flex items-center gap-1">
              <UIcon name="i-heroicons-chat-bubble-left" class="w-4 h-4" />
              <span>{{ data.comment_count || 0 }}</span>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="flex gap-2">
            <UButton flex-1 variant="outline" icon="i-heroicons-heart" @click="handleAction('likeMoment')">
              点赞
            </UButton>
            <UButton flex-1 variant="outline" icon="i-heroicons-chat-bubble-left" @click="handleAction('commentMoment')">
              评论
            </UButton>
            <UButton variant="ghost" icon="i-heroicons-share" square @click="handleAction('shareMoment')" />
          </div>
        </div>

        <!-- 自定义内容 -->
        <div v-else-if="type === 'custom'">
          <slot name="content" />
        </div>

        <!-- 空状态 -->
        <div v-else class="text-center py-8">
          <UIcon name="i-heroicons-document" class="w-12 h-12 text-neutral-400 mx-auto mb-3" />
          <p class="text-neutral-500">暂无详细信息</p>
        </div>
      </div>
    </template>

    <template #footer v-if="showFooter">
      <div class="flex justify-between">
        <UButton variant="ghost" @click="isOpen = false">
          关闭
        </UButton>
        <slot name="footer-actions" />
      </div>
    </template>
  </USlideover>
</template>

<script setup lang="ts">
// Props
interface Props {
  open?: boolean
  type: 'user' | 'group' | 'moment' | 'custom'
  data?: any
  title?: string
  side?: 'left' | 'right'
  showFooter?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
  side: 'right',
  showFooter: false
})

// Emits
const emit = defineEmits<{
  'update:open': [value: boolean]
  'action': [action: string, data?: any]
}>()

// 响应式状态
const isOpen = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

// 计算属性
const title = computed(() => {
  if (props.title) return props.title
  
  switch (props.type) {
    case 'user':
      return '用户详情'
    case 'group':
      return '群组详情'
    case 'moment':
      return '动态详情'
    default:
      return '详细信息'
  }
})

const carouselItems = computed(() => {
  if (!props.data?.images) return []
  return props.data.images.map((image: string, index: number) => ({
    src: image,
    alt: `动态图片 ${index + 1}`
  }))
})

const headerIcon = computed(() => {
  switch (props.type) {
    case 'user':
      return 'i-heroicons-user'
    case 'group':
      return 'i-heroicons-user-group'
    case 'moment':
      return 'i-heroicons-photo'
    default:
      return 'i-heroicons-document'
  }
})

// 工具函数
const formatDate = (dateString: string) => {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  })
}

// 处理操作
const handleAction = (action: string) => {
  emit('action', action, props.data)
}
</script>
