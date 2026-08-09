<script lang="ts">
	import { X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	// SheetShell — 底部弹层统一外壳（handle + 标题栏 + 关闭按钮 + 内容槽）。
	// 供 CreateGroupSheet / NewConversationSheet 等全屏 bottom sheet 复用，消除逐字重复的外壳标记。

	let {
		title,
		onClose,
		children
	}: {
		title: string;
		onClose: () => void;
		children: Snippet;
	} = $props();
</script>

<div class="flex h-full flex-col bg-surface-primary">
	<!-- Handle -->
	<div class="flex shrink-0 justify-center pt-3 pb-1">
		<div class="rounded-full bg-border-default w-9 h-1" ></div>
	</div>

	<!-- Header -->
	<div class="flex shrink-0 items-center justify-between px-5 py-3 border-b border-border-default" >
		<h2 class="text-lg text-text-primary">{title}</h2>
		<button
			type="button"
			onclick={onClose}
			class="hover-btn scale-btn hit-slop flex items-center justify-center rounded-full border border-border-default bg-surface-primary size-8 shadow-raised-sm focus-ring"
			aria-label="关闭"
		>
			<X class="size-3.5 text-text-secondary"   strokeWidth={2} />
		</button>
	</div>

	{@render children()}
</div>
