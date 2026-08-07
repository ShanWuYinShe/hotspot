<script lang="ts">
	import type { NearbyUser } from './nearby-users';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';

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
				<p class="mt-1 flex items-center gap-1.5 text-xxs">
					<span style="color:var(--cst-accent-default);">#{u.tag}</span>
					<span style="color:var(--cst-text-secondary);opacity:0.5;">·</span>
					<span style="color:var(--cst-text-secondary);">{u.distance}</span>
				</p>
			{/snippet}
			{#snippet actions()}
				<button
					type="button"
					class="hover-btn tap-93 px-3 py-1 rounded-full outline-none transition-all text-xs"
					style={followed
						? 'background:transparent;color:var(--cst-text-secondary);border:1px solid var(--cst-border-strong);'
						: 'background:var(--cst-accent-default);color:var(--cst-text-on-accent);border:1px solid transparent;'}
					onclick={() => onToggle(u.id)}
				>
					{followed ? '已关注' : '关注'}
				</button>
			{/snippet}
		</UserCardRow>
	{/each}
</ListContainer>
