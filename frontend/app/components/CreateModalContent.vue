<template>
  <UModal :title="currentTitle" size="lg">
    <template #body>
      <!-- 主菜单 -->
      <div v-if="currentView === 'menu'" class="space-y-3">
        <!-- 创建群组 -->
        <UButton
          @click="switchToView('group')"
          variant="ghost"
          class="w-full justify-start p-4 h-auto min-h-[60px] touch-manipulation transition-colors duration-200"
        >
          <div class="flex items-center space-x-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/30 flex-shrink-0">
              <UIcon name="i-heroicons-user-group" class="h-5 w-5 text-blue-600 dark:text-blue-400" />
            </div>
            <div class="text-left flex-1 min-w-0">
              <div class="font-medium text-neutral-900 dark:text-white truncate">创建群组</div>
              <div class="text-sm text-neutral-600 dark:text-neutral-400 truncate">邀请朋友加入群聊</div>
            </div>
          </div>
        </UButton>

        <!-- 发布动态 -->
        <UButton
          @click="switchToView('moment')"
          variant="ghost"
          class="w-full justify-start p-4 h-auto min-h-[60px] touch-manipulation transition-colors duration-200"
        >
          <div class="flex items-center space-x-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-100 dark:bg-purple-900/30 flex-shrink-0">
              <UIcon name="i-heroicons-photo" class="h-5 w-5 text-purple-600 dark:text-purple-400" />
            </div>
            <div class="text-left flex-1 min-w-0">
              <div class="font-medium text-neutral-900 dark:text-white truncate">发布动态</div>
              <div class="text-sm text-neutral-600 dark:text-neutral-400 truncate">分享你的精彩时刻</div>
            </div>
          </div>
        </UButton>

        <!-- 寻找附近 -->
        <UButton
          @click="handleNearbyClick"
          variant="ghost"
          class="w-full justify-start p-4 h-auto min-h-[60px] touch-manipulation transition-colors duration-200"
        >
          <div class="flex items-center space-x-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/30 flex-shrink-0">
              <UIcon name="i-heroicons-map-pin" class="h-5 w-5 text-green-600 dark:text-green-400" />
            </div>
            <div class="text-left flex-1 min-w-0">
              <div class="font-medium text-neutral-900 dark:text-white truncate">寻找附近</div>
              <div class="text-sm text-neutral-600 dark:text-neutral-400 truncate">发现身边的人和群组</div>
            </div>
          </div>
        </UButton>
      </div>

      <!-- 创建群组表单 -->
      <div v-else-if="currentView === 'group'">
        <CreateGroupForm @success="onGroupCreated" @cancel="backToMenu" />
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
// Props
interface Props {
  open?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  open: false
});

// Emits
const emit = defineEmits<{
  'update:open': [value: boolean];
  'success': [data?: { type: string; data?: any }];
  'close': [];
}>();

// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen;

const toast = useToast()

// 视图状态管理
type ViewType = 'menu' | 'group' | 'moment'
const currentView = ref<ViewType>('menu')

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
}

function backToMenu() {
  currentView.value = 'menu'
}

function closeModal() {
  currentView.value = 'menu'
  emit('close')
}

// 处理寻找附近点击
function handleNearbyClick() {
  closeModal()
  navigateTo('/users/nearby')
}

// 创建成功回调
function onGroupCreated(group: { id: number; name: string }) {
  toast.add({
    title: '群组创建成功',
    description: `"${group.name}" 群组已创建`,
    color: 'success'
  })
  closeModal()
  emit('success', { type: 'group', data: group })
  navigateTo(`/groups/${group.id}`)
}

function onMomentCreated() {
  toast.add({
    title: '动态发布成功',
    color: 'success'
  })
  closeModal()
  emit('success', { type: 'moment' })
  navigateTo('/moments')
}

// 当组件卸载时重置视图状态
onUnmounted(() => {
  currentView.value = 'menu'
})
</script>

