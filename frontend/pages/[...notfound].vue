<template>
  <div class="error-page">
    <!-- 游戏背景层 -->
    <div class="game-background">
      <div class="game-bg-pattern"></div>
      <div class="game-bg-stars"></div>
      <div class="game-bg-glow"></div>
    </div>
    
    <div class="error-container">
      <div class="error-content">
        <div class="game-panel game-panel-epic">
          <div class="error-icon">
            <div class="icon-frame">
              <Icon name="mdi:map-marker-question" class="error-icon-svg" />
              <div class="icon-glow"></div>
            </div>
          </div>
          
          <div class="error-text">
            <h1 class="error-code game-text-enhanced">404</h1>
            <h2 class="error-title game-text-enhanced">迷失在虚空中</h2>
            <p class="error-desc game-text-enhanced">
              你似乎走入了一个未知的领域。<br>
              这个传送门已经失效或者从未存在过。
            </p>
          </div>
          
          <div class="action-buttons">
            <button 
              class="game-button game-button-primary"
              @click="goHome"
            >
              <Icon name="mdi:home" class="w-5 h-5 mr-2" />
              <span class="game-text-enhanced">返回主城</span>
            </button>
            <button 
              v-if="pageState.canGoBack"
              class="game-button game-button-secondary"
              @click="goBack"
            >
              <Icon name="mdi:arrow-left" class="w-5 h-5 mr-2" />
              <span class="game-text-enhanced">返回上一个位置</span>
            </button>
          </div>
          
          <div class="error-tips game-panel game-panel-rare p-4 mt-6">
            <div class="tip-header">
              <Icon name="mdi:lightbulb" class="w-5 h-5 text-game-rare" />
              <span class="game-text-enhanced">探索提示</span>
            </div>
            <div class="tip-content">
              <p class="game-text-enhanced">
                检查URL拼写，或使用导航菜单探索其他区域。
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// Vue 相关导入
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'

// Composables 导入
// 导入组件

// TypeScript 接口定义
interface PageState {
  canGoBack: boolean
}

// 页面配置
definePageMeta({
  layout: false  // 禁用默认布局，使用自定义背景
})

// 路由管理
const router = useRouter()

// 页面状态管理
const pageState = ref<PageState>({
  canGoBack: false
})

// 页面加载时检查历史记录
onMounted(() => {
  pageState.value.canGoBack = window.history.length > 1
})

// 事件处理函数
const goHome = (): void => {
  router.push('/')
}

const goBack = (): void => {
  if (pageState.value.canGoBack) {
    router.go(-1)
  } else {
    goHome()
  }
}
</script>

<style scoped lang="scss">
.error-page {
  min-height: 100vh;
  background: var(--game-bg-primary);
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 游戏背景 */
.game-background {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: var(--z-base);
  pointer-events: none;
}

.game-bg-pattern {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: 
    radial-gradient(circle at 30% 40%, rgba(255, 107, 53, 0.15) 0%, transparent 50%),
    radial-gradient(circle at 70% 60%, rgba(138, 43, 226, 0.15) 0%, transparent 50%),
    radial-gradient(circle at 50% 50%, rgba(255, 215, 0, 0.1) 0%, transparent 70%);
  background-size: 800px 800px, 600px 600px, 1000px 1000px;
  animation: backgroundMove 30s ease-in-out infinite;
}

.game-bg-stars {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: 
    radial-gradient(2px 2px at 20px 30px, rgba(255, 255, 255, 0.3), transparent),
    radial-gradient(2px 2px at 40px 70px, rgba(255, 107, 53, 0.4), transparent),
    radial-gradient(1px 1px at 90px 40px, rgba(138, 43, 226, 0.4), transparent);
  background-repeat: repeat;
  background-size: 200px 100px;
  animation: starTwinkle 4s ease-in-out infinite alternate;
}

.game-bg-glow {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: radial-gradient(ellipse at center, rgba(138, 43, 226, 0.1) 0%, transparent 70%);
  animation: glow 6s ease-in-out infinite alternate;
}

@keyframes backgroundMove {
  0%, 100% { transform: translateX(0) translateY(0); }
  25% { transform: translateX(-15px) translateY(-10px); }
  50% { transform: translateX(15px) translateY(-5px); }
  75% { transform: translateX(-10px) translateY(15px); }
}

@keyframes starTwinkle {
  0% { opacity: 0.3; }
  100% { opacity: 0.8; }
}

@keyframes glow {
  0% { opacity: 0.5; }
  100% { opacity: 0.8; }
}

/* 错误页面容器 */
.error-container {
  position: relative;
  z-index: var(--z-content);
  padding: 2rem;
  width: 100%;
  max-width: 600px;
  margin: 0 auto;
}

.error-content {
  .game-panel {
    padding: 3rem 2rem;
    text-align: center;
    position: relative;
  }
}

/* 错误图标 */
.error-icon {
  margin-bottom: 2rem;
  
  .icon-frame {
    position: relative;
    display: inline-block;
    
    .error-icon-svg {
      width: 120px;
      height: 120px;
      color: var(--game-epic);
      position: relative;
      z-index: var(--z-elevated);
      animation: float 3s ease-in-out infinite;
    }
    
    .icon-glow {
      position: absolute;
      top: -20px;
      left: -20px;
      width: calc(100% + 40px);
      height: calc(100% + 40px);
      background: radial-gradient(circle, rgba(138, 43, 226, 0.3) 0%, transparent 70%);
      border-radius: 50%;
      z-index: var(--z-content);
      animation: pulse 2s ease-in-out infinite;
    }
  }
}

/* 错误文本 */
.error-text {
  margin-bottom: 2rem;
}

.error-code {
  font-size: 4rem;
  font-weight: bold;
  margin: 0 0 1rem 0;
  color: var(--game-epic);
  text-shadow: 0 0 20px rgba(138, 43, 226, 0.5);
  animation: pulse 3s ease-in-out infinite;
}

.error-title {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0 0 1rem 0;
  color: var(--game-text-primary);
}

.error-desc {
  font-size: 1rem;
  line-height: 1.6;
  color: var(--game-text-secondary);
  margin: 0;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 1rem;
  justify-content: center;
  flex-wrap: wrap;
  margin-bottom: 2rem;
}

/* 错误提示 */
.error-tips {
  text-align: left;
  
  .tip-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
    font-weight: 600;
  }
  
  .tip-content {
    p {
      margin: 0;
      font-size: 0.9rem;
      line-height: 1.4;
    }
  }
}

/* 动画效果 */
@keyframes float {
  0%, 100% { transform: translateY(0px); }
  50% { transform: translateY(-10px); }
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.8; transform: scale(1.05); }
}

/* 响应式设计 */
@media (max-width: 768px) {
  .error-container {
    padding: 1rem;
  }
  
  .error-content .game-panel {
    padding: 2rem 1rem;
  }
  
  .error-icon .icon-frame .error-icon-svg {
    width: 80px;
    height: 80px;
  }
  
  .error-code {
    font-size: 3rem;
  }
  
  .error-title {
    font-size: 1.25rem;
  }
  
  .error-desc {
    font-size: 0.9rem;
  }
  
  .action-buttons {
    flex-direction: column;
    gap: 0.5rem;
  }
  
  .game-button {
    width: 100%;
    justify-content: center;
  }
}
</style>

