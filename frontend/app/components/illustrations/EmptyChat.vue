<template>
  <svg :class="svgClass" viewBox="0 0 200 160" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="empty chat illustration">
    <defs>
      <linearGradient id="chatGradient" x1="0" x2="1" y1="0" y2="1">
        <stop offset="0%" stop-color="#ddd6fe" />
        <stop offset="100%" stop-color="#e0e7ff" />
      </linearGradient>
      <linearGradient id="bubbleGradient1" x1="0" x2="1" y1="0" y2="1">
        <stop offset="0%" stop-color="#8b5cf6" />
        <stop offset="100%" stop-color="#7c3aed" />
      </linearGradient>
      <linearGradient id="bubbleGradient2" x1="0" x2="1" y1="0" y2="1">
        <stop offset="0%" stop-color="#06b6d4" />
        <stop offset="100%" stop-color="#0891b2" />
      </linearGradient>
    </defs>
    
    <!-- 背景 -->
    <rect x="0" y="0" width="200" height="160" rx="16" fill="url(#chatGradient)" opacity="0.3" />
    
    <!-- 手机框架 -->
    <g transform="translate(50, 20)">
      <rect x="0" y="0" width="100" height="120" rx="12" fill="#fff" opacity="0.9" stroke="#e2e8f0" stroke-width="1"/>
      
      <!-- 聊天气泡 -->
      <g opacity="0.6">
        <!-- 左侧气泡 -->
        <rect x="8" y="20" width="40" height="16" rx="8" fill="url(#bubbleGradient1)" opacity="0.7"/>
        <rect x="8" y="40" width="32" height="16" rx="8" fill="url(#bubbleGradient1)" opacity="0.5"/>
        
        <!-- 右侧气泡 -->
        <rect x="52" y="60" width="40" height="16" rx="8" fill="url(#bubbleGradient2)" opacity="0.7"/>
        <rect x="60" y="80" width="32" height="16" rx="8" fill="url(#bubbleGradient2)" opacity="0.5"/>
      </g>
      
      <!-- 输入框 -->
      <rect x="8" y="100" width="84" height="12" rx="6" fill="#f1f5f9" stroke="#e2e8f0" stroke-width="1"/>
    </g>
    
    <!-- 装饰性元素 -->
    <circle cx="30" cy="40" r="3" fill="#8b5cf6" opacity="0.3">
      <animate attributeName="opacity" values="0.3;0.7;0.3" dur="2s" repeatCount="indefinite"/>
    </circle>
    <circle cx="170" cy="50" r="2" fill="#06b6d4" opacity="0.4">
      <animate attributeName="opacity" values="0.4;0.8;0.4" dur="2.5s" repeatCount="indefinite"/>
    </circle>
    <circle cx="25" cy="130" r="2.5" fill="#f59e0b" opacity="0.3">
      <animate attributeName="opacity" values="0.3;0.6;0.3" dur="3s" repeatCount="indefinite"/>
    </circle>
  </svg>
</template>

<script setup lang="ts">
interface Props { 
  size?: 'sm' | 'md' | 'lg' 
}

const props = withDefaults(defineProps<Props>(), { size: 'md' })

const svgClass = computed(() => ({
  sm: 'w-32 h-26',
  md: 'w-48 h-38',
  lg: 'w-64 h-51'
} as const)[props.size])
</script>
