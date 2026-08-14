<script lang="ts">
	import { goto } from '$app/navigation';
	import { Search as SearchIcon, Users, UsersRound, Sparkles, Map, ChevronRight } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import { platform } from '@/stores/platform.svelte';
	import { ROUTES } from '@/lib/routes';

	// 附近门户页（平板/移动端顶级导航「附近」tab）：搜索入口 + 三分区入口卡。
	// 内容流由各专区页承担（首页为大纲页），本页不拉取数据，避免与首页重复展示
	// 同一组附近的人/群组/动态。

	const SECTIONS: { id: string; icon: typeof Users; title: string; description: string; href: string }[] = [
		{ id: 'people', icon: Users, title: '附近的人', description: '认识 5km 内的新朋友', href: ROUTES.nearbyPeople },
		{ id: 'groups', icon: UsersRound, title: '附近群组', description: '加入身边的兴趣小组', href: ROUTES.nearbyGroups },
		{ id: 'moments', icon: Sparkles, title: '附近动态', description: '看看周边正在发生什么', href: ROUTES.nearbyMoments }
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
			titleSize={device === 'desktop' ? 20 : device === 'tablet' ? 24 : 22}
		>
			{#snippet trailing()}
				<ActionIconButton icon={Map} label="地图" onClick={() => void goto(ROUTES.map)} variant="ghost" size={36} />
			{/snippet}
		</PageHeader>
	{/snippet}

	<!-- 搜索入口（门户第一入口） -->
	<button
		type="button"
		class="hover-btn flex w-full items-center gap-3 rounded-md border border-border-default bg-surface-primary p-3 text-left focus-ring"
		onclick={() => void goto(ROUTES.search)}
	>
		<SearchIcon class="size-5 shrink-0 text-accent-default" strokeWidth={1.75} />
		<span class="min-w-0 flex-1 py-2 text-md text-text-secondary">搜索用户、群组、动态、地点…</span>
	</button>

	<!-- 三分区入口卡（内容在各专区页，这里只做导航门户） -->
	<div class="mt-5 grid gap-3">
		{#each SECTIONS as section (section.id)}
			<button
				type="button"
				class="hover-btn flex w-full items-center gap-3.5 rounded-md border border-border-default bg-surface-primary p-4 text-left focus-ring"
				onclick={() => void goto(section.href)}
			>
				<span class="flex shrink-0 items-center justify-center rounded-full icon-badge size-9">
					<section.icon class="size-5 text-accent-default" strokeWidth={1.75} />
				</span>
				<span class="min-w-0 flex-1">
					<span class="block text-md font-medium text-text-primary">{section.title}</span>
					<span class="mt-0.5 block text-xs text-text-secondary">{section.description}</span>
				</span>
				<ChevronRight class="size-4 shrink-0 text-text-secondary" strokeWidth={1.75} />
			</button>
		{/each}
	</div>
</StandardPageLayout>
