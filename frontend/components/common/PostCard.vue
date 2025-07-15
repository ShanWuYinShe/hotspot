<template>
  <div class="post-card">
    <!-- 帖子头部 -->
    <div class="post-header">
      <div class="user-info" @click="handleUserClick">
        <div class="user-avatar clickable">
          <img v-if="post.author.avatar" :src="post.author.avatar" :alt="post.author.name" />
          <span v-else>{{ post.author.name.charAt(0) }}</span>
        </div>
        <div class="user-details">
          <div class="user-name clickable text-responsive-base text-gold">{{ post.author.name }}</div>
          <div class="post-meta">
            <span class="post-time text-responsive-sm">{{ formatTime(post.created_at) }}</span>
            <span v-if="post.location" class="post-location text-responsive-sm text-bright">
              <Icon name="mdi:map-marker" size="16" />
              {{ post.location }}
            </span>
          </div>
        </div>
      </div>
      <div class="post-actions">
        <NDropdown trigger="click">
          <NButton type="default" class="more-btn">
            <Icon name="mdi:dots-horizontal" size="16" />
          </NButton>
          <template #dropdown>
            <NDropdownMenu>
              <NDropdownItem @click="handleSave">
                <Icon name="mdi:bookmark" size="16" />
                {{ post.isSaved ? '取消收藏' : '收藏' }}
              </NDropdownItem>
              <NDropdownItem @click="handleShare">
                <Icon name="mdi:share" size="16" />
                分享
              </NDropdownItem>
              <NDropdownItem @click="handleReport">
                <Icon name="mdi:flag" size="16" />
                举报
              </NDropdownItem>
            </NDropdownMenu>
          </template>
        </NDropdown>
        <template #fallback>
          <NButton type="default" class="more-btn">
            <Icon name="mdi:dots-horizontal" size="16" />
          </NButton>
        </template>
      </div>
    </div>

    <!-- 帖子内容 -->
    <div class="post-content">
      <div class="post-text text-responsive-base">{{ post.content }}</div>

      <!-- 心情和标签 -->
      <div v-if="post.mood || (post.tags && post.tags.length > 0)" class="post-meta-info">
        <div v-if="post.mood" class="post-mood">
          <span class="mood-icon">😊</span>
          <span class="mood-text text-responsive-sm text-electric">{{ post.mood }}</span>
        </div>
        <div v-if="post.tags && post.tags.length > 0" class="post-tags">
          <span v-for="tag in post.tags" :key="tag" class="tag text-responsive-sm text-bright">#{{ tag }}</span>
        </div>
      </div>

      <!-- 图片列表 -->
      <div v-if="post.images && post.images.length > 0" class="post-images">
        <div :class="['images-grid', `grid-${Math.min(post.images.length, 4)}`]">
          <div v-for="(image, index) in displayImages" :key="index" class="image-item" @click="openImagePreview(index)">
            <img :src="image" :alt="`图片 ${index + 1}`" />
            <div v-if="index === 3 && post.images.length > 4" class="more-images-overlay">
              <span>+{{ post.images.length - 4 }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 帖子互动数据 -->
    <div class="post-stats">
      <div class="stat-item">
        <Icon name="mdi:heart" size="16" />
        <span class="text-responsive-sm text-electric">{{ post.likes_count }}</span>
      </div>
      <div class="stat-item">
        <Icon name="mdi:message" size="16" />
        <span class="text-responsive-sm text-electric">{{ post.comments_count }}</span>
      </div>
      <div v-if="post.shares_count" class="stat-item">
        <Icon name="mdi:share" size="16" />
        <span class="text-responsive-sm text-electric">{{ post.shares_count }}</span>
      </div>
    </div>

    <!-- 帖子操作按钮 -->
    <div class="post-actions-bar">
      <NButton :type="post.isLiked ? 'primary' : 'default'" :loading="isLiking" @click="handleLike"
        class="action-btn like-btn" plain>
        <Icon name="mdi:post.isLiked ? 'heart-filled' : 'heart'" />
        <span>{{ post.isLiked ? '已赞' : '点赞' }}</span>
      </NButton>

      <NButton @click="handleComment" class="action-btn comment-btn" plain>
        <Icon name="mdi:message" size="16" />
        <span>评论</span>
      </NButton>

      <NButton @click="handleShare" class="action-btn share-btn" plain>
        <Icon name="mdi:share" size="16" />
        <span>分享</span>
      </NButton>
    </div>

    <!-- 评论列表 -->
    <div v-if="post.comments && post.comments.length > 0" class="comments-section">
      <div class="comments-header">
        <span class="comments-count">{{ post.comments_count }} 条评论</span>
        <NButton type="default" @click="toggleComments">
          {{ showComments ? '收起' : '展开' }}
        </NButton>
      </div>

      <transition name="comments">
        <div v-show="showComments" class="comments-list">
          <div v-for="comment in displayComments" :key="comment.id" class="comment-item">
            <div class="comment-avatar">
              <img v-if="comment.author.avatar" :src="comment.author.avatar" :alt="comment.author.name" />
              <span v-else>{{ comment.author.name.charAt(0) }}</span>
            </div>
            <div class="comment-content">
              <div class="comment-header">
                <span class="comment-author">{{ comment.author.name }}</span>
                <span class="comment-time">{{ formatTime(comment.created_at) }}</span>
              </div>
              <div class="comment-text">{{ comment.content }}</div>

              <!-- 显示回复 -->
              <div v-if="comment.replies && comment.replies.length > 0" class="comment-replies">
                <div v-for="reply in comment.replies.slice(0, 2)" :key="reply.id" class="reply-item">
                  <span class="reply-author">{{ reply.author.name }}</span>
                  <span class="reply-content">{{ reply.content }}</span>
                  <span class="reply-time">{{ formatTime(reply.created_at) }}</span>
                </div>

                <div v-if="comment.replies.length > 2" class="more-replies">
                  <NButton type="default" size="small" @click="showAllReplies(comment)">
                    查看全部 {{ comment.replies.length }} 条回复
                  </NButton>
                </div>
              </div>

              <div class="comment-actions">
                <NButton type="default" size="small" @click="handleCommentLike(comment)"
                  :class="{ liked: comment.isLiked }">
                  <Icon name="mdi:comment.isLiked ? 'heart-filled' : 'heart'" />
                  {{ comment.likes_count }}
                </NButton>
                <NButton type="default" size="small" @click="handleCommentReply(comment)">
                  回复
                </NButton>
              </div>
            </div>
          </div>

          <div v-if="post.comments.length > 3 && !showAllComments" class="load-more-comments">
            <NButton type="default" @click="loadMoreComments">
              查看更多评论 ({{ post.comments.length - 3 }})
            </NButton>
          </div>
        </div>
      </transition>
    </div>

    <!-- 图片预览模态框 -->
    <ImagePreviewModal v-model="showImagePreview" :images="post.images || []" :initial-index="previewImageIndex" />

    <!-- 评论回复模态框 -->
    <CommentReplyModal v-model="showReplyModal" :reply-to="currentReplyComment" @submit="handleSubmitReply" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { NButton, NDropdown, useMessage } from 'naive-ui'
import type { DropdownOption } from 'naive-ui'
import ImagePreviewModal from './ImagePreviewModal.vue'
import CommentReplyModal from './CommentReplyModal.vue'
import type { Post, Comment } from '~/types'

// API实例
const message = useMessage()

interface Props {
  post: Post
}

interface Emits {
  (e: 'like', post: Post): void
  (e: 'comment', post: Post): void
  (e: 'share', post: Post): void
  (e: 'follow', user: any): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 响应式数据
const isLiking = ref<boolean>(false)
const showComments = ref<boolean>(false)
const showAllComments = ref<boolean>(false)
const showImagePreview = ref<boolean>(false)
const previewImageIndex = ref<number>(0)
const showReplyModal = ref<boolean>(false)
const currentReplyComment = ref<Comment | null>(null)

// 计算属性
const displayImages = computed(() => {
  if (!props.post.images) return []
  return props.post.images.slice(0, 4)
})

const displayComments = computed(() => {
  if (!props.post.comments) return []
  return showAllComments.value ? props.post.comments : props.post.comments.slice(0, 3)
})

// 方法
const formatTime = (dateString: string): string => {
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`

  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}天前`

  return date.toLocaleDateString()
}

const handleLike = async () => {
  if (isLiking.value) return

  isLiking.value = true
  try {
    emit('like', props.post)
  } finally {
    setTimeout(() => {
      isLiking.value = false
    }, 300)
  }
}

const handleComment = () => {
  emit('comment', props.post)
}

const handleShare = () => {
  emit('share', props.post)
}

const handleSave = () => {
  // 切换收藏状态
  props.post.isSaved = !props.post.isSaved
  message.success(props.post.isSaved ? '收藏成功' : '取消收藏')
}

const handleReport = () => {
  message.info('举报功能开发中...')
}

const toggleComments = () => {
  showComments.value = !showComments.value
}

const loadMoreComments = () => {
  showAllComments.value = true
}

const handleCommentLike = async (comment: Comment) => {
  try {
    // TODO: 评论点赞功能暂未开放，等待后端支持
    message.warning('评论功能暂未开放')
  } catch (error) {
    console.error('评论点赞失败:', error)
    message.error('操作失败，请重试')
  }
}

const handleCommentReply = (comment: Comment) => {
  currentReplyComment.value = comment
  showReplyModal.value = true
}

const handleSubmitReply = async (content: string, replyTo: Comment) => {
  try {
    // TODO: 评论回复功能暂未开放，等待后端支持
    message.warning('评论功能暂未开放')

    // 关闭回复模态框
    showReplyModal.value = false
    currentReplyComment.value = null
  } catch (error) {
    console.error('发送回复失败:', error)
    message.error('发送回复失败，请重试')
  }
}

const showAllReplies = (comment: Comment) => {
  // 这里可以打开一个新的对话框显示所有回复
  // 或者展开显示所有回复
  message.info('查看全部回复功能开发中...')
}

const openImagePreview = (index: number) => {
  previewImageIndex.value = index
  showImagePreview.value = true
}

const handleUserClick = async () => {
  // 导航到用户资料页面
  const userId = props.post.author.id
  if (!userId) {
    message.warning('用户信息不完整')
    return
  }

  try {
    // 确保userId是数字类型
    const userIdNum = typeof userId === 'string' ? parseInt(userId) : userId

    if (isNaN(userIdNum)) {
      throw new Error('Invalid user ID')
    }

    // 使用Nuxt的导航功能
    await navigateTo(`/user/${userIdNum}`)
  } catch (error) {
    console.error('用户资料导航失败:', error)
    message.error('无法打开用户资料页面')
  }
}
</script>

<style scoped lang="scss">
.post-card {
  background: var(--el-bg-color);
  border-radius: 12px;
  padding: 1.5rem;
  margin-bottom: 1rem;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid var(--el-border-color-light);
  transition: all 0.3s ease;

  &:hover {
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
  }
}

.post-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;

  .user-info {
    display: flex;
    gap: 0.75rem;
    flex: 1;
  }

  .user-avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: var(--el-color-primary);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-weight: 600;
    overflow: hidden;
    flex-shrink: 0;
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      transform: scale(1.05);
    }

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }

  .user-details {
    flex: 1;
    min-width: 0;

    .user-name {
      font-size: 0.9rem;
      font-weight: 600;
      color: var(--el-text-color-primary);
      margin-bottom: 0.25rem;
      cursor: pointer;
      transition: color 0.2s ease;

      &:hover {
        color: var(--el-color-primary);
      }
    }

    .post-meta {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      font-size: 0.8rem;
      color: var(--el-text-color-secondary);

      .post-location {
        display: flex;
        align-items: center;
        gap: 0.25rem;
      }
    }
  }

  .post-actions {
    .more-btn {
      color: var(--el-text-color-secondary);
      padding: 0.5rem;
    }
  }
}

.post-content {
  margin-bottom: 1rem;

  .post-text {
    font-size: 0.9rem;
    line-height: 1.6;
    color: var(--el-text-color-primary);
    margin-bottom: 0.75rem;
    white-space: pre-wrap;
  }

  .post-meta-info {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    margin-bottom: 0.75rem;

    .post-mood {
      display: flex;
      align-items: center;
      gap: 0.25rem;
      padding: 0.25rem 0.5rem;
      background: var(--el-color-primary-light-9);
      border-radius: 12px;
      font-size: 0.8rem;

      .mood-icon {
        font-size: 1rem;
      }
    }

    .post-tags {
      display: flex;
      flex-wrap: wrap;
      gap: 0.5rem;

      .tag {
        color: var(--el-color-primary);
        font-size: 0.8rem;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s ease;

        &:hover {
          color: var(--el-color-primary-dark-2);
        }
      }
    }
  }
}

.post-images {
  margin-bottom: 0.75rem;

  .images-grid {
    display: grid;
    gap: 0.5rem;
    border-radius: 8px;
    overflow: hidden;

    &.grid-1 {
      grid-template-columns: 1fr;
      max-height: 400px;
    }

    &.grid-2 {
      grid-template-columns: 1fr 1fr;
      max-height: 300px;
    }

    &.grid-3 {
      grid-template-columns: 2fr 1fr;
      grid-template-rows: 1fr 1fr;
      max-height: 300px;

      .image-item:first-child {
        grid-row: 1 / 3;
      }
    }

    &.grid-4 {
      grid-template-columns: 1fr 1fr;
      grid-template-rows: 1fr 1fr;
      max-height: 300px;
    }

    .image-item {
      position: relative;
      background: var(--el-fill-color-light);
      border-radius: 4px;
      overflow: hidden;
      cursor: pointer;
      transition: all 0.2s ease;

      &:hover {
        transform: scale(1.02);
      }

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        transition: all 0.2s ease;
      }

      .more-images-overlay {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.6);
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
        font-size: 1.25rem;
        font-weight: 600;
      }
    }
  }
}

.post-stats {
  display: flex;
  gap: 1.5rem;
  margin-bottom: 0.75rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--el-border-color-lighter);

  .stat-item {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    color: var(--el-text-color-secondary);
    font-size: 0.8rem;
  }
}

.post-actions-bar {
  display: flex;
  gap: 1rem;

  .action-btn {
    flex: 1;
    height: 36px;
    font-size: 0.85rem;
    border-radius: 18px;
    transition: all 0.2s ease;

    &.like-btn {
      &:not(.is-plain) {
        background: var(--el-color-danger);
        border-color: var(--el-color-danger);
        color: white;

        &:hover {
          background: var(--el-color-danger-dark-2);
          border-color: var(--el-color-danger-dark-2);
        }
      }
    }

    &:hover {
      transform: translateY(-2px);
    }
  }
}

.comments-section {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--el-border-color-lighter);

  .comments-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.75rem;

    .comments-count {
      font-size: 0.85rem;
      color: var(--el-text-color-secondary);
      font-weight: 500;
    }
  }

  .comments-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .comment-item {
    display: flex;
    gap: 0.75rem;

    .comment-avatar {
      width: 32px;
      height: 32px;
      border-radius: 50%;
      background: var(--el-color-primary-light-3);
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-size: 0.8rem;
      font-weight: 600;
      overflow: hidden;
      flex-shrink: 0;

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
    }

    .comment-content {
      flex: 1;
      min-width: 0;

      .comment-header {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        margin-bottom: 0.25rem;

        .comment-author {
          font-size: 0.8rem;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }

        .comment-time {
          font-size: 0.75rem;
          color: var(--el-text-color-secondary);
        }
      }

      .comment-text {
        font-size: 0.85rem;
        line-height: 1.5;
        color: var(--el-text-color-primary);
        margin-bottom: 0.5rem;
      }

      .comment-replies {
        margin: 0.5rem 0;
        padding-left: 1rem;
        border-left: 2px solid var(--el-border-color-lighter);

        .reply-item {
          display: flex;
          align-items: baseline;
          gap: 0.5rem;
          margin-bottom: 0.25rem;
          font-size: 0.8rem;

          .reply-author {
            color: var(--el-color-primary);
            font-weight: 600;
            flex-shrink: 0;
          }

          .reply-content {
            color: var(--el-text-color-primary);
            flex: 1;
          }

          .reply-time {
            color: var(--el-text-color-secondary);
            font-size: 0.75rem;
            flex-shrink: 0;
          }
        }

        .more-replies {
          margin-top: 0.25rem;

          .el-button {
            font-size: 0.75rem;
            color: var(--el-color-primary);
            padding: 0;
            height: auto;
          }
        }
      }

      .comment-actions {
        display: flex;
        gap: 1rem;

        .el-button {
          padding: 0;
          height: auto;
          font-size: 0.75rem;
          color: var(--el-text-color-secondary);

          &.liked {
            color: var(--el-color-danger);
          }

          &:hover {
            color: var(--el-color-primary);
          }
        }
      }
    }
  }

  .load-more-comments {
    text-align: center;
    padding-top: 0.5rem;

    .el-button {
      font-size: 0.8rem;
      color: var(--el-color-primary);
    }
  }
}

// 过渡动画
.comments-enter-active,
.comments-leave-active {
  transition: all 0.3s ease;
}

.comments-enter-from,
.comments-leave-to {
  opacity: 0;
  max-height: 0;
  overflow: hidden;
}

.comments-enter-to,
.comments-leave-from {
  opacity: 1;
  max-height: 500px;
}

// 响应式设计
@media (max-width: 768px) {
  .post-card {
    padding: 1rem;
    margin-bottom: 0.75rem;
    border-radius: 8px;
  }

  .post-header {
    .user-avatar {
      width: 36px;
      height: 36px;
    }

    .user-details {
      .user-name {
        font-size: 0.85rem;
      }

      .post-meta {
        font-size: 0.75rem;
      }
    }
  }

  .post-content {
    .post-text {
      font-size: 0.85rem;
    }
  }

  .post-actions-bar {
    gap: 0.5rem;

    .action-btn {
      height: 32px;
      font-size: 0.8rem;
    }
  }

  .post-images {
    .images-grid {
      &.grid-1 {
        max-height: 300px;
      }

      &.grid-2,
      &.grid-3,
      &.grid-4 {
        max-height: 250px;
      }
    }
  }
}

// 深色模式适配
.dark {
  .post-card {
    background: var(--el-bg-color);
    border-color: var(--el-border-color);

    &:hover {
      box-shadow: 0 4px 20px rgba(255, 255, 255, 0.05);
    }
  }

  .post-meta-info {
    .post-mood {
      background: var(--el-color-primary-light-8);
    }
  }
}
</style>
