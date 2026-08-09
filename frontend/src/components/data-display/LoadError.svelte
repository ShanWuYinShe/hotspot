<script lang="ts">
	import EmptyState from './EmptyState.svelte';
	import Button from '@/components/ui/button.svelte';

	// LoadError — 数据加载失败统一状态（标题/描述/重试按钮）。
	// 替代各页面手写「静默转空态」：失败时展示原因并提供重试，不再伪装成“暂无内容”。

	let { message = '加载失败，请稍后重试', onRetry }: { message?: string; onRetry?: () => void } = $props();
</script>

<EmptyState bare title="加载失败" description={message}>
	{#snippet action()}
		{#if onRetry}
			<Button type="button" variant="secondary" size="sm" onclick={onRetry}>重试</Button>
		{/if}
	{/snippet}
</EmptyState>
