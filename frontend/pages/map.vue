<!--
地图页面 - 全屏高德地图视图
显示用户位置、附近用户、群组等信息
-->
<template>
  <div class="map-page">
    <!-- 地图主容器 -->
    <div class="map-container">
      <AmapView 
        map-height="100vh" 
        :show-controls="true" 
        :show-location-info="true" 
        :is-fullscreen="true"
        :nearby-users="nearbyUsers" 
        :nearby-groups="nearbyGroups" 
        @back="goBack" 
        @location-update="handleLocationUpdate"
        @user-click="handleUserClick" 
        @group-click="handleGroupClick" 
      />
    </div>

    <!-- 顶部工具栏 -->
    <div class="map-toolbar">
      <div class="toolbar-left">
        <NButton 
          text 
          size="large" 
          @click="goBack"
          class="toolbar-btn"
        >
          <template #icon>
            <Icon name="mdi:arrow-left" />
          </template>
        </NButton>
        <div class="toolbar-title">地图探索</div>
      </div>
      <div class="toolbar-right">
        <NButton 
          text 
          size="large" 
          @click="refreshData" 
          :loading="isRefreshing"
          class="toolbar-btn"
        >
          <template #icon>
            <Icon name="mdi:refresh" />
          </template>
        </NButton>
      </div>
    </div>

    <!-- 浮动操作按钮 -->
    <div class="floating-actions">
      <NButton 
        circle 
        type="primary" 
        size="large"
        @click="centerToMyLocation"
        class="fab-btn"
      >
        <template #icon>
          <Icon name="mdi:crosshairs-gps" />
        </template>
      </NButton>
      
      <NButton 
        circle 
        type="info" 
        size="large"
        @click="showNearbyUsers"
        class="fab-btn"
      >
        <template #icon>
          <Icon name="mdi:account-search" />
        </template>
      </NButton>
      
      <NButton 
        circle 
        type="success" 
        size="large"
        @click="showNearbyGroups"
        class="fab-btn"
      >
        <template #icon>
          <Icon name="mdi:account-group" />
        </template>
      </NButton>
    </div>

    <!-- 用户详情弹窗 -->
    <NModal 
      v-model:show="showUserDetail" 
      :title="selectedUser?.nickname || '用户详情'" 
      style="width: 90%; max-width: 400px;"
      preset="card"
    >
      <div v-if="selectedUser" class="user-detail-card">
        <div class="user-header">
          <div class="user-avatar">
            <NAvatar 
              :size="60" 
              :src="selectedUser.avatar_url || defaultImages.userAvatar" 
            />
          </div>
          <div class="user-info">
            <h3 class="user-name">{{ selectedUser.nickname }}</h3>
            <div class="user-meta">
              <div class="meta-item">
                <Icon name="mdi:map-marker" />
                <span>{{ formatDistance(selectedUser) }}</span>
              </div>
              <div class="meta-item" v-if="selectedUser.timestamp">
                <Icon name="mdi:clock" />
                <span>{{ formatTime(selectedUser.timestamp) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="user-actions">
          <NButton 
            type="primary" 
            @click="sendFriendRequest" 
            :loading="isSendingRequest"
            block
          >
            <template #icon>
              <Icon name="mdi:account-plus" />
            </template>
            添加好友
          </NButton>
          
          <NButton 
            type="default" 
            @click="startChat"
            block
            style="margin-top: 8px;"
          >
            <template #icon>
              <Icon name="mdi:chat" />
            </template>
            私聊
          </NButton>
        </div>
      </div>
    </NModal>

    <!-- 群组详情弹窗 -->
    <NModal 
      v-model:show="showGroupDetail" 
      :title="selectedGroup?.name || '群组详情'" 
      style="width: 90%; max-width: 500px;"
      preset="card"
    >
      <div v-if="selectedGroup" class="group-detail-card">
        <div class="group-header">
          <div class="group-avatar">
            <NAvatar 
              :size="60" 
              :src="selectedGroup.avatar_url || defaultImages.groupAvatar"
            >
              {{ selectedGroup.name?.charAt(0) || 'G' }}
            </NAvatar>
          </div>
          <div class="group-info">
            <h3 class="group-name">{{ selectedGroup.name }}</h3>
            <p class="group-description" v-if="selectedGroup.description">
              {{ selectedGroup.description }}
            </p>
            <div class="group-stats">
              <div class="stat-item">
                <Icon name="mdi:account-group" />
                <span>{{ selectedGroup.member_count || 0 }} 成员</span>
              </div>
              <div class="stat-item">
                <Icon name="mdi:map-marker" />
                <span>{{ formatDistance(selectedGroup) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="group-actions">
          <NButton 
            type="primary" 
            @click="joinGroup" 
            :loading="isJoiningGroup" 
            :disabled="selectedGroup.is_member"
            block
          >
            <template #icon>
              <Icon name="mdi:account-plus" />
            </template>
            {{ selectedGroup.is_member ? '已加入' : '加入群组' }}
          </NButton>
          
          <NButton 
            type="default" 
            @click="viewGroupDetail"
            block
            style="margin-top: 8px;"
          >
            <template #icon>
              <Icon name="mdi:eye" />
            </template>
            查看详情
          </NButton>
        </div>
      </div>
    </NModal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NButton, NModal, NAvatar, useMessage } from 'naive-ui'
import { useRouter } from 'vue-router'
import AmapView from '~/components/map/AmapView.vue'
import type { UserLocationPin } from '~/types/models/user'
import type { GroupInfo } from '~/types/models/group'
import { useLocationService, useUserService, useGroupService } from '~/services'
import { defaultImages } from '@/utils/assets'

// 页面元数据
definePageMeta({
  layout: 'default',
  title: '地图探索',
  middleware: 'auth'
})

// API实例和路由
const message = useMessage()
const router = useRouter()

// 服务实例
const { getCurrentLocation } = useLocationService()
const { searchUsersByLocation } = useUserService()
const { searchGroupsByLocation } = useGroupService()

// 响应式数据
const nearbyUsers = ref<UserLocationPin[]>([])
const nearbyGroups = ref<GroupInfo[]>([])
const selectedUser = ref<UserLocationPin | null>(null)
const selectedGroup = ref<GroupInfo | null>(null)

// 状态管理
const isRefreshing = ref<boolean>(false)
const showUserDetail = ref<boolean>(false)
const showGroupDetail = ref<boolean>(false)
const isSendingRequest = ref<boolean>(false)
const isJoiningGroup = ref<boolean>(false)

// 页面生命周期
onMounted(async () => {
  await loadMapData()

  // 监听位置跟踪启用/禁用
  window.addEventListener('location-tracking-enable', (event: any) => {
    console.log('位置跟踪已启用，阈值:', event.detail.threshold)
    // 地图可以根据阈值设置重新加载数据
    refreshData()
  })
  
  window.addEventListener('location-tracking-disable', () => {
    console.log('位置跟踪已禁用')
  })
  
  // 监听自动刷新
  window.addEventListener('app-auto-refresh', () => {
    refreshData()
  })
})

// 方法实现
const goBack = () => {
  router.back()
}

const loadMapData = async () => {
  try {
    await Promise.all([
      loadNearbyUsers(),
      loadNearbyGroups()
    ])
  } catch (error) {
    console.error('加载地图数据失败:', error)
    message.error('加载地图数据失败')
  }
}

const loadNearbyUsers = async () => {
  try {
    const response = await searchUsersByLocation({
      latitude: 39.9042, // 默认北京坐标
      longitude: 116.4074,
      radius: 5000,
      limit: 50
    })
    // 转换用户数据为地图标点格式
    nearbyUsers.value = (response?.items || []).map(user => ({
      user_id: user.user_id,
      nickname: user.nickname,
      avatar_url: undefined, // UserInfo 中没有头像字段，使用默认
      latitude: user.latitude || 39.9042,
      longitude: user.longitude || 116.4074,
      timestamp: new Date().toISOString()
    }))
  } catch (error) {
    console.error('加载附近用户失败:', error)
  }
}

const loadNearbyGroups = async () => {
  try {
    const response = await searchGroupsByLocation({
      latitude: 39.9042, // 默认北京坐标
      longitude: 116.4074,
      radius: 5000,
      limit: 20
    })
    nearbyGroups.value = response?.items || []
  } catch (error) {
    console.error('加载附近群组失败:', error)
  }
}

const refreshData = async () => {
  if (isRefreshing.value) return
  
  isRefreshing.value = true
  try {
    await loadMapData()
    message.success('数据刷新成功')
  } catch (error) {
    message.error('刷新失败')
  } finally {
    isRefreshing.value = false
  }
}

const centerToMyLocation = async () => {
  try {
    const location = await getCurrentLocation()
    if (location) {
      message.success('已定位到当前位置')
      // AmapView 组件会处理位置更新
    }
  } catch (error) {
    message.error('定位失败，请检查位置权限')
  }
}

const showNearbyUsers = () => {
  message.info(`找到 ${nearbyUsers.value.length} 个附近用户`)
}

const showNearbyGroups = () => {
  message.info(`找到 ${nearbyGroups.value.length} 个附近群组`)
}

const handleLocationUpdate = (location: { latitude: number; longitude: number }) => {
  // 位置更新时重新加载数据
  loadMapData()
}

const handleUserClick = (user: UserLocationPin) => {
  selectedUser.value = user
  showUserDetail.value = true
}

const handleGroupClick = (group: GroupInfo) => {
  selectedGroup.value = group
  showGroupDetail.value = true
}

const sendFriendRequest = async () => {
  if (!selectedUser.value) return
  
  isSendingRequest.value = true
  try {
    // TODO: 实现添加好友API调用
    message.success('好友请求已发送')
    showUserDetail.value = false
  } catch (error) {
    message.error('发送好友请求失败')
  } finally {
    isSendingRequest.value = false
  }
}

const startChat = () => {
  if (!selectedUser.value) return
  
  // 跳转到私聊页面
  router.push(`/chat/user/${selectedUser.value.user_id}`)
  showUserDetail.value = false
}

const joinGroup = async () => {
  if (!selectedGroup.value) return
  
  isJoiningGroup.value = true
  try {
    // TODO: 实现加入群组API调用
    message.success('已成功加入群组')
    showGroupDetail.value = false
  } catch (error) {
    message.error('加入群组失败')
  } finally {
    isJoiningGroup.value = false
  }
}

const viewGroupDetail = () => {
  if (!selectedGroup.value) return
  
  router.push(`/group/${selectedGroup.value.group_id}`)
  showGroupDetail.value = false
}

const formatDistance = (item: UserLocationPin | GroupInfo): string => {
  if ('distance' in item && item.distance !== undefined) {
    return item.distance < 1000 
      ? `${Math.round(item.distance)}m` 
      : `${(item.distance / 1000).toFixed(1)}km`
  }
  return '未知距离'
}

const formatTime = (timestamp: string): string => {
  if (!timestamp) return ''
  
  const now = new Date()
  const time = new Date(timestamp)
  const diff = now.getTime() - time.getTime()
  const minutes = Math.floor(diff / 60000)
  
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (minutes < 1440) return `${Math.floor(minutes / 60)}小时前`
  
  return time.toLocaleDateString()
}
</script>

<style scoped lang="scss">
.map-page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--color-background-primary);
  position: relative;
  overflow: hidden;
}

/* 游戏背景 */
.game-background {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: var(--z-base);
  pointer-events: none;
}

.game-bg-pattern {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background:
    radial-gradient(circle at 10% 20%, rgba(255, 107, 53, 0.1) 0%, transparent 50%),
    radial-gradient(circle at 90% 80%, rgba(138, 43, 226, 0.1) 0%, transparent 50%),
    radial-gradient(circle at 50% 50%, rgba(255, 215, 0, 0.05) 0%, transparent 70%);
  background-size: 1200px 1200px, 800px 800px, 600px 600px;
  animation: backgroundMove 30s ease-in-out infinite;
}

.game-bg-stars {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background:
    radial-gradient(1px 1px at 20px 30px, rgba(255, 255, 255, 0.2), transparent),
    radial-gradient(1px 1px at 40px 70px, rgba(255, 107, 53, 0.3), transparent),
    radial-gradient(1px 1px at 90px 40px, rgba(138, 43, 226, 0.3), transparent);
  background-repeat: repeat;
  background-size: 150px 100px;
  animation: starTwinkle 4s ease-in-out infinite alternate;
}

@keyframes backgroundMove {

  0%,
  100% {
    transform: translateX(0) translateY(0) rotate(0deg);
  }

  33% {
    transform: translateX(-10px) translateY(-10px) rotate(1deg);
  }

  66% {
    transform: translateX(10px) translateY(-5px) rotate(-1deg);
  }
}

@keyframes starTwinkle {
  0% {
    opacity: 0.3;
  }

  100% {
    opacity: 0.7;
  }
}

.map-container {
  flex: 1;
  position: relative;
  z-index: var(--z-content);
}

.map-toolbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  background: rgba(13, 13, 39, 0.95);
  border-bottom: 2px solid var(--game-border);
  backdrop-filter: blur(10px);
  z-index: var(--z-elevated);
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.2);
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.toolbar-btn {
  background: none;
  border: none;
  color: var(--game-text-secondary);
  font-size: 1.2rem;
  cursor: pointer;
  transition: all 0.3s ease;

  &:hover {
    color: var(--game-text-primary);
    transform: scale(1.1);
  }
}

.toolbar-title {
  color: var(--game-text-primary);
  font-weight: bold;
  font-size: 1.1rem;
  text-shadow: 0 0 10px rgba(138, 43, 226, 0.5);
}

.toolbar-right {
  display: flex;
  align-items: center;
}

.floating-actions {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  z-index: var(--z-elevated);
}

.fab-btn {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 2px solid var(--color-border-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;

  &:hover {
    transform: scale(1.1);
    border-color: var(--color-primary-500);
    box-shadow: 0 0 20px rgba(59, 130, 246, 0.4);
  }

  &:active {
    transform: scale(0.95);
  }

  &.primary {
    background: linear-gradient(135deg, var(--color-primary-500), var(--color-primary-600));
    border-color: var(--color-primary-500);
    color: white;
    box-shadow: 0 0 20px rgba(59, 130, 246, 0.4);
  }

  &.secondary {
    background: linear-gradient(135deg, var(--color-secondary-500), var(--color-secondary-600));
    border-color: var(--color-secondary-500);
    color: white;
    box-shadow: 0 0 20px rgba(99, 102, 241, 0.4);
  }

  &.success {
    background: linear-gradient(135deg, var(--color-success-500), var(--color-success-600));
    border-color: var(--color-success-500);
    color: white;
    box-shadow: 0 0 20px rgba(34, 197, 94, 0.4);
  }

  &.warning {
    background: linear-gradient(135deg, var(--color-warning-500), var(--color-warning-600));
    border-color: var(--color-warning-500);
    color: white;
    box-shadow: 0 0 20px rgba(245, 158, 11, 0.4);
  }
}

.fab-tooltip {
  position: absolute;
  top: -30px;
  left: 50%;
  transform: translateX(-50%);
  background-color: rgba(0, 0, 0, 0.7);
  color: white;
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 0.75rem;
  white-space: nowrap;
  opacity: 0;
  transition: opacity 0.3s ease-in-out;
  pointer-events: none;
  z-index: var(--z-elevated);
}

.floating-actions .fab-btn:hover .fab-tooltip {
  opacity: 1;
}

/* 现代对话框样式 */
.modern-dialog {
  :deep(.el-dialog) {
    background: var(--color-bg-elevated);
    border: 2px solid var(--color-border-primary);
    border-radius: 1rem;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
    backdrop-filter: blur(10px);
  }

  :deep(.el-dialog__header) {
    background: linear-gradient(135deg, rgba(59, 130, 246, 0.1), rgba(99, 102, 241, 0.1));
    border-bottom: 1px solid var(--color-border-primary);
    padding: 1rem;
  }

  :deep(.el-dialog__title) {
    color: var(--color-text-primary);
    font-weight: 600;
    font-size: 1.1rem;
  }

  :deep(.el-dialog__body) {
    padding: 1.5rem;
  }

  :deep(.el-dialog__close) {
    color: var(--color-text-secondary);
    font-size: 1.2rem;
    transition: all 0.3s ease;

    &:hover {
      color: var(--color-text-primary);
      transform: scale(1.1);
    }
  }
}

/* 用户详情样式 */
.user-detail-card {
  .user-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1.5rem;
    padding-bottom: 1.5rem;
    border-bottom: 1px solid var(--game-border);

    .user-avatar {
      width: 60px;
      height: 60px;
      border-radius: 50%;
      overflow: hidden;
      flex-shrink: 0;

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
    }

    .user-info {
      flex: 1;

      .user-name {
        margin: 0 0 0.5rem 0;
        font-size: 1.25rem;
        color: var(--color-text-primary);
        font-weight: bold;
      }

      .user-meta {
        display: flex;
        gap: 1rem;
        flex-wrap: wrap;

        .meta-item {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          color: var(--color-text-secondary);
          font-size: 0.875rem;

          .el-icon {
            font-size: 1.1rem;
          }
        }
      }
    }
  }

  .user-actions {
    display: flex;
    gap: 1rem;
    justify-content: center;
    flex-wrap: wrap;
  }
}

/* 群组详情样式 */
.group-detail-card {
  .group-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1.5rem;
    padding-bottom: 1.5rem;
    border-bottom: 1px solid var(--game-border);

    .group-avatar {
      width: 60px;
      height: 60px;
      border-radius: 12px;
      overflow: hidden;
      flex-shrink: 0;

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
    }

    .group-info {
      flex: 1;

      .group-name {
        margin: 0 0 0.5rem 0;
        font-size: 1.125rem;
        color: var(--color-text-primary);
        font-weight: bold;
      }

      .group-description {
        margin: 0 0 1rem 0;
        color: var(--color-text-secondary);
        font-size: 0.875rem;
        line-height: 1.4;
      }

      .group-stats {
        display: flex;
        gap: 1rem;
        flex-wrap: wrap;

        .stat-item {
          display: flex;
          align-items: center;
          gap: 0.25rem;

          span {
            color: var(--color-text-secondary);
            font-size: 0.8125rem;
          }
        }
      }
    }
  }

  .group-actions {
    display: flex;
    gap: 1rem;
    justify-content: center;
    flex-wrap: wrap;
  }
}

/* 现代化按钮样式 */
.action-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  border: none;
  border-radius: 0.5rem;
  font-weight: 500;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.3s ease;

  &.primary {
    background: linear-gradient(135deg, var(--color-primary-500), var(--color-primary-600));
    color: white;
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);

    &:hover {
      background: linear-gradient(135deg, var(--color-primary-600), var(--color-primary-700));
      transform: translateY(-2px);
      box-shadow: 0 6px 20px rgba(59, 130, 246, 0.4);
    }

    &:disabled {
      opacity: 0.6;
      cursor: not-allowed;
      transform: none;
    }
  }

  &.secondary {
    background: var(--color-bg-secondary);
    color: var(--color-text-primary);
    border: 1px solid var(--color-border-primary);

    &:hover {
      background: var(--color-bg-tertiary);
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    }
  }
}

/* 现代化浮动操作按钮 */
.floating-actions {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  z-index: var(--z-elevated);
}

.fab-btn {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--color-bg-elevated);
  border: 2px solid var(--color-border-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
  color: var(--color-text-primary);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);

  &:hover {
    transform: scale(1.1);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.2);
  }

  &:active {
    transform: scale(0.95);
  }

  &.primary {
    background: linear-gradient(135deg, var(--color-primary-500), var(--color-primary-600));
    border-color: var(--color-primary-500);
    color: white;
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
  }

  &.secondary {
    background: linear-gradient(135deg, var(--color-secondary-500), var(--color-secondary-600));
    border-color: var(--color-secondary-500);
    color: white;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
  }

  &.success {
    background: linear-gradient(135deg, var(--color-success-500), var(--color-success-600));
    border-color: var(--color-success-500);
    color: white;
    box-shadow: 0 4px 12px rgba(34, 197, 94, 0.3);
  }

  &.warning {
    background: linear-gradient(135deg, var(--color-warning-500), var(--color-warning-600));
    border-color: var(--color-warning-500);
    color: white;
    box-shadow: 0 4px 12px rgba(245, 158, 11, 0.3);
  }
}

.fab-tooltip {
  position: absolute;
  top: -40px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, 0.8);
  color: white;
  padding: 0.5rem 0.75rem;
  border-radius: 0.5rem;
  font-size: 0.75rem;
  white-space: nowrap;
  opacity: 0;
  transition: opacity 0.3s ease;
  pointer-events: none;
  z-index: var(--z-elevated);

  &::after {
    content: '';
    position: absolute;
    top: 100%;
    left: 50%;
    transform: translateX(-50%);
    border: 4px solid transparent;
    border-top-color: rgba(0, 0, 0, 0.8);
  }
}

.fab-btn:hover .fab-tooltip {
  opacity: 1;
}

@keyframes avatarGlow {
  0% {
    box-shadow: 0 0 10px rgba(50, 205, 50, 0.5);
  }

  100% {
    box-shadow: 0 0 20px rgba(50, 205, 50, 0.8);
  }
}

@keyframes epicGlow {
  0% {
    box-shadow: 0 0 10px rgba(138, 43, 226, 0.5);
  }

  100% {
    box-shadow: 0 0 20px rgba(138, 43, 226, 0.8);
  }
}

@keyframes float {

  0%,
  100% {
    transform: translateY(0px);
  }

  50% {
    transform: translateY(-3px);
  }
}

@keyframes pulse {

  0%,
  100% {
    opacity: 0.3;
    transform: scale(1);
  }

  50% {
    opacity: 0.6;
    transform: scale(1.05);
  }
}

/* 响应式设计 */
@media (max-width: 768px) {
  .map-toolbar {
    padding: 0.8rem 1rem;
  }

  .floating-actions {
    bottom: 1rem;
    right: 1rem;
  }

  .fab-btn {
    width: 40px;
    height: 40px;
    font-size: 1.2rem;
  }

  .fab-tooltip {
    top: -25px;
    font-size: 0.6rem;
  }

  .user-detail-card,
  .group-detail-card {

    .user-header,
    .group-header {
      flex-direction: column;
      text-align: center;
      gap: 0.8rem;
      padding-bottom: 1rem;

      .user-avatar,
      .group-avatar {
        width: 50px;
        height: 50px;
      }

      .user-info,
      .group-info {

        .user-name,
        .group-name {
          font-size: 1.1rem;
        }

        .user-meta,
        .group-stats {
          justify-content: center;
        }
      }
    }

    .user-actions,
    .group-actions {
      flex-direction: column;
      gap: 0.5rem;
    }
  }
}
</style>
