<template>
  <!-- 创建内容统一Modal -->
  <UModal v-model:open="isOpen" :title="currentTitle" size="lg">
    <template #body>
      <!-- 主菜单 -->
      <div v-if="currentView === 'menu'" class="space-y-3">
        <!-- 创建群组 -->
        <UButton
          @click="switchToView('group')"
          variant="ghost"
          class="w-full justify-start p-4 h-auto"
        >
          <div class="flex items-center space-x-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900">
              <UIcon name="i-heroicons-user-group" class="h-5 w-5 text-blue-600 dark:text-blue-400" />
            </div>
            <div class="text-left">
              <div class="font-medium text-neutral-900 dark:text-white">创建群组</div>
              <div class="text-sm text-neutral-600 dark:text-neutral-400">邀请朋友加入群聊</div>
            </div>
          </div>
        </UButton>

        <!-- 发布动态 -->
        <UButton
          @click="switchToView('moment')"
          variant="ghost"
          class="w-full justify-start p-4 h-auto"
        >
          <div class="flex items-center space-x-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900">
              <UIcon name="i-heroicons-photo" class="h-5 w-5 text-green-600 dark:text-green-400" />
            </div>
            <div class="text-left">
              <div class="font-medium text-neutral-900 dark:text-white">发布动态</div>
              <div class="text-sm text-neutral-600 dark:text-neutral-400">分享你的精彩时刻</div>
            </div>
          </div>
        </UButton>

        <!-- 寻找附近 -->
        <UButton
          to="/nearby-users"
          @click="isOpen = false"
          variant="ghost"
          class="w-full justify-start p-4 h-auto"
        >
          <div class="flex items-center space-x-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-orange-100 dark:bg-orange-900">
              <UIcon name="i-heroicons-map-pin" class="h-5 w-5 text-orange-600 dark:text-orange-400" />
            </div>
            <div class="text-left">
              <div class="font-medium text-neutral-900 dark:text-white">寻找附近</div>
              <div class="text-sm text-neutral-600 dark:text-neutral-400">发现身边的人和群组</div>
            </div>
          </div>
        </UButton>
      </div>

      <!-- 创建群组表单 -->
      <div v-else-if="currentView === 'group'">
        <CreateGroupModal 
          v-model:open="groupModalOpen" 
          @success="onGroupCreated"
          :embedded="true"
        />
      </div>

      <!-- 发布动态表单 -->
      <div v-else-if="currentView === 'moment'">
        <CreateMomentForm @success="onMomentCreated" />
      </div>
    </template>
    
    <template #footer v-if="currentView !== 'menu'">
      <div class="flex justify-between">
        <UButton @click="backToMenu" variant="ghost">
          <UIcon name="i-heroicons-arrow-left" class="h-4 w-4 mr-2" />
          返回
        </UButton>
        <UButton @click="closeModal" variant="ghost">
          取消
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
// 使用 defineModel 简化双向绑定
const isOpen = defineModel<boolean>('open', { default: false })
const toast = useSmartToast()

// 视图状态管理
type ViewType = 'menu' | 'group' | 'moment'
const currentView = ref<ViewType>('menu')
const groupModalOpen = ref(false)

// 动态标题
const currentTitle = computed(() => {
  switch (currentView.value) {
    case 'group': return '创建群组'
    case 'moment': return '发布动态'
    default: return '创建新内容'
  }
})

// 视图切换
function switchToView(view: ViewType) {
  currentView.value = view
  if (view === 'group') {
    groupModalOpen.value = true
  }
}

function backToMenu() {
  currentView.value = 'menu'
  groupModalOpen.value = false
}

function closeModal() {
  isOpen.value = false
  currentView.value = 'menu'
  groupModalOpen.value = false
}

// 寻找附近：已改为按钮 to="/nearby-users"，此函数不再需要

// 创建成功回调
function onGroupCreated(group: any) {
  toast.success({
    title: '群组创建成功',
    description: `"${group.name}" 群组已创建`
  })
  closeModal()
  navigateTo(`/groups/${group.group_id}`)
}

function onMomentCreated() {
  toast.success('动态发布成功')
  closeModal()
  navigateTo('/moments')
}

// 监听模态框关闭，重置视图
watch(isOpen, (newVal) => {
  if (!newVal) {
    currentView.value = 'menu'
    groupModalOpen.value = false
  }
})
</script>
