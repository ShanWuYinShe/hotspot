<template>
  <!-- 详情侧边面板 - 使用 USlideover -->
  <USlideover :side="side" :overlay="true">
    <template #header>
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <UIcon :name="headerIcon" class="w-5 h-5 text-primary-500" />
          <h2 class="text-lg font-semibold">{{ title }}</h2>
        </div>
        <UButton variant="ghost" icon="i-heroicons-x-mark" size="sm" square @click="handleClose" />
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
              <UBadge :color="data.is_temp ? 'warning' : 'success'" size="sm">
                {{ data.is_temp ? '临时账户' : '正式用户' }}
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
                <p class="text-2xl font-bold text-primary-600">{{ data.friends_count || 0 }}</p>
                <p class="text-sm text-neutral-500">好友</p>
              </div>
            </div>
          </UCard>

          <!-- 操作按钮 -->
          <div class="space-y-2">
            <UButton block @click="handleViewProfile">
              <UIcon name="i-heroicons-user" class="w-4 h-4 mr-2" />
              查看详细资料
            </UButton>
            <UButton variant="outline" block @click="handleSendMessage">
              <UIcon name="i-heroicons-chat-bubble-left-right" class="w-4 h-4 mr-2" />
              发送消息
            </UButton>
          </div>
        </div>

        <!-- 群组详情 -->
        <div v-else-if="type === 'group' && data" class="space-y-4">
          <!-- 群组头像和基本信息 -->
          <div class="text-center">
            <UAvatar 
              :alt="data.name" 
              size="2xl" 
              class="mx-auto mb-4 ring-4 ring-blue-100 dark:ring-blue-900"
            >
              {{ data.name?.charAt(0) || 'G' }}
            </UAvatar>
            <h3 class="text-xl font-bold text-neutral-900 dark:text-white">{{ data.name }}</h3>
            <p class="text-neutral-600 dark:text-neutral-400">{{ data.description || '暂无描述' }}</p>
            <div class="flex justify-center mt-2">
              <UBadge :color="data.is_public ? 'success' : 'warning'" size="sm">
                {{ data.is_public ? '公开群组' : '私密群组' }}
              </UBadge>
            </div>
          </div>

          <!-- 群组统计 -->
          <UCard>
            <template #header>
              <h4 class="font-semibold">群组信息</h4>
            </template>
            <div class="space-y-3">
              <div class="flex justify-between">
                <span class="text-neutral-600">成员数量</span>
                <span class="font-semibold">{{ data.member_count || 0 }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-neutral-600">创建时间</span>
                <span class="font-semibold">{{ formatDate(data.created_at) }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-neutral-600">最后活跃</span>
                <span class="font-semibold">{{ formatDate(data.last_active_at) }}</span>
              </div>
            </div>
          </UCard>

          <!-- 操作按钮 -->
          <div class="space-y-2">
            <UButton block @click="handleJoinGroup">
              <UIcon name="i-heroicons-user-plus" class="w-4 h-4 mr-2" />
              加入群组
            </UButton>
            <UButton variant="outline" block @click="handleViewGroup">
              <UIcon name="i-heroicons-eye" class="w-4 h-4 mr-2" />
              查看群组
            </UButton>
          </div>
        </div>

        <!-- 动态详情 -->
        <div v-else-if="type === 'moment' && data" class="space-y-4">
          <!-- 动态作者信息 -->
          <div class="flex items-center gap-3">
            <UAvatar :alt="data.author?.nickname" size="md">
              {{ data.author?.nickname?.charAt(0) || 'U' }}
            </UAvatar>
            <div class="flex-1">
              <p class="font-semibold text-neutral-900 dark:text-white">{{ data.author?.nickname }}</p>
              <p class="text-sm text-neutral-500">{{ formatDate(data.created_at) }}</p>
            </div>
          </div>

          <!-- 动态内容 -->
          <UCard>
            <template #header>
              <h4 class="font-semibold">动态内容</h4>
            </template>
            <div class="space-y-3">
              <p class="text-neutral-700 dark:text-neutral-300">{{ data.content }}</p>
              
              <!-- 动态图片 -->
              <div v-if="data.images && data.images.length > 0">
                <UCarousel
                  v-if="data.images.length > 1"
                  v-slot="{ item }"
                  :items="data.images"
                  arrows
                  dots
                  class="w-full"
                >
                  <img 
                    :src="String(item)" 
                    :alt="'动态图片'"
                    class="w-full h-48 object-cover rounded-lg"
                  />
                </UCarousel>
                <img 
                  v-else
                  :src="data.images[0]" 
                  alt="动态图片"
                  class="w-full h-48 object-cover rounded-lg"
                />
              </div>
            </div>
          </UCard>

          <!-- 动态统计 -->
          <UCard>
            <template #header>
              <h4 class="font-semibold">互动数据</h4>
            </template>
            <div class="grid grid-cols-2 gap-4 text-center">
              <div>
                <p class="text-2xl font-bold text-error-600">{{ data.likes_count || 0 }}</p>
                <p class="text-sm text-neutral-500">点赞</p>
              </div>
              <div>
                <p class="text-2xl font-bold text-primary-600">{{ data.comments_count || 0 }}</p>
                <p class="text-sm text-neutral-500">评论</p>
              </div>
            </div>
          </UCard>

          <!-- 操作按钮 -->
          <div class="space-y-2">
            <UButton block @click="handleLikeMoment">
              <UIcon name="i-heroicons-heart" class="w-4 h-4 mr-2" />
              {{ data.is_liked ? '取消点赞' : '点赞' }}
            </UButton>
            <UButton variant="outline" block @click="handleViewMoment">
              <UIcon name="i-heroicons-eye" class="w-4 h-4 mr-2" />
              查看详情
            </UButton>
          </div>
        </div>

        <!-- 空状态 -->
        <div v-else class="text-center py-8">
          <UIcon name="i-heroicons-document-text" class="w-12 h-12 mx-auto text-neutral-400 mb-4" />
          <p class="text-neutral-500">暂无详细信息</p>
        </div>
      </div>
    </template>
  </USlideover>
</template>

<script setup lang="ts">
// Props
interface Props {
  open?: boolean;
  type?: 'user' | 'group' | 'moment';
  data?: any;
  side?: 'left' | 'right';
}

const props = withDefaults(defineProps<Props>(), {
  open: false,
  type: 'user',
  data: null,
  side: 'right'
});

// Emits
const emit = defineEmits<{
  'update:open': [value: boolean];
  'close': [];
  'action': [action: string, data?: any];
}>();

// useOverlay 会自动管理 Slideover 状态，无需手动绑定 isOpen

// 计算属性
const title = computed(() => {
  switch (props.type) {
    case 'user': return '用户详情';
    case 'group': return '群组详情';
    case 'moment': return '动态详情';
    default: return '详情';
  }
});

const headerIcon = computed(() => {
  switch (props.type) {
    case 'user': return 'i-heroicons-user';
    case 'group': return 'i-heroicons-user-group';
    case 'moment': return 'i-heroicons-photo';
    default: return 'i-heroicons-document-text';
  }
});

// 处理关闭
const handleClose = () => {
  emit('close');
};

// 格式化日期
const formatDate = (dateString: string) => {
  if (!dateString) return '未知';
  const date = new Date(dateString);
  return date.toLocaleDateString('zh-CN');
};

// 用户操作
const handleViewProfile = () => {
  if (props.data?.user_id) {
    emit('action', 'viewProfile', props.data);
    handleClose();
    navigateTo(`/users/${props.data.user_id}`);
  }
};

const handleSendMessage = () => {
  if (props.data?.user_id) {
    emit('action', 'sendMessage', props.data);
    handleClose();
    navigateTo(`/chat/users/${props.data.user_id}`);
  }
};

// 群组操作
const handleJoinGroup = () => {
  emit('action', 'joinGroup', props.data);
  // 这里可以添加加入群组的逻辑
};

const handleViewGroup = () => {
  if (props.data?.group_id) {
    emit('action', 'viewGroup', props.data);
    handleClose();
    navigateTo(`/groups/${props.data.group_id}`);
  }
};

// 动态操作
const handleLikeMoment = () => {
  emit('action', 'likeMoment', props.data);
  // 这里可以添加点赞逻辑
};

const handleViewMoment = () => {
  if (props.data?.moment_id) {
    emit('action', 'viewMoment', props.data);
    handleClose();
    navigateTo(`/moments/${props.data.moment_id}`);
  }
};
</script>

