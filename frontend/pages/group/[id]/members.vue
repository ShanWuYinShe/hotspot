<template>
  <div class="group-members-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-navigation">
          <BackButton />
        </div>

        <div class="header-info">
          <div class="group-avatar">
            <NAvatar :size="48" :src="groupInfo?.avatar_url">
              {{ groupInfo?.name?.charAt(0) || 'G' }}
            </NAvatar>
          </div>
          <div class="group-details">
            <h2 class="group-name heading-lg text-gradient-fire">{{ groupInfo?.name }}</h2>
            <div class="member-stats">
              <span class="member-count text-responsive-base text-electric">{{ memberList.length }} 名成员</span>
            </div>
          </div>
        </div>

        <div class="header-actions">
          <NInput v-model:value="searchKeyword" placeholder="搜索成员" clearable class="search-input" @input="handleSearch">
            <template #prefix>
              <NIcon size="16">
                <Search />
              </NIcon>
            </template>
          </NInput>
          <NButton v-if="isGroupAdmin" type="primary" @click="showInviteDialog = true" class="invite-btn">
            <template #icon>
              <NIcon size="16">
                <Plus />
              </NIcon>
            </template>
            邀请成员
          </NButton>
        </div>
      </div>
    </div>

    <!-- 主要内容区域 -->
    <div class="members-container">
      <!-- 成员统计和筛选 -->
      <div class="members-stats">
        <div class="stats-cards">
          <div class="stat-card">
            <div class="stat-icon">👑</div>
            <div class="stat-info">
              <div class="stat-number text-responsive-xl text-gold">{{ adminMembers.length }}</div>
              <div class="stat-label text-responsive-sm text-electric">管理员</div>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-icon">👥</div>
            <div class="stat-info">
              <div class="stat-number text-responsive-xl text-gold">{{ regularMembers.length }}</div>
              <div class="stat-label text-responsive-sm text-electric">普通成员</div>
            </div>
          </div>
          <div class="stat-card">
            <div class="stat-icon">🔥</div>
            <div class="stat-info">
              <div class="stat-number text-responsive-xl text-gold">{{ activeMembers.length }}</div>
              <div class="stat-label text-responsive-sm text-electric">活跃成员</div>
            </div>
          </div>
        </div>

        <div class="filter-tabs">
          <NTabs v-model:value="activeFilter" @update:value="handleFilterChange">
            <NTabPane name="all" tab="全部成员">
              <template #tab>
                <span class="tab-label">
                  全部成员
                  <NBadge :value="memberList.length" class="tab-badge" />
                </span>
              </template>
            </NTabPane>
            <NTabPane name="admin" tab="管理员">
              <template #tab>
                <span class="tab-label">
                  管理员
                  <NBadge :value="adminMembers.length" class="tab-badge" />
                </span>
              </template>
            </NTabPane>
            <NTabPane name="member" tab="普通成员">
              <template #tab>
                <span class="tab-label">
                  普通成员
                  <NBadge :value="regularMembers.length" class="tab-badge" />
                </span>
              </template>
            </NTabPane>
          </NTabs>
        </div>
      </div>

      <!-- 管理员区域 -->
      <div v-if="adminMembers.length > 0 && (activeFilter === 'all' || activeFilter === 'admin')"
        class="member-section">
        <div class="section-header">
          <div class="section-title">
            <NIcon size="18">
              <PersonOutline />
            </NIcon>
            <h4>管理员</h4>
            <span class="member-count">({{ adminMembers.length }})</span>
          </div>
        </div>
        <div class="members-grid">
          <div v-for="member in adminMembers" :key="member.user_id" class="member-card admin"
            @click="handleMemberClick(member)">
            <div class="member-avatar">
              <NAvatar :src="member.avatar_url" :size="56">
                {{ member.nickname?.charAt(0) }}
              </NAvatar>
              <div class="role-badge admin">
                <NIcon size="14">
                  <PersonOutline />
                </NIcon>
              </div>
              <div class="online-indicator" v-if="isOnline(member)"></div>
            </div>
            <div class="member-info">
              <div class="member-name">{{ member.nickname }}</div>
              <div class="member-role">{{ getRoleText(member.role) }}</div>
              <div class="member-status">
                <span class="join-date">{{ formatJoinDate(member.join_time) }}</span>
              </div>
            </div>
            <div v-if="canManageMember(member)" class="member-actions">
              <NDropdown @select="(command: string) => handleMemberAction(command, member)" trigger="click">
                <template #trigger>
                  <NButton class="action-btn" circle>
                    <template #icon>
                      <NIcon size="16">
                        <MoreHorizontal />
                      </NIcon>
                    </template>
                  </NButton>
                </template>
                <template #default>
                  <div class="dropdown-option" @click="handleMemberAction(`profile-${member.user_id}`, member)">
                    <NIcon size="16">
                      <PersonOutline />
                    </NIcon>
                    查看资料
                  </div>
                  <div class="dropdown-option" @click="handleMemberAction(`message-${member.user_id}`, member)">
                    <NIcon size="16">
                      <ChatbubbleOutline />
                    </NIcon>
                    发送消息
                  </div>
                  <div v-if="member.role !== 'admin' && isGroupOwner" class="dropdown-option"
                    @click="handleMemberAction(`promote-${member.user_id}`, member)">
                    <NIcon size="16">
                      <ArrowUp />
                    </NIcon>
                    设为管理员
                  </div>
                  <div v-if="member.role === 'admin' && isGroupOwner && member.user_id !== groupInfo?.owner_id"
                    class="dropdown-option" @click="handleMemberAction(`demote-${member.user_id}`, member)">
                    <NIcon size="16">
                      <ArrowDown />
                    </NIcon>
                    取消管理员
                  </div>
                  <div v-if="member.user_id !== groupInfo?.owner_id" class="dropdown-option danger-item"
                    @click="handleMemberAction(`remove-${member.user_id}`, member)">
                    <NIcon size="16">
                      <PersonRemove />
                    </NIcon>
                    移除成员
                  </div>
                </template>
              </NDropdown>
            </div>
          </div>
        </div>
      </div>

      <!-- 普通成员区域 -->
      <div v-if="regularMembers.length > 0 && (activeFilter === 'all' || activeFilter === 'member')"
        class="member-section">
        <div class="section-header">
          <div class="section-title">
            <NIcon size="16">
              <People />
            </NIcon>
            <h4>普通成员</h4>
            <span class="member-count">({{ filteredRegularMembers.length }})</span>
          </div>
        </div>

        <!-- 加载状态 -->
        <div v-if="loading" class="loading-state">
          <NSkeleton :repeat="5" animated />
        </div>

        <!-- 空状态 -->
        <div v-else-if="filteredRegularMembers.length === 0" class="empty-state">
          <div class="empty-icon">👥</div>
          <h3>暂无成员</h3>
          <p v-if="searchKeyword">没有找到匹配的成员</p>
          <p v-else>群组暂无普通成员</p>
        </div>

        <!-- 下拉刷新 -->
        <div v-else class="members-grid">
          <div v-for="member in filteredRegularMembers" :key="member.user_id" class="member-card"
            @click="handleMemberClick(member)">
            <div class="member-avatar">
              <NAvatar :src="member.avatar_url" :size="56">
                {{ member.nickname?.charAt(0) }}
              </NAvatar>
              <div class="online-indicator" v-if="isOnline(member)"></div>
            </div>
            <div class="member-info">
              <div class="member-name">{{ member.nickname }}</div>
              <div class="member-role">{{ getRoleText(member.role) }}</div>
              <div class="member-status">
                <span class="join-date">{{ formatJoinDate(member.join_time) }}</span>
              </div>
            </div>
            <div v-if="canManageMember(member)" class="member-actions">
              <NDropdown @select="(command: string) => handleMemberAction(command, member)" trigger="click">
                <template #trigger>
                  <NButton class="action-btn" circle>
                    <template #icon>
                      <NIcon size="16">
                        <MoreHorizontal />
                      </NIcon>
                    </template>
                  </NButton>
                </template>
                <template #default>
                  <div class="dropdown-option" @click="handleMemberAction(`profile-${member.user_id}`, member)">
                    <NIcon size="16">
                      <PersonOutline />
                    </NIcon>
                    查看资料
                  </div>
                  <div class="dropdown-option" @click="handleMemberAction(`message-${member.user_id}`, member)">
                    <NIcon size="16">
                      <ChatbubbleOutline />
                    </NIcon>
                    发送消息
                  </div>
                  <div v-if="isGroupOwner" class="dropdown-option"
                    @click="handleMemberAction(`promote-${member.user_id}`, member)">
                    <NIcon size="16">
                      <ArrowUp />
                    </NIcon>
                    设为管理员
                  </div>
                  <div v-if="isGroupAdmin" class="dropdown-option"
                    @click="handleMemberAction(`mute-${member.user_id}`, member)"
                    :class="{ disabled: member.is_muted || false }">
                    <NIcon size="16">
                      <VolumeOff />
                    </NIcon>
                    {{ (member.is_muted || false) ? '已禁言' : '禁言' }}
                  </div>
                  <div v-if="isGroupAdmin" class="dropdown-option danger-item"
                    @click="handleMemberAction(`remove-${member.user_id}`, member)">
                    <NIcon size="16">
                      <PersonRemove />
                    </NIcon>
                    移除成员
                  </div>
                </template>
              </NDropdown>
            </div>
          </div>
        </div>

        <!-- 加载更多 -->
        <div v-if="hasMore" class="load-more">
          <NButton @click="loadMore" :loading="loadingMore" block>
            加载更多成员
          </NButton>
        </div>
      </div>
    </div>

    <!-- 邀请成员对话框 -->
    <NModal v-model:show="showInviteDialog" title="邀请成员" preset="dialog" :closable="true">
      <div class="invite-content">
        <div class="invite-methods">
          <NButton type="primary" @click="copyInviteLink" block>
            <template #icon>
              <NIcon>
                <LinkOutline />
              </NIcon>
            </template>
            复制邀请链接
          </NButton>
          <NButton @click="showQRCode = true" block>
            <template #icon>
              <NIcon>
                <QrCodeOutline />
              </NIcon>
            </template>
            生成二维码
          </NButton>
          <NButton @click="shareToWeChat" block>
            <template #icon>
              <NIcon>
                <ShareOutline />
              </NIcon>
            </template>
            分享到微信
          </NButton>
        </div>

        <NDivider>或搜索用户邀请</NDivider>

        <NInput v-model:value="inviteSearchKeyword" placeholder="输入用户名或手机号" @keyup.enter="searchUsers">
          <template #suffix>
            <NButton @click="searchUsers" text>
              <template #icon>
                <NIcon>
                  <Search />
                </NIcon>
              </template>
            </NButton>
          </template>
        </NInput>

        <div v-if="searchedUsers.length > 0" class="searched-users">
          <div v-for="user in searchedUsers" :key="user.id" class="user-item">
            <NAvatar :src="user.avatar" :size="32">
              {{ user.nickname?.charAt(0) }}
            </NAvatar>
            <div class="user-info">
              <div class="user-name">{{ user.nickname }}</div>
              <div class="user-desc">{{ user.phone || user.email }}</div>
            </div>
            <NButton size="small" type="primary" @click="inviteUser(user)" :disabled="user.isInvited">
              {{ user.isInvited ? '已邀请' : '邀请' }}
            </NButton>
          </div>
        </div>
      </div>
    </NModal>

    <!-- 二维码对话框 -->
    <NModal v-model:show="showQRCode" title="群组二维码" preset="dialog" :closable="true">
      <div class="qr-code-content">
        <div class="qr-code-placeholder">
          <NIcon size="80">
            <QrCodeOutline />
          </NIcon>
          <p>扫码加入群组</p>
        </div>
        <div class="qr-tips">
          <p>请将二维码分享给好友</p>
          <p>好友扫码即可加入群组</p>
        </div>
      </div>
    </NModal>

    <!-- 浮动操作按钮 -->
    <FloatingActionButton v-if="isGroupAdmin" :actions="fabActions" @action-click="handleFabAction"
      class="compact-only" />

    <!-- 创建群组对话框 -->
    <CreateGroupDialog v-model="showCreateGroupDialog" @success="handleCreateGroupSuccess" />

    <!-- 新消息对话框 -->
    <NewMessageDialog v-model="showNewMessageDialog" @success="handleNewMessageSuccess" />
  </div>
</template>

<script setup lang="ts">
// 禁用 SSR 以避免 Naive UI 组件的 document.head 错误
definePageMeta({
  ssr: false
})

import { ref, computed, onMounted } from 'vue'
import {
  NButton, NIcon, NInput, NAvatar, NDropdown, NModal,
  NSkeleton, NTabs, NTabPane, NBadge, NDivider, useMessage, useDialog
} from 'naive-ui'
import {
  Search, Add as Plus, PersonOutline, ChatbubbleOutline,
  ArrowUp, ArrowDown, PersonRemove, People,
  EllipsisHorizontal as MoreHorizontal, VolumeOff, LinkOutline, QrCodeOutline, ShareOutline
} from '@vicons/ionicons5'

// 项目组件导入
import BackButton from '~/components/common/BackButton.vue'
import FloatingActionButton from '~/components/common/FloatingActionButton.vue'
import CreateGroupDialog from '~/components/group/CreateGroupDialog.vue'
import NewMessageDialog from '~/components/message/NewMessageDialog.vue'

// Stores
import { useAuthStore } from '~/stores/auth'

// Services
import { useGroupService } from '~/services'

// 服务实例
const { searchGroupsById, getGroupMembers } = useGroupService()

// Utils
import { extractErrorMessage } from '~/utils/error-handler'

// Types
import type { GroupInfo, GroupMember } from '~/types/models/group'

// Composables
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

// 消息和对话框API
const message = useMessage()
const dialog = useDialog()

// 技能栏功能状态
const batchMode = ref(false)

// 技能栏功能函数
// 聚焦搜索框
const focusSearch = () => {
  const searchInput = document.querySelector('.search-input input') as HTMLInputElement
  if (searchInput) {
    searchInput.focus()
  }
}

// 切换批量管理模式
const toggleBatchMode = () => {
  batchMode.value = !batchMode.value
  message.info(batchMode.value ? '进入批量管理模式' : '退出批量管理模式')
}

// 筛选在线成员
const filterOnlineMembers = () => {
  // TODO: 实现筛选在线成员功能
  message.info('筛选在线成员功能开发中...')
}

// 筛选管理员成员
const filterAdminMembers = () => {
  // TODO: 实现筛选管理员成员功能
  message.info('筛选管理员成员功能开发中...')
}

// 响应式数据
const groupId = computed(() => route.params.id as string)
const currentUserId = computed(() => authStore.user?.user_id)
const currentUserRole = computed(() => groupInfo.value?.user_role || 'member')

const groupInfo = ref<GroupInfo | null>(null)
const memberList = ref<GroupMember[]>([])
const searchKeyword = ref('')
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(true)
const showInviteDialog = ref(false)
const showQRCode = ref(false)
const showCreateGroupDialog = ref(false)
const showNewMessageDialog = ref(false)
const inviteSearchKeyword = ref('')
const searchedUsers = ref<any[]>([])

// 权限检查
const isGroupAdmin = computed(() => {
  return currentUserRole.value === 'admin' || currentUserRole.value === 'owner'
})

const isGroupOwner = computed(() => {
  return currentUserRole.value === 'owner'
})

// 管理员成员
const adminMembers = computed(() => {
  return memberList.value.filter(member =>
    member.role === 'owner' || member.role === 'admin'
  )
})

// 普通成员
const regularMembers = computed(() => {
  return memberList.value.filter(member => member.role === 'member')
})

// 活跃成员 (最近7天内活跃)
const activeMembers = computed(() => {
  const sevenDaysAgo = new Date()
  sevenDaysAgo.setDate(sevenDaysAgo.getDate() - 7)

  return memberList.value.filter(member => {
    if (!member.last_active_at) return false
    const lastActive = new Date(member.last_active_at)
    return lastActive >= sevenDaysAgo
  })
})

// 筛选相关
const activeFilter = ref('all')

// 普通成员（过滤后）
const filteredRegularMembers = computed(() => {
  let members = memberList.value.filter(member => member.role === 'member')

  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    members = members.filter(member =>
      member.nickname?.toLowerCase().includes(keyword)
    )
  }

  return members
})

// 浮动操作按钮
const fabActions = computed(() => [
  {
    id: 'invite',
    label: '邀请成员',
    icon: Plus,
    color: '#409EFF'
  },
  {
    id: 'search',
    label: '搜索成员',
    icon: Search,
    color: '#67C23A'
  }
])

// 生命周期
onMounted(async () => {
  await loadGroupInfo()
  await loadMembers()
})

// 加载群组信息
const loadGroupInfo = async () => {
  try {
    const response = await searchGroupsById({ group_id: groupId.value })
    groupInfo.value = response
  } catch (error) {
    console.error('加载群组信息失败:', error)
    message.error('加载群组信息失败')
  }
}

// 加载成员列表
const loadMembers = async (cursor?: number) => {
  try {
    loading.value = true
    const response = await getGroupMembers({
      group_id: groupId.value,
      cursor,
      limit: 50
    })

    if (cursor) {
      memberList.value = [...memberList.value, ...response.items]
    } else {
      memberList.value = response.items
    }

    hasMore.value = response.has_more
  } catch (error) {
    console.error('加载成员失败:', error)
    message.error('加载成员失败')
  } finally {
    loading.value = false
  }
}

// 加载更多成员
const loadMore = async () => {
  if (!hasMore.value || loadingMore.value) return

  loadingMore.value = true
  const lastMember = memberList.value[memberList.value.length - 1]
  await loadMembers(lastMember?.user_id)
  loadingMore.value = false
}

// 检查是否可以管理成员
const canManageMember = (member: GroupMember) => {
  if (member.user_id === currentUserId.value) return false
  if (!isGroupAdmin.value) return false
  if (currentUserRole.value === 'owner') return true
  if (currentUserRole.value === 'admin' && member.role === 'member') return true
  return false
}

// 检查成员是否在线
const isOnline = (member: GroupMember) => {
  if (!member.last_active_at) return false
  const lastActive = new Date(member.last_active_at).getTime()
  const now = new Date().getTime()
  return (now - lastActive) < 5 * 60 * 1000 // 5分钟内活跃视为在线
}

// 格式化角色文本
const getRoleText = (role: string) => {
  switch (role) {
    case 'owner': return '群主'
    case 'admin': return '管理员'
    case 'member': return '成员'
    default: return '成员'
  }
}

// 格式化加入时间
const formatJoinTime = (joinTime: string) => {
  const date = new Date(joinTime)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (days === 0) {
    return '今天'
  } else if (days === 1) {
    return '昨天'
  } else if (days < 30) {
    return `${days}天前`
  } else {
    return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
  }
}

// 格式化加入日期 (用于新模板)
const formatJoinDate = (joinTime: string) => {
  if (!joinTime) return '最近加入'
  const date = new Date(joinTime)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (days === 0) {
    return '今天加入'
  } else if (days === 1) {
    return '昨天加入'
  } else if (days < 30) {
    return `${days}天前加入`
  } else {
    return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })
  }
}

// 处理筛选变化
const handleFilterChange = (filterName: string) => {
  activeFilter.value = filterName
}

// 搜索处理
const handleSearch = () => {
  // 搜索逻辑已在计算属性中处理
}

// 返回上一页
const goBack = () => {
  router.back()
}

// 格式化最后活跃时间
const formatLastActive = (lastActive: string) => {
  if (!lastActive) return '从未活跃'

  const date = new Date(lastActive)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (minutes < 1) {
    return '刚刚活跃'
  } else if (minutes < 60) {
    return `${minutes}分钟前活跃`
  } else if (hours < 24) {
    return `${hours}小时前活跃`
  } else if (days < 7) {
    return `${days}天前活跃`
  } else {
    return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
  }
}

// 处理成员点击
const handleMemberClick = (member: GroupMember) => {
  navigateTo(`/user/${member.user_id}`)
}

// 处理成员操作
const handleMemberAction = async (command: string, member: GroupMember) => {
  const [action, userId] = command.split('-')

  switch (action) {
    case 'profile':
      navigateTo(`/user/${userId}`)
      break
    case 'message':
      navigateTo(`/chat/${userId}`)
      break
    case 'promote':
      await handlePromoteMember(member)
      break
    case 'demote':
      await handleDemoteMember(member)
      break
    case 'mute':
      await handleMuteMember(member)
      break
    case 'remove':
      await handleKickMember(member)
      break
  }
}

// 提升为管理员
const handlePromoteMember = async (member: GroupMember) => {
  try {
    await new Promise<void>((resolve, reject) => {
      dialog.warning({
        title: '提升管理员',
        content: `确定要将 ${member.nickname} 设为管理员吗？`,
        positiveText: '确定',
        negativeText: '取消',
        onPositiveClick: () => {
          resolve()
        },
        onNegativeClick: () => {
          reject()
        }
      })
    })

    // TODO: 调用API提升权限
    member.role = 'admin'
    message.success('已设为管理员')
  } catch (error) {
    // 用户取消
  }
}

// 取消管理员
const handleDemoteMember = async (member: GroupMember) => {
  try {
    await new Promise<void>((resolve, reject) => {
      dialog.warning({
        title: '取消管理员',
        content: `确定要取消 ${member.nickname} 的管理员权限吗？`,
        positiveText: '确定',
        negativeText: '取消',
        onPositiveClick: () => {
          resolve()
        },
        onNegativeClick: () => {
          reject()
        }
      })
    })

    // TODO: 调用API降级权限
    member.role = 'member'
    message.success('已取消管理员权限')
  } catch (error) {
    // 用户取消
  }
}

// 禁言成员
const handleMuteMember = async (member: GroupMember) => {
  try {
    await new Promise<void>((resolve, reject) => {
      dialog.warning({
        title: '禁言成员',
        content: `确定要禁言 ${member.nickname} 吗？禁言后该成员将无法在群内发送消息。`,
        positiveText: '确定',
        negativeText: '取消',
        onPositiveClick: () => {
          resolve()
        },
        onNegativeClick: () => {
          reject()
        }
      })
    })

    // TODO: 调用API禁言
    if (member.is_muted !== undefined) {
      member.is_muted = true
    }
    message.success('已禁言该成员')
  } catch (error) {
    // 用户取消
  }
}

// 移除成员
const handleKickMember = async (member: GroupMember) => {
  try {
    await new Promise<void>((resolve, reject) => {
      dialog.error({
        title: '移除成员',
        content: `确定要将 ${member.nickname} 移出群组吗？该操作不可撤销。`,
        positiveText: '确定移除',
        negativeText: '取消',
        onPositiveClick: () => {
          resolve()
        },
        onNegativeClick: () => {
          reject()
        }
      })
    })

    // TODO: 调用API移除成员
    const index = memberList.value.findIndex(m => m.user_id === member.user_id)
    if (index > -1) {
      memberList.value.splice(index, 1)
    }
    message.success('已移除该成员')
  } catch (error) {
    // 用户取消
  }
}

// 搜索用户
const searchUsers = async () => {
  if (!inviteSearchKeyword.value.trim()) return

  try {
    // TODO: 调用搜索用户接口
    searchedUsers.value = [
      {
        id: '1',
        nickname: '张三',
        phone: '138****8888',
        avatar: '',
        isInvited: false
      }
    ]
  } catch (error) {
    message.error('搜索用户失败')
  }
}

// 邀请用户
const inviteUser = async (user: any) => {
  try {
    // TODO: 调用邀请API
    user.isInvited = true
    message.success(`已向 ${user.nickname} 发送邀请`)
  } catch (error) {
    message.error('邀请失败')
  }
}

// 复制邀请链接
const copyInviteLink = async () => {
  try {
    const link = `${window.location.origin}/group/${groupId.value}/join`
    await navigator.clipboard.writeText(link)
    message.success('邀请链接已复制到剪贴板')
    showInviteDialog.value = false
  } catch (error) {
    message.error('复制失败')
  }
}

// 分享到微信
const shareToWeChat = () => {
  message.info('微信分享功能开发中...')
}

// 下拉刷新
const handleRefresh = async () => {
  await loadMembers()
}

// 浮动操作按钮处理
const handleFabAction = (action: any) => {
  switch (action.id) {
    case 'invite':
      showInviteDialog.value = true
      break
    case 'search':
      // 聚焦搜索框
      nextTick(() => {
        const searchInput = document.querySelector('.search-input input') as HTMLInputElement
        searchInput?.focus()
      })
      break
  }
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

// 设置页面标题
useHead({
  title: computed(() => `${groupInfo.value?.name || '群组'} - 成员列表 - HotSpot`)
})
</script>

<style lang="scss" scoped>
.group-members-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
}

.page-header {
  background: var(--bg-color, white);
  border-bottom: 1px solid var(--border-color, #e2e8f0);
  padding: 24px;
  box-shadow: var(--shadow-sm, 0 2px 8px rgba(0, 0, 0, 0.05));

  .header-content {
    max-width: 1200px;
    margin: 0 auto;
    display: flex;
    align-items: center;
    gap: 24px;

    .header-navigation {
      .back-btn {
        background: var(--fill-color-light, #f1f5f9);
        border: 1px solid var(--border-color, #e2e8f0);
        color: var(--text-color-secondary, #64748b);
        border-radius: 12px;
        padding: 8px 12px;
        transition: all 0.3s ease;

        &:hover {
          background: var(--border-color, #e2e8f0);
          border-color: var(--border-color-dark, #cbd5e1);
          transform: translateX(-2px);
        }
      }
    }

    .header-info {
      display: flex;
      align-items: center;
      gap: 16px;
      flex: 1;

      .group-details {
        .group-name {
          margin: 0 0 8px 0;
          font-size: 24px;
          font-weight: 700;
          color: var(--text-color-primary, #1e293b);
        }

        .member-stats {
          display: flex;
          gap: 16px;

          .member-count {
            font-size: 14px;
            color: var(--text-color-secondary, #64748b);
            font-weight: 500;
          }
        }
      }
    }

    .header-actions {
      display: flex;
      align-items: center;
      gap: 16px;

      .search-input {
        width: 280px;
      }

      .invite-btn {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        border: none;
        border-radius: 12px;
        padding: 12px 20px;
        font-weight: 600;

        &:hover {
          transform: translateY(-1px);
          box-shadow: 0 6px 20px rgba(102, 126, 234, 0.3);
        }
      }
    }
  }
}

.members-container {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.members-stats {
  margin-bottom: 32px;

  .stats-cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 20px;
    margin-bottom: 24px;

    .stat-card {
      background: var(--bg-color, white);
      border-radius: 16px;
      padding: 24px;
      box-shadow: var(--shadow-sm, 0 4px 12px rgba(0, 0, 0, 0.05));
      border: 1px solid var(--border-color-light, #f1f5f9);
      transition: all 0.3s ease;

      &:hover {
        transform: translateY(-2px);
        box-shadow: var(--shadow-md, 0 8px 24px rgba(0, 0, 0, 0.1));
      }

      display: flex;
      align-items: center;
      gap: 16px;

      .stat-icon {
        font-size: 32px;
        padding: 12px;
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        min-width: 56px;
        height: 56px;
      }

      .stat-info {
        .stat-number {
          font-size: 28px;
          font-weight: 700;
          color: var(--text-color-primary, #1e293b);
          margin-bottom: 4px;
        }

        .stat-label {
          font-size: 14px;
          color: var(--text-color-secondary, #64748b);
          font-weight: 500;
        }
      }
    }
  }

  .filter-tabs {
    background: var(--bg-color, white);
    border-radius: 16px;
    padding: 4px;
    box-shadow: var(--shadow-sm, 0 4px 12px rgba(0, 0, 0, 0.05));
  }
}

.member-section {
  background: var(--bg-color, white);
  border-radius: 16px;
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: var(--shadow-sm, 0 4px 12px rgba(0, 0, 0, 0.05));
  border: 1px solid var(--border-color-light, #f1f5f9);

  .section-header {
    margin-bottom: 24px;

    .section-title {
      display: flex;
      align-items: center;
      gap: 12px;

      h4 {
        margin: 0;
        font-size: 18px;
        font-weight: 700;
        color: var(--text-color-primary, #1e293b);
      }

      .member-count {
        background: var(--fill-color-light, #f1f5f9);
        color: var(--text-color-secondary, #64748b);
        padding: 4px 12px;
        border-radius: 20px;
        font-size: 12px;
        font-weight: 600;
      }
    }
  }

  .members-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 20px;

    .member-card {
      background: var(--fill-color-light, #f8fafc);
      border: 1px solid var(--border-color, #e2e8f0);
      border-radius: 16px;
      padding: 20px;
      transition: all 0.3s ease;
      cursor: pointer;

      &:hover {
        transform: translateY(-2px);
        box-shadow: var(--shadow-md, 0 8px 24px rgba(0, 0, 0, 0.1));
        border-color: var(--border-color-dark, #cbd5e1);
      }

      &.admin {
        background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
        border-color: #f59e0b;

        &:hover {
          box-shadow: 0 8px 24px rgba(245, 158, 11, 0.2);
        }
      }

      display: flex;
      align-items: center;
      gap: 16px;

      .member-avatar {
        position: relative;
        flex-shrink: 0;

        .role-badge {
          position: absolute;
          bottom: -2px;
          right: -2px;
          width: 20px;
          height: 20px;
          background: #f59e0b;
          border: 2px solid white;
          border-radius: 50%;
          display: flex;
          align-items: center;
          justify-content: center;

          &.admin {
            background: #f59e0b;
          }
        }

        .online-indicator {
          position: absolute;
          top: -2px;
          right: -2px;
          width: 16px;
          height: 16px;
          background: #10b981;
          border: 3px solid white;
          border-radius: 50%;
          animation: pulse 2s infinite;
        }
      }

      .member-info {
        flex: 1;
        min-width: 0;

        .member-name {
          font-size: 16px;
          font-weight: 600;
          color: var(--text-color-primary, #1e293b);
          margin-bottom: 4px;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }

        .member-role {
          font-size: 13px;
          color: #f59e0b;
          font-weight: 600;
          margin-bottom: 6px;
        }

        .member-status {
          .join-date {
            font-size: 12px;
            color: var(--text-color-secondary, #64748b);
          }
        }
      }

      .member-actions {
        .action-btn {
          background: rgba(255, 255, 255, 0.8);
          border: 1px solid var(--border-color, #e2e8f0);
          color: var(--text-color-secondary, #64748b);
          width: 32px;
          height: 32px;
          border-radius: 8px;

          &:hover {
            background: white;
            border-color: var(--border-color-dark, #cbd5e1);
            color: var(--text-color-primary, #1e293b);
          }
        }
      }
    }
  }
}

.loading-state,
.empty-state {
  padding: 40px 20px;
  text-align: center;

  .empty-icon {
    font-size: 48px;
    margin-bottom: 16px;
  }

  h3 {
    margin: 0 0 8px 0;
    color: var(--text-color-secondary, #666);
  }

  p {
    margin: 0;
    color: var(--text-color-tertiary, #999);
    font-size: 14px;
  }
}

.load-more {
  margin-top: 20px;
}

.invite-content {
  .invite-methods {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-bottom: 20px;
  }

  .searched-users {
    margin-top: 16px;
    max-height: 300px;
    overflow-y: auto;

    .user-item {
      display: flex;
      align-items: center;
      padding: 12px 0;
      border-bottom: 1px solid var(--border-color-light, #f0f0f0);

      &:last-child {
        border-bottom: none;
      }

      .user-info {
        flex: 1;
        margin-left: 12px;

        .user-name {
          font-size: 14px;
          font-weight: 600;
          margin-bottom: 4px;
        }

        .user-desc {
          font-size: 12px;
          color: var(--text-color-secondary, #666);
        }
      }
    }
  }
}

.qr-code-content {
  text-align: center;

  .qr-code-placeholder {
    padding: 40px;
    border: 2px dashed var(--border-color, #ddd);
    border-radius: 8px;
    margin-bottom: 20px;

    p {
      margin: 16px 0 0 0;
      color: var(--text-color-secondary, #666);
    }
  }

  .qr-tips {
    p {
      margin: 8px 0;
      font-size: 12px;
      color: var(--text-color-tertiary, #999);
    }
  }
}

// 动画
@keyframes pulse {
  0% {
    transform: scale(1);
    opacity: 1;
  }

  50% {
    transform: scale(1.1);
    opacity: 0.7;
  }

  100% {
    transform: scale(1);
    opacity: 1;
  }
}

// 移动端适配
@media (max-width: 768px) {
  .page-header {
    padding: 16px;

    .header-content {
      flex-direction: column;
      gap: 16px;
      align-items: stretch;

      .header-info {
        .group-details {
          .group-name {
            font-size: 20px;
          }
        }
      }

      .header-actions {
        .search-input {
          width: 100%;
        }
      }
    }
  }

  .members-container {
    padding: 16px;
  }

  .members-stats {
    .stats-cards {
      grid-template-columns: 1fr;
      gap: 12px;

      .stat-card {
        padding: 16px;
      }
    }
  }

  .member-section {
    .members-grid {
      grid-template-columns: 1fr;
      gap: 12px;

      .member-card {
        padding: 16px;
      }
    }
  }
}

// 危险操作样式
:deep(.danger-item) {
  color: var(--color-error, #f56565);

  &:hover {
    background-color: var(--color-error-50, #fef2f2);
    color: var(--color-error, #f56565);
  }
}
</style>
