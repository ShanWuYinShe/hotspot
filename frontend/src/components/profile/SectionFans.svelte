<script lang="ts">
	import { goto } from '$app/navigation';
	import { UserPlus, Check } from '@lucide/svelte';
	import ProfileUserSection from '@/components/profile/ProfileUserSection.svelte';
	import type { SectionFilter } from '@/components/profile/section-filter';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';

	// 粉丝分区：ProfileUserSection 骨架 + UserCardRow 行（含统计行与待回关徽章）。

	let { embedded = false }: { embedded?: boolean } = $props();

	let fanFilter = $state<'all' | 'mutual' | 'pending'>('all');

	const { followers } = useRelationships();
	let followedBack = $state<Set<string>>(new Set(followers.filter((f) => f.isMutual).map((f) => f.id)));

	const toggleFollow = (id: string) => {
		const next = new Set(followedBack);
		next.has(id) ? next.delete(id) : next.add(id);
		followedBack = next;
	};

	const filtered = $derived(
		followers.filter((f) => {
			const isMutual = followedBack.has(f.id);
			if (fanFilter === 'mutual') return isMutual;
			if (fanFilter === 'pending') return !isMutual;
			return true;
		})
	);
	const mutualCount = $derived([...followedBack].length);
	const pendingCount = $derived(followers.length - mutualCount);

	const fanLabel = (joinedDays: number) =>
		joinedDays <= 1 ? '今天' : joinedDays <= 7 ? `${joinedDays}天前` : `${Math.floor(joinedDays / 7)}周前`;

	const FILTERS: SectionFilter[] = [
		{ id: 'all', label: '全部' },
		{ id: 'mutual', label: '互相关注' },
		{ id: 'pending', label: '待回关' }
	];
</script>

<ProfileUserSection
	title="粉丝"
	subtitle="关注你的人，优先处理待回关关系"
	count={followers.length}
	countSuffix=" 人"
	{embedded}
	stats={[
		{ label: '总粉丝', value: followers.length },
		{ label: '互相关注', value: mutualCount, tone: 'positive' },
		{ label: '待回关', value: pendingCount, tone: 'warning' }
	]}
	filters={FILTERS}
	onFilter={(id) => (fanFilter = id as 'all' | 'mutual' | 'pending')}
	emptyTitle="还没有粉丝"
	emptyDescription="分享你的瞬间，让更多人发现你。"
	emptyIcon={UserPlus}
	items={filtered}
>
	{#snippet headerExtra()}
		{#if pendingCount > 0}
			<span class="rounded-full px-2 py-0.5 text-xs text-status-warning" style="background:var(--cst-status-warning-bg);border:var(--cst-border-width-hairline) solid var(--cst-status-warning-border);">
				{pendingCount} 待回关
			</span>
		{/if}
	{/snippet}
	{#snippet children(f)}
		{@const isMutual = followedBack.has(f.id)}
		<UserCardRow
			name={f.name}
			avatar={f.avatar}
			tone={embedded ? 'card' : 'row'}
			onClick={() => void goto(`${ROUTES.user(f.id)}?from=relationships`)}
			ariaLabel={`查看 ${f.name} 的主页`}
		>
			{#snippet nameBadge()}
				{#if isMutual}
					<span class="flex items-center gap-0.5 badge text-status-online" style="background:var(--cst-status-online-bg);">
						<Check class="size-2.5"  strokeWidth={2.5} />
						互关
					</span>
				{/if}
			{/snippet}
			{#snippet subtitle()}
				<p class="mt-0.5 text-text-secondary text-xs">
					{f.handle} · {fanLabel(f.joinedDays)}关注了你
				</p>
			{/snippet}
			{#snippet actions()}
				<button
					type="button"
					class="hover-btn tap-92 flex shrink-0 items-center gap-1 rounded-full px-3 py-1 text-xs focus-ring {isMutual ? 'bg-transparent text-text-secondary' : 'bg-accent-default text-text-on-accent'}"
					style={isMutual ? 'border:var(--cst-border-width-hairline) solid var(--cst-border-strong);' : 'border:var(--cst-border-width-hairline) solid transparent;'}
					onclick={(e) => {
						e.stopPropagation();
						toggleFollow(f.id);
					}}
				>
					{#if isMutual}
						<Check class="size-2.5"  strokeWidth={2.5} />
						互关
					{:else}
						<UserPlus class="size-2.5"  strokeWidth={2} />
						回关
					{/if}
				</button>
			{/snippet}
		</UserCardRow>
	{/snippet}
</ProfileUserSection>
