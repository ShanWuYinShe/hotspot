<script lang="ts">
	import { goto } from '$app/navigation';
	import { UserPlus, Check, Heart } from '@lucide/svelte';
	import ProfileUserSection from '@/components/profile/ProfileUserSection.svelte';
	import type { SectionFilter } from '@/components/profile/section-filter';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 关注分区：ProfileUserSection 骨架 + UserCardRow 行。

	let { embedded = false }: { embedded?: boolean } = $props();

	let followingFilter = $state<'all' | 'mutual'>('all');
	let unfollowedIds = $state<Set<string>>(new Set());

	const { following } = useRelationships();

	const toggleUnfollow = (id: string) => {
		const next = new Set(unfollowedIds);
		next.has(id) ? next.delete(id) : next.add(id);
		unfollowedIds = next;
	};

	const filtered = $derived(followingFilter === 'mutual' ? following.filter((f) => f.isMutual) : following);

	const FILTERS: SectionFilter[] = [
		{ id: 'all', label: '全部' },
		{ id: 'mutual', label: '互关' }
	];
</script>

<ProfileUserSection
	title="关注"
	subtitle="你主动订阅的动态与创作者"
	count={following.length}
	countSuffix=" 人"
	headerNote="点击卡片查看主页"
	{embedded}
	filters={FILTERS}
	filterValue={followingFilter}
	onFilter={(id) => (followingFilter = id as 'all' | 'mutual')}
	emptyTitle="还没有关注"
	emptyDescription="去发现附近的人，找到有趣的创作者。"
	emptyIcon={Heart}
	items={filtered}
>
	{#snippet children(f)}
		<UserCardRow
			name={f.name}
			avatar={f.avatar}
			tone={embedded ? 'card' : 'row'}
			onClick={() => void goto(`${ROUTES.user(f.id)}?from=relationships`)}
			ariaLabel={`查看 ${f.name} 的主页`}
			style={`opacity:${unfollowedIds.has(f.id) ? 0.55 : 1};`}
		>
			{#snippet subtitle()}
				<p class="truncate mt-px text-text-secondary text-xs">{f.desc}</p>
			{/snippet}
			{#snippet actions()}
				<button
					type="button"
					class="hover-btn tap-93 flex shrink-0 items-center gap-1 rounded-full px-3 py-1 text-xs outline-none {unfollowedIds.has(f.id) ? 'bg-accent-default text-text-on-accent' : 'bg-transparent text-text-secondary'}"
					style={unfollowedIds.has(f.id) ? 'border:1px solid transparent;' : 'border:1px solid var(--cst-border-strong);'}
					onclick={(e) => {
						e.stopPropagation();
						toggleUnfollow(f.id);
					}}
				>
					{#if unfollowedIds.has(f.id)}
						<UserPlus class="size-2.5"  strokeWidth={2} />
						关注
					{:else}
						<Check class="size-2.5"  strokeWidth={2.5} />
						已关注
					{/if}
				</button>
			{/snippet}
		</UserCardRow>
	{/snippet}
</ProfileUserSection>
