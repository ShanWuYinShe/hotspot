<script lang="ts">
	import { goto } from '$app/navigation';
	import { UserRound } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Button from '@/components/ui/button.svelte';

	// NeedsAuthNotice — 未登录占位（区别于 LoadError 的错误态）。
	// 10100（未登录/会话失效）是正常业务态：重试无意义，正确动作是引导登录。
	// 与 requireAuth() 弹层互补：弹层拦截「写操作」，本组件承接「整页/分区只读内容」。
	// spotlight：透传 EmptyState 的场景化空态装饰层（整页/整块内容的访客引导启用；
	// 窄面板内不启用，保持克制）。
	let {
		title = '登录后查看',
		description = '登录后即可查看你的内容与关系',
		spotlight = false,
		class: className = ''
	}: {
		title?: string;
		description?: string;
		spotlight?: boolean;
		class?: string;
	} = $props();
</script>

<EmptyState bare {title} {description} {spotlight} class={className}>
	{#snippet icon()}
		<UserRound class="text-accent-default" size={64} strokeWidth={1.5} />
	{/snippet}
	{#snippet action()}
		<Button type="button" size="md" onclick={() => void goto('/login')}>去登录</Button>
	{/snippet}
</EmptyState>
