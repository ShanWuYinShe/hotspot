<script lang="ts">
	import { goto } from '$app/navigation';
	import { Search as SearchIcon, Users, UsersRound, Sparkles, Map as MapIcon, ChevronRight } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import { platform } from '@/stores/platform.svelte';
	import { ROUTES } from '@/lib/routes';

	// 附近门户页（平板/移动端顶级导航「附近」tab）：搜索入口 + 分区入口列表。
	// 内容流由各专区页承担（首页为大纲页），本页不拉取数据，避免与首页重复展示
	// 同一组附近的人/群组/动态。
	// 视觉契约：门户入口收敛为「一个搜索框 + 一张列表卡」——每区一张独立
	// 浮起卡会让移动端呈现四散的「框框」；行级导航交给 ListContainer/ListItem。

	const SECTIONS: { id: string; icon: typeof Users; title: string; description: string; href: string }[] = [
		{ id: 'people', icon: Users, title: '附近的人', description: '认识 5km 内的新朋友', href: ROUTES.nearbyPeople },
		{ id: 'groups', icon: UsersRound, title: '附近群组', description: '加入身边的兴趣小组', href: ROUTES.nearbyGroups },
		{ id: 'moments', icon: Sparkles, title: '附近动态', description: '看看周边正在发生什么', href: ROUTES.nearbyMoments },
		{ id: 'map', icon: MapIcon, title: '地图浏览', description: '在地图上查看周边热点', href: ROUTES.map }
	];

	const device = $derived(platform.platform);
</script>

<svelte:head>
	<title>附近 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		<PageHeader
			title="附近"
			subtitle="发现周边有趣的人和事"
			sticky
			titleSize={device === 'desktop' ? 'xl' : '3xl'}
			subtitleSize={device === 'desktop' ? 'xs' : 'sm'}
		/>
	{/snippet}

	<!-- 搜索入口（门户第一入口）：卡片式搜索框，与下方列表卡保持统一层次 -->
	<button
		type="button"
		class="hover-btn flex w-full items-center gap-3 rounded-md border border-border-default bg-surface-primary px-4 py-3 text-left shadow-raised-sm focus-ring hover:border-border-strong"
		onclick={() => void goto(ROUTES.search)}
	>
		<SearchIcon class="size-5 shrink-0 text-accent-default" strokeWidth={1.75} />
		<span class="min-w-0 flex-1 text-md text-text-secondary">搜索用户、群组、动态、地点…</span>
	</button>

	<!-- 分区入口列表：唯一一张 bordered 列表卡（行间分隔线，无逐行卡片框） -->
	<ListContainer bordered class="mt-4" ariaLabel="附近分区入口">
		{#each SECTIONS as section (section.id)}
			<ListItem class="list-row-standard flex items-center gap-3.5 px-4" onClick={() => void goto(section.href)} ariaLabel={section.title}>
				<span class="flex shrink-0 items-center justify-center rounded-full icon-badge size-9">
					<section.icon class="size-5 text-accent-default" strokeWidth={1.75} />
				</span>
				<span class="min-w-0 flex-1 py-3">
					<span class="block text-md font-medium text-text-primary">{section.title}</span>
					<span class="mt-0.5 block text-xs text-text-secondary">{section.description}</span>
				</span>
				<ChevronRight class="size-4 shrink-0 text-text-secondary" strokeWidth={1.75} />
			</ListItem>
		{/each}
	</ListContainer>
</StandardPageLayout>
