<template>
  <div class="max-w-4xl mx-auto p-6 space-y-8">
    <div class="text-center">
      <h1 class="text-3xl font-bold text-neutral-900 dark:text-neutral-100 mb-2">
        Nuxt UI v3 高级组件使用指南
      </h1>
      <p class="text-neutral-600 dark:text-neutral-400">
        UDrawer、UContextMenu、USlideover 的最佳实践和使用方法
      </p>
    </div>

    <!-- UDrawer 使用指南 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-bars-3" class="w-6 h-6 text-primary-500" />
          <h2 class="text-xl font-semibold">UDrawer 使用指南</h2>
        </div>
      </template>

      <div class="space-y-6">
        <div>
          <h3 class="text-lg font-semibold mb-3">最佳实践</h3>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UAlert 
              icon="i-heroicons-check-circle"
              color="success"
              title="推荐做法"
              description="使用 UDrawer 作为移动端主导航，提供清晰的层级结构"
            />
            <UAlert 
              icon="i-heroicons-light-bulb"
              color="info"
              title="使用场景"
              description="设置面板、筛选器、通知中心、移动端菜单"
            />
          </div>
        </div>

        <div>
          <h3 class="text-lg font-semibold mb-3">组件特性</h3>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="space-y-2">
              <h4 class="font-medium">支持的方向</h4>
              <ul class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
                <li>• <code>left</code> - 左侧滑出（推荐用于导航）</li>
                <li>• <code>right</code> - 右侧滑出（推荐用于设置）</li>
                <li>• <code>top</code> - 顶部滑出（推荐用于通知）</li>
                <li>• <code>bottom</code> - 底部滑出（推荐用于操作面板）</li>
              </ul>
            </div>
            <div class="space-y-2">
              <h4 class="font-medium">内置功能</h4>
              <ul class="text-sm text-neutral-600 dark:text-neutral-400 space-y-1">
                <li>• 自动遮罩层管理</li>
                <li>• 键盘 ESC 关闭</li>
                <li>• 点击外部关闭</li>
                <li>• 滑动手势支持</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </UCard>

    <!-- UContextMenu 使用指南 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-cursor-arrow-rays" class="w-6 h-6 text-success-500" />
          <h2 class="text-xl font-semibold">UContextMenu 使用指南</h2>
        </div>
      </template>

      <div class="space-y-6">
        <div>
          <h3 class="text-lg font-semibold mb-3">支持的菜单类型</h3>
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <div class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
              <h4 class="font-medium mb-2">用户菜单</h4>
              <p class="text-sm text-neutral-600 dark:text-neutral-400">
                查看资料、发送消息、添加好友、举报等操作
              </p>
            </div>
            <div class="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
              <h4 class="font-medium mb-2">群组菜单</h4>
              <p class="text-sm text-neutral-600 dark:text-neutral-400">
                查看群组、加入群组、分享链接、举报等操作
              </p>
            </div>
            <div class="p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
              <h4 class="font-medium mb-2">动态菜单</h4>
              <p class="text-sm text-neutral-600 dark:text-neutral-400">
                点赞、评论、分享、编辑、删除等操作
              </p>
            </div>
          </div>
        </div>
      </div>
    </UCard>

    <!-- USlideover 使用指南 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-rectangle-stack" class="w-6 h-6 text-warning-500" />
          <h2 class="text-xl font-semibold">USlideover 使用指南</h2>
        </div>
      </template>

      <div class="space-y-6">
        <div>
          <h3 class="text-lg font-semibold mb-3">适用场景对比</h3>
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-neutral-200 dark:border-neutral-700">
                  <th class="text-left p-3 font-semibold">场景</th>
                  <th class="text-left p-3 font-semibold">UModal</th>
                  <th class="text-left p-3 font-semibold">UDrawer</th>
                  <th class="text-left p-3 font-semibold">USlideover</th>
                </tr>
              </thead>
              <tbody class="text-neutral-600 dark:text-neutral-400">
                <tr class="border-b border-neutral-100 dark:border-neutral-800">
                  <td class="p-3 font-medium">详情查看</td>
                  <td class="p-3">❌ 遮挡内容</td>
                  <td class="p-3">⚠️ 占用全屏</td>
                  <td class="p-3">✅ 最佳选择</td>
                </tr>
                <tr class="border-b border-neutral-100 dark:border-neutral-800">
                  <td class="p-3 font-medium">表单编辑</td>
                  <td class="p-3">✅ 适合简单表单</td>
                  <td class="p-3">⚠️ 适合复杂表单</td>
                  <td class="p-3">✅ 适合中等表单</td>
                </tr>
                <tr class="border-b border-neutral-100 dark:border-neutral-800">
                  <td class="p-3 font-medium">导航菜单</td>
                  <td class="p-3">❌ 不适合</td>
                  <td class="p-3">✅ 最佳选择</td>
                  <td class="p-3">❌ 不适合</td>
                </tr>
                <tr>
                  <td class="p-3 font-medium">设置面板</td>
                  <td class="p-3">⚠️ 适合简单设置</td>
                  <td class="p-3">✅ 适合复杂设置</td>
                  <td class="p-3">✅ 适合中等设置</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </UCard>

    <!-- 实际项目集成 -->
    <UCard>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon name="i-heroicons-rocket-launch" class="w-6 h-6 text-purple-500" />
          <h2 class="text-xl font-semibold">项目集成指南</h2>
        </div>
      </template>

      <div class="space-y-6">
        <div>
          <h3 class="text-lg font-semibold mb-3">已集成的组件</h3>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UAlert 
              icon="i-heroicons-check-circle"
              color="success"
              title="MobileDrawer.vue"
              description="移动端导航抽屉，包含用户信息、主要功能、设置等"
            />
            <UAlert 
              icon="i-heroicons-check-circle"
              color="success"
              title="ContextMenuWrapper.vue"
              description="右键菜单包装器，支持用户、群组、动态等类型"
            />
            <UAlert 
              icon="i-heroicons-check-circle"
              color="success"
              title="DetailSlideover.vue"
              description="详情侧边面板，支持用户、群组、动态详情展示"
            />
            <UAlert 
              icon="i-heroicons-information-circle"
              color="info"
              title="AppNavigation.vue"
              description="已更新移动端导航，集成了 MobileDrawer"
            />
          </div>
        </div>

        <div>
          <h3 class="text-lg font-semibold mb-3">使用建议</h3>
          <div class="space-y-4">
            <div class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
              <h4 class="font-medium mb-2 flex items-center gap-2">
                <UIcon name="i-heroicons-device-phone-mobile" class="w-4 h-4" />
                移动端优先
              </h4>
              <p class="text-sm text-neutral-600 dark:text-neutral-400">
                在移动端使用 UDrawer 替代传统的下拉菜单，提供更好的触摸体验。
              </p>
            </div>

            <div class="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
              <h4 class="font-medium mb-2 flex items-center gap-2">
                <UIcon name="i-heroicons-cursor-arrow-rays" class="w-4 h-4" />
                右键菜单增强
              </h4>
              <p class="text-sm text-neutral-600 dark:text-neutral-400">
                为所有可交互的卡片和列表项添加右键菜单，提供快速操作入口。
              </p>
            </div>

            <div class="p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
              <h4 class="font-medium mb-2 flex items-center gap-2">
                <UIcon name="i-heroicons-rectangle-stack" class="w-4 h-4" />
                详情面板
              </h4>
              <p class="text-sm text-neutral-600 dark:text-neutral-400">
                使用 USlideover 替代模态框来展示详情，保持上下文连续性。
              </p>
            </div>
          </div>
        </div>

        <div>
          <h3 class="text-lg font-semibold mb-3">下一步计划</h3>
          <div class="space-y-2">
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-check-circle" class="w-4 h-4 text-success-500" />
              <span class="text-sm">在更多页面中集成右键菜单</span>
            </div>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-check-circle" class="w-4 h-4 text-success-500" />
              <span class="text-sm">优化移动端用户体验</span>
            </div>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-check-circle" class="w-4 h-4 text-success-500" />
              <span class="text-sm">添加手势支持和动画效果</span>
            </div>
          </div>
        </div>

        <div>
          <h3 class="text-lg font-semibold mb-3">体验演示</h3>
          <div class="flex gap-3">
            <UButton to="/demo/ui-components" icon="i-heroicons-play">
              查看组件演示
            </UButton>
            <UButton variant="outline" @click="openMobileDrawer">
              体验移动端抽屉
            </UButton>
          </div>
        </div>
      </div>
    </UCard>
  </div>

  <!-- 移动端导航抽屉 -->
  <MobileDrawer v-model:open="mobileDrawerOpen" />
</template>

<script setup lang="ts">
// 页面元数据
definePageMeta({
  title: 'UI组件使用指南'
})

// 响应式状态
const mobileDrawerOpen = ref(false)

// 打开移动端抽屉
const openMobileDrawer = () => {
  mobileDrawerOpen.value = true
}
</script>
