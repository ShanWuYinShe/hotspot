<script lang="ts">
	import type { Snippet } from 'svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import ListItem from '@/components/list/ListItem.svelte';

	// UserCardRow — 用户卡片行（好友/关注/粉丝列表共用）。
	// 统一：ListItem 外壳 + Avatar（ring + 可选角标）+ 名字 + 可选徽章。
	// 副信息（handle/desc/时间）、缩略图、动作按钮由调用方 snippet 注入。
	// tone: 'row'（List bordered 内，avatar 44）/ 'card'（embedded 网格，avatar 48）。

	let {
		name,
		avatar,
		tone = 'row',
		avatarBadge,
		nameBadge,
		subtitle,
		thumbnail,
		actions,
		onClick,
		ariaLabel,
		style
	}: {
		name: string;
		avatar: string;
		tone?: 'row' | 'card';
		avatarBadge?: Snippet;
		nameBadge?: Snippet;
		subtitle?: Snippet;
		thumbnail?: Snippet;
		actions?: Snippet;
		/** 整行点击（不传则行不可点击，仅行内动作交互，如附近用户行的关注按钮）。 */
		onClick?: () => void;
		ariaLabel?: string;
		style?: string;
	} = $props();
</script>

<ListItem
	tone={tone === 'card' ? 'card' : undefined}
	class="flex items-center gap-3 {tone === 'card' ? 'p-3.5' : 'p-3'}"
	{onClick}
	{ariaLabel}
	{style}
>
	<Avatar ring src={avatar} name={name} size={tone === 'card' ? 48 : 44}>
		{#snippet badge()}
			{#if avatarBadge}
				{@render avatarBadge()}
			{/if}
		{/snippet}
	</Avatar>
	<div class="min-w-0 flex-1">
		<div class="flex items-center gap-2">
			<span class="text-text-primary text-sm">{name}</span>
			{#if nameBadge}
				{@render nameBadge()}
			{/if}
		</div>
		{#if subtitle}
			{@render subtitle()}
		{/if}
	</div>
	{#if thumbnail}
		{@render thumbnail()}
	{/if}
	{#if actions}
		{@render actions()}
	{/if}
</ListItem>
