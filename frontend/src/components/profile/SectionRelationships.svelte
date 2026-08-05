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

	// 关系聚合页（/me/relationships，桌面导航「关系」与通知页深链的落地页）：
	// 好友 / 关注 / 粉丝 三 tab。对齐全站样板（nearby/moments）：StandardPageLayout
	// + sticky PageHeader + below 下划线 Tab 栏（随 header 吸顶）；tab 状态存 URL
	// query（?tab=friends|following|fans，默认 friends），可刷新/分享/直达；
	// 三个分区组件以 embedded 模式内嵌（隐藏各自 header，卡片网格布局）。

	type RelTab = 'friends' | 'following' | 'fans';
	const TABS: { id: RelTab; label: string }[] = [
		{ id: 'friends', label: '好友' },
		{ id: 'following', label: '关注' },
		{ id: 'fans', label: '粉丝' }
	];

	const { friends, following, followers } = useRelationships();

	const activeTab = $derived<RelTab>(
		TABS.some((t) => t.id === page.url.searchParams.get('tab'))
			? (page.url.searchParams.get('tab') as RelTab)
			: 'friends'
	);
	const activeCount = $derived(
		activeTab === 'friends' ? friends.length : activeTab === 'following' ? following.length : followers.length
	);

	const switchTab = (tab: RelTab) => {
		if (tab === activeTab) return;
		void goto(`${ROUTES.meSection('relationships')}?tab=${tab}`);
	};

	const device = $derived(platform.platform);
</script>

<!-- ═══ 样板 header：sticky PageHeader + below 下划线 Tab 栏（对齐 nearby/moments）═══ -->
{#snippet relHeader()}
	<PageHeader
		title="关系"
		sticky
		titleSize={device === 'desktop' ? 20 : device === 'tablet' ? 24 : 22}
		contentClassName="pt-4 pb-0"
		onBack={() => void goto(ROUTES.me)}
	>
		{#snippet trailing()}
			<span class="text-text-secondary text-xs">{activeCount} 人</span>
		{/snippet}
		{#snippet below()}
			{@render relTabBar()}
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
		onchange={(id) => switchTab(id as RelTab)}
	/>
{/snippet}

<!-- ═══ 样板 layout：StandardPageLayout（sticky header + ContentColumn 全宽 gutter）═══
     embedded Section 自带 px-[var(--page-gutter)]，负 margin 抵消 ContentColumn 的
     gutter，避免双重内边距（视觉 gutter 与 header/正文保持一致）。 -->
<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		{@render relHeader()}
	{/snippet}

	<div class="-mx-[var(--page-gutter)]">
		{#if activeTab === 'friends'}
			<SectionFriends embedded />
		{:else if activeTab === 'following'}
			<SectionFollowing embedded />
		{:else}
			<SectionFans embedded />
		{/if}
	</div>
</StandardPageLayout>
