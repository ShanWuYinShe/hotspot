<template>
  <div class="container mx-auto px-4 py-8">
    <div class="max-w-2xl mx-auto">
      <!-- 页面标题 -->
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-2xl font-bold text-gray-900">动态广场</h1>
        <UButton @click="createMoment">
          <UIcon name="i-heroicons-plus" class="w-4 h-4 mr-2" />
          发布动态
        </UButton>
      </div>

      <!-- 发布动态框 -->
      <UCard class="mb-6">
        <div class="flex space-x-3">
          <UAvatar 
            :src="currentUser?.avatar_url" 
            :alt="currentUser?.nickname"
            size="md"
          >
            {{ currentUser?.nickname?.charAt(0) || 'U' }}
          </UAvatar>
          <div class="flex-1">
            <UTextarea
              v-model="newMoment"
              placeholder="分享你的精彩时刻..."
              :rows="3"
              class="mb-3"
            />
            <div class="flex justify-between items-center">
              <div class="flex space-x-2">
                <UButton variant="ghost" size="sm" @click="addImage">
                  <UIcon name="i-heroicons-photo" class="w-4 h-4 mr-1" />
                  图片
                </UButton>
                <UButton variant="ghost" size="sm" @click="addLocation">
                  <UIcon name="i-heroicons-map-pin" class="w-4 h-4 mr-1" />
                  位置
                </UButton>
              </div>
              <UButton 
                :disabled="!newMoment.trim()"
                @click="publishMoment"
                :loading="publishing"
              >
                发布
              </UButton>
            </div>
          </div>
        </div>
      </UCard>

      <!-- 动态列表 -->
      <div class="space-y-6">
        <UCard 
          v-for="moment in moments" 
          :key="moment.id"
          class="hover:shadow-md transition-shadow"
        >
          <!-- 用户信息 -->
          <div class="flex items-center space-x-3 mb-3">
            <UAvatar 
              :src="moment.user.avatar_url" 
              :alt="moment.user.nickname"
              size="md"
            >
              {{ moment.user.nickname.charAt(0) }}
            </UAvatar>
            <div class="flex-1">
              <h3 class="font-semibold text-gray-900">
                {{ moment.user.nickname }}
              </h3>
              <p class="text-sm text-gray-600">
                {{ formatTime(moment.created_at) }}
                <span v-if="moment.location" class="ml-2">
                  <UIcon name="i-heroicons-map-pin" class="w-3 h-3 inline" />
                  {{ moment.location }}
                </span>
              </p>
            </div>
            <UDropdown :items="getMomentActions(moment)">
              <UButton 
                variant="ghost" 
                size="sm" 
                icon="i-heroicons-ellipsis-horizontal"
              />
            </UDropdown>
          </div>

          <!-- 动态内容 -->
          <div class="mb-4">
            <p class="text-gray-900 leading-relaxed whitespace-pre-wrap">
              {{ moment.content }}
            </p>
          </div>

          <!-- 图片网格 -->
          <div v-if="moment.images && moment.images.length > 0" class="mb-4">
            <div 
              :class="[
                'grid gap-2',
                moment.images.length === 1 ? 'grid-cols-1' : 
                moment.images.length === 2 ? 'grid-cols-2' :
                'grid-cols-3'
              ]"
            >
              <div 
                v-for="(image, index) in moment.images.slice(0, 9)" 
                :key="index"
                class="relative aspect-square bg-gray-100 rounded-lg overflow-hidden cursor-pointer"
                @click="previewImage(moment.images, index)"
              >
                <img 
                  :src="image.url" 
                  :alt="`图片 ${index + 1}`"
                  class="w-full h-full object-cover hover:scale-105 transition-transform"
                />
                <div 
                  v-if="index === 8 && moment.images.length > 9"
                  class="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center text-white font-semibold"
                >
                  +{{ moment.images.length - 9 }}
                </div>
              </div>
            </div>
          </div>

          <!-- 互动按钮 -->
          <div class="flex items-center space-x-6 pt-3 border-t">
            <UButton 
              variant="ghost" 
              size="sm"
              :class="moment.is_liked ? 'text-red-500' : 'text-gray-600'"
              @click="toggleLike(moment)"
            >
              <UIcon :name="moment.is_liked ? 'i-heroicons-heart-solid' : 'i-heroicons-heart'" class="w-4 h-4 mr-1" />
              {{ moment.like_count }}
            </UButton>
            
            <UButton variant="ghost" size="sm" @click="showComments(moment)">
              <UIcon name="i-heroicons-chat-bubble-left-right" class="w-4 h-4 mr-1" />
              {{ moment.comment_count }}
            </UButton>
            
            <UButton variant="ghost" size="sm" @click="shareMoment(moment)">
              <UIcon name="i-heroicons-share" class="w-4 h-4 mr-1" />
              分享
            </UButton>
          </div>
        </UCard>
      </div>

      <!-- 加载更多 -->
      <div v-if="hasMoreMoments" class="text-center mt-8">
        <UButton 
          variant="outline" 
          @click="loadMoreMoments"
          :loading="loadingMore"
        >
          加载更多
        </UButton>
      </div>

      <!-- 空状态 -->
      <div v-if="moments.length === 0" class="text-center py-12">
        <UIcon name="i-heroicons-photo" class="w-16 h-16 text-gray-400 mx-auto mb-4" />
        <h3 class="text-lg font-medium text-gray-900 mb-2">暂无动态</h3>
        <p class="text-gray-600 mb-6">成为第一个分享精彩时刻的人吧！</p>
        <UButton @click="createMoment">发布第一条动态</UButton>
      </div>
    </div>
  </div>
</template>

<script setup>
// 页面元数据
definePageMeta({
  title: '动态广场',
  requiresAuth: true
})

const authStore = useAuthStore()
const toast = useToast()

// 当前用户
const currentUser = computed(() => authStore.currentUser)

// 数据状态
const newMoment = ref('')
const publishing = ref(false)
const loadingMore = ref(false)
const hasMoreMoments = ref(true)

// 模拟动态数据
const moments = ref([
  {
    id: 1,
    user: {
      id: 1,
      nickname: '张小明',
      avatar_url: ''
    },
    content: '今天天气真不错，和朋友们一起去公园野餐了！生活真美好 ✨',
    images: [
      { url: 'https://picsum.photos/400/400?random=1', alt: '野餐照片1' },
      { url: 'https://picsum.photos/400/400?random=2', alt: '野餐照片2' },
      { url: 'https://picsum.photos/400/400?random=3', alt: '野餐照片3' }
    ],
    location: '中央公园',
    created_at: new Date(Date.now() - 1000 * 60 * 30),
    like_count: 15,
    comment_count: 3,
    is_liked: false
  },
  {
    id: 2,
    user: {
      id: 2,
      nickname: '李小红',
      avatar_url: ''
    },
    content: '刚刚完成了5公里的晨跑，感觉整个人都充满了活力！💪',
    images: [],
    location: '江滨公园',
    created_at: new Date(Date.now() - 1000 * 60 * 60 * 2),
    like_count: 8,
    comment_count: 1,
    is_liked: true
  },
  {
    id: 3,
    user: {
      id: 3,
      nickname: '王大强',
      avatar_url: ''
    },
    content: '新学了一道菜，味道还不错！下次请大家来品尝 🍝',
    images: [
      { url: 'https://picsum.photos/400/400?random=4', alt: '美食照片' }
    ],
    location: null,
    created_at: new Date(Date.now() - 1000 * 60 * 60 * 5),
    like_count: 12,
    comment_count: 5,
    is_liked: false
  }
])

// 格式化时间
const formatTime = (date) => {
  const now = new Date()
  const diff = now - date
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (minutes < 60) {
    return `${minutes}分钟前`
  } else if (hours < 24) {
    return `${hours}小时前`
  } else {
    return `${days}天前`
  }
}

// 获取动态操作菜单
const getMomentActions = (moment) => {
  const actions = []
  
  if (moment.user.id === currentUser.value?.user_id) {
    actions.push([{
      label: '删除',
      icon: 'i-heroicons-trash',
      click: () => deleteMoment(moment)
    }])
  } else {
    actions.push([{
      label: '举报',
      icon: 'i-heroicons-flag',
      click: () => reportMoment(moment)
    }])
  }
  
  return actions
}

// 发布动态
const publishMoment = async () => {
  if (!newMoment.value.trim() || publishing.value) return
  
  publishing.value = true
  
  try {
    // 模拟发布动态
    const moment = {
      id: Date.now(),
      user: {
        id: currentUser.value?.user_id,
        nickname: currentUser.value?.nickname || '用户',
        avatar_url: currentUser.value?.avatar_url
      },
      content: newMoment.value.trim(),
      images: [],
      location: null,
      created_at: new Date(),
      like_count: 0,
      comment_count: 0,
      is_liked: false
    }
    
    moments.value.unshift(moment)
    newMoment.value = ''
    
    toast.add({
      title: '发布成功',
      color: 'success'
    })
    
  } catch (error) {
    toast.add({
      title: '发布失败',
      description: '请重试',
      color: 'error'
    })
  } finally {
    publishing.value = false
  }
}

// 切换点赞
const toggleLike = async (moment) => {
  try {
    moment.is_liked = !moment.is_liked
    moment.like_count += moment.is_liked ? 1 : -1
    
    // 这里应该调用 API 更新点赞状态
    
  } catch (error) {
    toast.add({
      title: '操作失败',
      description: '请重试',
      color: 'error'
    })
  }
}

// 显示评论
const showComments = (moment) => {
  toast.add({
    title: '功能开发中',
    description: '评论功能即将上线',
    color: 'info'
  })
}

// 分享动态
const shareMoment = (moment) => {
  toast.add({
    title: '功能开发中',
    description: '分享功能即将上线',
    color: 'info'
  })
}

// 创建动态
const createMoment = () => {
  // 滚动到发布框
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// 添加图片
const addImage = () => {
  toast.add({
    title: '功能开发中',
    description: '图片上传功能即将上线',
    color: 'info'
  })
}

// 添加位置
const addLocation = () => {
  toast.add({
    title: '功能开发中',
    description: '位置选择功能即将上线',
    color: 'info'
  })
}

// 预览图片
const previewImage = (images, index) => {
  toast.add({
    title: '功能开发中',
    description: '图片预览功能即将上线',
    color: 'info'
  })
}

// 删除动态
const deleteMoment = (moment) => {
  toast.add({
    title: '确认删除',
    description: '确定要删除这条动态吗？',
    color: 'warning'
  })
}

// 举报动态
const reportMoment = (moment) => {
  toast.add({
    title: '功能开发中',
    description: '举报功能即将上线',
    color: 'info'
  })
}

// 加载更多动态
const loadMoreMoments = async () => {
  if (loadingMore.value) return
  
  loadingMore.value = true
  
  try {
    // 模拟加载更多
    await new Promise(resolve => setTimeout(resolve, 1000))
    hasMoreMoments.value = false
    
  } catch (error) {
    toast.add({
      title: '加载失败',
      description: '请重试',
      color: 'error'
    })
  } finally {
    loadingMore.value = false
  }
}
</script>
