<template>
  <svg :class="svgClass" viewBox="0 0 200 140" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="empty illustration">
    <defs>
      <linearGradient id="g1" x1="0" x2="1" y1="0" y2="1">
        <stop offset="0%" stop-color="#d1fae5" />
        <stop offset="100%" stop-color="#bfdbfe" />
      </linearGradient>
    </defs>
    <rect x="0" y="0" width="200" height="140" rx="12" fill="url(#g1)" opacity="0.25" />
    <g transform="translate(24,22)">
      <rect x="0" y="0" width="152" height="84" rx="10" fill="#fff" opacity="0.8" />
      <circle cx="24" cy="24" r="12" fill="#10B981" opacity="0.15" />
      <circle cx="48" cy="40" r="8" fill="#6366F1" opacity="0.15" />
      <circle cx="110" cy="30" r="10" fill="#F59E0B" opacity="0.15" />
      <rect x="20" y="58" width="112" height="8" rx="4" fill="#E5E7EB" />
      <rect x="40" y="72" width="72" height="6" rx="3" fill="#E5E7EB" />
    </g>
  </svg>
</template>

<script setup lang="ts">
interface Props { size?: 'sm'|'md'|'lg' }
const props = withDefaults(defineProps<Props>(), { size: 'md' })
const svgClass = computed(() => ({ sm: 'w-32 h-24', md: 'w-48 h-36', lg: 'w-64 h-44' } as const)[props.size])
</script>
