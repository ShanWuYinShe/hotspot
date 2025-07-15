<template>
  <div class="unified-page">
    <div class="unified-container">
      <!-- 页面标题 -->
      <div class="unified-card">
        <div class="unified-card__title">
          <Icon name="mdi:home" />
          欢迎来到 HotSpot
        </div>
        <div class="unified-card__content">
          <p>基于位置的社交平台，发现身边有趣的人和事</p>
        </div>
      </div>

      <!-- 位置状态卡片 -->
      <div class="unified-card">
        <div class="unified-card__title">
          <Icon name="mdi:map-marker" />
          位置状态
        </div>
        <div class="unified-card__content">
          <div class="status-row">
            <div class="status-info">
              <p class="status-text">
                {{ isLocationSharing ? '位置共享已开启' : '位置共享已关闭' }}
              </p>
            </div>
            <NButton :type="isLocationSharing ? 'error' : 'primary'" @click="toggleLocationSharing"
              :loading="isLocationLoading" size="small">
              {{ isLocationSharing ? '关闭共享' : '开启共享' }}
            </NButton>
          </div>
        </div>
      </div>

      <!-- 附近用户 -->
      <div class="unified-card">
        <div class="unified-card__title">
          <Icon name="mdi:account-group" />
          附近用户
          <NButton type="primary" size="small" @click="loadNearbyUsers" :loading="isLoadingUsers"
            style="margin-left: auto;">
            <template #icon>
              <Icon name="mdi:refresh" />
            </template>
            刷新
          </NButton>
        </div>

        <div class="unified-card__content">
          <div v-if="nearbyUsers.length > 0" class="user-grid">
            <div v-for="user in nearbyUsers" :key="user.user_id" class="user-item"
              @click="navigateToUser(user.user_id)">
              <NAvatar :size="40" :src="user.avatar_url || defaultImages.userAvatar" class="user-avatar">
                {{ user.nickname?.charAt(0) || 'U' }}
              </NAvatar>
              <div class="user-info">
                <h4 class="user-name">{{ user.nickname }}</h4>
                <p class="user-distance">{{ user.distance }}m</p>
              </div>
            </div>
          </div>

          <div v-else class="unified-empty">
            <div class="unified-empty__icon">
              <Icon name="mdi:account-group" />
            </div>
            <h4 class="unified-empty__title">暂无附近用户</h4>
            <p class="unified-empty__description">开启位置共享后可以发现附近的用户</p>
          </div>
        </div>
      </div>

      <!-- 附近群组 -->
      <div class="unified-card">
        <div class="unified-card__title">
          <Icon name="mdi:account-multiple" />
          附近群组
          <NButton type="primary" size="small" @click="loadNearbyGroups" :loading="isLoadingGroups"
            style="margin-left: auto;">
            <template #icon>
              <Icon name="mdi:refresh" />
            </template>
            刷新
          </NButton>
        </div>

        <div class="unified-card__content">
          <div v-if="nearbyGroups.length > 0" class="group-grid">
            <div v-for="group in nearbyGroups" :key="group.group_id" class="group-item"
              @click="navigateToGroup(group.group_id)">
              <NAvatar :size="40" :src="group.avatar_url || defaultImages.groupAvatar" class="group-avatar">
                {{ group.name?.charAt(0) || 'G' }}
              </NAvatar>
              <div class="group-info">
                <h4 class="group-name">{{ group.name }}</h4>
                <p class="group-description">{{ group.description }}</p>
                <div class="group-stats">
                  <span class="group-members">{{ group.member_count }} 成员</span>
                  <span class="group-distance">{{ group.distance }}m</span>
                </div>
              </div>
            </div>
          </div>

          <div v-else class="unified-empty">
            <div class="unified-empty__icon">
              <Icon name="mdi:account-multiple" />
            </div>
            <h4 class="unified-empty__title">暂无附近群组</h4>
            <p class="unified-empty__description">开启位置共享后可以发现附近的群组</p>
          </div>
        </div>
      </div>

      <!-- 快速操作 -->
      <div class="action-grid">
        <NButton type="primary" size="large" @click="showCreateGroupDialog = true"
          class="unified-btn unified-btn--primary">
          <template #icon>
            <Icon name="mdi:plus" />
          </template>
          创建群组
        </NButton>

        <NButton type="success" size="large" @click="showNewMessageDialog = true"
          class="unified-btn unified-btn--success">
          <template #icon>
            <Icon name="mdi:message-text" />
          </template>
          发起聊天
        </NButton>
      </div>
    </div>

    <!-- 对话框 -->
    <CreateGroupDialog v-model="showCreateGroupDialog" @success="handleCreateGroupSuccess" />

    <NewMessageDialog v-model="showNewMessageDialog" @success="handleNewMessageSuccess" />
  </div>
</template>

<script setup lang="ts">
// Vue 3 Composition API
import { ref, computed, onMounted } from 'vue'

// Nuxt 3 composables
import { useRouter } from 'vue-router'

// Naive UI 组件
import { NAvatar, NButton, useMessage } from 'naive-ui'

// 项目组件
import CreateGroupDialog from '@/components/group/CreateGroupDialog.vue'
import NewMessageDialog from '@/components/message/NewMessageDialog.vue'

// 服务层
import { useUserService } from '@/services/user.service'
import { useGroupService } from '@/services/group.service'

// Store
import { useAuthStore } from '@/stores/auth'

// 工具函数
import { defaultImages } from '@/utils/assets'

// 设置集成

// 类型定义
import type { UserListItem } from '@/types/models/user'
import type { GroupInfo } from '@/types/models/group'

// 页面元数据
definePageMeta({
  layout: 'default',
  title: '首页'
})

// 路由
const router = useRouter()

// 服务实例
const { searchUsersByLocation } = useUserService()
const { searchGroupsByLocation } = useGroupService()

// 状态管理
const authStore = useAuthStore()
const message = useMessage()

// 响应式状态
const isLocationSharing = ref(false)
const isLocationLoading = ref(false)
const isLoadingUsers = ref(false)
const isLoadingGroups = ref(false)
const nearbyUsers = ref<UserListItem[]>([])
const nearbyGroups = ref<GroupInfo[]>([])
const showCreateGroupDialog = ref(false)
const showNewMessageDialog = ref(false)

// 计算属性
const userNickname = computed(() => {
  return authStore.user?.nickname || '朋友'
})

// 生命周期钩子
onMounted(async () => {
  await loadPageData()

  // 监听设置触发的自动刷新
  window.addEventListener('app-auto-refresh', () => {
    loadPageData()
  })

  // 监听手动刷新
  window.addEventListener('app-manual-refresh', () => {
    loadPageData()
  })
})

// 方法定义
const loadPageData = async () => {
  try {
    await Promise.all([
      loadNearbyUsers(),
      loadNearbyGroups(),
      checkLocationSharing()
    ])
  } catch (error) {
    console.error('加载页面数据失败:', error)
    message.error('加载页面数据失败')
  }
}

const loadNearbyUsers = async () => {
  if (isLoadingUsers.value) return

  isLoadingUsers.value = true
  try {
    const response = await searchUsersByLocation({
      latitude: 0, // TODO: 使用实际位置
      longitude: 0,
      radius: 5000, // 5km
      limit: 20
    })
    nearbyUsers.value = response?.items || []
  } catch (error) {
    console.error('加载附近用户失败:', error)
    message.error('加载附近用户失败')
    nearbyUsers.value = []
  } finally {
    isLoadingUsers.value = false
  }
}

const loadNearbyGroups = async () => {
  if (isLoadingGroups.value) return

  isLoadingGroups.value = true
  try {
    const response = await searchGroupsByLocation({
      latitude: 0, // TODO: 使用实际位置
      longitude: 0,
      radius: 5000, // 5km
      limit: 20
    })
    nearbyGroups.value = response?.items || []
  } catch (error) {
    console.error('加载附近群组失败:', error)
    message.error('加载附近群组失败')
    nearbyGroups.value = []
  } finally {
    isLoadingGroups.value = false
  }
}

const checkLocationSharing = async () => {
  try {
    // TODO: 实现位置共享状态检查
    isLocationSharing.value = false
  } catch (error) {
    console.error('检查位置共享状态失败:', error)
  }
}

const toggleLocationSharing = async () => {
  if (isLocationLoading.value) return

  isLocationLoading.value = true
  try {
    isLocationSharing.value = !isLocationSharing.value
    // TODO: 实现位置共享切换逻辑

    message.success(
      isLocationSharing.value ? '位置共享已开启' : '位置共享已关闭'
    )
  } catch (error) {
    console.error('切换位置共享失败:', error)
    message.error('切换位置共享失败')
    // 回滚状态
    isLocationSharing.value = !isLocationSharing.value
  } finally {
    isLocationLoading.value = false
  }
}

const navigateToUser = (userId: number) => {
  router.push(`/user/${userId}`)
}

const navigateToGroup = (groupId: string) => {
  router.push(`/group/${groupId}`)
}

const handleCreateGroupSuccess = (group: any) => {
  showCreateGroupDialog.value = false
  message.success('群组创建成功')
  navigateToGroup(group.group_id)
}

const handleNewMessageSuccess = async (conversation: any) => {
  showNewMessageDialog.value = false
  message.success('聊天创建成功')
  await router.push(`/chat/${conversation.conversation_id}`)
}
</script>

<style scoped>
/* 状态行样式 */
.status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.status-info {
  flex: 1;
}

.status-text {
  margin: 0;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

/* 用户网格 */
.user-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 1rem;
}

.user-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;

  &:hover {
    background: var(--card-bg-hover);
    border-color: var(--card-border-hover);
    transform: translateY(-1px);
  }
}

.user-avatar {
  flex-shrink: 0;
}

.user-info {
  flex: 1;
  min-width: 0;
}

.user-name {
  margin: 0 0 0.25rem 0;
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-distance {
  margin: 0;
  font-size: 0.8rem;
  color: var(--text-muted);
}

/* 群组网格 */
.group-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}

.group-item {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1rem;
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;

  &:hover {
    background: var(--card-bg-hover);
    border-color: var(--card-border-hover);
    transform: translateY(-1px);
  }
}

.group-avatar {
  flex-shrink: 0;
}

.group-info {
  flex: 1;
  min-width: 0;
}

.group-name {
  margin: 0 0 0.5rem 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-description {
  margin: 0 0 0.75rem 0;
  font-size: 0.85rem;
  color: var(--text-secondary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
}

.group-stats {
  display: flex;
  align-items: center;
  gap: 1rem;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.group-members,
.group-distance {
  margin: 0;
}

/* 快速操作网格 */
.action-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-top: 1rem;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .user-grid {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 0.75rem;
  }

  .group-grid {
    grid-template-columns: 1fr;
    gap: 0.75rem;
  }

  .user-item,
  .group-item {
    padding: 0.75rem;
  }

  .group-item {
    gap: 0.75rem;
  }

  .action-grid {
    grid-template-columns: 1fr;
    gap: 0.75rem;
  }

  .status-row {
    flex-direction: column;
    align-items: stretch;
    gap: 0.75rem;
  }
}

@media (max-width: 480px) {
  .user-grid {
    grid-template-columns: 1fr 1fr;
    gap: 0.5rem;
  }

  .user-item {
    padding: 0.5rem;
    gap: 0.5rem;
  }

  .user-name {
    font-size: 0.8rem;
  }

  .user-distance {
    font-size: 0.7rem;
  }
}
</style>
