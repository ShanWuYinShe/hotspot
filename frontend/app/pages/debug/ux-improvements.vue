<template>
  <div class="max-w-4xl mx-auto p-6 space-y-8">
    <div class="text-center">
      <h1 class="text-3xl font-bold text-neutral-900 dark:text-neutral-100 mb-2">
        用户体验优化总览
      </h1>
      <p class="text-neutral-600 dark:text-neutral-400">
        HotSpot 前端项目的用户体验改进汇总
      </p>
    </div>

    <!-- 导航优化 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-bars-3" class="w-6 h-6 text-primary-500" />
          <h2 class="text-xl font-semibold">导航体验优化</h2>
        </div>
      </template>

      <div class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div class="space-y-2">
            <h3 class="font-medium text-neutral-900 dark:text-neutral-100">✅ 已完成</h3>
            <ul class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
              <li>• 修复语言选择框显示当前选项</li>
              <li>• 添加语言切换加载状态和反馈</li>
              <li>• 优化消息按钮的动画效果</li>
              <li>• 改进移动端和桌面端的响应式设计</li>
            </ul>
          </div>
          <div class="space-y-2">
            <h3 class="font-medium text-neutral-900 dark:text-neutral-100">🎯 特性</h3>
            <ul class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
              <li>• 防重复切换保护</li>
              <li>• 成功/失败状态提示</li>
              <li>• 加载状态视觉反馈</li>
              <li>• 本地存储语言偏好</li>
            </ul>
          </div>
        </div>
      </div>
    </UCard>

    <!-- 组件库 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-cube" class="w-6 h-6 text-success-500" />
          <h2 class="text-xl font-semibold">通用组件库</h2>
        </div>
      </template>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <!-- LoadingState 组件 -->
        <div class="space-y-3">
          <h3 class="font-medium flex items-center gap-2">
            <UIcon name="i-heroicons-arrow-path" class="w-4 h-4" />
            LoadingState
          </h3>
          <div class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
            <p>• 统一的加载状态展示</p>
            <p>• 支持自定义图标和文本</p>
            <p>• 可选进度条显示</p>
            <p>• 取消操作支持</p>
          </div>
          <LoadingState 
            title="示例加载"
            description="这是一个加载状态示例"
            icon="i-heroicons-photo"
            class="!py-4"
          />
        </div>

        <!-- EmptyState 组件 -->
        <div class="space-y-3">
          <h3 class="font-medium flex items-center gap-2">
            <UIcon name="i-heroicons-inbox" class="w-4 h-4" />
            EmptyState
          </h3>
          <div class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
            <p>• 优雅的空状态展示</p>
            <p>• 自定义插图支持</p>
            <p>• 引导用户操作</p>
            <p>• 多种样式变体</p>
          </div>
          <EmptyState 
            title="暂无数据"
            description="这是一个空状态示例"
            action-text="添加内容"
            class="!py-4"
          />
        </div>

        <!-- ErrorState 组件 -->
        <div class="space-y-3">
          <h3 class="font-medium flex items-center gap-2">
            <UIcon name="i-heroicons-exclamation-triangle" class="w-4 h-4" />
            ErrorState
          </h3>
          <div class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
            <p>• 友好的错误状态展示</p>
            <p>• 重试和返回操作</p>
            <p>• 错误详情展开</p>
            <p>• 客服联系入口</p>
          </div>
          <ErrorState 
            title="加载失败"
            description="这是一个错误状态示例"
            class="!py-4"
            @retry="() => {}"
          />
        </div>
      </div>
    </UCard>

    <!-- 交互增强 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-cursor-arrow-ripple" class="w-6 h-6 text-warning-500" />
          <h2 class="text-xl font-semibold">交互体验增强</h2>
        </div>
      </template>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- UButton Loading 示例 -->
        <div class="space-y-3">
          <h3 class="font-medium">UButton Loading 功能</h3>
          <div class="space-y-2">
            <UButton 
              :loading="buttonLoading"
              @click="simulateLoading"
              icon="i-heroicons-paper-airplane"
            >
              {{ buttonLoading ? '处理中...' : '点击测试' }}
            </UButton>
            <p class="text-xs text-neutral-500">
              使用UButton内置的loading功能，更简洁高效
            </p>
          </div>
        </div>

        <!-- 简单输入框示例 -->
        <div class="space-y-3">
          <h3 class="font-medium">增强输入框</h3>
          <div class="space-y-2">
            <label class="text-sm font-medium text-neutral-900 dark:text-neutral-100">
              用户名
            </label>
            <UInput
              v-model="inputValue"
              placeholder="请输入用户名"
              icon="i-heroicons-user"
              :maxlength="20"
            />
            <p class="text-xs text-neutral-500">
              支持字符计数、验证、清除等功能 ({{ inputValue.length }}/20)
            </p>
          </div>
        </div>
      </div>
    </UCard>

    <!-- Composables -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-code-bracket" class="w-6 h-6 text-info-500" />
          <h2 class="text-xl font-semibold">Composables 工具库</h2>
        </div>
      </template>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div class="space-y-4">
          <h3 class="font-medium">useErrorHandler</h3>
          <ul class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
            <li>• 统一的错误处理机制</li>
            <li>• 智能错误分类和提示</li>
            <li>• API、网络、权限错误处理</li>
            <li>• 成功、警告、信息提示</li>
          </ul>
          <UButton @click="testErrorHandler" size="sm" variant="outline">
            测试错误处理
          </UButton>
        </div>

        <div class="space-y-4">
          <h3 class="font-medium">useUXEnhancements</h3>
          <ul class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
            <li>• 防抖和节流处理</li>
            <li>• 剪贴板和分享功能</li>
            <li>• 设备检测和适配</li>
            <li>• 时间格式化和工具函数</li>
          </ul>
          <div class="flex gap-2">
            <UButton @click="testCopy" size="sm" variant="outline">
              测试复制
            </UButton>
            <UButton @click="testShare" size="sm" variant="outline">
              测试分享
            </UButton>
          </div>
        </div>
      </div>
    </UCard>

    <!-- 设备信息 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-device-phone-mobile" class="w-6 h-6 text-neutral-500" />
          <h2 class="text-xl font-semibold">设备适配信息</h2>
        </div>
      </template>

      <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-center">
        <div class="space-y-2">
          <p class="text-sm font-medium">设备类型</p>
          <UBadge color="neutral" class="hidden sm:inline-flex">
            桌面端
          </UBadge>
          <UBadge color="primary" class="sm:hidden">
            移动端
          </UBadge>
        </div>
        <div class="space-y-2">
          <p class="text-sm font-medium">触摸设备</p>
          <UBadge :color="isTouchDevice ? 'success' : 'neutral'">
            {{ isTouchDevice ? '是' : '否' }}
          </UBadge>
        </div>
        <div class="space-y-2">
          <p class="text-sm font-medium">响应式检测</p>
          <UBadge color="success">
            使用CSS
          </UBadge>
        </div>
        <div class="space-y-2">
          <p class="text-sm font-medium">响应式方案</p>
          <UBadge color="success">
            Tailwind CSS
          </UBadge>
        </div>
      </div>
    </UCard>
  </div>
</template>

<script setup lang="ts">
const { handleError, showSuccess, showWarning } = useErrorHandler()
const { copyToClipboard, shareContent, isTouchDevice } = useUXEnhancements()

const buttonLoading = ref(false)
const inputValue = ref('')

// 模拟加载
const simulateLoading = async () => {
  buttonLoading.value = true
  await new Promise(resolve => setTimeout(resolve, 2000))
  buttonLoading.value = false
}

// 测试错误处理
const testErrorHandler = () => {
  const errors = [
    { status: 401, message: '登录已过期' },
    { status: 404, message: '内容不存在' },
    { status: 500, message: '服务器错误' },
    { name: 'NetworkError', message: '网络连接失败' }
  ]
  
  const randomError = errors[Math.floor(Math.random() * errors.length)]
  handleError(randomError, '测试错误')
}

// 测试复制
const testCopy = async () => {
  await copyToClipboard('这是一个测试文本', '测试文本已复制')
}

// 测试分享
const testShare = async () => {
  const success = await shareContent({
    title: 'HotSpot 用户体验优化',
    text: '查看 HotSpot 项目的用户体验改进',
    url: window.location.href
  })
  
  if (!success) {
    showWarning('分享失败', '您的浏览器不支持原生分享功能')
  }
}

// 使用 Tailwind CSS 响应式类代替 JavaScript 检测
</script>
