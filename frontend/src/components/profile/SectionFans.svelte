<script lang="ts">
	import { goto } from '$app/navigation';
	import { UserPlus, Check } from '@lucide/svelte';
	import ProfileUserSection from '@/components/profile/ProfileUserSection.svelte';
	import type { SectionFilter } from '@/components/profile/section-filter';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import { useRelationships, type MappedFollower } from '@/lib/profile-data.svelte';
	import { follow } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import { ROUTES } from '@/lib/routes';

	// 粉丝分区：ProfileUserSection 骨架 + UserCardRow 行（含统计行与待回关徽章）。

	let { embedded = false }: { embedded?: boolean } = $props();

	let fanFilter = $state<'all' | 'mutual' | 'pending'>('all');

	// getter 对象解构会拿到非响应式快照，$derived 桥接
	const rel = useRelationships();
	const followers = $derived(rel.followers);
	const loading = $derived(rel.loading);

	// 回关状态以服务端 isMutual 为基准，本地 overrides 承载乐观切换（API 失败回滚）
	let followOverrides = $state<Record<string, boolean>>({});
	let pendingIds = $state<Set<string>>(new Set());

	const isMutualWith = (f: MappedFollower) => followOverrides[f.id] ?? f.isMutual;

	const toggleFollow = async (f: MappedFollower) => {
		const prev = isMutualWith(f);
		const next = !prev;
		followOverrides = { ...followOverrides, [f.id]: next };
		pendingIds = new Set([...pendingIds, f.id]);
		try {
			// spec: users/set_follow——回关 active:true，取消回关 active:false
			const res = await follow({ user_id: f.id, active: next });
			if (res.code !== 0) throw new Error(res.message ?? '操作失败');
			if (next) showToast(`已关注 ${f.name}`, 'success');
		} catch {
			followOverrides = { ...followOverrides, [f.id]: prev };
			showToast('操作失败，请重试', 'danger');
		} finally {
			const rest = new Set(pendingIds);
			rest.delete(f.id);
			pendingIds = rest;
		}
	};

	const filtered = $derived(
		followers.filter((f) => {
			const isMutual = isMutualWith(f);
			if (fanFilter === 'mutual') return isMutual;
			if (fanFilter === 'pending') return !isMutual;
			return true;
		})
	);
	const mutualCount = $derived(followers.filter((f) => isMutualWith(f)).length);
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
	filterValue={fanFilter}
	onFilter={(id) => (fanFilter = id as 'all' | 'mutual' | 'pending')}
	emptyTitle="还没有粉丝"
	emptyDescription="分享你的瞬间，让更多人发现你。"
	emptyIcon={UserPlus}
	loading={loading}
	error={rel.error}
	onRetry={rel.retry}
	needsAuth={rel.needsAuth}
	items={filtered}
>
	{#snippet headerExtra()}
		{#if pendingCount > 0}
			<span class="inline-flex h-5 items-center justify-center rounded-full px-2 text-xxs font-medium leading-none text-status-warning" style="background:var(--cst-status-warning-bg);border:var(--cst-border-width-hairline) solid var(--cst-status-warning-border);">
				{pendingCount} 待回关
			</span>
		{/if}
	{/snippet}
	{#snippet children(f)}
		{@const isMutual = isMutualWith(f)}
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
					disabled={pendingIds.has(f.id)}
					class="hover-btn tap-92 hit-slop flex shrink-0 items-center gap-1 rounded-full px-3 py-1 text-xs focus-ring disabled:opacity-muted {isMutual ? 'bg-transparent text-text-secondary hover:bg-interactive-hover' : 'bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md'}"
					style={isMutual ? 'border:var(--cst-border-width-hairline) solid var(--cst-border-strong);' : 'border:var(--cst-border-width-hairline) solid transparent;'}
					onclick={(e) => {
						e.stopPropagation();
						void toggleFollow(f);
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
