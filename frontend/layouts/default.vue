<template>
  <div class="app-layout">
    <!-- 页面内容 -->
    <main class="app-main">
      <slot />
    </main>

    <!-- 底部导航栏 -->
    <AppNavigationBar />
  </div>
</template>

<script setup lang="ts">
import AppNavigationBar from '~/components/common/AppNavigationBar.vue'
</script>

<style scoped lang="scss">
.app-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--color-bg-primary);
  margin: 0;
  padding: 0;
  position: relative; // 确保定位正确
}

.app-main {
  flex: 1;
  padding: 0; // 移除所有padding，让页面内容自己控制间距
  margin: 0; // 移除所有margin
  padding-bottom: 70px; // 仅保留底部导航栏空间
  
  // 确保内容从顶部开始
  > * {
    margin-top: 0;
  }
  
  // 第一个元素不应有顶部margin
  > *:first-child {
    margin-top: 0 !important;
    padding-top: 0;
  }
  
  @media (max-width: 768px) {
    padding-bottom: 60px; // 移动端稍小一些
  }
}

// 深色模式适配
@media (prefers-color-scheme: dark) {
  .app-layout {
    background: var(--color-bg-primary);
  }
}
</style>