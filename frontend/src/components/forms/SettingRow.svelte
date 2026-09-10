<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';
	import ListItem from '@/components/list/ListItem.svelte';

	// SettingRow — 设置行（对齐 Figma Make SettingsPage）：
	// 左侧 icon-badge 圆形容器 + 标题/副标题 + 右侧控件（Toggle/按钮/箭头等），
	// 可选下方扩展区（extension：主题选择 grid 等换行内容，属于该行的一部分，
	// 随行 hover 一起高亮）。
	// 行交互统一走 ListItem 原语（hover 填充 / 键盘 / aria 同全站行语言），
	// 本组件只负责设置行的内容编排（此前为手写 hover-bg 行，近重复已收敛）。

	let {
		title,
		description,
		badge,
		icon,
		onRowClick,
		children,
		extension,
		class: className = ''
	}: {
		title: string;
		description?: string;
		/** 标题右侧小标（如「仅本机」），用于标注该项的生效范围/状态，缺省不渲染。 */
		badge?: Snippet;
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

<ListItem onClick={onRowClick} class={cn('px-4 py-3.5', className)}>
	<div class="flex items-center gap-3">
		<div class="flex items-center justify-center rounded-full shrink-0 icon-badge size-9">
			{@render icon()}
		</div>
		<div class="flex-1 min-w-0">
			<div class="flex items-center gap-2">
				<p class="text-sm text-text-primary">{title}</p>
				{#if badge}
					{@render badge()}
				{/if}
			</div>
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
</ListItem>
