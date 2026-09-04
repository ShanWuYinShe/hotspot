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
		onRowClick,
		children,
		extension,
		class: className = ''
	}: {
		title: string;
		description?: string;
		/** 左侧图标（放入 size-9 icon-badge 圆形容器）。 */
		icon: Snippet;
		/** 整行点击代理（如行点击即切换右侧 Toggle，M3 设置行规范）；缺省为纯展示行。 */
		onRowClick?: () => void;
		/** 右侧控件区（Toggle/下拉/箭头等）。 */
		children?: Snippet;
		/** 行下方扩展区（换行内容，如主题选择 grid）。 */
		extension?: Snippet;
		class?: string;
	} = $props();
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_to_interactive_role, a11y_interactive_supports_focus, a11y_no_noninteractive_tabindex -->
<div
	class={cn('px-4 py-3.5', onRowClick && 'hover-bg cursor-pointer', className)}
	role={onRowClick ? 'button' : undefined}
	tabindex={onRowClick ? 0 : undefined}
	onclick={onRowClick}
	onkeydown={(e) => {
		if (!onRowClick) return;
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onRowClick();
		}
	}}
>
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
			<!-- 行点击代理存在时，右侧控件的点击不冒泡（避免 Toggle 双切换）。
			     纯事件隔离容器，无交互语义（键盘事件由行与控件各自处理）。 -->
			<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
			<span class="flex shrink-0 items-center" onclick={onRowClick ? (e) => e.stopPropagation() : undefined}>
				{@render children()}
			</span>
		{/if}
	</div>
	{#if extension}
		<div class="mt-3">{@render extension()}</div>
	{/if}
</div>
