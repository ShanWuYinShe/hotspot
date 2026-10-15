<script lang="ts">
import { formatRelativeDays } from '@/lib/utils';
	import { goto } from '$app/navigation';
	import { Check, UserPlus } from '@lucide/svelte';
	import FollowButton from '@/components/ui/FollowButton.svelte';
	import ProfileUserSection from '@/components/profile/ProfileUserSection.svelte';
	import type { SectionFilter } from '@/components/profile/section-filter';
	import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import { useRelationships, type MappedFollower } from '@/lib/profile-data.svelte';
	import { createOptimisticFollow, type OptimisticFollow } from '@/lib/relationships.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { ROUTES } from '@/lib/routes';

	// 粉丝分区：ProfileUserSection 骨架 + UserCardRow 行（含统计行与待回关徽章）。

	// followState 由调用方（关系聚合页）注入共享，跨 tab 切换保持一致；
	// 独立使用时回退本地实例（行为不变）。
	let { embedded = false, followState = createOptimisticFollow(), searchQ = '' }: { embedded?: boolean; followState?: OptimisticFollow; searchQ?: string } = $props();

	let fanFilter = $state<'all' | 'mutual' | 'pending'>('all');

	// getter 对象解构会拿到非响应式快照，$derived 桥接
	const rel = useRelationships();
	const followers = $derived(rel.followers);
	const loading = $derived(rel.loading);

	// 回关状态以服务端 isMutual 为基准，乐观切换与回滚由共享模块承载

	const isMutualWith = (f: MappedFollower) => followState.isFollowed(f.id, f.isMutual);

	const toggleFollow = async (f: MappedFollower) => {
		const r = await followState.toggle(f.id, f.isMutual);
		if (r.ok && r.followed) showToast(`已关注 ${f.name}`, 'success');
	};

	const filtered = $derived(
		followers
			.filter((f) => {
				const isMutual = isMutualWith(f);
				if (fanFilter === 'mutual') return isMutual;
				if (fanFilter === 'pending') return !isMutual;
				return true;
			})
			// 搜索词过滤（r12-r18 头部统一：四段一致支持搜索）
			.filter((f) => searchQ === '' || f.name.toLowerCase().includes(searchQ.toLowerCase()))
	);
	const mutualCount = $derived(followers.filter((f) => isMutualWith(f)).length);
	const pendingCount = $derived(followers.length - mutualCount);

	const fanLabel = formatRelativeDays;

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
		// 零值弱化:无互相关注/待回关时不亮状态色(对齐 4428f3ad 零在线弱化;Gemini 审查 r18)
		{ label: '互相关注', value: mutualCount, tone: mutualCount > 0 ? 'positive' : undefined },
		{ label: '待回关', value: pendingCount, tone: pendingCount > 0 ? 'warning' : undefined }
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
				<FollowButton
					active={isMutual}
					pending={followState.isPending(f.id)}
					labelActive="互关"
					labelInactive="回关"
					onToggle={() => void toggleFollow(f)}
				/>
			{/snippet}
		</UserCardRow>
	{/snippet}
</ProfileUserSection>
