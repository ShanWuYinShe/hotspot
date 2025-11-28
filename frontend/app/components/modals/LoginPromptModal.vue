<template>
  <UModal v-model="isOpen" :ui="{ width: 'sm:max-w-md' }">
    <UCard>
      <!-- 头部 -->
      <template #header>
        <div class="flex items-center justify-between">
          <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">
            需要登录
          </h3>
          <UButton
            color="gray"
            variant="ghost"
            icon="i-heroicons-x-mark-20-solid"
            @click="close"
          />
        </div>
      </template>

      <!-- 内容 -->
      <div class="space-y-4">
        <!-- 图标 -->
        <div class="flex justify-center">
          <div class="w-16 h-16 bg-primary-50 dark:bg-primary-900/20 rounded-full flex items-center justify-center">
            <UIcon name="i-heroicons-lock-closed" class="w-8 h-8 text-neutral-600 dark:text-neutral-300" />
          </div>
        </div>

        <!-- 提示信息 -->
        <div class="text-center space-y-2">
          <p class="text-base text-neutral-700 dark:text-neutral-300">
            {{ message }}
          </p>
          <p class="text-sm text-neutral-500 dark:text-neutral-400">
            登录后即可解锁所有功能
          </p>
        </div>

        <!-- 功能列表 -->
        <div class="bg-gray-50 dark:bg-gray-800/50 rounded-lg p-4">
          <ul class="space-y-2 text-sm text-neutral-600 dark:text-neutral-400">
            <li class="flex items-center gap-2">
              <UIcon name="i-heroicons-check-circle" class="w-4 h-4 text-green-500" />
              发布和分享生活瞬间
            </li>
            <li class="flex items-center gap-2">
              <UIcon name="i-heroicons-check-circle" class="w-4 h-4 text-green-500" />
              与好友互动和聊天
            </li>
            <li class="flex items-center gap-2">
              <UIcon name="i-heroicons-check-circle" class="w-4 h-4 text-green-500" />
              加入兴趣群组
            </li>
            <li class="flex items-center gap-2">
              <UIcon name="i-heroicons-check-circle" class="w-4 h-4 text-green-500" />
              发现附近的人和地点
            </li>
          </ul>
        </div>
      </div>

      <!-- 底部操作 -->
      <template #footer>
        <div class="flex flex-col gap-3">
          <!-- 登录按钮 -->
          <UButton
            block
            size="lg"
            @click="goToLogin"
          >
            立即登录
          </UButton>

          <!-- 注册引导 -->
          <div class="text-center text-sm text-neutral-500 dark:text-neutral-400">
            还没有账号？
            <UButton
              variant="link"
              :padded="false"
              @click="goToRegister"
            >
              立即注册
            </UButton>
          </div>

          <!-- 游客继续浏览 -->
          <UButton
            block
            color="gray"
            variant="ghost"
            @click="close"
          >
            继续浏览
          </UButton>
        </div>
      </template>
    </UCard>
  </UModal>
</template>

<script setup lang="ts">
const props = defineProps<{
  actionName?: string
  redirectPath?: string
}>()

const isOpen = ref(true)
const router = useRouter()

// 生成友好的提示信息
const message = computed(() => {
  if (props.actionName) {
    return `登录后即可${props.actionName}`
  }
  return '登录后即可使用完整功能'
})

// 关闭模态框
const close = () => {
  isOpen.value = false
}

// 跳转到登录页
const goToLogin = () => {
  close()
  const query: Record<string, string> = {
    reason: 'action_required'
  }
  if (props.redirectPath) {
    query.redirect = props.redirectPath
  }
  if (props.actionName) {
    query.action = props.actionName
  }
  router.push({
    path: '/auth/login',
    query
  })
}

// 跳转到注册页
const goToRegister = () => {
  close()
  const query: Record<string, string> = {}
  if (props.redirectPath) {
    query.redirect = props.redirectPath
  }
  router.push({
    path: '/auth/register',
    query
  })
}
</script>
