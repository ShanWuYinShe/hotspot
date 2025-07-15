<!-- 实时连接状态组件 -->
<template>
  <div v-if="!isConnected" class="connection-status">
    <div class="connection-status-content">
      <NIcon class="status-icon"><Warning /></NIcon>
      <span class="status-text">{{ statusMessage }}</span>
      <NButton
        v-if="canReconnect"
        size="small"
        @click="handleReconnect"
        class="reconnect-btn"
      >
        重新连接
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NIcon, NButton } from 'naive-ui'
import { Warning } from '@vicons/ionicons5'

interface Props {
  isConnected?: boolean
  statusMessage?: string
  canReconnect?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  isConnected: true,
  statusMessage: '连接已断开，请检查网络',
  canReconnect: true
})

const emit = defineEmits<{
  reconnect: []
}>()

// 重新连接
const handleReconnect = () => {
  emit('reconnect')
}
</script>

<style scoped lang="scss">
.connection-status {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: var(--z-notification);
  background-color: rgba(250, 173, 20, 0.9);
  padding: 8px 16px;
  color: #fff;
  transition: all 0.3s ease;
  
  .connection-status-content {
    display: flex;
    align-items: center;
    justify-content: center;
    max-width: 1200px;
    margin: 0 auto;
    
    .status-icon {
      margin-right: 8px;
      font-size: 18px;
    }
    
    .status-text {
      flex: 1;
      font-size: 14px;
    }
    
    .reconnect-btn {
      margin-left: 16px;
      background-color: rgba(255, 255, 255, 0.2);
      border: none;
      color: white;
      
      &:hover {
        background-color: rgba(255, 255, 255, 0.3);
      }
    }
  }
}

@media (max-width: 768px) {
  .connection-status {
    padding: 6px 12px;
    
    .connection-status-content {
      .status-text {
        font-size: 12px;
      }
    }
  }
}
</style>
