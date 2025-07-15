<template>
  <div class="user-detail-page">
    <div class="user-detail-container">
      <!-- 页面头部 -->
      <div class="page-header">
        <BackButton />
        <h2 class="page-title">用户资料</h2>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="loading-state">
        <NSkeleton :rows="10" animated />
      </div>

      <!-- 用户信息 -->
      <div v-else-if="user" class="user-profile">
        <!-- 用户头部信息 -->
        <div class="user-header">
          <div class="user-cover">
            <!-- 背景图 -->
            <div class="cover-image">
              <div class="default-cover"></div>
            </div>

            <!-- 用户基本信息 -->
            <div class="user-basic-info">
              <div class="avatar-section">
                <NAvatar :size="100" :src="user.avatar_url" class="user-avatar">
                  {{ user.nickname?.charAt(0) || 'U' }}
                </NAvatar>

                <!-- 在线状态 -->
                <div class="online-indicator"></div>
              </div>

              <div class="user-info">
                <h1 class="username text-responsive-2xl text-gold">{{ user.nickname }}</h1>
                <p class="user-id text-responsive-sm text-bright">ID: {{ user.user_id }}</p>

                <!-- 位置信息 -->
                <!-- 注释：用户个人资料中暂不显示位置信息，因为UserProfile类型中没有location_name字段 -->
                <!-- <div v-if="user.location_name" class="location-info">
                  <NIcon><Location /></NIcon>
                  <span>{{ user.location_name }}</span>
                </div> -->
              </div>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="user-actions">
            <NButton v-if="!isCurrentUser" type="primary" @click="handleSendMessage" :icon="ChatDotRound">
              发消息
            </NButton>
            <NButton v-if="!isCurrentUser" @click="handleAddFriend" :icon="UserFilled">
              加好友
            </NButton>
            <NButton v-if="isCurrentUser" @click="handleEditProfile" :icon="Edit">
              编辑资料
            </NButton>
          </div>

          <!-- 用户统计 -->
          <div class="user-stats">
            <div class="stat-item" @click="handleStatsClick('moments')">
              <div class="stat-number text-responsive-xl text-electric">{{ userStats.moments || 0 }}</div>
              <div class="stat-label text-responsive-sm text-bright">动态</div>
            </div>
            <div class="stat-divider"></div>
            <div class="stat-item" @click="handleStatsClick('groups')">
              <div class="stat-number text-responsive-xl text-electric">{{ userStats.groups || 0 }}</div>
              <div class="stat-label text-responsive-sm text-bright">群组</div>
            </div>
            <div class="stat-divider"></div>
            <div class="stat-item" @click="handleStatsClick('friends')">
              <div class="stat-number text-responsive-xl text-electric">{{ userStats.friends || 0 }}</div>
              <div class="stat-label text-responsive-sm text-bright">好友</div>
            </div>
          </div>
        </div>

        <!-- 内容标签页 -->
        <div class="content-tabs">
          <UTabs v-model="activeTab" @tab-change="handleTabChange">
            <UTabPane label="动态" name="moments">
              <template #label>
                <span class="tab-label">
                  动态
                  <NBadge v-if="contentCounts.moments > 0" :value="contentCounts.moments" :max="99"
                    class="tab-badge" />
                </span>
              </template>
            </UTabPane>
            <UTabPane label="相册" name="photos">
              <template #label>
                <span class="tab-label">
                  相册
                  <NBadge v-if="contentCounts.photos > 0" :value="contentCounts.photos" :max="99" class="tab-badge" />
                </span>
              </template>
            </UTabPane>
          </UTabs>
        </div>

        <!-- 下拉刷新 -->
        <PullRefresh @refresh="handleRefresh">
          <!-- 内容区域 -->
          <div class="content-area">
            <!-- 动态列表 -->
            <div v-if="activeTab === 'moments'" class="moments-content">
              <div v-if="contentLoading && !moments.length" class="loading-state">
                <NSkeleton :rows="3" animated />
              </div>

              <div v-else-if="moments.length === 0" class="empty-state">
                <div class="empty-icon">📍</div>
                <h3>暂无动态</h3>
                <p v-if="isCurrentUser">开始您的第一条动态吧</p>
                <p v-else">该用户还没有动态记录</p>
              </div>

              <div v-else class="moments-list">
                <div v-for="moment in moments" :key="moment.id" class="moment-item">
                  <div class="moment-header">
                    <div class="moment-user">{{ moment.user?.nickname || '用户' }}</div>
                    <div class="moment-time">{{ formatTime(moment.created_at) }}</div>
                  </div>
                  <div class="moment-content">
                    <p>{{ moment.content }}</p>
                    <div v-if="moment.location_name" class="moment-location">
                      📍 {{ moment.location_name }}
                    </div>
                  </div>
                  <div class="moment-actions">
                    <NButton text @click="handleLike(moment)">
                      <NIcon>
                        <Star />
                      </NIcon>
                      {{ moment.likes_count || 0 }}
                    </NButton>
                    <NButton text @click="handleComment(moment)">
                      <NIcon>
                        <ChatDotRound />
                      </NIcon>
                      {{ moment.comments_count || 0 }}
                    </NButton>
                  </div>
                </div>
              </div>
            </div>

            <!-- 相册 -->
            <div v-else-if="activeTab === 'photos'" class="photos-content">
              <div v-if="contentLoading && !photos.length" class="loading-state">
                <NSkeleton :rows="3" animated />
              </div>

              <div v-else-if="photos.length === 0" class="empty-state">
                <div class="empty-icon">📸</div>
                <h3>暂无照片</h3>
                <p v-if="isCurrentUser">上传您的精彩照片</p>
                <p v-else>该用户还没有上传照片</p>
              </div>

              <div v-else class="photos-grid">
                <div v-for="(photo, index) in photos" :key="photo.id" class="photo-item"
                  @click="handleImagePreview(photos.map(p => p.url), index)">
                  <img :src="photo.thumbnail || photo.url" :alt="photo.alt" />
                  <div class="photo-overlay">
                    <NIcon>
                      <View />
                    </NIcon>
                  </div>
                </div>
              </div>
            </div>

            <!-- 加载更多 -->
            <div v-if="hasMore" class="load-more">
              <NButton @click="loadMore" :loading="contentLoading" block>
                加载更多
              </NButton>
            </div>
          </div>
        </PullRefresh>
      </div>

      <!-- 用户不存在 -->
      <div v-else class="user-not-found">
        <div class="error-icon">😔</div>
        <h3>用户不存在</h3>
        <p>该用户可能已被删除或不存在</p>
        <NButton type="primary" @click="handleBack">
          返回
        </NButton>
      </div>
    </div>

    <!-- 浮动操作按钮 -->
    <FloatingActionButton v-if="!isCurrentUser" :actions="fabActions" @action="handleFabAction" class="compact-only" />

    <!-- 创建群组对话框 -->
    <CreateGroupDialog v-model="showCreateGroupDialog" @success="handleCreateGroupSuccess" />

    <!-- 新消息对话框 -->
    <NewMessageDialog v-model="showNewMessageDialog" @success="handleNewMessageSuccess" />
  </div>
</template>

<script setup lang="ts">
// 手动导入 Vue API
import { ref, computed, onMounted, watch } from 'vue'

// 手动导入 Naive UI 组件
import { NAvatar, NButton, NTabs, NSkeleton, NTabPane, useMessage } from 'naive-ui'

// API实例
const message = useMessage()

// 手动导入 Naive UI 图标
import { Chatbubble as ChatDotRound, Person as UserFilled, CreateOutline as Edit, Location, Calendar, Add as Plus, Search, Share, Star, Eye as View } from '@vicons/ionicons5'

// 导入组件
import { useAuthStore } from '~/stores/auth'
import { useUserStore } from '~/stores/user'
import { useUserService, useMomentService } from '~/services'
import type { UserProfile } from '~/types/models/user'

// 服务实例
const { searchUsersById } = useUserService()
const { getUserMoments } = useMomentService()

// 手动导入组件
import BackButton from '~/components/common/BackButton.vue'
import FloatingActionButton from '~/components/common/FloatingActionButton.vue'
import CreateGroupDialog from '~/components/group/CreateGroupDialog.vue'
import NewMessageDialog from '~/components/message/NewMessageDialog.vue'
import PullRefresh from '~/components/common/PullRefresh.vue'
import { extractErrorMessage } from '~/utils/error-handler'

// 页面配置
definePageMeta({
  title: '用户详情'
})

// 路由参数
const route = useRoute()
const userId = computed(() => route.params.id as string)

// 状态管理
const authStore = useAuthStore()
const userStore = useUserStore()
const router = useRouter()

// 页面状态
const loading = ref(false)
const contentLoading = ref(false)
const activeTab = ref('moments')
const hasMore = ref(true)
const showCreateGroupDialog = ref(false)
const showNewMessageDialog = ref(false)

// 用户数据
const user = ref<UserProfile | null>(null)

// 用户统计数据
const userStats = ref({
  moments: 0,
  groups: 0,
  friends: 0
})

// 内容数据
const moments = ref<any[]>([])
const photos = ref<any[]>([])

// 分页状态
const currentPage = ref(1)
const pageSize = 20

// 是否为当前用户
const isCurrentUser = computed(() => {
  return authStore.user?.user_id?.toString() === userId.value
})

// 内容计数
const contentCounts = computed(() => {
  return {
    moments: moments.value.length,
    photos: photos.value.length
  }
})

// 浮动操作按钮配置
const fabActions = ref([
  {
    id: 'message',
    label: '发消息',
    icon: 'ChatDotRound',
    color: '#409EFF'
  },
  {
    id: 'friend',
    label: '加好友',
    icon: 'UserFilled',
    color: '#67C23A'
  },
  {
    id: 'share',
    label: '分享',
    icon: 'Share',
    color: '#E6A23C'
  }
])

// 事件处理
const handleBack = () => {
  router.back()
}

const handleMore = () => {
  message.info('更多功能开发中...')
}

const handleSendMessage = () => {
  if (!authStore.isAuthenticated) {
    navigateTo('/auth')
    return
  }

  navigateTo(`/chat?userId=${userId.value}`)
}

const handleAddFriend = async () => {
  if (!authStore.isAuthenticated) {
    navigateTo('/auth')
    return
  }

  try {
    // TODO: Call real API for adding friend when implemented
    message.info('好友功能开发中...')
    // Example: await friendService.sendFriendRequest({ user_id: parseInt(userId.value) })
    // if (user.value) {
    //   user.value.isFriend = true
    // }
    // message.success('好友请求已发送')
  } catch (error: any) {
    const errorMessage = extractErrorMessage(error, '发送好友请求失败')
    console.error('发送好友请求失败:', error)
    message.error(errorMessage)
  }
}

const handleEditProfile = () => {
  navigateTo('/profile/edit')
}

const handleStatsClick = (type: string) => {
  switch (type) {
    case 'moments':
      activeTab.value = 'moments'
      break
    case 'groups':
      // 如果是当前用户，导航到我的群组页面
      if (isCurrentUser.value) {
        navigateTo('/my-groups')
      } else {
        // 其他用户的群组列表功能待开发
        message.info('查看用户群组功能开发中...')
      }
      break
    case 'friends':
      // 好友列表功能待开发
      message.info('好友功能开发中...')
      break
  }
}

const handleTabChange = (tabName: string | number) => {
  activeTab.value = String(tabName)
  currentPage.value = 1 // Reset pagination when changing tabs
  loadTabContent()
}

const handleRefresh = async () => {
  currentPage.value = 1 // Reset pagination on refresh
  await loadUserData()
  await loadTabContent()
}

// Dialog 成功处理器
const handleCreateGroupSuccess = async (group: any) => {
  message.success('群组创建成功！')
  await navigateTo(`/group/${group.group_id}/chat`)
}

const handleMomentSuccess = async (moment: any) => {
  message.success('动态创建成功！')
  await navigateTo(`/moment/${moment.id}`)
}

const handleNewMessageSuccess = async (conversation: any) => {
  message.success('消息发送成功！')
  await navigateTo(`/chat/${conversation.conversation_id}`)
}

const loadMore = async () => {
  if (contentLoading.value || !hasMore.value) return

  currentPage.value++
  contentLoading.value = true
  try {
    if (activeTab.value === 'moments') {
      const userIdNum = parseInt(userId.value)
      if (!isNaN(userIdNum)) {
        const response = await getUserMoments({ user_id: userIdNum, limit: pageSize })
        moments.value.push(...(response?.items || []))
        hasMore.value = response?.items && response?.items.length === pageSize || false
      }
    }
    // TODO: Add pagination for activities and photos when APIs are available
  } catch (error: any) {
    const errorMessage = extractErrorMessage(error, '加载失败')
    console.error('加载更多内容失败:', error)
    message.error(errorMessage)
    currentPage.value-- // Revert page increment on error
  } finally {
    contentLoading.value = false
  }
}

const handleFabAction = (actionId: string) => {
  switch (actionId) {
    case 'message':
      handleSendMessage()
      break
    case 'friend':
      handleAddFriend()
      break
    case 'share':
      message.info('分享功能开发中...')
      break
  }
}

const handleLike = async (activity: any) => {
  try {
    activity.likes = (activity.likes || 0) + 1
    message.success('已点赞')
  } catch (error) {
    message.error('点赞失败')
  }
}

const handleComment = (activity: any) => {
  message.info('评论功能开发中...')
}

const handleImagePreview = (images: string[], index: number) => {
  // 这里可以打开图片预览组件
  message.info('图片预览功能开发中...')
}

// 格式化距离
const formatDistance = (distance: number) => {
  if (distance < 1000) {
    return `${Math.round(distance)}m`
  } else {
    return `${(distance / 1000).toFixed(1)}km`
  }
}

// 格式化时间
const formatTime = (timestamp: number) => {
  const now = new Date().getTime()
  const diff = now - timestamp

  if (diff < 1000 * 60) {
    return '刚刚'
  } else if (diff < 1000 * 60 * 60) {
    return `${Math.floor(diff / (1000 * 60))}分钟前`
  } else if (diff < 1000 * 60 * 60 * 24) {
    return `${Math.floor(diff / (1000 * 60 * 60))}小时前`
  } else {
    return new Date(timestamp).toLocaleDateString()
  }
}

// 加载用户数据
const loadUserData = async () => {
  loading.value = true
  try {
    const userIdNum = parseInt(userId.value)
    if (isNaN(userIdNum)) {
      throw new Error('Invalid user ID')
    }

    const userData = await searchUsersById({ user_id: userIdNum })
    user.value = userData

    // Load user statistics
    try {
      // Get user moments count
      const momentsResponse = await getUserMoments({ user_id: userIdNum, limit: 1 })
      userStats.value.moments = momentsResponse.items.length || 0

      // TODO: Load groups and friends count when APIs are available
      userStats.value.groups = 0
      userStats.value.friends = 0
    } catch (statsError) {
      console.warn('Failed to load user stats:', statsError)
    }
  } catch (error: any) {
    console.error('加载用户数据失败:', error)
    const errorMessage = extractErrorMessage(error, '加载用户数据失败')
    message.error(errorMessage)
    user.value = null
  } finally {
    loading.value = false
  }
}

// 加载标签页内容
const loadTabContent = async () => {
  contentLoading.value = true
  try {
    if (activeTab.value === 'moments') {
      // Load user moments
      const userIdNum = parseInt(userId.value)
      if (!isNaN(userIdNum)) {
        const response = await getUserMoments({ user_id: userIdNum, limit: pageSize })
        if (currentPage.value === 1) {
          moments.value = response?.items || []
        } else {
          moments.value.push(...(response?.items || []))
        }
        hasMore.value = response?.items && response?.items.length === pageSize || false
      }
    } else if (activeTab.value === 'photos') {
      // TODO: Load user photos when API is available
      photos.value = []
      message.info('相册功能开发中...')
    }
  } catch (error: any) {
    const errorMessage = extractErrorMessage(error, '加载内容失败')
    console.error('加载内容失败:', error)
    message.error(errorMessage)
  } finally {
    contentLoading.value = false
  }
}

// 页面加载时获取数据
onMounted(async () => {
  await loadUserData()
  await loadTabContent()
})

// 监听路由变化
watch(() => route.params.id, async (newId) => {
  if (newId) {
    await loadUserData()
    await loadTabContent()
  }
})
</script>

<style scoped lang="scss">
.user-detail-page {
  min-height: 100vh;
  background: var(--color-bg-primary);
}

.user-detail-container {
  max-width: 800px;
  margin: 0 auto;
  max-height: calc(100vh - 100px);
  /* 增加显示空间 */
  min-height: calc(100vh - 140px);
  overflow-y: auto;
  scroll-behavior: smooth;
  -webkit-overflow-scrolling: touch;
  padding: 1rem;

  /* 使用隐藏滚动条的设置 */
  &::-webkit-scrollbar {
    width: 0px;
    background: transparent;
  }

  scrollbar-width: none;
  -ms-overflow-style: none;

  /* 滚动体验优化 */
  overscroll-behavior-y: contain;
  touch-action: pan-y;
  padding-bottom: 2rem;

  @media (max-width: 768px) {
    max-height: calc(100vh - 80px);
    /* 移动端更少留白 */
    min-height: calc(100vh - 100px);
    padding: 0.5rem;
    padding-bottom: 1.5rem;
  }
}

// 页面头部
.page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 24px;
  background: var(--pixel-bg-primary);
  border-bottom: 1px solid var(--pixel-border-primary);
  position: sticky;
  top: 0;
  z-index: var(--z-sticky);
  backdrop-filter: blur(10px);

  @media (max-width: 768px) {
    padding: 12px 16px;
  }
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--pixel-text-primary);
  margin: 0;

  @media (max-width: 768px) {
    font-size: 16px;
  }
}

// 加载状态
.loading-state {
  padding: 2rem;
}

// 用户头部
.user-header {
  background: var(--el-bg-color);
  border-radius: 12px;
  overflow: hidden;
  margin-bottom: 1rem;
  box-shadow: var(--el-box-shadow-light);
}

.user-cover {
  position: relative;
}

.cover-image {
  height: 200px;
  background: linear-gradient(135deg, var(--el-color-primary-light-7), var(--el-color-primary-light-5));

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.default-cover {
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.user-basic-info {
  padding: 1.5rem;
  margin-top: -50px;
  position: relative;
  z-index: var(--z-content);
}

.avatar-section {
  position: relative;
  display: inline-block;
  margin-bottom: 1rem;
}

.user-avatar {
  border: 4px solid var(--el-bg-color);
  box-shadow: var(--el-box-shadow);
}

.online-indicator {
  position: absolute;
  bottom: 5px;
  right: 5px;
  width: 20px;
  height: 20px;
  background: var(--el-color-success);
  border: 3px solid var(--el-bg-color);
  border-radius: 50%;
}

.user-info {
  margin-top: 1rem;
}

.username {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  margin: 0 0 0.5rem 0;
}

.user-id {
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
  margin: 0 0 1rem 0;
}

.user-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.user-tag {
  border-radius: 12px;
}

.location-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  margin-bottom: 1rem;
}

.distance {
  color: var(--el-color-primary);
  font-weight: 500;
}

.user-bio {
  color: var(--el-text-color-regular);
  line-height: 1.6;
  margin: 0;
}

// 操作按钮
.user-actions {
  display: flex;
  gap: 1rem;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

// 用户统计
.user-stats {
  display: flex;
  padding: 1.5rem;
}

.stat-item {
  flex: 1;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    transform: translateY(-2px);
  }
}

.stat-number {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--el-color-primary);
  margin-bottom: 0.25rem;
}

.stat-label {
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}

.stat-divider {
  width: 1px;
  background: var(--el-border-color-lighter);
  margin: 0 1rem;
}

// 内容标签页
.content-tabs {
  background: var(--el-bg-color);
  border-radius: 12px;
  padding: 0 1rem;
  margin-bottom: 1rem;
  box-shadow: var(--el-box-shadow-light);
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.tab-badge {
  :deep(.el-badge__content) {
    font-size: 10px;
    min-width: 16px;
    height: 16px;
    line-height: 16px;
  }
}

// 内容区域
.content-area {
  background: var(--el-bg-color);
  border-radius: 12px;
  min-height: 300px;
  box-shadow: var(--el-box-shadow-light);
}

// 加载和空状态
.empty-state {
  padding: 3rem 2rem;
  text-align: center;
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
  opacity: 0.6;
}

.empty-state h3 {
  color: var(--el-text-color-primary);
  margin-bottom: 0.5rem;
}

.empty-state p {
  color: var(--el-text-color-secondary);
}

// 动态列表
.activities-list {
  padding: 1rem;
}

.activity-item {
  padding: 1.5rem;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: none;
  }
}

.activity-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.activity-time {
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}

.activity-type {
  font-size: 0.75rem;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  padding: 0.25rem 0.5rem;
  border-radius: 8px;
}

.activity-content {
  margin-bottom: 1rem;

  p {
    color: var(--el-text-color-primary);
    line-height: 1.6;
    margin: 0 0 1rem 0;
  }
}

.activity-images {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 0.5rem;
}

.activity-image {
  width: 100%;
  height: 120px;
  object-fit: cover;
  border-radius: 8px;
  cursor: pointer;
  transition: transform 0.2s ease;

  &:hover {
    transform: scale(1.02);
  }
}

.activity-actions {
  display: flex;
  gap: 1rem;
}

// 动态列表
.moments-list {
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.moment-item {
  border-radius: 8px;
  overflow: hidden;
  background: var(--el-bg-color);
  padding: 1rem;
  box-shadow: var(--el-box-shadow-light);
}

.moment-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.moment-user {
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.moment-time {
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}

.moment-content {
  margin-bottom: 1rem;

  p {
    color: var(--el-text-color-primary);
    line-height: 1.6;
    margin: 0 0 0.5rem 0;
  }
}

.moment-location {
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
  margin-top: 0.5rem;
}

.moment-actions {
  display: flex;
  gap: 1rem;
}

// 相册网格
.photos-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 1rem;
  padding: 1rem;
}

.photo-item {
  position: relative;
  aspect-ratio: 1;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    transition: transform 0.2s ease;
  }

  &:hover {
    img {
      transform: scale(1.05);
    }

    .photo-overlay {
      opacity: 1;
    }
  }
}

.photo-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s ease;

  .el-icon {
    color: white;
    font-size: 1.5rem;
  }
}

// 加载更多
.load-more {
  padding: 1rem;
  border-top: 1px solid var(--el-border-color-lighter);
}

// 用户不存在
.user-not-found {
  text-align: center;
  padding: 3rem 2rem;
}

.error-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}

.user-not-found h3 {
  color: var(--el-text-color-primary);
  margin-bottom: 0.5rem;
}

.user-not-found p {
  color: var(--el-text-color-secondary);
  margin-bottom: 2rem;
}

// 响应式设计
@media (max-width: 768px) {
  .user-detail-container {
    padding: 0.5rem;
  }

  .user-basic-info {
    padding: 1rem;
  }

  .user-actions {
    padding: 1rem;
    flex-direction: column;
    gap: 0.5rem;
  }

  .user-stats {
    padding: 1rem;
  }

  .stat-number {
    font-size: 1.25rem;
  }

  .photos-grid {
    grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
    gap: 0.5rem;
  }

  .activity-images {
    grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
  }

  .activity-image {
    height: 100px;
  }
}

// 深色模式适配
.dark {

  .user-header,
  .content-tabs,
  .content-area {
    background: var(--el-bg-color);
    border-color: var(--el-border-color);
  }

  .default-cover {
    background: linear-gradient(135deg, #4c63d2 0%, #5a67d8 100%);
  }
}

// 动画效果
.user-header,
.content-tabs,
.content-area {
  animation: slideUp 0.3s ease-out;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
