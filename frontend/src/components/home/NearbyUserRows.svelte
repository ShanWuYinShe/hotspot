<script lang="ts">
	import type { NearbyUser } from './nearby-users';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import Button from '@/components/ui/button.svelte';

	// 首页的「附近用户」行列表：ListContainer + UserCardRow
	// （在线角标 + #tag · 距离副标题 + 关注按钮）。

	let {
		users,
		followingIds,
		followPending,
		onToggle
	}: {
		users: NearbyUser[];
		followingIds: Set<string>;
		/** 关注请求进行中的用户 id（按钮禁点防重复提交） */
		followPending?: Set<string>;
		onToggle: (id: string) => void;
	} = $props();
</script>

<ListContainer bordered={false} class="divide-y divide-border-subtle/30">
	{#each users as u (u.id)}
		{@const followed = followingIds.has(u.id)}
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
				{@const pending = followPending?.has(u.id) ?? false}
				<!-- 双态 hover 差异（四态契约）：未关注 = accent 光晕抬档；已关注 = 描边加深 + 淡底填充 -->
				<Button
					type="button"
					variant={followed ? 'outline' : 'primary'}
					size="sm"
					aria-pressed={followed}
					disabled={pending}
					class="hit-slop rounded-full px-3 py-1 {followed
						? 'border-border-strong text-text-secondary hover:border-accent-border hover:bg-interactive-hover hover:text-text-primary'
						: ''}"
					onclick={() => onToggle(u.id)}
				>
					{followed ? '已关注' : '关注'}
				</Button>
			{/snippet}
		</UserCardRow>
	{/each}
</ListContainer>
