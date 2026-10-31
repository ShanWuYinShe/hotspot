<script lang="ts">
	import type { Snippet } from 'svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import ListItem from '@/components/list/ListItem.svelte';

	// EntityRow —— 通用实体卡片行原语（用户/群组/成员/申请/邀请通用）：
	// - 统一基于 ListItem 外壳 + Avatar（ring + 可选角标/在线点）+ 名字 + 可选副信息/动作；
	// - avatarShape 区分实体类型：用户 circle（默认）/ 群组 rounded（圆角方头像）；
	// - tone: 'row'（标准列表行，Avatar 44）/ 'card'（独立卡片行，Avatar 48）；
	// - 动作按钮（FollowButton / JoinButton / Button / ActionIconButton）经 actions snippet 注入。

	let {
		name,
		avatar,
		tone = 'row',
		avatarShape = 'circle',
		avatarBadge,
		nameBadge,
		subtitle,
		meta,
		thumbnail,
		actions,
		onClick,
		ariaLabel,
		style,
		class: className = ''
	}: {
		name: string;
		avatar: string;
		tone?: 'row' | 'card';
		avatarShape?: 'circle' | 'rounded';
		avatarBadge?: Snippet;
		nameBadge?: Snippet;
		subtitle?: Snippet;
		meta?: Snippet;
		thumbnail?: Snippet;
		actions?: Snippet;
		onClick?: () => void;
		ariaLabel?: string;
		style?: string;
		class?: string;
	} = $props();
</script>

<ListItem
	tone={tone === 'card' ? 'card' : undefined}
	class="flex items-center gap-3 {tone === 'card' ? 'p-3.5' : 'p-3'} {className}"
	{onClick}
	ariaLabel={ariaLabel ?? name}
	{style}
>
	<Avatar ring shape={avatarShape} src={avatar} name={name} size={tone === 'card' ? 48 : 44}>
		{#snippet badge()}
			{#if avatarBadge}
				{@render avatarBadge()}
			{/if}
		{/snippet}
	</Avatar>
	<div class="min-w-0 flex-1">
		<div class="flex items-center gap-2">
			<span class="text-text-primary text-sm font-medium">{name}</span>
			{#if nameBadge}
				{@render nameBadge()}
			{/if}
		</div>
		{#if subtitle}
			{@render subtitle()}
		{/if}
	</div>
	{#if meta}
		<span class="flex shrink-0 items-center">{@render meta()}</span>
	{/if}
	{#if thumbnail}
		{@render thumbnail()}
	{/if}
	{#if actions}
		{@render actions()}
	{/if}
</ListItem>
