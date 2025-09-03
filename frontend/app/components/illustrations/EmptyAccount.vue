<template>
  <svg :class="svgClass" viewBox="0 0 200 140" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="empty account illustration">
    <defs>
      <linearGradient id="ga" x1="0" x2="1" y1="0" y2="1">
        <stop offset="0%" stop-color="#cffafe" />
        <stop offset="100%" stop-color="#e9d5ff" />
      </linearGradient>
    </defs>
    <rect x="0" y="0" width="200" height="140" rx="12" fill="url(#ga)" opacity="0.25" />
    <g transform="translate(24,22)">
      <rect x="0" y="0" width="152" height="96" rx="10" fill="#fff" opacity="0.9" />
      <!-- avatar circle -->
      <circle cx="40" cy="40" r="16" fill="#A78BFA" opacity="0.2" />
      <circle cx="40" cy="36" r="6" fill="#A78BFA" opacity="0.35" />
      <rect x="30" y="48" width="20" height="8" rx="4" fill="#A78BFA" opacity="0.25" />
      <!-- text lines -->
      <rect x="70" y="26" width="60" height="8" rx="4" fill="#E5E7EB" />
      <rect x="70" y="42" width="44" height="8" rx="4" fill="#E5E7EB" />
      <rect x="70" y="58" width="52" height="8" rx="4" fill="#E5E7EB" />
    </g>
  </svg>
</template>

<script setup lang="ts">
interface Props { size?: 'sm'|'md'|'lg' }
const props = withDefaults(defineProps<Props>(), { size: 'md' })
const svgClass = computed(() => ({ sm: 'w-32 h-24', md: 'w-48 h-36', lg: 'w-64 h-44' } as const)[props.size])
</script>
