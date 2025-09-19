<template>
  <svg :class="svgClass" viewBox="0 0 200 140" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="empty search illustration">
    <defs>
      <linearGradient id="searchGradient" x1="0" x2="1" y1="0" y2="1">
        <stop offset="0%" stop-color="#fef3c7" />
        <stop offset="100%" stop-color="#fde68a" />
      </linearGradient>
      <linearGradient id="magnifierGradient" x1="0" x2="1" y1="0" y2="1">
        <stop offset="0%" stop-color="#f59e0b" />
        <stop offset="100%" stop-color="#d97706" />
      </linearGradient>
    </defs>
    
    <!-- 背景 -->
    <rect x="0" y="0" width="200" height="140" rx="16" fill="url(#searchGradient)" opacity="0.3" />
    
    <!-- 搜索框 -->
    <g transform="translate(50, 40)">
      <rect x="0" y="0" width="100" height="20" rx="10" fill="#fff" opacity="0.9" stroke="#e2e8f0" stroke-width="1"/>
      <circle cx="15" cy="10" r="3" fill="#9ca3af" opacity="0.6"/>
      <rect x="25" y="8" width="60" height="4" rx="2" fill="#e5e7eb" opacity="0.5"/>
    </g>
    
    <!-- 放大镜 -->
    <g transform="translate(90, 80)">
      <circle cx="0" cy="0" r="12" fill="none" stroke="url(#magnifierGradient)" stroke-width="3" opacity="0.7">
        <animate attributeName="stroke-width" values="3;4;3" dur="2s" repeatCount="indefinite"/>
      </circle>
      <line x1="8.5" y1="8.5" x2="16" y2="16" stroke="url(#magnifierGradient)" stroke-width="3" stroke-linecap="round" opacity="0.7">
        <animate attributeName="stroke-width" values="3;4;3" dur="2s" repeatCount="indefinite"/>
      </line>
    </g>
    
    <!-- 搜索结果占位符 -->
    <g transform="translate(30, 90)" opacity="0.3">
      <rect x="0" y="0" width="140" height="8" rx="4" fill="#d1d5db"/>
      <rect x="0" y="12" width="120" height="8" rx="4" fill="#d1d5db"/>
      <rect x="0" y="24" width="100" height="8" rx="4" fill="#d1d5db"/>
    </g>
    
    <!-- 装饰性搜索图标 -->
    <g opacity="0.2">
      <circle cx="30" cy="30" r="2" fill="#f59e0b">
        <animate attributeName="opacity" values="0.2;0.5;0.2" dur="3s" repeatCount="indefinite"/>
      </circle>
      <circle cx="170" cy="35" r="1.5" fill="#d97706">
        <animate attributeName="opacity" values="0.2;0.4;0.2" dur="2.5s" repeatCount="indefinite"/>
      </circle>
      <circle cx="25" cy="120" r="2.5" fill="#fbbf24">
        <animate attributeName="opacity" values="0.2;0.6;0.2" dur="4s" repeatCount="indefinite"/>
      </circle>
    </g>
  </svg>
</template>

<script setup lang="ts">
interface Props { 
  size?: 'sm' | 'md' | 'lg' 
}

const props = withDefaults(defineProps<Props>(), { size: 'md' })

const svgClass = computed(() => ({
  sm: 'w-32 h-22',
  md: 'w-48 h-34',
  lg: 'w-64 h-45'
} as const)[props.size])
</script>
