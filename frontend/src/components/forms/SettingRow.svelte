<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// SettingRow — 设置行（对齐 Figma Make SettingsPage）：
	// 左侧 icon-badge 圆形容器 + 标题/副标题 + 右侧控件（Toggle/按钮/箭头等），
	// 可选下方扩展区（extension：主题选择 grid 等换行内容）。
	// 消除 settings 页 9 处手写重复（icon-badge + flex + title + desc 结构）。

	let {
		title,
		description,
		icon,
		children,
		extension,
		class: className = ''
	}: {
		title: string;
		description?: string;
		/** 左侧图标（放入 size-9 icon-badge 圆形容器）。 */
		icon: Snippet;
		/** 右侧控件区（Toggle/下拉/箭头等）。 */
		children?: Snippet;
		/** 行下方扩展区（换行内容，如主题选择 grid）。 */
		extension?: Snippet;
		class?: string;
	} = $props();
</script>

<div class={cn('px-4 py-3.5', className)}>
	<div class="flex items-center gap-3">
		<div class="flex items-center justify-center rounded-full shrink-0 icon-badge size-9">
			{@render icon()}
		</div>
		<div class="flex-1 min-w-0">
			<p class="text-sm text-text-primary">{title}</p>
			{#if description}
				<p class="text-xs text-text-secondary">{description}</p>
			{/if}
		</div>
		{#if children}
			{@render children()}
		{/if}
	</div>
	{#if extension}
		<div class="mt-3">{@render extension()}</div>
	{/if}
</div>
