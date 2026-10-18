<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Users } from '@lucide/svelte';
	import type { NearbyUser } from './nearby-users';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import FollowButton from '@/components/ui/FollowButton.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';

	// 首页的「附近用户」行列表：ListContainer + UserCardRow
	// （在线角标 + #tag · 距离副标题 + 关注按钮）。

	let {
		users,
		isFollowed,
		isPending,
		onToggle,
		emptyHint,
		columns = 1
	}: {
		users: NearbyUser[];
		/** 该用户当前是否已关注（乐观态由父级的共享关注状态承载） */
		isFollowed: (id: string) => boolean;
		/** 关注请求进行中判断（按钮禁点防重复提交） */
		isPending?: (id: string) => boolean;
		onToggle: (id: string) => void;
		/** 空态补充说明（如「正在按「南天门」5km 范围筛选」），缺省不渲染。 */
		emptyHint?: string;
		/** 2 = 卡片双列网格（桌面右栏，r01：单列行过宽拉扯关注按钮距离）；缺省单列行列表 */
		columns?: 1 | 2;
	} = $props();
</script>

{#if users.length === 0}
	<!-- 空列表契约（skill hotspot-frontend-layout）：空态一律 EmptyState，
		 不渲染空 bordered 容器（线上反馈：空框 + 大片空白观感差）。 -->
	<EmptyState
		bare
		title="附近暂时没有匹配的用户"
		description={emptyHint}
	>
		{#snippet icon()}
			<Users size={24} color="var(--cst-text-secondary)" class="opacity-subtle" />
		{/snippet}
	</EmptyState>
{:else if columns === 2}
	<!-- 双列卡片网格（对齐 SectionFriends 好友网格模式）：桌面右栏用，单列行过宽 -->
	<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
		{#each users as u (u.id)}
			{@const followed = isFollowed(u.id)}
			<!-- 卡壳由 tone="card" 的 ListItem 承担；外层再包手写卡会造成双描边/双投影（卡片套卡片） -->
			<UserCardRow name={u.name} avatar={u.avatar} tone="card">
					{#snippet avatarBadge()}
						{#if u.isOnline}
							<OnlineDot size="md" />
						{/if}
					{/snippet}
					{#snippet subtitle()}
						{@const hasTag = Boolean(u.tag)}
						{@const hasDistance = Boolean(u.distance)}
						{#if hasTag || hasDistance}
							<p class="mt-1 flex items-center gap-1.5 text-xxs">
								{#if hasTag}<span class="text-accent-default">#{u.tag}</span>{/if}
								{#if hasTag && hasDistance}<span class="opacity-muted text-text-secondary">·</span>{/if}
								{#if hasDistance}<span class="text-text-secondary">{u.distance}</span>{/if}
							</p>
						{/if}
					{/snippet}
					{#snippet actions()}
						{@const pending = isPending?.(u.id) ?? false}
						<FollowButton active={followed} pending={pending} onToggle={() => onToggle(u.id)} />
					{/snippet}
			</UserCardRow>
		{/each}
	</div>
{:else}
		<ListContainer bordered>
	{#each users as u (u.id)}
		{@const followed = isFollowed(u.id)}
		<UserCardRow name={u.name} avatar={u.avatar}>
			{#snippet avatarBadge()}
				{#if u.isOnline}
					<OnlineDot size="md" />
				{/if}
			{/snippet}
			{#snippet subtitle()}
				<!-- tag/距离缺省时不渲染对应分段与分隔点，避免出现悬空的「#」或「·」 -->
				{@const hasTag = Boolean(u.tag)}
				{@const hasDistance = Boolean(u.distance)}
				{#if hasTag || hasDistance}
					<p class="mt-1 flex items-center gap-1.5 text-xxs">
						{#if hasTag}<span class="text-accent-default">#{u.tag}</span>{/if}
						{#if hasTag && hasDistance}<span class="opacity-muted text-text-secondary">·</span>{/if}
						{#if hasDistance}<span class="text-text-secondary">{u.distance}</span>{/if}
					</p>
				{/if}
			{/snippet}
			{#snippet actions()}
				{@const pending = isPending?.(u.id) ?? false}
				<FollowButton active={followed} pending={pending} onToggle={() => onToggle(u.id)} />
			{/snippet}
		</UserCardRow>
	{/each}
	</ListContainer>
{/if}
