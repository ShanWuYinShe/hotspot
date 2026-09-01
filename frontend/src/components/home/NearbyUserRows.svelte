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
		onToggle
	}: {
		users: NearbyUser[];
		followingIds: Set<string>;
		onToggle: (id: string) => void;
	} = $props();
</script>

<ListContainer bordered>
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
				<Button
					type="button"
					variant={followed ? 'outline' : 'primary'}
					size="sm"
					class="hit-slop rounded-full px-3 py-1 {followed ? 'border-border-strong text-text-secondary' : ''}"
					onclick={() => onToggle(u.id)}
				>
					{followed ? '已关注' : '关注'}
				</Button>
			{/snippet}
		</UserCardRow>
	{/each}
</ListContainer>
