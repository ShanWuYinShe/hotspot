<script lang="ts">
	import { ImageOff } from '@lucide/svelte';

	// 无图媒体占位：占满父容器，主题化底色 + 图标，替代 <img src=""> 的破图。
	// 父容器需自带尺寸与圆角（aspect-* / size-*），本组件 stretch 填充。
	let { iconSize = 20, label = '' }: { iconSize?: number; label?: string } = $props();
</script>

<div class="flex h-full w-full flex-col items-center justify-center gap-1.5 bg-accent-soft" aria-hidden={!label}>
	<ImageOff size={iconSize} class="text-accent-default" strokeWidth={1.5} />
	{#if label}
		<span class="text-xxs text-text-secondary">{label}</span>
	{/if}
</div>
