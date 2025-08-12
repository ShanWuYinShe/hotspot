<template>
  <div class="container mx-auto px-4 py-8">
    <div class="max-w-2xl mx-auto">
      <!-- 页面标题 -->
      <div class="mb-6">
        <h1 class="text-2xl font-bold text-gray-900">设置</h1>
      </div>

      <!-- 账户设置 -->
      <UCard class="mb-6">
        <template #header>
          <h3 class="font-semibold flex items-center">
            <UIcon name="i-heroicons-user-circle" class="w-5 h-5 mr-2" />
            账户设置
          </h3>
        </template>
        
        <div class="space-y-4">
          <!-- 昵称设置 -->
          <div class="flex items-center justify-between">
            <div>
              <div class="font-medium">昵称</div>
              <div class="text-sm text-gray-600">{{ user?.nickname || '未设置' }}</div>
            </div>
            <UButton variant="outline" size="sm" @click="editNickname">
              编辑
            </UButton>
          </div>

          <!-- 头像设置 -->
          <div class="flex items-center justify-between">
            <div class="flex items-center">
              <div>
                <div class="font-medium">头像</div>
                <div class="text-sm text-gray-600">点击更换头像</div>
              </div>
            </div>
            <UAvatar 
              :src="user?.avatar_url" 
              :alt="user?.nickname"
              size="md"
              class="cursor-pointer"
              @click="changeAvatar"
            >
              {{ user?.nickname?.charAt(0) || 'U' }}
            </UAvatar>
          </div>

          <!-- 绑定状态 -->
          <div v-if="user?.is_temp" class="border-t pt-4">
            <UAlert 
              color="warning" 
              variant="soft"
              title="临时用户"
              description="绑定手机号或邮箱以保存您的数据"
            >
              <template #actions>
                <UButton color="warning" variant="solid" size="xs" @click="bindAccount">
                  立即绑定
                </UButton>
              </template>
            </UAlert>
          </div>
        </div>
      </UCard>

      <!-- 隐私设置 -->
      <UCard class="mb-6">
        <template #header>
          <h3 class="font-semibold flex items-center">
            <UIcon name="i-heroicons-shield-check" class="w-5 h-5 mr-2" />
            隐私设置
          </h3>
        </template>
        
        <div class="space-y-4">
          <!-- 位置共享 -->
          <div class="flex items-center justify-between">
            <div>
              <div class="font-medium">位置共享</div>
              <div class="text-sm text-gray-600">允许其他用户查看我的位置</div>
            </div>
            <UToggle v-model="settings.shareLocation" />
          </div>

          <!-- 在线状态 -->
          <div class="flex items-center justify-between">
            <div>
              <div class="font-medium">在线状态</div>
              <div class="text-sm text-gray-600">显示我的在线状态</div>
            </div>
            <UToggle v-model="settings.showOnlineStatus" />
          </div>

          <!-- 消息通知 -->
          <div class="flex items-center justify-between">
            <div>
              <div class="font-medium">消息通知</div>
              <div class="text-sm text-gray-600">接收新消息推送</div>
            </div>
            <UToggle v-model="settings.messageNotifications" />
          </div>
        </div>
      </UCard>

      <!-- 功能设置 -->
      <UCard class="mb-6">
        <template #header>
          <h3 class="font-semibold flex items-center">
            <UIcon name="i-heroicons-cog-6-tooth" class="w-5 h-5 mr-2" />
            功能设置
          </h3>
        </template>
        
        <div class="space-y-4">
          <!-- 主题模式 -->
          <div class="flex items-center justify-between">
            <div>
              <div class="font-medium">主题模式</div>
              <div class="text-sm text-gray-600">切换明暗主题</div>
            </div>
            <USelect 
              v-model="settings.theme" 
              :options="themeOptions"
              size="sm"
              class="w-32"
            />
          </div>

          <!-- 语言设置 -->
          <div class="flex items-center justify-between">
            <div>
              <div class="font-medium">语言</div>
              <div class="text-sm text-gray-600">界面显示语言</div>
            </div>
            <USelect 
              v-model="settings.language" 
              :options="languageOptions"
              size="sm"
              class="w-32"
            />
          </div>

          <!-- 地图提供商 -->
          <div class="flex items-center justify-between">
            <div>
              <div class="font-medium">地图提供商</div>
              <div class="text-sm text-gray-600">选择地图服务</div>
            </div>
            <USelect 
              v-model="settings.mapProvider" 
              :options="mapProviderOptions"
              size="sm"
              class="w-32"
            />
          </div>
        </div>
      </UCard>

      <!-- 其他设置 -->
      <UCard class="mb-6">
        <template #header>
          <h3 class="font-semibold flex items-center">
            <UIcon name="i-heroicons-information-circle" class="w-5 h-5 mr-2" />
            其他
          </h3>
        </template>
        
        <div class="space-y-1">
          <UButton variant="ghost" block justify="start" @click="clearCache">
            <UIcon name="i-heroicons-trash" class="w-5 h-5 mr-3" />
            清除缓存
          </UButton>
          
          <UButton variant="ghost" block justify="start" @click="showAbout">
            <UIcon name="i-heroicons-information-circle" class="w-5 h-5 mr-3" />
            关于应用
          </UButton>
          
          <UButton variant="ghost" block justify="start" @click="showHelp">
            <UIcon name="i-heroicons-question-mark-circle" class="w-5 h-5 mr-3" />
            帮助与支持
          </UButton>
        </div>
      </UCard>

      <!-- 危险操作 -->
      <UCard>
        <template #header>
          <h3 class="font-semibold text-red-600 flex items-center">
            <UIcon name="i-heroicons-exclamation-triangle" class="w-5 h-5 mr-2" />
            危险操作
          </h3>
        </template>
        
        <div class="space-y-1">
          <UButton 
            variant="ghost" 
            color="red" 
            block 
            justify="start" 
            @click="deleteAccount"
          >
            <UIcon name="i-heroicons-trash" class="w-5 h-5 mr-3" />
            删除账户
          </UButton>
        </div>
      </UCard>
    </div>
  </div>
</template>

<script setup>
// 页面元数据
definePageMeta({
  title: '设置',
  requiresAuth: true
})

const authStore = useAuthStore()
const toast = useToast()

// 当前用户信息
const user = computed(() => authStore.currentUser)

// 设置数据
const settings = reactive({
  shareLocation: true,
  showOnlineStatus: true,
  messageNotifications: true,
  theme: 'system',
  language: 'zh-CN',
  mapProvider: 'gaode'
})

// 选项数据
const themeOptions = [
  { label: '跟随系统', value: 'system' },
  { label: '浅色模式', value: 'light' },
  { label: '深色模式', value: 'dark' }
]

const languageOptions = [
  { label: '简体中文', value: 'zh-CN' },
  { label: 'English', value: 'en-US' }
]

const mapProviderOptions = [
  { label: '高德地图', value: 'gaode' },
  { label: '百度地图', value: 'baidu' },
  { label: '腾讯地图', value: 'tencent' }
]

// 编辑昵称
const editNickname = () => {
  toast.add({
    title: '功能开发中',
    description: '昵称编辑功能即将上线',
    color: 'info'
  })
}

// 更换头像
const changeAvatar = () => {
  toast.add({
    title: '功能开发中',
    description: '头像上传功能即将上线',
    color: 'info'
  })
}

// 绑定账户
const bindAccount = () => {
  toast.add({
    title: '功能开发中',
    description: '账户绑定功能即将上线',
    color: 'info'
  })
}

// 清除缓存
const clearCache = () => {
  toast.add({
    title: '缓存已清除',
    color: 'success'
  })
}

// 显示关于
const showAbout = () => {
  toast.add({
    title: 'HotSpot v1.0.0',
    description: '基于位置的社交平台',
    color: 'info'
  })
}

// 显示帮助
const showHelp = () => {
  toast.add({
    title: '功能开发中',
    description: '帮助文档即将上线',
    color: 'info'
  })
}

// 删除账户
const deleteAccount = () => {
  toast.add({
    title: '危险操作',
    description: '此功能需要额外确认',
    color: 'error'
  })
}

// 监听设置变化并保存
watch(settings, (newSettings) => {
  // 这里可以保存设置到本地存储或服务器
  console.log('Settings updated:', newSettings)
}, { deep: true })
</script>
