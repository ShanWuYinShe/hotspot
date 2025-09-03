<template>
  <svg :class="svgClass" viewBox="0 0 200 140" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="empty groups illustration">
    <defs>
      <linearGradient id="gg" x1="0" x2="1" y1="0" y2="1">
        <stop offset="0%" stop-color="#fde68a" />
        <stop offset="100%" stop-color="#fef3c7" />
      </linearGradient>
    </defs>
    <rect x="0" y="0" width="200" height="140" rx="12" fill="url(#gg)" opacity="0.25" />
    <g transform="translate(20,26)">
      <rect x="0" y="0" width="160" height="88" rx="10" fill="#fff" opacity="0.9" />
      <circle cx="40" cy="40" r="16" fill="#F59E0B" opacity="0.15" />
      <circle cx="80" cy="40" r="16" fill="#10B981" opacity="0.15" />
      <circle cx="120" cy="40" r="16" fill="#6366F1" opacity="0.15" />
      <rect x="24" y="64" width="112" height="8" rx="4" fill="#E5E7EB" />
    </g>
  </svg>
</template>

<script setup lang="ts">
interface Props { size?: 'sm'|'md'|'lg' }
const props = withDefaults(defineProps<Props>(), { size: 'md' })
const svgClass = computed(() => ({ sm: 'w-32 h-24', md: 'w-48 h-36', lg: 'w-64 h-44' } as const)[props.size])
</script>
