<template>
  <button 
    v-if="shouldShowBackButton"
    @click="goBack" 
    :class="['pixel-back-btn', size]"
    v-tooltip="{ customText: tooltipText }"
  >
    <NIcon :size="iconSize">
      <ArrowBack />
    </NIcon>
  </button>
</template>

<script setup lang="ts">
import { NIcon } from 'naive-ui'
import { ArrowBack } from '@vicons/ionicons5'

interface Props {
  size?: 'small' | 'medium' | 'large'
  tooltipText?: string
}

const props = withDefaults(defineProps<Props>(), {
  size: 'medium',
  tooltipText: '返回上一页'
})

// 导入导航功能
const router = useRouter()

// 返回功能
const shouldShowBackButton = computed(() => {
  return router.currentRoute.value.path !== '/'
})

const goBack = () => {
  router.back()
}

// 根据尺寸计算图标大小
const iconSize = computed(() => {
  switch (props.size) {
    case 'small': return '16'
    case 'large': return '28'
    default: return '20'
  }
})
</script>

<style scoped>
/* 基础返回按钮样式 */
.pixel-back-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(64, 224, 208, 0.1) 0%, rgba(64, 224, 208, 0.2) 100%);
  border: 2px solid #40e0d0;
  border-radius: 8px;
  color: #40e0d0;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.pixel-back-btn:hover {
  background: linear-gradient(135deg, rgba(64, 224, 208, 0.2) 0%, rgba(64, 224, 208, 0.3) 100%);
  border-color: #5ff5dc;
  color: #5ff5dc;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(64, 224, 208, 0.3);
}

.pixel-back-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 6px rgba(64, 224, 208, 0.2);
}

.pixel-back-btn::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(45deg, transparent 30%, rgba(255, 255, 255, 0.1) 50%, transparent 70%);
  transform: translateX(-100%);
  transition: transform 0.3s ease;
}

.pixel-back-btn:hover::before {
  transform: translateX(100%);
}

/* 尺寸变体 */
.pixel-back-btn.small {
  width: 32px;
  height: 32px;
}

.pixel-back-btn.medium {
  width: 40px;
  height: 40px;
}

.pixel-back-btn.large {
  width: 48px;
  height: 48px;
}
</style>
