<template>
  <div class="unified-page">
    <div class="unified-container">
      <!-- 页面标题 -->
      <div class="unified-card">
        <div class="unified-card__title">
          <Icon name="mdi:camera" />
          动态广场
        </div>
        <div class="unified-card__content">
          <p>分享生活，发现精彩时刻</p>
        </div>
      </div>
    </div>

    <!-- 发布动态卡片 -->
    <div class="unified-card">
      <div class="unified-card__title">
        <Icon name="mdi:plus-circle" />
        发布动态
      </div>
      <div class="unified-card__content">
        <div class="publish-section">
          <div class="user-header">
            <NAvatar :size="40" :src="currentUser?.avatar || defaultImages.userAvatar" class="user-avatar">
              {{ currentUser?.nickname?.charAt(0) || 'U' }}
            </NAvatar>
            <div class="user-info">
              <h4 class="user-name">
                <ClientOnly fallback="未登录">
                  {{ currentUser?.nickname || '未登录' }}
                </ClientOnly>
              </h4>
              <p class="user-subtitle">分享你的精彩时刻</p>
            </div>
          </div>

          <div class="publish-form">
            <NInput v-model="newPostContent" type="textarea" placeholder="分享你的想法..." :rows="3" maxlength="500"
              show-word-limit resize="none" class="publish-input" />

            <div class="publish-actions">
              <div class="publish-options">
                <NButton type="default" size="small" @click="showLocationDialog = true" class="unified-btn">
                  <template #icon>
                    <Icon name="mdi:map-marker" />
                  </template>
                  添加位置
                </NButton>

                <NButton type="default" size="small" @click="showImageUploader = true" class="unified-btn">
                  <template #icon>
                    <Icon name="mdi:image" />
                  </template>
                  添加图片
                </NButton>
              </div>

              <NButton type="primary" @click="publishPost" :loading="isPublishing" :disabled="!newPostContent.trim()"
                class="unified-btn unified-btn--primary">
                <template #icon>
                  <Icon name="mdi:send" />
                </template>
                发布
              </NButton>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 动态列表 -->
    <div class="unified-card">
      <div class="unified-card__title">
        <Icon name="mdi:timeline" />
        最新动态
        <NButton type="primary" size="small" @click="refreshPosts" :loading="isLoading" style="margin-left: auto;"
          class="unified-btn">
          <template #icon>
            <Icon name="mdi:refresh" />
          </template>
          刷新
        </NButton>
      </div>

      <div class="unified-card__content">
        <div v-if="posts.length > 0" class="posts-list">
          <div v-for="post in posts" :key="post.id" class="post-item">
            <div class="post-header">
              <div class="post-user">
                <NAvatar :size="36" :src="defaultImages.userAvatar" class="post-avatar">
                  {{ post.nickname?.charAt(0) || 'U' }}
                </NAvatar>
                <div class="post-user-info">
                  <h4 class="post-user-name">{{ post.nickname || '匿名用户' }}</h4>
                  <p class="post-time">{{ formatTime(post.created_at) }}</p>
                </div>
              </div>

              <div v-if="post.user_id === currentUser?.user_id" class="post-actions">
                <NDropdown :options="[
                  { label: '编辑', key: 'edit', props: { onClick: () => editPost(post) } },
                  { label: '删除', key: 'delete', props: { onClick: () => deletePost(post) } }
                ]">
                  <NButton text size="small">
                    <Icon name="mdi:dots-horizontal" />
                  </NButton>
                </NDropdown>
              </div>
            </div>

            <div class="post-content">
              <p class="post-text">{{ post.description }}</p>

              <div v-if="post.location_name" class="post-location">
                <Icon name="mdi:map-marker" />
                <span>{{ post.location_name }}</span>
              </div>

              <!-- 动态统计 -->
              <div class="post-stats">
                <div class="stat-item">
                  <NButton text size="small" @click="toggleLike(post)" :type="post.liked_by_me ? 'primary' : 'default'"
                    class="stat-btn">
                    <template #icon>
                      <Icon :name="post.liked_by_me ? 'mdi:heart' : 'mdi:heart-outline'" />
                    </template>
                    {{ post.likes_count || 0 }}
                  </NButton>
                </div>

                <div class="stat-item">
                  <NButton text size="small" @click="showComments(post)" class="stat-btn">
                    <template #icon>
                      <Icon name="mdi:comment-outline" />
                    </template>
                    评论
                  </NButton>
                </div>

                <div class="stat-item">
                  <NButton text size="small" @click="sharePost(post)" class="stat-btn">
                    <template #icon>
                      <Icon name="mdi:share-variant-outline" />
                    </template>
                    分享
                  </NButton>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="isLoading" class="unified-loading">
          <div class="unified-loading__spinner"></div>
          <span class="unified-loading__text">加载动态中...</span>
        </div>

        <div v-else class="unified-empty">
          <div class="unified-empty__icon">
            <Icon name="mdi:camera" />
          </div>
          <h4 class="unified-empty__title">暂无动态</h4>
          <p class="unified-empty__description">成为第一个发布动态的人吧</p>
        </div>
      </div>

      <!-- 位置选择对话框 -->
      <LocationShare v-model="showLocationDialog" @selected="handleLocationSelected" />

      <!-- 图片上传器 -->
      <ImageUploader v-model:visible="showImageUploader" @uploaded="handleImageUploaded" />

      <!-- 图片预览 -->
      <ImagePreviewModal v-model="showImagePreview" :images="[previewImageUrl]" :initial-index="0" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { NInput, NButton, NDropdown, NAvatar, useMessage } from 'naive-ui'
import ImagePreviewModal from '@/components/common/ImagePreviewModal.vue'
import LocationShare from '@/components/common/LocationShare.vue'
import ImageUploader from '@/components/common/ImageUploader.vue'

// API实例
const message = useMessage()
import { useMomentStore } from '@/stores/moment'
import { useAuthStore } from '@/stores/auth'
import { defaultImages } from '@/utils/assets'
import type { CheckInInfo } from '~/types/api/checkin'

// 页面元数据
definePageMeta({
  layout: 'default',
  title: '动态',
  middleware: 'auth'
})

// 响应式数据
const newPostContent = ref<string>('')
const isPublishing = ref<boolean>(false)
const isLoading = ref<boolean>(false)
const isRefreshing = ref<boolean>(false)
const posts = ref<CheckInInfo[]>([])
const moments = ref<CheckInInfo[]>([])
const showImagePreview = ref<boolean>(false)
const previewImageUrl = ref<string>('')
const showLocationDialog = ref<boolean>(false)
const showImageUploader = ref<boolean>(false)

// Store
const momentStore = useMomentStore()
const authStore = useAuthStore()

// 当前用户
const currentUser = computed(() => authStore.user)

// 页面同步
onMounted(() => {
  // usePageSync('dynamic')
  loadPosts()
})

// 加载动态
const loadPosts = async () => {
  isLoading.value = true
  try {
    // 获取附近动态，使用默认位置
    await momentStore.fetchNearbyMoments({
      latitude: 39.9042, // 北京坐标作为默认值
      longitude: 116.4074,
      radius: 10000, // 10km范围
      limit: 20
    })
    posts.value = [...(momentStore.nearbyMoments || [])] as CheckInInfo[]
    moments.value = [...(momentStore.nearbyMoments || [])] as CheckInInfo[]
  } catch (error) {
    console.error('加载动态失败:', error)
    message.error('加载动态失败')
    posts.value = []
    moments.value = []
  } finally {
    isLoading.value = false
  }
}

// 发布动态
const publishPost = async () => {
  if (!newPostContent.value.trim()) {
    message.warning('请输入动态内容')
    return
  }

  isPublishing.value = true
  try {
    await momentStore.createMoment({
      description: newPostContent.value.trim(),
      latitude: 39.9042, // 默认位置
      longitude: 116.4074,
      location_name: '默认位置',
      tags: []
    })

    message.success('动态发布成功')
    newPostContent.value = ''

    // 重新加载动态列表
    await loadPosts()
  } catch (error) {
    console.error('发布动态失败:', error)
    message.error('发布动态失败')
  } finally {
    isPublishing.value = false
  }
}

// 刷新动态
const refreshPosts = async () => {
  isRefreshing.value = true
  try {
    await loadPosts()
    message.success('动态已刷新')
  } catch (error) {
    console.error('刷新动态失败:', error)
    message.error('刷新失败')
  } finally {
    isRefreshing.value = false
  }
}

// 切换点赞
const toggleLike = async (post: any) => {
  try {
    await momentStore.toggleLike(post.checkin_id || post.id.toString())
    message.success(post.liked_by_me ? '取消点赞成功' : '点赞成功')
  } catch (error) {
    console.error('点赞操作失败:', error)
    message.error('操作失败')
  }
}

// 显示评论
const showComments = (post: any) => {
  // TODO: 实现评论功能
  message.info('评论功能开发中')
}

// 分享动态
const sharePost = (post: any) => {
  // TODO: 实现分享功能
  message.info('分享功能开发中')
}

// 编辑动态
const editPost = (post: any) => {
  // TODO: 实现编辑功能
  message.info('编辑功能开发中')
}

// 删除动态
const deletePost = (post: any) => {
  // TODO: 实现删除功能
  message.info('删除功能开发中')
}

// 处理位置选择
const handleLocationSelected = (location: any) => {
  // TODO: 实现位置选择功能
  message.info('位置选择功能开发中')
}

// 处理图片上传
const handleImageUploaded = (images: any) => {
  // TODO: 实现图片上传功能
  message.info('图片上传功能开发中')
}

// 预览图片
const previewImage = (imageUrl: string) => {
  previewImageUrl.value = imageUrl
  showImagePreview.value = true
}

// 格式化时间
const formatTime = (timestamp: string) => {
  if (!timestamp) return ''

  const now = new Date()
  const time = new Date(timestamp)
  const diff = now.getTime() - time.getTime()
  const minutes = Math.floor(diff / 60000)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (minutes < 1440) return `${Math.floor(minutes / 60)}小时前`
  if (minutes < 10080) return `${Math.floor(minutes / 1440)}天前`

  return time.toLocaleDateString()
}
</script>

<style scoped>
/* 发布区域样式 */
.publish-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.user-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.user-avatar {
  flex-shrink: 0;
}

.user-info {
  flex: 1;
}

.user-name {
  margin: 0 0 0.25rem 0;
  font-size: 0.95rem;
  font-weight: 600;
  color: var(--text-primary);
}

.user-subtitle {
  margin: 0;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.publish-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.publish-input {
  width: 100%;
}

.publish-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.publish-options {
  display: flex;
  gap: 0.5rem;
}

/* 动态列表样式 */
.posts-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.post-item {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: 8px;
  padding: 1rem;
  transition: all 0.3s ease;
}

.post-item:hover {
  background: var(--card-bg-hover);
  border-color: var(--card-border-hover);
}

.post-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
}

.post-user {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.post-avatar {
  flex-shrink: 0;
}

.post-user-info {
  flex: 1;
}

.post-user-name {
  margin: 0 0 0.25rem 0;
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
}

.post-time {
  margin: 0;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.post-content {
  margin-bottom: 0.75rem;
}

.post-text {
  margin: 0 0 0.75rem 0;
  color: var(--text-secondary);
  line-height: 1.5;
}

.post-location {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.post-footer {
  border-top: 1px solid var(--card-border);
  padding-top: 0.75rem;
}

.post-stats {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.stat-btn {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.8rem;
  color: var(--text-muted);
  transition: color 0.3s ease;
}

.stat-btn:hover {
  color: var(--text-secondary);
}

.stat-btn.liked {
  color: var(--skill-color-3);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .publish-actions {
    flex-direction: column;
    align-items: stretch;
    gap: 0.75rem;
  }

  .publish-options {
    justify-content: center;
  }

  .post-item {
    padding: 0.75rem;
  }

  .post-header {
    margin-bottom: 0.5rem;
  }

  .post-content {
    margin-bottom: 0.5rem;
  }

  .post-footer {
    padding-top: 0.5rem;
  }

  .post-stats {
    gap: 0.75rem;
  }
}

@media (max-width: 480px) {
  .user-header {
    gap: 0.5rem;
  }

  .user-name {
    font-size: 0.9rem;
  }

  .user-subtitle {
    font-size: 0.75rem;
  }

  .post-user {
    gap: 0.5rem;
  }

  .post-user-name {
    font-size: 0.85rem;
  }

  .post-time {
    font-size: 0.7rem;
  }

  .post-stats {
    gap: 0.5rem;
  }

  .stat-btn {
    font-size: 0.75rem;
  }
}
</style>
