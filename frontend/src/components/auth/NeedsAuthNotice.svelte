<script lang="ts">
	import type { Component } from 'svelte';
	import { goto } from '$app/navigation';
	import { UserRound } from '@lucide/svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Button from '@/components/ui/button.svelte';
	import { ROUTES } from '@/lib/routes';

	// NeedsAuthNotice — 未登录占位（区别于 LoadError 的错误态）。
	// 10100（未登录/会话失效）是正常业务态：重试无意义，正确动作是引导登录。
	// 与 requireAuth() 弹层互补：弹层拦截「写操作」，本组件承接「整页/分区只读内容」。
	//
	// R13：从「一个图标 + 两行字 + 去登录」升级为访客引导面板——
	// 图标软底 + 标题 + 描述 + 3 条价值点（页面按语境传入）+ 主 CTA（登录/注册）+
	// 次 CTA（先随便逛逛，去首页/附近）。整页级空态默认开启 spotlight 装饰层。
	let {
		title = '登录后查看',
		description = '登录后即可查看你的内容与关系',
		highlights = [],
		primaryLabel = '登录 / 注册',
		secondaryLabel = '先随便逛逛',
		onSecondary,
		spotlight = true,
		class: className = ''
	}: {
		title?: string;
		description?: string;
		/** 3 条以内的价值点（icon + 文案），用于说明"登录后能得到什么"。 */
		highlights?: { icon: Component; text: string }[];
		primaryLabel?: string;
		secondaryLabel?: string;
		/** 次 CTA 行为；不传则回首页（访客可浏览的公共内容）。 */
		onSecondary?: () => void;
		spotlight?: boolean;
		class?: string;
	} = $props();

	function goSecondary() {
		if (onSecondary) onSecondary();
		else void goto(ROUTES.home);
	}
</script>

<EmptyState {spotlight} iconTone="accent" {title} {description} class={className}>
	{#snippet icon()}
		<UserRound class="size-8" strokeWidth={1.5} />
	{/snippet}
	{#snippet extra()}
		{#if highlights.length > 0}
			<ul class="mt-1 flex w-full max-w-90 flex-col gap-2 px-1 text-left">
				{#each highlights as item (item.text)}
					<li class="flex items-center gap-2.5">
						<span
							class="flex size-8 shrink-0 items-center justify-center rounded-full bg-surface-island text-accent-default shadow-xs"
						>
							<item.icon class="size-4" strokeWidth={1.75} />
						</span>
						<span class="text-xs leading-relaxed text-text-secondary">{item.text}</span>
					</li>
				{/each}
			</ul>
		{/if}
	{/snippet}
	{#snippet action()}
		<div class="mt-2 flex flex-wrap items-center justify-center gap-2.5">
			<Button type="button" size="md" onclick={() => void goto(ROUTES.login)}>{primaryLabel}</Button>
			<Button type="button" size="md" variant="ghost" onclick={goSecondary}>{secondaryLabel}</Button>
		</div>
	{/snippet}
</EmptyState>
