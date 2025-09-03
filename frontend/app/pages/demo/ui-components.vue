<template>
  <div class="max-w-4xl mx-auto p-6 space-y-8">
    <div class="text-center">
      <h1 class="text-3xl font-bold text-neutral-900 dark:text-neutral-100 mb-2">
        Nuxt UI v3 高级组件演示
      </h1>
      <p class="text-neutral-600 dark:text-neutral-400">
        UDrawer、UContextMenu、USlideover 组件的使用示例
      </p>
    </div>

    <!-- UDrawer 演示 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-bars-3" class="w-6 h-6 text-primary-500" />
          <h2 class="text-xl font-semibold">UDrawer - 抽屉组件</h2>
        </div>
      </template>

      <div class="space-y-4">
        <p class="text-neutral-600 dark:text-neutral-400">
          抽屉组件适用于移动端导航、设置面板等场景。点击下面的按钮体验不同方向的抽屉。
        </p>
        
        <div class="flex flex-wrap gap-3">
          <UButton @click="leftDrawerOpen = true" icon="i-heroicons-arrow-right">
            左侧抽屉
          </UButton>
          <UButton @click="rightDrawerOpen = true" icon="i-heroicons-arrow-left">
            右侧抽屉
          </UButton>
          <UButton @click="topDrawerOpen = true" icon="i-heroicons-arrow-down">
            顶部抽屉
          </UButton>
          <UButton @click="bottomDrawerOpen = true" icon="i-heroicons-arrow-up">
            底部抽屉
          </UButton>
        </div>

        <!-- 移动端导航抽屉演示 -->
        <div class="mt-4 p-4 bg-neutral-50 dark:bg-neutral-800 rounded-lg">
          <h4 class="font-medium mb-2">移动端导航抽屉</h4>
          <p class="text-sm text-neutral-600 dark:text-neutral-400 mb-3">
            在移动端，点击左上角的菜单按钮即可打开导航抽屉
          </p>
          <UButton @click="mobileDrawerOpen = true" variant="outline" size="sm">
            打开移动端导航
          </UButton>
        </div>
      </div>
    </UCard>

    <!-- UContextMenu 演示 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-cursor-arrow-rays" class="w-6 h-6 text-success-500" />
          <h2 class="text-xl font-semibold">UContextMenu - 右键菜单</h2>
        </div>
      </template>

      <div class="space-y-4">
        <p class="text-neutral-600 dark:text-neutral-400">
          右键菜单提供快速操作入口。在下面的卡片上右键点击体验不同类型的菜单。
        </p>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <!-- 用户卡片 -->
          <ContextMenuWrapper 
            type="user" 
            :data="demoUser" 
            @action="handleContextAction"
          >
            <UCard class="cursor-pointer hover:shadow-md transition-shadow">
              <div class="flex items-center gap-3">
                <UAvatar alt="演示用户" size="md">D</UAvatar>
                <div>
                  <p class="font-semibold">演示用户</p>
                  <p class="text-sm text-neutral-500">右键查看菜单</p>
                </div>
              </div>
            </UCard>
          </ContextMenuWrapper>

          <!-- 群组卡片 -->
          <ContextMenuWrapper 
            type="group" 
            :data="demoGroup" 
            @action="handleContextAction"
          >
            <UCard class="cursor-pointer hover:shadow-md transition-shadow">
              <div class="flex items-center gap-3">
                <div class="w-12 h-12 bg-blue-500 rounded-lg flex items-center justify-center">
                  <UIcon name="i-heroicons-user-group" class="w-6 h-6 text-white" />
                </div>
                <div>
                  <p class="font-semibold">演示群组</p>
                  <p class="text-sm text-neutral-500">右键查看菜单</p>
                </div>
              </div>
            </UCard>
          </ContextMenuWrapper>
        </div>
      </div>
    </UCard>

    <!-- USlideover 演示 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-rectangle-stack" class="w-6 h-6 text-warning-500" />
          <h2 class="text-xl font-semibold">USlideover - 侧边面板</h2>
        </div>
      </template>

      <div class="space-y-4">
        <p class="text-neutral-600 dark:text-neutral-400">
          侧边面板适用于详情展示、表单编辑等场景。点击下面的按钮体验不同类型的面板。
        </p>

        <div class="flex flex-wrap gap-3">
          <UButton @click="openUserDetail" icon="i-heroicons-user">
            用户详情
          </UButton>
          <UButton @click="openGroupDetail" icon="i-heroicons-user-group">
            群组详情
          </UButton>
          <UButton @click="openMomentDetail" icon="i-heroicons-photo">
            动态详情
          </UButton>
          <UButton @click="openCustomDetail" icon="i-heroicons-cog-6-tooth">
            自定义面板
          </UButton>
        </div>
      </div>
    </UCard>

    <!-- 实际应用场景 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-sparkles" class="w-6 h-6 text-purple-500" />
          <h2 class="text-xl font-semibold">实际应用场景</h2>
        </div>
      </template>

      <div class="space-y-4">
        <p class="text-neutral-600 dark:text-neutral-400">
          这些组件在实际应用中的使用场景：
        </p>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
            <UIcon name="i-heroicons-bars-3" class="w-8 h-8 text-blue-500 mb-2" />
            <h4 class="font-semibold mb-2">UDrawer</h4>
            <ul class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
              <li>• 移动端主导航</li>
              <li>• 设置面板</li>
              <li>• 筛选器</li>
              <li>• 通知中心</li>
            </ul>
          </div>

          <div class="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
            <UIcon name="i-heroicons-cursor-arrow-rays" class="w-8 h-8 text-green-500 mb-2" />
            <h4 class="font-semibold mb-2">UContextMenu</h4>
            <ul class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
              <li>• 卡片快速操作</li>
              <li>• 列表项菜单</li>
              <li>• 文本选择菜单</li>
              <li>• 图片操作菜单</li>
            </ul>
          </div>

          <div class="p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
            <UIcon name="i-heroicons-rectangle-stack" class="w-8 h-8 text-purple-500 mb-2" />
            <h4 class="font-semibold mb-2">USlideover</h4>
            <ul class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
              <li>• 详情查看</li>
              <li>• 表单编辑</li>
              <li>• 购物车</li>
              <li>• 聊天面板</li>
            </ul>
          </div>
        </div>
      </div>
    </UCard>
  </div>

  <!-- 抽屉组件实例 -->
  <UDrawer v-model:open="leftDrawerOpen" side="left">
    <template #header>
      <h3 class="text-lg font-semibold">左侧抽屉</h3>
    </template>
    <template #body>
      <div class="p-4">
        <p>这是一个从左侧滑出的抽屉。</p>
        <p class="mt-2 text-sm text-neutral-500">适用于主导航、菜单等场景。</p>
      </div>
    </template>
  </UDrawer>

  <UDrawer v-model:open="rightDrawerOpen" side="right">
    <template #header>
      <h3 class="text-lg font-semibold">右侧抽屉</h3>
    </template>
    <template #body>
      <div class="p-4">
        <p>这是一个从右侧滑出的抽屉。</p>
        <p class="mt-2 text-sm text-neutral-500">适用于设置面板、详情等场景。</p>
      </div>
    </template>
  </UDrawer>

  <UDrawer v-model:open="topDrawerOpen" side="top">
    <template #header>
      <h3 class="text-lg font-semibold">顶部抽屉</h3>
    </template>
    <template #body>
      <div class="p-4">
        <p>这是一个从顶部滑出的抽屉。</p>
        <p class="mt-2 text-sm text-neutral-500">适用于通知、搜索等场景。</p>
      </div>
    </template>
  </UDrawer>

  <UDrawer v-model:open="bottomDrawerOpen" side="bottom">
    <template #header>
      <h3 class="text-lg font-semibold">底部抽屉</h3>
    </template>
    <template #body>
      <div class="p-4">
        <p>这是一个从底部滑出的抽屉。</p>
        <p class="mt-2 text-sm text-neutral-500">适用于操作面板、筛选器等场景。</p>
      </div>
    </template>
  </UDrawer>

  <!-- 移动端导航抽屉 -->
  <MobileDrawer v-model:open="mobileDrawerOpen" />

  <!-- 详情侧边面板 -->
  <DetailSlideover 
    v-model:open="slideoverOpen" 
    :type="slideoverType" 
    :data="slideoverData"
    @action="handleSlideoverAction"
  >
    <template v-if="slideoverType === 'custom'" #content>
      <div class="space-y-4">
        <h3 class="text-lg font-semibold">自定义面板内容</h3>
        <p>这是一个自定义的侧边面板，您可以在这里放置任何内容。</p>
        <UAlert 
          icon="i-heroicons-information-circle"
          title="提示"
          description="这个面板支持完全自定义的内容和布局。"
        />
      </div>
    </template>
  </DetailSlideover>
</template>

<script setup lang="ts">
// 页面元数据
definePageMeta({
  title: 'UI组件演示'
})

// 响应式状态
const leftDrawerOpen = ref(false)
const rightDrawerOpen = ref(false)
const topDrawerOpen = ref(false)
const bottomDrawerOpen = ref(false)
const mobileDrawerOpen = ref(false)

const slideoverOpen = ref(false)
const slideoverType = ref<'user' | 'group' | 'moment' | 'custom'>('user')
const slideoverData = ref<any>(null)

// Toast 通知
const toast = useSmartToast()

// 演示数据
const demoUser = {
  user_id: 1,
  nickname: '演示用户',
  email: 'demo@example.com',
  is_temporary: false,
  moments_count: 42,
  friends_count: 128
}

const demoGroup = {
  group_id: 1,
  name: '演示群组',
  description: '这是一个用于演示的群组',
  member_count: 256,
  is_private: false,
  created_at: new Date().toISOString()
}

const demoMoment = {
  moment_id: 1,
  content: '这是一条演示动态，展示了动态的基本信息和操作。',
  user: demoUser,
  like_count: 15,
  comment_count: 8,
  created_at: new Date().toISOString(),
  images: []
}

// 处理右键菜单操作
const handleContextAction = (action: string, data?: any) => {
  toast.info({
    title: '操作触发',
    description: `执行了操作: ${action}`
  })
  console.log('Context action:', action, data)
}

// 打开用户详情
const openUserDetail = () => {
  slideoverType.value = 'user'
  slideoverData.value = demoUser
  slideoverOpen.value = true
}

// 打开群组详情
const openGroupDetail = () => {
  slideoverType.value = 'group'
  slideoverData.value = demoGroup
  slideoverOpen.value = true
}

// 打开动态详情
const openMomentDetail = () => {
  slideoverType.value = 'moment'
  slideoverData.value = demoMoment
  slideoverOpen.value = true
}

// 打开自定义面板
const openCustomDetail = () => {
  slideoverType.value = 'custom'
  slideoverData.value = null
  slideoverOpen.value = true
}

// 处理侧边面板操作
const handleSlideoverAction = (action: string, data?: any) => {
  toast.success({
    title: '操作成功',
    description: `执行了操作: ${action}`
  })
  console.log('Slideover action:', action, data)
}
</script>
