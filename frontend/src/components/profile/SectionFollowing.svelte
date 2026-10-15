<script lang="ts">
	import { goto } from '$app/navigation';
	import { Heart } from '@lucide/svelte';
	import ProfileUserSection from '@/components/profile/ProfileUserSection.svelte';
	import type { SectionFilter } from '@/components/profile/section-filter';
	import FollowButton from '@/components/ui/FollowButton.svelte';
import UserCardRow from '@/components/profile/UserCardRow.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { createOptimisticFollow, type OptimisticFollow } from '@/lib/relationships.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { ROUTES } from '@/lib/routes';

	// 关注分区：ProfileUserSection 骨架 + UserCardRow 行。

	// followState 由调用方（关系聚合页）注入共享，tab 内切换即时同步行与计数；
	// 独立使用时回退本地实例（行为不变）。
	let { embedded = false, followState = createOptimisticFollow(), searchQ = '' }: { embedded?: boolean; followState?: OptimisticFollow; searchQ?: string } = $props();

	let followingFilter = $state<'all' | 'mutual'>('all');

	// getter 对象解构会拿到非响应式快照，$derived 桥接
	const rel = useRelationships();
	const following = $derived(rel.following);
	const loading = $derived(rel.loading);

	// 可见行：服务端列表为基准，取关成功的行立即移除（对齐黑名单 tab 的
	// 解除即消失；此前残留变灰行 + 计数不变，空态在切页内永远不可达）。
	// override 生效中（请求飞行）行已翻转，失败回滚后自动恢复。
	const isFollowed = (id: string) => followState.isFollowed(id, true);
	const visibleFollowing = $derived(following.filter((f) => isFollowed(f.id)));
	const visibleCount = $derived(visibleFollowing.length);

	const toggleUnfollow = async (id: string, name: string) => {
		const r = await followState.toggle(id, true);
		if (r.ok && !r.followed) showToast(`已取消关注 ${name}`, 'success');
	};

	// 搜索词过滤（r12-r18 头部统一：四段一致支持搜索）
	const filtered = $derived(
		(visibleFollowing.filter((f) => (followingFilter === 'mutual' ? f.isMutual : true))).filter((f) =>
			searchQ === '' || f.name.toLowerCase().includes(searchQ.toLowerCase())
		)
	);

	const FILTERS: SectionFilter[] = [
		{ id: 'all', label: '全部' },
		{ id: 'mutual', label: '互关' }
	];
</script>

<ProfileUserSection
	title="关注"
	subtitle="你主动订阅的动态与创作者"
	count={visibleCount}
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

		>
			{#snippet subtitle()}
				<p class="truncate mt-px text-text-secondary text-xs">{f.desc}</p>
			{/snippet}
			{#snippet actions()}
				<FollowButton
					active={isFollowed(f.id)}
					pending={followState.isPending(f.id)}
					onToggle={() => void toggleUnfollow(f.id, f.name)}
				/>
			{/snippet}
		</UserCardRow>
	{/snippet}
</ProfileUserSection>
