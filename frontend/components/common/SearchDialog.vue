<template>
  <NDialog
    v-model="visible"
    title="搜索"
    :show-close="true"
    :close-on-click-modal="true"
    destroy-on-close
    append-to-body
    class="search-dialog responsive-dialog"
  >
    <!-- 搜索输入框 -->
    <div class="search-input-container">
      <NInput
        ref="searchInputRef"
        v-model="searchQuery"
        placeholder="搜索用户、群组、动态或消息..."
        size="large"
        clearable
        @input="handleSearch"
        @keyup.enter="handleEnterSearch"
        class="search-input"
      >
        <template #prefix>
          <Icon name="mdi:magnify" size="16" />
        </template>
      </NInput>
    </div>

    <!-- 搜索类型切换 -->
    <div class="search-tabs">
      <el-radio-group v-model="searchType" @change="handleSearch">
        <el-radio-button value="all">全部</el-radio-button>
        <el-radio-button value="users">用户</el-radio-button>
        <el-radio-button value="groups">群组</el-radio-button>
        <el-radio-button value="moments">动态</el-radio-button>
        <el-radio-button value="messages">消息</el-radio-button>
      </el-radio-group>
    </div>

    <!-- 搜索历史（当没有搜索时显示） -->
    <div v-if="!searchQuery && searchHistory.length > 0" class="search-history">
      <div class="section-header">
        <h4>搜索历史</h4>
        <NButton text type="primary" @click="clearHistory">清除</NButton>
      </div>
      <div class="history-list">
        <div
          v-for="(item, index) in searchHistory"
          :key="index"
          class="history-item"
          @click="searchQuery = item; handleSearch()"
        >
          <Icon name="mdi:clock" size="16" />
          <span>{{ item }}</span>
          <NButton 
            text 
            type="error" 
            @click.stop="removeHistoryItem(index)"
            class="remove-btn"
          >
            <Icon name="mdi:close" size="16" />
          </NButton>
        </div>
      </div>
    </div>

    <!-- 快速搜索建议（当没有搜索时显示） -->
    <div v-if="!searchQuery" class="quick-searches">
      <div class="section-header">
        <h4>快速搜索</h4>
      </div>
      <div class="quick-search-list">
        <UTag
          v-for="suggestion in quickSearches"
          :key="suggestion"
          class="quick-search-tag"
          @click="searchQuery = suggestion; handleSearch()"
        >
          {{ suggestion }}
        </UTag>
      </div>
    </div>

    <!-- 搜索结果 -->
    <div v-if="searchQuery" class="search-results">
      <!-- 加载状态 -->
      <div v-if="loading" class="loading-container">
        <NSkeleton :rows="3" animated />
      </div>

      <!-- 搜索结果为空 -->
      <div v-else-if="!hasResults" class="empty-results">
        <UEmpty description="未找到相关结果" />
      </div>

      <!-- 用户结果 -->
      <div v-if="results.users.length > 0" class="result-section">
        <h4 class="result-title">用户</h4>
        <div class="user-results">
          <div
            v-for="user in results.users"
            :key="user.user_id"
            class="user-item"
            @click="handleUserClick(user)"
          >
            <NAvatar :src="user.avatar_url" :size="40">
              {{ user.nickname?.charAt(0) }}
            </NAvatar>
            <div class="user-info">
              <div class="user-name">{{ user.nickname }}</div>
              <div class="user-desc">{{ user.location_name || '无描述' }}</div>
            </div>
            <NButton type="primary" text>查看</NButton>
          </div>
        </div>
      </div>

      <!-- 群组结果 -->
      <div v-if="results.groups.length > 0" class="result-section">
        <h4 class="result-title">群组</h4>
        <div class="group-results">
          <div
            v-for="group in results.groups"
            :key="group.group_id"
            class="group-item"
            @click="handleGroupClick(group)"
          >
            <NAvatar :src="group.avatar_url" :size="40" shape="square">
              {{ group.name?.charAt(0) }}
            </NAvatar>
            <div class="group-info">
              <div class="group-name">{{ group.name }}</div>
              <div class="group-desc">{{ group.description || '暂无描述' }}</div>
              <div class="group-meta">
                <span>{{ group.member_count || 0 }} 成员</span>
                <span v-if="group.location_name">· {{ group.location_name }}</span>
              </div>
            </div>
            <NButton type="primary" text>加入</NButton>
          </div>
        </div>
      </div>

      <!-- 动态结果 -->
      <div v-if="results.moments.length > 0" class="result-section">
        <h4 class="result-title">动态</h4>
        <div class="moment-results">
          <div
            v-for="moment in results.moments"
            :key="moment.id"
            class="moment-item"
            @click="handleMomentClick(moment)"
          >
            <div class="moment-info">
              <div class="moment-content">{{ moment.description || '无描述' }}</div>
              <div class="moment-meta">
                <span>{{ moment.nickname || '用户' }}</span>
                <span v-if="moment.location_name">· {{ moment.location_name }}</span>
                <span>· {{ formatTime(moment.created_at) }}</span>
              </div>
            </div>
            <NButton type="primary" text>查看</NButton>
          </div>
        </div>
      </div>

      <!-- 消息结果 -->
      <div v-if="results.messages.length > 0" class="result-section">
        <h4 class="result-title">消息</h4>
        <div class="message-results">
          <div
            v-for="message in results.messages"
            :key="message.id"
            class="message-item"
            @click="handleMessageClick(message)"
          >
            <div class="message-info">
              <div class="message-content">{{ message.content }}</div>
              <div class="message-meta">
                <span>{{ message.sender_nickname }}</span>
                <span>· {{ formatTime(message.created_at) }}</span>
              </div>
            </div>
            <NButton type="primary" text>查看</NButton>
          </div>
        </div>
      </div>
    </div>
  </NDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { NDialog, NInput, NButton, NTag, NSkeleton, NEmpty, NAvatar } from 'naive-ui'
import { useMessage } from 'naive-ui'
import { Search, Time as Clock, Close } from '@vicons/ionicons5'
// 导入组件
import { useUserService, useGroupService, useMomentService } from '~/services'
import type { UserListItem } from '~/types/models/user'
import type { GroupInfo } from '~/types/models/group'
import type { CheckInInfo } from '~/types/models/checkin'

// API实例
const message = useMessage()

// 服务实例
const { searchUsersByName } = useUserService()
const { searchGroupsByName } = useGroupService()
const { searchMomentsByTags } = useMomentService()

interface SearchResults {
  users: UserListItem[]
  groups: GroupInfo[]
  moments: CheckInInfo[]
  messages: any[]
}

interface Props {
  modelValue: boolean
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 搜索输入框引用
const searchInputRef = ref<HTMLInputElement | null>(null)

// 对话框显示状态
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// 搜索相关状态
const searchQuery = ref<string>('')
const searchType = ref<'all' | 'users' | 'groups' | 'moments' | 'messages'>('all')
const loading = ref<boolean>(false)
const searchTimer = ref<NodeJS.Timeout>()

// 搜索结果
const results = ref<SearchResults>({
  users: [],
  groups: [],
  moments: [],
  messages: []
})

// 搜索历史和快捷搜索
const searchHistory = ref<string[]>([])
const quickSearches = ref<string[]>([
  '附近的人',
  '热门群组',
  '最新动态',
  '我的好友'
])

// 计算是否有结果
const hasResults = computed(() => {
  return results.value.users.length > 0 ||
         results.value.groups.length > 0 ||
         results.value.moments.length > 0 ||
         results.value.messages.length > 0
})

// 监听对话框开启
watch(visible, (newVal) => {
  if (newVal) {
    loadSearchHistory()
  } else {
    // 关闭时清空搜索
    searchQuery.value = ''
    results.value = { users: [], groups: [], moments: [], messages: [] }
  }
})

// 加载搜索历史
const loadSearchHistory = () => {
  const history = localStorage.getItem('search_history')
  if (history) {
    searchHistory.value = JSON.parse(history)
  }
}

// 保存搜索历史
const saveSearchHistory = (query: string) => {
  if (!query.trim()) return
  
  // 移除重复项并添加到开头
  const newHistory = [query, ...searchHistory.value.filter(item => item !== query)]
  // 只保留最近10条
  searchHistory.value = newHistory.slice(0, 10)
  localStorage.setItem('search_history', JSON.stringify(searchHistory.value))
}

// 清除搜索历史
const clearHistory = () => {
  searchHistory.value = []
  localStorage.removeItem('search_history')
}

// 移除单条历史记录
const removeHistoryItem = (index: number) => {
  searchHistory.value.splice(index, 1)
  localStorage.setItem('search_history', JSON.stringify(searchHistory.value))
}

// 处理搜索输入
const handleSearch = () => {
  if (searchTimer.value) {
    clearTimeout(searchTimer.value)
  }
  
  searchTimer.value = setTimeout(() => {
    if (searchQuery.value.trim()) {
      performSearch()
    } else {
      // 清空结果
      results.value = { users: [], groups: [], moments: [], messages: [] }
    }
  }, 300) // 防抖300ms
}

// 处理回车搜索
const handleEnterSearch = () => {
  if (searchQuery.value.trim()) {
    saveSearchHistory(searchQuery.value)
    performSearch()
  }
}

// 执行搜索
const performSearch = async () => {
  if (!searchQuery.value.trim()) return
  
  loading.value = true
  
  try {
    await realSearch()
  } catch (error) {
    console.error('搜索失败:', error)
    message.error('搜索失败，请重试')
  } finally {
    loading.value = false
  }
}

// 真实搜索API
const realSearch = async () => {
  const query = searchQuery.value.trim()
  
  // 清空之前的结果
  results.value = { users: [], groups: [], moments: [], messages: [] }
  
  // 并发执行搜索
  const searchPromises = []
  
  // 用户搜索
  if (searchType.value === 'all' || searchType.value === 'users') {
    searchPromises.push(
      searchUsersByName({
        keyword: query,
        cursor: 0,
        limit: 10
      }).then((response: any) => {
        results.value.users = response.items || []
      }).catch((error: any) => {
        console.error('用户搜索失败:', error)
      })
    )
  }
  
  // 群组搜索
  if (searchType.value === 'all' || searchType.value === 'groups') {
    searchPromises.push(
      searchGroupsByName({
        keyword: query,
        cursor: 0,
        limit: 10
      }).then((response: any) => {
        results.value.groups = response.items || []
      }).catch((error: any) => {
        console.error('群组搜索失败:', error)
      })
    )
  }
  
  // 动态搜索 - 使用标签搜索
  if (searchType.value === 'all' || searchType.value === 'moments') {
    searchPromises.push(
      searchMomentsByTags({
        tags: [query], // 将查询词作为标签搜索
        latitude: 0,   // 默认位置，可以后续改进
        longitude: 0,
        cursor: 0,
        limit: 10
      }).then((response: any) => {
        results.value.moments = response.items || []
      }).catch((error: any) => {
        console.error('动态搜索失败:', error)
        results.value.moments = []
      })
    )
  }
  
  // 消息搜索（暂时跳过，因为API还未实现）
  // if (searchType.value === 'all' || searchType.value === 'messages') {
  //   searchPromises.push(
  //     messageService.searchMessages({
  //       query,
  //       limit: 10
  //     }).then(response => {
  //       results.value.messages = response.items || []
  //     }).catch(error => {
  //       console.error('消息搜索失败:', error)
  //     })
  //   )
  // }
  
  // 等待所有搜索完成
  await Promise.all(searchPromises)
}

// 格式化时间
const formatTime = (dateString: string | Date) => {
  const now = new Date()
  const date = typeof dateString === 'string' ? new Date(dateString) : dateString
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  return date.toLocaleDateString()
}

// 处理点击事件
const handleUserClick = (user: any) => {
  visible.value = false
  navigateTo(`/user/${user.user_id}`)
}

const handleGroupClick = (group: any) => {
  visible.value = false
  navigateTo(`/group/${group.group_id}/chat`)
}

const handleMomentClick = (moment: any) => {
  visible.value = false
  navigateTo(`/moment/${moment.id}`)
}

const handleMessageClick = (message: any) => {
  visible.value = false
  navigateTo('/messages')
}

onMounted(() => {
  loadSearchHistory()
})
</script>

<style scoped lang="scss">
@use '~/assets/styles/mixins' as *;

.search-dialog {
  .search-input-container {
    margin-bottom: 20px;
    
    .search-input {
      :deep(.el-input__inner) {
        border-radius: 12px;
        padding-left: 45px;
      }
    }
  }
  
  .search-tabs {
    margin-bottom: 20px;
    
    :deep(.el-radio-group) {
      width: 100%;
      display: flex;
      
      .el-radio-button {
        flex: 1;
        
        .el-radio-button__inner {
          width: 100%;
          border-radius: 8px;
          margin: 0 2px;
        }
      }
    }
  }
  
  .search-history, .quick-searches {
    margin-bottom: 20px;
    
    .section-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;
      
      h4 {
        margin: 0;
        font-size: 14px;
        color: var(--color-text-primary);
      }
    }
    
    .history-list {
      .history-item {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        border-radius: 8px;
        cursor: pointer;
        transition: background-color 0.2s;
        
        &:hover {
          background: var(--color-bg-tertiary);
        }
        
        .el-icon {
          color: var(--color-text-tertiary);
          font-size: 14px;
        }
        
        span {
          flex: 1;
          font-size: 14px;
        }
        
        .remove-btn {
          opacity: 0;
          transition: opacity 0.2s;
        }
        
        &:hover .remove-btn {
          opacity: 1;
        }
      }
    }
    
    .quick-search-list {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      
      .quick-search-tag {
        cursor: pointer;
        border-radius: 16px;
        
        &:hover {
          background: var(--color-primary-50);
          border-color: var(--color-primary-300);
        }
      }
    }
  }
  
  .search-results {
    .loading-container {
      padding: 20px 0;
    }
    
    .empty-results {
      padding: 40px 20px;
      text-align: center;
    }
    
    .result-section {
      margin-bottom: 24px;
      
      .result-title {
        margin: 0 0 12px 0;
        font-size: 16px;
        font-weight: 600;
        color: var(--color-text-primary);
      }
      
      .user-item, .group-item, .moment-item, .message-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px;
        border-radius: 12px;
        cursor: pointer;
        transition: background-color 0.2s;
        
        &:hover {
          background: var(--color-bg-tertiary);
        }
        
        .user-info, .group-info, .moment-info, .message-info {
          flex: 1;
          min-width: 0;
          
          .user-name, .group-name, .moment-content, .message-content {
            font-weight: 500;
            color: var(--color-text-primary);
            margin-bottom: 4px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
          
          .user-desc, .group-desc, .group-meta, .moment-meta, .message-meta {
            font-size: 12px;
            color: var(--color-text-secondary);
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
          
          .group-meta, .moment-meta, .message-meta {
            margin-top: 4px;
            
            span {
              margin-right: 4px;
            }
          }
        }
      }
    }
  }
}

/* 响应式对话框样式 - 替代 JavaScript 检测 */
:deep(.responsive-dialog) {
  width: 600px;
  
  @include respond-below(md) {
    width: 100vw !important;
    height: 100vh !important;
    margin: 0 !important;
    max-width: none !important;
    max-height: none !important;
    border-radius: 0 !important;
  }
}

// 移动端适配
@media (max-width: 768px) {
  .search-dialog {
    :deep(.el-dialog) {
      margin: 0;
      height: 100vh;
      max-height: none;
      border-radius: 0;
    }
    
    :deep(.el-dialog__body) {
      padding: 16px;
      max-height: calc(100vh - 120px);
      overflow-y: auto;
    }
    
    .search-tabs {
      :deep(.el-radio-group) {
        .el-radio-button {
          .el-radio-button__inner {
            font-size: 12px;
            padding: 8px 12px;
          }
        }
      }
    }
  }
}
</style>