<template>
  <div class="app-page profile-page">
    <div class="page-header">
      <h1 class="page-title">个人资料</h1>
      <p class="page-subtitle">管理您的个人信息</p>
    </div>

    <div class="app-container">
      <!-- 用户信息卡片 -->
      <div class="content-card profile-card">
        <div class="profile-header">
          <div class="profile-header__avatar">
            <NAvatar :size="120" :src="userInfo?.avatar_url || defaultImages.userAvatar">
              <img :src="defaultImages.userAvatar" />
            </NAvatar>
            <NButton class="avatar-edit-btn" type="primary" size="small" circle @click="showProfileEditDialog = true">
              <NIcon>
                <Edit />
              </NIcon>
            </NButton>
          </div>

          <div class="profile-header__info">
            <h2 class="profile-name">{{ userInfo?.nickname || '未设置昵称' }}</h2>
            <p class="profile-id">ID: {{ userInfo?.user_id }}</p>
            <p class="profile-bio">{{ userInfo?.bio || '这个人很神秘，什么都没有留下' }}</p>

            <div class="profile-stats">
              <div class="stat-item">
                <div class="stat-value">{{ userStats.total_checkins }}</div>
                <div class="stat-label">签到次数</div>
              </div>
              <div class="stat-item">
                <div class="stat-value">{{ userStats.total_groups }}</div>
                <div class="stat-label">加入群组</div>
              </div>
              <div class="stat-item">
                <div class="stat-value">{{ userStats.total_messages }}</div>
                <div class="stat-label">发送消息</div>
              </div>
            </div>
          </div>
        </div>

        <div class="profile-actions">
          <NButton type="primary" @click="showProfileEditDialog = true">
            <NIcon>
              <Edit />
            </NIcon>
            编辑资料
          </NButton>

          <NButton type="default" @click="() => navigateTo('/settings')">
            <NIcon>
              <Setting />
            </NIcon>
            设置
          </NButton>
        </div>
      </div>

      <!-- 最近活动 -->
      <div class="content-card">
        <div class="card-header">
          <h3 class="card-title">最近活动</h3>
          <NButton type="primary" size="small" @click="loadRecentActivities" :loading="isLoadingActivities">
            <NIcon>
              <Refresh />
            </NIcon>
            刷新
          </NButton>
        </div>

        <div v-if="recentActivities.length > 0" class="activity-list">
          <div v-for="activity in recentActivities" :key="activity.id" class="activity-item">
            <div class="activity-icon">
              <NIcon>
                <component :is="getActivityIcon(activity.type)" />
              </NIcon>
            </div>

            <div class="activity-content">
              <h4 class="activity-title">{{ activity.title }}</h4>
              <p class="activity-description">{{ activity.description }}</p>
              <span class="activity-time">{{ formatTime(activity.created_at) }}</span>
            </div>
          </div>
        </div>

        <div v-else class="empty-state">
          <div class="empty-state__icon">
            <NIcon>
              <Clock />
            </NIcon>
          </div>
          <h4 class="empty-state__title">暂无活动记录</h4>
          <p class="empty-state__description">开始使用应用，记录您的精彩时刻</p>
        </div>
      </div>

      <!-- 我的群组 -->
      <div class="content-card">
        <div class="card-header">
          <h3 class="card-title">我的群组</h3>
          <NButton type="primary" size="small" @click="() => navigateTo('/group')">
            查看全部
          </NButton>
        </div>

        <div v-if="myGroups.length > 0" class="content-grid content-grid--3-cols">
          <div v-for="group in myGroups.slice(0, 6)" :key="group.group_id" class="group-card"
            @click="navigateToGroup(group.group_id)">
            <div class="group-avatar">
              <img :src="defaultImages.groupAvatar" :alt="group.name">
            </div>
            <div class="group-info">
              <h4 class="group-name">{{ group.name }}</h4>
              <p class="group-members">{{ group.member_count }} 成员</p>
            </div>
          </div>
        </div>

        <div v-else class="empty-state">
          <div class="empty-state__icon">
            <NIcon>
              <UserFilled />
            </NIcon>
          </div>
          <h4 class="empty-state__title">暂未加入群组</h4>
          <p class="empty-state__description">加入群组，与更多朋友交流</p>
          <NButton type="primary" @click="() => navigateTo('/group')">
            发现群组
          </NButton>
        </div>
      </div>
    </div>

    <!-- 编辑资料对话框 -->
    <ProfileEditDialog v-model="showProfileEditDialog" :user-info="userInfo" @success="handleProfileUpdateSuccess" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NButton, NIcon, NAvatar, useMessage } from 'naive-ui'
import { CreateOutline as Edit, Person as User, Star, Settings as Setting, Refresh, Time as Clock, Person as UserFilled } from '@vicons/ionicons5'
import ProfileEditDialog from '~/components/common/ProfileEditDialog.vue'
import { defaultImages } from '~/utils/assets'
import type { UserProfile } from '~/types/models/user'
import type { GroupInfo } from '~/types/api/group'

// API实例
const message = useMessage()

// 页面元数据
definePageMeta({
  layout: 'default',
  title: '个人资料',
  middleware: 'auth'
})

// 响应式数据
const showProfileEditDialog = ref<boolean>(false)
const userInfo = ref<UserProfile | null>(null)
const userStats = ref<{
  total_checkins: number,
  total_groups: number,
  total_messages: number
}>({
  total_checkins: 0,
  total_groups: 0,
  total_messages: 0
})
const recentActivities = ref<any[]>([])
const isLoadingActivities = ref<boolean>(false)
const myGroups = ref<GroupInfo[]>([])

// Store
const authStore = useAuthStore()
const userStore = useUserStore()
const momentStore = useMomentStore()

// 页面同步
onMounted(() => {
  loadUserData()
})

// 加载用户数据
const loadUserData = async () => {
  try {
    userInfo.value = authStore.user

    // 加载统计信息
    await loadStats()

    // 加载最近动态
    await loadRecentMoments()
  } catch (error) {
    console.error('加载用户数据失败:', error)
  }
}

// 加载统计信息
const loadStats = async () => {
  try {
    // TODO: 实现用户统计信息获取
    userStats.value = {
      total_checkins: 0,
      total_groups: 0,
      total_messages: 0
    }
  } catch (error) {
    console.error('加载统计信息失败:', error)
    userStats.value = {
      total_checkins: 0,
      total_groups: 0,
      total_messages: 0
    }
  }
}

// 加载最近动态
const loadRecentMoments = async () => {
  try {
    if (userInfo.value?.user_id) {
      const data = await momentStore.fetchUserMoments({ user_id: userInfo.value.user_id, limit: 5 })
    }
  } catch (error) {
    console.error('加载最近动态失败:', error)
  }
}

// 加载最近活动
const loadRecentActivities = async () => {
  isLoadingActivities.value = true
  try {
    // TODO: 实现最近活动获取
    recentActivities.value = []
  } catch (error) {
    console.error('加载最近活动失败:', error)
    message.error('加载最近活动失败')
  } finally {
    isLoadingActivities.value = false
  }
}

// 处理资料更新
const handleProfileUpdated = async (updatedInfo: UserProfile) => {
  userInfo.value = updatedInfo
  message.success('资料更新成功')

  // 重新加载数据
  await loadUserData()
}

// 处理资料更新成功
const handleProfileUpdateSuccess = async () => {
  await loadUserData()
  showProfileEditDialog.value = false
}

// 获取活动图标
const getActivityIcon = (type: string) => {
  switch (type) {
    case 'moment':
      return Star
    case 'group':
      return UserFilled
    case 'message':
      return User
    default:
      return Clock
  }
}

// 导航到群组详情
const navigateToGroup = (groupId: string) => {
  navigateTo(`/group/${groupId}`)
}

// 格式化日期
const formatDate = (timestamp: string | number | Date) => {
  if (!timestamp) return '未知'
  return new Date(timestamp).toLocaleDateString()
}

// 格式化时间
const formatTime = (timestamp: string | number | Date) => {
  if (!timestamp) return ''

  const now = new Date()
  const time = new Date(timestamp)
  const diff = now.getTime() - time.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (days === 0) return '今天'
  if (days === 1) return '昨天'
  if (days < 7) return `${days}天前`

  return time.toLocaleDateString()
}
</script>