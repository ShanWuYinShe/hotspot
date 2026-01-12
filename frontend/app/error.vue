<template>
  <div class="min-h-screen flex items-center justify-center p-4">
    <div class="w-full max-w-md">
      <UEmpty
        :icon="isNotFound ? 'i-lucide-map' : 'i-lucide-triangle-alert'"
        :title="`${error?.statusCode || 500} · ${errorTitle}`"
        :description="errorMessage"
        variant="subtle"
        :actions="[
          { label: '返回上一页', icon: 'i-lucide-arrow-left', onClick: handleBack },
          { label: '回到首页', icon: 'i-lucide-house', variant: 'outline', color: 'neutral', onClick: handleHome }
        ]"
      />

      <UCard v-if="isDev && error?.message" variant="soft" class="mt-6">
        <div class="space-y-2">
          <UBadge color="neutral" variant="soft" size="xs">message</UBadge>
          <pre class="text-xs font-mono break-all whitespace-pre-wrap bg-neutral-100 dark:bg-neutral-800 p-4 rounded-md text-neutral-900 dark:text-neutral-100 overflow-x-auto">{{ error.message }}</pre>

          <template v-if="error.stack">
            <UBadge color="neutral" variant="soft" size="xs">stack</UBadge>
            <pre class="text-xs font-mono break-all whitespace-pre-wrap bg-neutral-100 dark:bg-neutral-800 p-4 rounded-md text-neutral-900 dark:text-neutral-100 overflow-x-auto">{{ error.stack }}</pre>
          </template>
        </div>
      </UCard>
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

const router = useRouter()
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
  // 利用 Vue Router 的历史状态判断是否有上一页
  if (router.options.history.state.back) {
    router.back()
  } else {
    navigateTo('/')
  }
}

const handleHome = () => {
  clearError({ redirect: '/' })
}
</script>
