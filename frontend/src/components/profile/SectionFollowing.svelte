<script lang="ts">
	import { goto } from '$app/navigation';
	import { UserPlus, Check, Heart } from '@lucide/svelte';
	import ProfileUserSection from '@/components/profile/ProfileUserSection.svelte';
	import type { SectionFilter } from '@/components/profile/section-filter';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { createOptimisticFollow } from '@/lib/relationships.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { ROUTES } from '@/lib/routes';

	// 关注分区：ProfileUserSection 骨架 + UserCardRow 行。

	let { embedded = false }: { embedded?: boolean } = $props();

	let followingFilter = $state<'all' | 'mutual'>('all');
	// 取关状态：服务端 following 列表即基准（全员初始已关注），乐观切换与回滚由共享模块承载
	const followState = createOptimisticFollow();

	// getter 对象解构会拿到非响应式快照，$derived 桥接
	const rel = useRelationships();
	const following = $derived(rel.following);
	const loading = $derived(rel.loading);

	const isUnfollowed = (id: string) => !followState.isFollowed(id, true);

	const toggleUnfollow = async (id: string, name: string) => {
		const r = await followState.toggle(id, true);
		if (r.ok && !r.followed) showToast(`已取消关注 ${name}`, 'success');
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
	{embedded}
	filters={FILTERS}
	filterValue={followingFilter}
	onFilter={(id) => (followingFilter = id as 'all' | 'mutual')}
	emptyTitle="还没有关注"
	emptyDescription="去发现附近的人，找到有趣的创作者。"
	emptyIcon={Heart}
	loading={loading}
	error={rel.error}
	onRetry={rel.retry}
	needsAuth={rel.needsAuth}
	items={filtered}
>
	{#snippet children(f)}
		<UserCardRow
			name={f.name}
			avatar={f.avatar}
			tone={embedded ? 'card' : 'row'}
			onClick={() => void goto(`${ROUTES.user(f.id)}?from=relationships`)}
			ariaLabel={`查看 ${f.name} 的主页`}
			style={`opacity:${isUnfollowed(f.id) ? 'var(--cst-opacity-muted)' : 'var(--cst-opacity-full)'};`}
		>
			{#snippet subtitle()}
				<p class="truncate mt-px text-text-secondary text-xs">{f.desc}</p>
			{/snippet}
			{#snippet actions()}
				<button
					type="button"
					aria-pressed={!isUnfollowed(f.id)}
					disabled={followState.isPending(f.id)}
					class="hover-btn tap-92 hit-slop flex shrink-0 items-center gap-1 rounded-full border px-3.5 py-1.5 text-xs focus-ring disabled:pointer-events-none disabled:opacity-faint
						{isUnfollowed(f.id)
						? 'border-transparent bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md'
						: 'border-border-strong bg-transparent text-text-secondary hover:border-accent-border hover:bg-interactive-hover hover:text-text-primary'}"
					onclick={(e) => {
						e.stopPropagation();
						void toggleUnfollow(f.id, f.name);
					}}
				>
					{#if isUnfollowed(f.id)}
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
