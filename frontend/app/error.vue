<template>
  <div class="min-h-screen flex items-center justify-center bg-surface-dark p-4">
    <div class="text-center max-w-md">
      <!-- 错误图标 -->
      <div class="mb-6">
        <div
          class="w-24 h-24 mx-auto rounded-full flex items-center justify-center"
          :class="isNotFound ? 'bg-primary-subtle' : 'bg-red-deep'"
        >
          <UIcon
            :name="isNotFound ? 'i-heroicons-map' : 'i-heroicons-exclamation-triangle'"
            class="icon-2xl"
            :class="isNotFound ? 'text-accent' : 'text-error'"
          />
        </div>
      </div>

      <!-- 错误码 -->
      <p class="text-6xl font-bold text-inverted mb-2">
        {{ error?.statusCode || 500 }}
      </p>

      <!-- 错误标题 -->
      <h1 class="text-2xl font-semibold text-primary-content mb-2">
        {{ errorTitle }}
      </h1>

      <!-- 错误描述 -->
      <p class="text-secondary-content mb-8">
        {{ errorMessage }}
      </p>

      <!-- 操作按钮 -->
      <div class="flex flex-col sm:flex-row gap-3 justify-center">
        <UButton
          size="lg"
          @click="handleBack"
        >
          <UIcon name="i-heroicons-arrow-left" class="w-4 h-4 mr-2" />
          返回上一页
        </UButton>
        <UButton
          size="lg"
          variant="outline"
          @click="handleHome"
        >
          <UIcon name="i-heroicons-home" class="w-4 h-4 mr-2" />
          回到首页
        </UButton>
      </div>

      <!-- 开发环境显示详细错误 -->
      <div
        v-if="isDev && error?.message"
        class="mt-8 p-4 bg-interactive rounded-lg text-left"
      >
        <p class="text-xs font-mono text-secondary-content break-all">
          {{ error.message }}
        </p>
        <p v-if="error.stack" class="text-xs font-mono text-secondary-content mt-2 break-all whitespace-pre-wrap">
          {{ error.stack }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface ErrorProps {
  statusCode?: number
  message?: string
  stack?: string
}

const props = defineProps<{
  error: ErrorProps
}>()

const isDev = import.meta.dev

const isNotFound = computed(() => props.error?.statusCode === 404)

const errorTitle = computed(() => {
  const code = props.error?.statusCode
  switch (code) {
    case 404:
      return '页面不存在'
    case 403:
      return '没有权限'
    case 401:
      return '请先登录'
    case 500:
      return '服务器错误'
    default:
      return '出错了'
  }
})

const errorMessage = computed(() => {
  const code = props.error?.statusCode
  switch (code) {
    case 404:
      return '你访问的页面可能已被移除或地址输入错误'
    case 403:
      return '你没有权限访问此页面'
    case 401:
      return '请登录后再访问此页面'
    case 500:
      return '服务器遇到了一些问题，请稍后再试'
    default:
      return '发生了一些意外情况，请稍后再试'
  }
})

const handleBack = () => {
  clearError()
  if (window.history.length > 1) {
    window.history.back()
  } else {
    navigateTo('/')
  }
}

const handleHome = () => {
  clearError({ redirect: '/' })
}
</script>




