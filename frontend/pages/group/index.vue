<template>
  <div class="app-page group-page">
    <div class="page-header">
      <h1 class="page-title">群组中心</h1>
      <p class="page-subtitle">发现和加入感兴趣的群组</p>
    </div>

    <div class="app-container">
      <!-- 筛选器卡片 -->
      <div class="content-card">
        <div class="card-header">
          <h3 class="card-title">筛选</h3>
          <div class="filter-actions">
            <NButton 
              :type="activeFilter === 'nearby' ? 'primary' : 'default'" 
              @click="handleFilterChange('nearby')"
              size="small"
            >
              附近群组
            </NButton>
            <NButton 
              :type="activeFilter === 'my' ? 'primary' : 'default'" 
              @click="handleFilterChange('my')"
              size="small"
            >
              我的群组
            </NButton>
            <NButton 
              :type="activeFilter === 'popular' ? 'primary' : 'default'" 
              @click="handleFilterChange('popular')"
              size="small"
            >
              热门群组
            </NButton>
          </div>
        </div>
        
        <div class="search-section">
          <NInput 
            v-model:value="searchKeyword" 
            placeholder="搜索群组..." 
            @keyup.enter="onSearch" 
            clearable 
            @clear="onSearch"
            class="search-input"
          >
            <template #suffix>
              <NButton @click="onSearch" text>
                <template #icon>
                  <NIcon>
                    <Search />
                  </NIcon>
                </template>
              </NButton>
            </template>
          </NInput>
          <NButton type="primary" @click="showCreateGroupDialog = true">
            <template #icon>
              <NIcon>
                <Plus />
              </NIcon>
            </template>
            创建群组
          </NButton>
        </div>
      </div>

      <!-- 群组列表卡片 -->
      <div class="content-card">
        <div class="card-header">
          <h3 class="card-title">
            {{ activeFilter === 'nearby' ? '附近群组' : 
               activeFilter === 'my' ? '我的群组' : '热门群组' }}
          </h3>
          <NButton 
            type="primary" 
            size="small" 
            @click="handleRefresh" 
            :loading="loading"
          >
            <template #icon>
              <NIcon>
                <Refresh />
              </NIcon>
            </template>
            刷新
          </NButton>
        </div>

        <div v-if="groups.length > 0" class="content-grid content-grid--2-cols">
          <div 
            v-for="group in groups" 
            :key="group.group_id" 
            class="group-card"
            @click="handleGroupClick(group)"
          >
            <div class="group-card__avatar">
              <img 
                :src="group.avatar_url || defaultImages.groupAvatar" 
                :alt="group.name"
              >
            </div>
            <div class="group-card__info">
              <h4 class="group-card__name">{{ group.name }}</h4>
              <p class="group-card__description">{{ group.description || '暂无描述' }}</p>
              <div class="group-card__stats">
                <span class="group-card__members">{{ group.member_count }} 成员</span>
                <span v-if="group.distance" class="group-card__distance">{{ group.distance }}m</span>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="!loading" class="empty-state">
          <div class="empty-state__icon">
            <NIcon size="48">
              <People />
            </NIcon>
          </div>
          <h4 class="empty-state__title">暂无群组</h4>
          <p class="empty-state__description">
            {{ activeFilter === 'nearby' ? '附近暂无群组' : 
               activeFilter === 'my' ? '您还没有加入任何群组' : '暂无热门群组' }}
          </p>
        </div>

        <div v-if="loading" class="loading-state">
          <NIcon class="loading-state__spinner" size="24">
            <Refresh />
          </NIcon>
          <span>加载中...</span>
        </div>

        <div v-if="hasMore && !loading" class="load-more-section">
          <NButton @click="loadMoreGroups" style="width: 100%;">
            加载更多群组
          </NButton>
        </div>
      </div>
    </div>

    <CreateGroupDialog v-model="showCreateGroupDialog" @created="handleGroupCreated" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NButton, NInput, NIcon, useMessage } from 'naive-ui'
import { Search, Add as Plus, Refresh, People } from '@vicons/ionicons5'

// 项目组件
import CreateGroupDialog from '~/components/group/CreateGroupDialog.vue'

// 服务
import { useGroupService } from '~/services/group.service'

// 类型
import type { GroupInfo } from '~/types/models/group'

// 工具函数
import { extractErrorMessage } from '~/utils/error-handler'
import { defaultImages } from '@/utils/assets'

// 路由
const router = useRouter()

// 服务实例
const {
  searchGroupsByLocation,
  getOwnedGroups,
  searchGroupsByName
} = useGroupService()

// 页面配置
definePageMeta({
  title: '群组',
  middleware: 'auth'
})

// API实例
const message = useMessage()

const groups = ref<GroupInfo[]>([])
const loading = ref(false)
const hasMore = ref(false)
const cursor = ref<number | string | undefined>(undefined)
const userStore = useUserStore()
const searchKeyword = ref('')
const showCreateGroupDialog = ref(false)
const showNewMessageDialog = ref(false)

// 过滤器状态
const activeFilter = ref('nearby')

// Store引用
const groupStore = useGroupStore()
const messageStore = useMessageStore()

// 推荐群组数据 - 使用真实API数据
const recommendedGroups = computed(() => {
  return groupStore.nearbyGroups?.slice(0, 4) || []
})

// 今日统计数据 - 从API获取
const todayStats = computed(() => ({
  newGroups: groupStore.nearbyGroups?.length || 0,
  activeUsers: userStore.nearbyUsers?.length || 0,
  messages: messageStore.unreadCounts?.total_unread_count || 0,
  nearbyGroups: groupStore.nearbyGroups?.length || 0
}))

// 浮动操作按钮配置
const fabActions = ref([
  {
    id: 'create',
    icon: Plus,
    text: '创建群组',
    color: '#1e88e5'
  },
  {
    id: 'nearby',
    icon: Search,
    text: '附近群组',
    color: '#4caf50'
  },
  {
    id: 'my-groups',
    icon: People,
    text: '我的群组',
    color: '#ff9800'
  }
])

// 处理过滤器变更
const handleFilterChange = (filter: string) => {
  activeFilter.value = filter
  switch (filter) {
    case 'nearby':
      fetchNearbyGroups()
      break
    case 'my':
      fetchUserGroups()
      break
    case 'popular':
      fetchPopularGroups()
      break
  }
}

const fetchNearbyGroups = async () => {
  if (!userStore.currentLocation) {
    console.error('用户位置信息不可用')
    return
  }

  loading.value = true
  try {
    const res = await searchGroupsByLocation({
      latitude: userStore.currentLocation.latitude,
      longitude: userStore.currentLocation.longitude,
      radius: 5000,
      cursor: undefined,
      limit: 10
    })
    groups.value = res.items
    hasMore.value = res.has_more
    cursor.value = res.next_cursor
  } finally {
    loading.value = false
  }
}

const fetchUserGroups = async () => {
  loading.value = true
  try {
    const res = await getOwnedGroups({
      limit: 10
    })
    groups.value = res.items
    hasMore.value = res.has_more
    cursor.value = res.next_cursor
  } finally {
    loading.value = false
  }
}

const fetchPopularGroups = async () => {
  if (!userStore.currentLocation) {
    message.warning('需要位置信息才能获取热门群组')
    return
  }

  loading.value = true
  try {
    // 获取附近群组并按成员数排序作为热门群组
    const res = await searchGroupsByLocation({
      latitude: userStore.currentLocation.latitude,
      longitude: userStore.currentLocation.longitude,
      radius: 10000, // 扩大范围到10km
      cursor: undefined,
      limit: 20
    })
    
    // 按成员数降序排序获取热门群组
    const sortedGroups = res.items.sort((a, b) => b.member_count - a.member_count)
    groups.value = sortedGroups
    hasMore.value = res.has_more
    cursor.value = res.next_cursor
  } catch (error) {
    message.error('获取热门群组失败')
  } finally {
    loading.value = false
  }
}

const loadMoreGroups = async () => {
  if (!cursor.value) return
  if (!userStore.currentLocation) {
    console.error('用户位置信息不可用')
    return
  }

  loading.value = true
  try {
    const res = await searchGroupsByLocation({
      latitude: userStore.currentLocation.latitude,
      longitude: userStore.currentLocation.longitude,
      radius: 5000,
      cursor: typeof cursor.value === 'number' ? cursor.value : undefined,
      limit: 10
    })
    groups.value = [...groups.value, ...res.items]
    hasMore.value = res.has_more
    cursor.value = res.next_cursor
  } finally {
    loading.value = false
  }
}

const onSearch = async () => {
  if (!searchKeyword.value) {
    fetchNearbyGroups()
    return
  }
  loading.value = true
  try {
    // 搜索群组
    const res = await searchGroupsByName({
      keyword: searchKeyword.value,
      cursor: undefined,
      limit: 10
    })
    groups.value = res.items
    hasMore.value = res.has_more
    cursor.value = res.next_cursor
  } catch (e: any) {
    const errorMessage = extractErrorMessage(e, '搜索失败')
    message.error(errorMessage)
  } finally {
    loading.value = false
  }
}

const handleGroupClick = (group: GroupInfo) => {
  // 跳转到群组聊天页
  navigateTo(`/group/${group.group_id}/chat`)
}

const handleCreateGroup = () => {
  // 显示创建群组对话框
  showCreateGroupDialog.value = true
}

// 创建群组成功后的处理
const handleCreateGroupSuccess = async (group: any) => {
  message.success('群组创建成功！')
  await navigateTo(`/group/${group.group_id}/chat`)
}

// 创建消息成功后的处理
const handleNewMessageSuccess = async (conversation: any) => {
  message.success('消息发送成功！')
  await navigateTo(`/chat/${conversation.conversation_id}`)
}

// 下拉刷新处理
const handleRefresh = async () => {
  await fetchNearbyGroups()
}

// 浮动操作按钮处理
const handleFabAction = (actionId: string) => {
  switch (actionId) {
    case 'create':
      handleCreateGroup()
      break
    case 'nearby':
      fetchNearbyGroups()
      break
    case 'my-groups':
      fetchUserGroups()
      break
  }
}

// 页面加载时自动拉取附近群组
onMounted(() => {
  fetchNearbyGroups()
})

const handleGroupCreated = () => {
  showCreateGroupDialog.value = false
  handleFilterChange('my')
}
</script>

<style scoped>
/* 内容网格 */
.content-grid {
  display: grid;
  gap: 1rem;
  margin-bottom: 1rem;
}

.content-grid--2-cols {
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
}

/* 群组卡片 */
.group-card {
  background: var(--bg-color, white);
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 12px;
  padding: 1rem;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--shadow-md, 0 8px 16px rgba(0, 0, 0, 0.1));
    border-color: var(--border-color-dark, #cbd5e1);
  }
  
  &__avatar {
    flex-shrink: 0;
    
    img {
      width: 48px;
      height: 48px;
      border-radius: 8px;
      object-fit: cover;
      border: 2px solid var(--border-color, #e5e7eb);
      transition: all 0.3s ease;
    }
    
    &:hover img {
      border-color: var(--color-primary, #f97316);
    }
  }
  
  &__info {
    flex: 1;
    min-width: 0;
  }
  
  &__name {
    color: var(--text-color-primary, #0f172a);
    font-size: 1rem;
    font-weight: 600;
    margin: 0 0 0.25rem 0;
    text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.1);
  }
  
  &__description {
    color: var(--text-color-secondary, #475569);
    font-size: 0.85rem;
    margin: 0 0 0.5rem 0;
    line-height: 1.4;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
  }
  
  &__stats {
    display: flex;
    align-items: center;
    gap: 1rem;
    font-size: 0.75rem;
    color: var(--text-color-tertiary, #64748b);
  }
  
  &__members,
  &__distance {
    margin: 0;
  }
}

/* 筛选器和搜索 */
.filter-actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.search-section {
  display: flex;
  gap: 1rem;
  align-items: center;
  margin-top: 1rem;
  
  .search-input {
    flex: 1;
  }
}

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  
  &__icon {
    margin-bottom: 1rem;
    color: var(--text-color-tertiary, #64748b);
  }
  
  &__title {
    color: var(--text-color-primary, #0f172a);
    font-size: 1.125rem;
    font-weight: 600;
    margin: 0 0 0.5rem 0;
  }
  
  &__description {
    color: var(--text-color-secondary, #475569);
    font-size: 0.875rem;
    margin: 0;
    line-height: 1.5;
  }
}

/* 加载状态 */
.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 2rem;
  color: var(--text-color-secondary, #475569);
  
  &__spinner {
    animation: spin 1s linear infinite;
  }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* 加载更多 */
.load-more-section {
  margin-top: 1rem;
  text-align: center;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .content-grid--2-cols {
    grid-template-columns: 1fr;
  }
  
  .group-card {
    padding: 0.75rem;
    gap: 0.75rem;
    
    &__avatar img {
      width: 40px;
      height: 40px;
    }
    
    &__name {
      font-size: 0.9rem;
    }
    
    &__description {
      font-size: 0.8rem;
    }
    
    &__stats {
      font-size: 0.7rem;
    }
  }
  
  .search-section {
    flex-direction: column;
    gap: 0.75rem;
    
    .search-input {
      width: 100%;
    }
  }
  
  .filter-actions {
    justify-content: center;
  }
}
</style>