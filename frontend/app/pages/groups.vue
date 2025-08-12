<template>
  <div class="container mx-auto px-4 py-8">
    <div class="max-w-4xl mx-auto">
      <!-- 页面标题和操作 -->
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-2xl font-bold text-gray-900">我的群组</h1>
        <UButton @click="createGroup">
          <UIcon name="i-heroicons-plus" class="w-4 h-4 mr-2" />
          创建群组
        </UButton>
      </div>

      <!-- 搜索和筛选 -->
      <div class="flex space-x-4 mb-6">
        <UInput
          v-model="searchQuery"
          placeholder="搜索群组..."
          class="flex-1"
        >
          <template #leading>
            <UIcon name="i-heroicons-magnifying-glass" class="w-4 h-4" />
          </template>
        </UInput>
        
        <USelect 
          v-model="activeFilter" 
          :options="filterOptions"
          placeholder="筛选"
          class="w-32"
        />
      </div>

      <!-- 群组列表 -->
      <div v-if="filteredGroups.length > 0" class="space-y-4">
        <UCard 
          v-for="group in filteredGroups" 
          :key="group.id"
          class="hover:shadow-md transition-shadow cursor-pointer"
          @click="enterGroup(group)"
        >
          <div class="flex items-center space-x-4">
            <!-- 群组头像 -->
            <UAvatar 
              :src="group.avatar_url" 
              :alt="group.name"
              size="lg"
            >
              {{ group.name.charAt(0) }}
            </UAvatar>

            <!-- 群组信息 -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center space-x-2 mb-1">
                <h3 class="font-semibold text-gray-900 truncate">
                  {{ group.name }}
                </h3>
                <UBadge 
                  v-if="group.is_owner" 
                  color="primary" 
                  variant="soft"
                  size="xs"
                >
                  群主
                </UBadge>
                <UBadge 
                  v-else-if="group.is_admin" 
                  color="green" 
                  variant="soft"
                  size="xs"
                >
                  管理员
                </UBadge>
              </div>
              
              <p class="text-sm text-gray-600 truncate mb-2">
                {{ group.description || '暂无群组描述' }}
              </p>
              
              <div class="flex items-center space-x-4 text-xs text-gray-500">
                <span class="flex items-center">
                  <UIcon name="i-heroicons-users" class="w-3 h-3 mr-1" />
                  {{ group.member_count }} 人
                </span>
                <span class="flex items-center">
                  <UIcon name="i-heroicons-map-pin" class="w-3 h-3 mr-1" />
                  {{ formatDistance(group.distance) }}
                </span>
                <span v-if="group.last_activity">
                  {{ formatTime(group.last_activity) }}
                </span>
              </div>
            </div>

            <!-- 未读消息提示 -->
            <div v-if="group.unread_count > 0" class="flex flex-col items-center">
              <UBadge color="red" variant="solid">
                {{ group.unread_count > 99 ? '99+' : group.unread_count }}
              </UBadge>
            </div>

            <!-- 操作菜单 -->
            <UDropdown :items="getGroupActions(group)">
              <UButton 
                variant="ghost" 
                size="sm" 
                icon="i-heroicons-ellipsis-vertical"
                @click.stop
              />
            </UDropdown>
          </div>
        </UCard>
      </div>

      <!-- 空状态 -->
      <div v-else class="text-center py-12">
        <UIcon name="i-heroicons-user-group" class="w-16 h-16 text-gray-400 mx-auto mb-4" />
        <h3 class="text-lg font-medium text-gray-900 mb-2">暂无群组</h3>
        <p class="text-gray-600 mb-6">
          {{ searchQuery ? '没有找到匹配的群组' : '加入或创建您的第一个群组开始聊天吧' }}
        </p>
        <div class="space-x-4">
          <UButton @click="createGroup">创建群组</UButton>
          <UButton variant="outline" @click="discoverGroups">发现群组</UButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
// 页面元数据
definePageMeta({
  title: '我的群组',
  requiresAuth: true
})

const toast = useToast()

// 搜索和筛选
const searchQuery = ref('')
const activeFilter = ref('all')

const filterOptions = [
  { label: '全部群组', value: 'all' },
  { label: '我创建的', value: 'owned' },
  { label: '我管理的', value: 'admin' },
  { label: '最近活跃', value: 'active' }
]

// 模拟群组数据
const groups = ref([
  {
    id: 1,
    name: '附近的咖啡爱好者',
    description: '分享最棒的咖啡店和咖啡文化',
    avatar_url: '',
    member_count: 128,
    distance: 0.5,
    last_activity: new Date(Date.now() - 1000 * 60 * 30), // 30分钟前
    unread_count: 5,
    is_owner: true,
    is_admin: false
  },
  {
    id: 2,
    name: '跑步小分队',
    description: '每天早上7点一起晨跑',
    avatar_url: '',
    member_count: 45,
    distance: 1.2,
    last_activity: new Date(Date.now() - 1000 * 60 * 60 * 2), // 2小时前
    unread_count: 0,
    is_owner: false,
    is_admin: true
  },
  {
    id: 3,
    name: '摄影交流群',
    description: '分享摄影作品和技巧',
    avatar_url: '',
    member_count: 89,
    distance: 2.1,
    last_activity: new Date(Date.now() - 1000 * 60 * 60 * 24), // 1天前
    unread_count: 12,
    is_owner: false,
    is_admin: false
  }
])

// 筛选后的群组
const filteredGroups = computed(() => {
  let result = groups.value

  // 按筛选条件过滤
  if (activeFilter.value === 'owned') {
    result = result.filter(group => group.is_owner)
  } else if (activeFilter.value === 'admin') {
    result = result.filter(group => group.is_admin)
  } else if (activeFilter.value === 'active') {
    result = result.filter(group => group.last_activity > new Date(Date.now() - 1000 * 60 * 60 * 24))
  }

  // 按搜索关键词过滤
  if (searchQuery.value) {
    result = result.filter(group => 
      group.name.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      group.description?.toLowerCase().includes(searchQuery.value.toLowerCase())
    )
  }

  return result
})

// 格式化距离
const formatDistance = (distance) => {
  if (distance < 1) {
    return `${Math.round(distance * 1000)}m`
  } else {
    return `${distance.toFixed(1)}km`
  }
}

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

// 获取群组操作菜单
const getGroupActions = (group) => {
  const actions = [
    [{
      label: '进入群聊',
      icon: 'i-heroicons-chat-bubble-left-right',
      click: () => enterGroup(group)
    }]
  ]

  if (group.is_owner) {
    actions.push([{
      label: '管理群组',
      icon: 'i-heroicons-cog-6-tooth',
      click: () => manageGroup(group)
    }])
  }

  actions.push([{
    label: '退出群组',
    icon: 'i-heroicons-arrow-right-on-rectangle',
    click: () => leaveGroup(group)
  }])

  return actions
}

// 创建群组
const createGroup = () => {
  toast.add({
    title: '功能开发中',
    description: '群组创建功能即将上线',
    color: 'info'
  })
}

// 发现群组
const discoverGroups = () => {
  toast.add({
    title: '功能开发中',
    description: '群组发现功能即将上线',
    color: 'info'
  })
}

// 进入群组
const enterGroup = (group) => {
  navigateTo(`/chat/group/${group.id}`)
}

// 管理群组
const manageGroup = (group) => {
  toast.add({
    title: '功能开发中',
    description: '群组管理功能即将上线',
    color: 'info'
  })
}

// 退出群组
const leaveGroup = (group) => {
  toast.add({
    title: '确认退出',
    description: `确定要退出群组"${group.name}"吗？`,
    color: 'warning'
  })
}
</script>
