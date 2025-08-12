<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center">
    <div class="text-center px-4">
      <!-- 404 图标 -->
      <div class="mb-8">
        <div class="w-32 h-32 bg-primary-100 rounded-full flex items-center justify-center mx-auto mb-6">
          <UIcon name="i-heroicons-question-mark-circle" class="w-16 h-16 text-primary-600" />
        </div>
      </div>

      <!-- 错误信息 -->
      <h1 class="text-6xl font-bold text-gray-900 mb-4">404</h1>
      <h2 class="text-2xl font-semibold text-gray-800 mb-4">页面未找到</h2>
      <p class="text-gray-600 mb-8 max-w-md mx-auto">
        抱歉，您访问的页面不存在或已被移动。请检查网址是否正确，或返回首页继续浏览。
      </p>

      <!-- 操作按钮 -->
      <div class="space-y-4 sm:space-y-0 sm:space-x-4 sm:flex sm:justify-center">
        <UButton size="lg" @click="navigateTo('/')" class="w-full sm:w-auto">
          <UIcon name="i-heroicons-home" class="w-5 h-5 mr-2" />
          返回首页
        </UButton>
        <UButton variant="outline" size="lg" @click="goBack" class="w-full sm:w-auto">
          <UIcon name="i-heroicons-arrow-left" class="w-5 h-5 mr-2" />
          返回上页
        </UButton>
      </div>

      <!-- 帮助链接 -->
      <div class="mt-12 text-sm text-gray-500">
        <p class="mb-2">需要帮助？</p>
        <div class="space-x-4">
          <UButton variant="ghost" size="sm" @click="navigateTo('/settings')">
            联系客服
          </UButton>
          <UButton variant="ghost" size="sm" @click="navigateTo('/settings')">
            意见反馈
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
// 页面元数据
definePageMeta({
  title: '页面未找到',
  layout: false // 使用无布局样式
})

// 返回上一页
const goBack = () => {
  if (history.length > 1) {
    history.back()
  } else {
    navigateTo('/')
  }
}
</script>
