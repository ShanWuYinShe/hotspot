<script lang="ts">
	/**
	 * 通用空状态组件
	 * 
	 * 用于显示列表为空时的统一UI组件
	 * 支持自定义图标、文案和操作按钮
	 */
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import type { ComponentType, Snippet } from 'svelte';

	interface Props {
		/** 图标组件 */
		icon: ComponentType;
		/** 标题 */
		title: string;
		/** 描述文字 */
		description?: string;
		/** 操作按钮文字 */
		actionText?: string;
		/** 操作按钮点击事件 */
		onAction?: () => void;
		/** 自定义CSS类 */
		class?: string;
		/** 自定义内容（通常用于按钮） */
		children?: Snippet;
	}

	let {
		icon: Icon,
		title,
		description,
		actionText,
		onAction,
		class: className = '',
		children
	}: Props = $props();
</script>

<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md {className}">
	<CardContent class="flex flex-col items-center justify-center py-16 text-center">
		<Icon class="mb-4 h-12 w-12 text-muted-foreground/50" />
		<h3 class="mb-2 text-lg font-medium">{title}</h3>
		{#if description}
			<p class="mb-4 text-sm text-muted-foreground">{description}</p>
		{/if}
		{#if actionText && onAction}
			<Button variant="outline" onclick={onAction}>
				{actionText}
			</Button>
		{/if}
		{@render children?.()}
	</CardContent>
</Card>
