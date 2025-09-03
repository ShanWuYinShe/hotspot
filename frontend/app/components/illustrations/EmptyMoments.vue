<template>
  <svg :class="svgClass" viewBox="0 0 200 140" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="empty moments illustration">
    <defs>
      <linearGradient id="gmo" x1="0" x2="1" y1="0" y2="1">
        <stop offset="0%" stop-color="#fbcfe8" />
        <stop offset="100%" stop-color="#cffafe" />
      </linearGradient>
    </defs>
    <rect x="0" y="0" width="200" height="140" rx="12" fill="url(#gmo)" opacity="0.25" />
    <g transform="translate(22,22)">
      <rect x="0" y="0" width="156" height="96" rx="10" fill="#fff" opacity="0.9" />
      <rect x="18" y="18" width="60" height="44" rx="6" fill="#DBEAFE" />
      <circle cx="48" cy="40" r="12" fill="#93C5FD" />
      <rect x="88" y="22" width="50" height="8" rx="4" fill="#E5E7EB" />
      <rect x="88" y="38" width="36" height="8" rx="4" fill="#E5E7EB" />
      <rect x="88" y="54" width="44" height="8" rx="4" fill="#E5E7EB" />
    </g>
  </svg>
</template>

<script setup lang="ts">
interface Props { size?: 'sm'|'md'|'lg' }
const props = withDefaults(defineProps<Props>(), { size: 'md' })
const svgClass = computed(() => ({ sm: 'w-32 h-24', md: 'w-48 h-36', lg: 'w-64 h-44' } as const)[props.size])
</script>
