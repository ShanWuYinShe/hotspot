<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SectionFriends from '@/components/profile/SectionFriends.svelte';
	import SectionFollowing from '@/components/profile/SectionFollowing.svelte';
	import SectionFans from '@/components/profile/SectionFans.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { useRelationships } from '@/lib/profile-data.svelte';
	import { ROUTES } from '@/lib/routes';
	import {
		DEFAULT_RELATIONSHIP_TAB,
		isRelationshipTab,
		relationshipsHref,
		type RelationshipTab
	} from '@/lib/relationships';

	// 关系聚合页（/me/relationships，桌面导航「关系」与通知页深链的落地页）：
	// 好友 / 关注 / 粉丝 三 tab。对齐全站样板（nearby/moments）：StandardPageLayout
	// + sticky PageHeader + below 胶囊 Tab 栏（随 header 吸顶）；tab 状态存 URL
	// query（?tab=friends|following|fans，默认 friends），可刷新/分享/直达；
	// 三个分区组件以 embedded 模式内嵌（隐藏各自 header，卡片网格布局）。
	// 三 tab 是关系数据的唯一 canonical 入口（旧 /me/friends 等已重定向到此）。

	const TABS: { id: RelationshipTab; label: string }[] = [
		{ id: 'friends', label: '好友' },
		{ id: 'following', label: '关注' },
		{ id: 'fans', label: '粉丝' }
	];

	// getter 对象解构会拿到非响应式快照，$derived 桥接
	const rel = useRelationships();
	const friends = $derived(rel.friends);
	const following = $derived(rel.following);
	const followers = $derived(rel.followers);

	const activeTab = $derived<RelationshipTab>(
		isRelationshipTab(page.url.searchParams.get('tab'))
			? (page.url.searchParams.get('tab') as RelationshipTab)
			: DEFAULT_RELATIONSHIP_TAB
	);
	const activeCount = $derived(
		activeTab === 'friends' ? friends.length : activeTab === 'following' ? following.length : followers.length
	);

	const switchTab = (tab: RelationshipTab) => {
		if (tab === activeTab) return;
		void goto(relationshipsHref(tab));
	};

	const device = $derived(platform.platform);
</script>

<!-- ═══ 样板 header：sticky PageHeader + below 胶囊 Tab 栏（对齐 nearby/moments）═══ -->
{#snippet relHeader()}
	<PageHeader
		title="关系"
		sticky
		titleSize={device === 'desktop' ? 'xl' : '2xl'}
		onBack={() => void goto(ROUTES.me)}
	>
		{#snippet trailing()}
			<span class="text-text-secondary text-xs" aria-live="polite">{activeCount} 人</span>
		{/snippet}
		{#snippet below()}
			<!-- TabBar 作为 below 首元素需带上间距（对齐 nearby/moments 样板 below 首元素 mt-3） -->
			<div class="mt-3">
				{@render relTabBar()}
			</div>
		{/snippet}
	</PageHeader>
{/snippet}

{#snippet relTabBar()}
	<TabBar
		items={TABS.map((t) => ({
			id: t.id,
			label: t.label,
			count: t.id === 'friends' ? friends.length : t.id === 'following' ? following.length : followers.length
		}))}
		value={activeTab}
		onChange={(id) => switchTab(id as RelationshipTab)}
	/>
{/snippet}

<!-- ═══ 样板 layout：StandardPageLayout（sticky header + ContentColumn 全宽 gutter）═══
     Section 组件为纯内容（不自建 gutter），gutter 由 ContentColumn 统一提供。 -->
<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		{@render relHeader()}
	{/snippet}

	{#if activeTab === 'friends'}
		<SectionFriends embedded />
	{:else if activeTab === 'following'}
		<SectionFollowing embedded />
	{:else}
		<SectionFans embedded />
	{/if}
</StandardPageLayout>
