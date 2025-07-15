<template>
  <div class="app-root">
    <NMessageProvider>
      <NNotificationProvider>
        <NDialogProvider>
          <NuxtLayout>
            <NuxtPage />
          </NuxtLayout>
        </NDialogProvider>
      </NNotificationProvider>
    </NMessageProvider>
    <NuxtRouteAnnouncer />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { NMessageProvider, NNotificationProvider, NDialogProvider } from 'naive-ui'

// 初始化多用户系统
const multiUserSystem = useMultiUserSystem()

// 应用启动时初始化多用户系统
onMounted(async () => {
  try {
    // 1. 初始化多用户系统
    await multiUserSystem.initializeSystem()
    
    // 2. 设置路由守卫
    multiUserSystem.setupRouteGuards()
    
    // 3. 启动升级引导系统
    multiUserSystem.checkUpgradeOpportunities()
    
    console.log('✅ 多用户系统初始化完成')
  } catch (error) {
    console.error('❌ 多用户系统初始化失败:', error)
  }
})
</script>

<style>
/* 全局样式重置，确保页面占满全屏 */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 0;
  overflow-x: hidden;
}

#__nuxt {
  width: 100%;
  min-height: 100vh;
  margin: 0;
  padding: 0;
}

.app-root {
  width: 100%;
  min-height: 100vh;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
}
</style>
