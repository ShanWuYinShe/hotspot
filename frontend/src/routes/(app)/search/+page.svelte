<script lang="ts">
	import { goto } from '$app/navigation';
	import { X } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import Button from '@/components/ui/button.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
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
</script>

<svelte:head>
	<title>搜索 · HotSpot</title>
</svelte:head>

<svelte:window onkeydown={(e) => data.handleSearchNavKeydown(e)} />

<div class="flex flex-col flex-1 min-h-full" data-search-root>
	<StandardPageLayout contentClassName="pt-4 pb-6">
		{#snippet header()}
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
							class={device === 'desktop' ? 'max-w-3xl 2xl:max-w-5xl' : ''}
							onEnter={(v) => {
								if (!v.trim()) return;
								data.remember(v.trim());
								data.commitSearchUrl();
							}}
						>
							{#snippet trailing()}
								{#if device === 'desktop'}
									<kbd
										class="flex shrink-0 items-center rounded-sm border border-border-default bg-surface-subtle px-1.5 py-0.5 text-xs font-medium text-text-secondary"
										aria-hidden="true"
									>{searchShortcut}</kbd>
								{/if}
							{/snippet}
						</SearchInput>
					</div>
				{/snippet}
			</PageHeader>
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
