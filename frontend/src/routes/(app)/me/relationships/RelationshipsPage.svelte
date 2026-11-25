<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SectionFriends from '@/components/profile/SectionFriends.svelte';
	import SectionFollowing from '@/components/profile/SectionFollowing.svelte';
	import SectionFans from '@/components/profile/SectionFans.svelte';
	import SectionBlocked from '@/components/profile/SectionBlocked.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { smartBack } from '@/lib/smart-back';
	import { ROUTES } from '@/lib/routes';
	import { auth } from '@/stores/auth.svelte';
	import {
		DEFAULT_RELATIONSHIP_TAB,
		isRelationshipTab,
		relationshipsHref,
		type RelationshipTab
	} from '@/lib/relationships';
	import { createOptimisticFollow } from '@/lib/relationships.svelte';
	import { isSearchNavKey, nextSearchNavIndex } from '@/lib/search-nav';

	// 关系聚合页（/me/relationships，桌面导航「关系」与通知页深链的落地页）：
	// 好友 / 关注 / 粉丝 三 tab。对齐全站样板（nearby/moments）：StandardPageLayout
	// + integrated PageHeader + below 胶囊 Tab 栏（随内容滚走）；tab 状态存 URL
	// query（?tab=friends|following|fans，默认 friends），可刷新/分享/直达；
	// 三个分区组件以 embedded 模式内嵌（隐藏各自 header，卡片网格布局）。
	// 三 tab 是关系数据的唯一 canonical 入口（旧 /me/friends 等已重定向到此）。

	const TABS: { id: RelationshipTab; label: string }[] = [
		{ id: 'friends', label: '好友' },
		{ id: 'following', label: '关注' },
		{ id: 'fans', label: '粉丝' },
		{ id: 'blocked', label: '黑名单' }
	];

	// getter 对象解构会拿到非响应式快照，$derived 桥接
	const rel = useRelationships();
	// 关系分区搜索词：页头行统一持有并下发四段（r12-r18 头部统一）
	let relSearch = $state('');
	const friends = $derived(rel.friends);
	const following = $derived(rel.following);
	const followers = $derived(rel.followers);

	// 页级共享的乐观关注实例：注入 Following/Fans 分区，tab 内切换即时同步
	// 行、分区计数与页头徽标；跨 tab 保持一致；页重挂载时重建（与数据同生命周期）。
	const followState = createOptimisticFollow();
	const followingCount = $derived(following.filter((f) => followState.isFollowed(f.id, true)).length);

	const activeTab = $derived<RelationshipTab>(
		isRelationshipTab(page.url.searchParams.get('tab'))
			? (page.url.searchParams.get('tab') as RelationshipTab)
			: DEFAULT_RELATIONSHIP_TAB
	);
	const switchTab = (tab: RelationshipTab) => {
		if (tab === activeTab) return;
		void goto(relationshipsHref(tab));
	};

	const device = $derived(platform.platform);

	// 关系列表键盘导航（对齐 /notify、/messages 的列表导航口径）：
	// 候选 = 当前 tab 的列表行覆盖按钮（[data-cover]）与行内动作按钮；焦点在页头搜索框时
	// ↓ 进入首行（与消息页搜索框一致），↑ 保留输入框光标行为；弹层打开时不接管。
	function relationshipNavCandidates(): HTMLElement[] {
		const root = document.querySelector<HTMLElement>('[data-relationship-list]');
		if (!root) return [];
		return Array.from(
			root.querySelectorAll<HTMLElement>('[data-list-item] [data-cover]')
		).filter((el) => el.tabIndex >= 0 && el.offsetParent !== null);
	}

	function handleRelationshipsKeydown(e: KeyboardEvent) {
		if (!isSearchNavKey(e.key)) return;
		if (document.querySelector('[role="dialog"]')) return;
		const active = document.activeElement as HTMLElement | null;
		if (!active) return;
		const isSearch = active instanceof HTMLInputElement && active.getAttribute('aria-label') === '搜索关系';
		const root = document.querySelector<HTMLElement>('[data-relationship-list]');
		const inside = !!root && root.contains(active);
		if (!isSearch && !inside) return;
		const nodes = relationshipNavCandidates();
		if (!nodes.length) return;
		if (isSearch) {
			if (e.key !== 'ArrowDown') return;
			e.preventDefault();
			nodes[0]?.focus();
			return;
		}
		const next = nextSearchNavIndex(nodes.indexOf(active), nodes.length, e.key);
		if (next === null) return;
		e.preventDefault();
		nodes[next]?.focus();
	}
</script>

<svelte:window onkeydown={handleRelationshipsKeydown} />

<!-- ═══ 样板 header：integrated PageHeader + below 胶囊 Tab 栏（对齐 nearby/moments）═══ -->
{#snippet relHeader()}
	<PageHeader
		variant="integrated"
		title="关系"
		islandMeta={TABS.find((t) => t.id === activeTab)?.label ?? '关系'}
		maxWidth="var(--cst-content-breath)"
		onBack={() => smartBack(ROUTES.me)}
	>
		{#snippet trailing()}
			<!-- 页头不再放计数：R13 评审 P2-10（右上悬空「6 人」与分区计数重复且无操作价值）。
			     计数由 Tab 徽标与分区标题承担。 -->
		{/snippet}
		{#snippet below()}
			<!-- 间距由 PageHeader integrated 变体统一提供（below 外层 mt-2.5）。
			     搜索框不放这里：integrated 页头在桌面端整条隐藏，放 below 会让桌面端完全不可达；
			     统一放到正文顶部（登录态可见），三端位置一致且键盘可达。 -->
			<div class="min-w-0">{@render relTabBar()}</div>
		{/snippet}
	</PageHeader>
{/snippet}

{#snippet relTabBar()}
	<TabBar
		items={TABS.map((t) => ({
			id: t.id,
			label: t.label,
			// 未登录不显示 0 计数（无效信息；登录后的真实 0 由分区空态表达，Gemini 审查 r12）
			count: auth.isAuthenticated
				? t.id === 'friends'
					? friends.length
					: t.id === 'following'
						? followingCount
						: t.id === 'fans'
							? followers.length
							: undefined
				: undefined
		}))}
		value={activeTab}
		onChange={(id) => switchTab(id as RelationshipTab)}
	/>
{/snippet}

<!-- ═══ 样板 layout：StandardPageLayout（integrated header + ContentColumn 全宽 gutter）═══
     Section 组件为纯内容（不自建 gutter），gutter 由 ContentColumn 统一提供。 -->
<StandardPageLayout contentMaxWidth="var(--cst-content-breath)" contentClassName="pt-4 pb-6">
	{#snippet header()}
		{@render relHeader()}
	{/snippet}

	{#if auth.isAuthenticated}
		<div class="mb-3 flex justify-end">
			<div class="w-full sm:w-72 sm:shrink-0">
				<SearchInput
					value={relSearch}
					onChange={(v) => (relSearch = v)}
					placeholder={activeTab === 'friends' ? '搜索好友…' : activeTab === 'following' ? '搜索关注…' : activeTab === 'fans' ? '搜索粉丝…' : '搜索黑名单…'}
					ariaLabel="搜索关系"
				/>
			</div>
		</div>
	{/if}
	<div data-relationship-list>
		{#if activeTab === 'friends'}
			<SectionFriends embedded searchQ={relSearch} />
		{:else if activeTab === 'following'}
			<SectionFollowing embedded {followState} searchQ={relSearch} />
		{:else if activeTab === 'fans'}
			<SectionFans embedded {followState} searchQ={relSearch} />
		{:else}
			<SectionBlocked embedded searchQ={relSearch} />
		{/if}
	</div>
</StandardPageLayout>