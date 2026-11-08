<script lang="ts">
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { ArrowUp, Clock, Flame, Search as SearchIcon, X } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import { headerIsland } from '@/stores/header-island.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import Button from '@/components/ui/button.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { smartBack } from '@/lib/smart-back';
	import { ROUTES } from '@/lib/routes';

	import SearchDiscovery from './shared/SearchDiscovery.svelte';
	import SearchResults from './shared/SearchResults.svelte';
	import { useSearchPageData } from './shared/useSearchPageData.svelte';

	const data = useSearchPageData();
	const device = $derived(platform.platform);

	const searchShortcut = $derived(
		typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘K' : 'Ctrl K'
	);

	function scrollToTop() {
		const scroller = document.getElementById('main-content');
		const reduced = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
		scroller?.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' });
	}

	const islandToken = Symbol('search-island');

	// 桌面端顶栏灵动岛：直接作为全局搜索交互中枢
	$effect(() => {
		if (device !== 'desktop') return;
		headerIsland.register(
			islandToken,
			untrack(() => ({
				title: '搜索',
				meta: data.islandMeta,
				icon: islandIcon,
				center: islandSearchCenter,
				trailing: islandSearchTrailing,
				expanded: islandSearchExpanded,
				onBack: () => smartBack(),
				backLabel: '返回',
				onTitleClick: scrollToTop
			}))
		);
		return () => headerIsland.release(islandToken);
	});

	$effect(() => {
		if (device !== 'desktop') return;
		headerIsland.update(islandToken, {
			title: '搜索',
			meta: data.islandMeta,
			icon: islandIcon,
			center: islandSearchCenter,
			trailing: islandSearchTrailing,
			expanded: islandSearchExpanded,
			onBack: () => smartBack(),
			backLabel: '返回',
			onTitleClick: scrollToTop
		});
	});
</script>

<svelte:head>
	<title>搜索 · HotSpot</title>
</svelte:head>

<svelte:window onkeydown={(e) => data.handleSearchNavKeydown(e)} />

{#snippet islandIcon()}
	<SearchIcon class="size-3.5 text-accent-default" strokeWidth={2} />
{/snippet}

{#snippet islandSearchCenter()}
	<div class="flex h-full w-full min-w-0 items-center gap-1.5 pr-1">
		<input
			type="text"
			inputmode="search"
			enterkeyhint="search"
			autocomplete="off"
			autocapitalize="off"
			spellcheck="false"
			data-search-input=""
			placeholder="搜索用户、瞬间、群组或话题..."
			aria-label="搜索用户、瞬间、群组、地点或话题"
			value={data.query}
			oninput={(e) => {
				const v = (e.currentTarget as HTMLInputElement).value;
				data.searchTag = null;
				data.query = v;
				if (!v.trim()) data.commitSearchUrl();
			}}
			onkeydown={(e) => {
				if (e.key === 'Enter') {
					e.preventDefault();
					if (!data.query.trim()) return;
					data.remember(data.query.trim());
					data.commitSearchUrl();
				}
			}}
			class="h-6 w-full min-w-0 bg-transparent text-xs text-text-primary placeholder:text-text-secondary/60 focus:outline-none"
		/>
		{#if data.query.trim()}
			<button
				type="button"
				class="flex size-4 shrink-0 items-center justify-center rounded-full text-text-secondary/70 hover:text-text-primary hover:bg-interactive-hover transition-colors"
				onclick={() => data.clearSearch()}
				aria-label="清空搜索"
				title="清空搜索"
			>
				<X class="size-3" />
			</button>
		{/if}
	</div>
{/snippet}

{#snippet islandSearchTrailing()}
	{#if !data.query.trim()}
		<kbd class="hidden sm:inline-flex items-center rounded border border-border-default/35 bg-surface-island/50 px-1 py-0.2 text-[10px] font-mono text-text-secondary/70 select-none">
			{searchShortcut}
		</kbd>
	{/if}
{/snippet}

{#snippet islandSearchExpanded()}
	<div class="flex flex-col gap-2.5">
		<div class="flex items-center justify-between text-xs pb-0.5">
			<span class="font-medium text-text-primary">搜索控制中枢</span>
			<span class="font-mono text-xxs text-accent-default">
				{data.keyword ? `共 ${data.resultsTotal} 条结果` : '即时探索与分类'}
			</span>
		</div>

		<!-- 搜索分类切换 -->
		<div class="flex justify-center w-full py-0.5">
			<TabBar
				variant="segmented"
				size="sm"
				items={data.tabs}
				value={data.tab}
				onChange={(id) => {
					data.tab = id as any;
					data.commitSearchUrl();
				}}
				class="w-full justify-center bg-surface-subtle/80"
			/>
		</div>

		<!-- 热门推荐词快速选择 -->
		{#if data.hotTopics.length > 0}
			<div class="flex flex-col gap-1.5 pt-0.5">
				<div class="flex items-center gap-1.5 text-xxs font-medium text-text-secondary">
					<Flame class="size-3 text-accent-default" />
					<span>热门话题</span>
				</div>
				<div class="flex flex-wrap gap-1.5 max-h-20 overflow-y-auto no-scrollbar">
					{#each data.hotTopics.slice(0, 6) as topic (topic)}
						<button
							type="button"
							class="rounded-full border border-border-default/30 bg-surface-subtle/70 px-2.5 py-0.5 text-xxs text-text-secondary hover:border-accent-border/50 hover:bg-accent-soft hover:text-text-accent-on-soft transition-all active:scale-95 cursor-pointer"
							onclick={() => data.chooseTopic(topic)}
						>
							#{topic}
						</button>
					{/each}
				</div>
			</div>
		{/if}

		<!-- 搜索历史 -->
		{#if data.recent.length > 0}
			<div class="flex flex-col gap-1.5 pt-0.5">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-1.5 text-xxs font-medium text-text-secondary">
						<Clock class="size-3 text-text-secondary/70" />
						<span>历史搜索</span>
					</div>
					<button
						type="button"
						class="text-xxs text-text-secondary/60 hover:text-accent-default transition-colors"
						onclick={() => (data.recent = [])}
					>
						清空
					</button>
				</div>
				<div class="flex flex-wrap gap-1.5 max-h-16 overflow-y-auto no-scrollbar">
					{#each data.recent.slice(0, 5) as r (r)}
						<button
							type="button"
							class="rounded-full border border-border-default/20 bg-surface-subtle/40 px-2.5 py-0.5 text-xxs text-text-secondary hover:text-text-primary hover:border-border-default/40 transition-all active:scale-95 cursor-pointer"
							onclick={() => data.choose(r)}
						>
							{r}
						</button>
					{/each}
				</div>
			</div>
		{/if}

		<!-- 快捷操作与回顶 -->
		<div class="flex items-center justify-between border-t border-border-default/15 pt-2 text-xxs text-text-secondary">
			<div class="flex items-center gap-1.5 text-text-secondary/70">
				<kbd class="rounded border border-border-default/30 bg-surface-island px-1 py-0.2 font-mono text-[10px]">↵</kbd>
				<span>检索</span>
				<span class="text-border-default/40">·</span>
				<kbd class="rounded border border-border-default/30 bg-surface-island px-1 py-0.2 font-mono text-[10px]">Esc</kbd>
				<span>收起</span>
			</div>
			<button
				type="button"
				class="group flex items-center gap-1 text-accent-default hover:underline transition-colors font-medium text-xxs"
				onclick={scrollToTop}
			>
				<span>平滑回顶</span>
				<ArrowUp class="size-3 transition-transform group-hover:-translate-y-0.5" />
			</button>
		</div>
	</div>
{/snippet}

<div class="flex flex-col flex-1 min-h-full" data-search-root>
	<StandardPageLayout contentClassName="pt-4 pb-6">
		{#snippet header()}
			{#if device !== 'desktop'}
				<PageHeader
					variant="integrated"
					title="搜索"
					onBack={() => smartBack()}
					islandMeta={data.islandMeta}
				>
					{#snippet islandTrailing()}
						{#if data.query.trim()}
							<ActionIconButton icon={X} label="清空搜索" variant="ghost" tone="muted" size={32} onClick={data.clearSearch} />
						{/if}
					{/snippet}

					{#snippet below()}
						<div class="mt-3 mb-2.5 w-full">
							<SearchInput
								value={data.query}
								onChange={(v) => {
									data.searchTag = null;
									data.query = v;
									if (!v.trim()) data.commitSearchUrl();
								}}
								placeholder="搜索用户、瞬间、群组、地点或话题"
								ariaLabel="搜索用户、瞬间、群组、地点或话题"
								autofocus
								onEnter={(v) => {
									if (!v.trim()) return;
									data.remember(v.trim());
									data.commitSearchUrl();
								}}
							/>
						</div>
					{/snippet}
				</PageHeader>
			{/if}
		{/snippet}

		{#if data.searchError}
			<LoadError message={data.searchError} onRetry={() => void data.doSearch(data.keyword)} />
		{:else if !data.keyword}
			{#if data.discoveryErrorView === 'full'}
				<LoadError message="推荐内容加载失败，请稍后重试" onRetry={data.retryDiscovery} />
			{:else}
				{#if data.discoveryErrorView === 'inline'}
					<div class="mb-6 flex items-center gap-3 rounded-md bg-surface-subtle px-3 py-2.5">
						<p class="min-w-0 flex-1 text-sm text-text-secondary" role="status">
							{data.discoveryAllFailed ? '推荐内容暂时无法加载，你仍可通过上方搜索框查找。' : '部分推荐内容加载失败。'}
						</p>
						<Button type="button" variant="secondary" size="sm" class="hit-slop shrink-0" onclick={data.retryDiscovery}>重试</Button>
					</div>
				{/if}
				<SearchDiscovery
					recent={data.recent}
					hotTopics={data.hotTopics}
					hotMoments={data.hotMoments}
					hotUsers={data.hotUsers}
					hotGroups={data.hotGroups}
					topicsLoading={data.momentsLoading}
					momentsLoading={data.momentsLoading}
					usersLoading={data.usersLoading}
					groupsLoading={data.groupsLoading}
					onChoose={data.choose}
					onChooseTopic={data.chooseTopic}
					onOpenMoment={(id) => void goto(ROUTES.moment(id))}
					onOpenUser={(id) => void goto(ROUTES.user(id))}
					onOpenGroup={(id) => void goto(ROUTES.group(id))}
					onSeeAllUsers={() => void goto(ROUTES.nearbyPeople)}
					onSeeAllGroups={() => void goto(ROUTES.nearbyGroups)}
					onClear={() => (data.recent = [])}
				/>
			{/if}
		{:else}
			<SearchResults
				query={data.query}
				tab={data.tab}
				tabs={data.tabs}
				loading={data.loading}
				hasResults={data.hasResults}
				results={data.results}
				suggestions={data.discoveryTopics.filter((t) => t.toLowerCase() !== data.keyword)}
				hotTopics={data.hotTopics}
				onChooseTopic={data.chooseTopic}
				onTabChange={(id) => {
					data.tab = id;
					data.commitSearchUrl();
				}}
				onChoose={data.choose}
			/>
		{/if}
	</StandardPageLayout>
</div>
