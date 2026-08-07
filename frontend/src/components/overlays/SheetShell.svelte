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
			class="hover-btn scale-btn flex items-center justify-center rounded-full bg-surface-primary outline-none shadow-raised-sm"
			style="width:32px;height:32px;border:1px solid var(--cst-border-default);"
		>
			<X class="size-3.5"  style="color:var(--cst-text-secondary)" strokeWidth={2} />
		</button>
	</div>

	{@render children()}
</div>
