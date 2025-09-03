<template>
  <svg :class="svgClass" viewBox="0 0 200 140" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="empty messages illustration">
    <defs>
      <linearGradient id="gm" x1="0" x2="1" y1="0" y2="1">
        <stop offset="0%" stop-color="#e0e7ff" />
        <stop offset="100%" stop-color="#dbeafe" />
      </linearGradient>
    </defs>
    <rect x="0" y="0" width="200" height="140" rx="12" fill="url(#gm)" opacity="0.25" />
    <g transform="translate(22,24)">
      <rect x="0" y="0" width="156" height="92" rx="10" fill="#fff" opacity="0.9" />
      <rect x="16" y="18" width="88" height="10" rx="5" fill="#E5E7EB" />
      <rect x="16" y="36" width="120" height="10" rx="5" fill="#E5E7EB" />
      <rect x="16" y="54" width="96" height="10" rx="5" fill="#E5E7EB" />
      <circle cx="132" cy="20" r="6" fill="#6366F1" opacity="0.2" />
      <circle cx="144" cy="20" r="6" fill="#10B981" opacity="0.2" />
    </g>
  </svg>
  </template>

<script setup lang="ts">
interface Props { size?: 'sm'|'md'|'lg' }
const props = withDefaults(defineProps<Props>(), { size: 'md' })
const svgClass = computed(() => ({ sm: 'w-32 h-24', md: 'w-48 h-36', lg: 'w-64 h-44' } as const)[props.size])
</script>
