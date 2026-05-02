<script lang="ts">
	import { page } from '$app/state';
	import { momentsService } from '$lib/api/services';
	import { navigateToNearbyBackTarget } from '../../nearby-back';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { platformStore } from '$lib/stores/platform.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { useLocationAwareLoad } from '$lib/hooks/use-location-aware-load.svelte';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { MomentItem } from '$lib/types/api';
	import Search from 'lucide-svelte/icons/search';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Image from 'lucide-svelte/icons/image';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import { t } from '$lib/i18n';

	const TABS = [
		{ id: 'hot', label: t('discover.hot') || '热门' },
		{ id: 'nearby', label: t('discover.nearby') || '附近' },
		{ id: 'following', label: t('discover.following') || '关注' },
	];

	const pagination = usePagination<MomentItem>({
		uniqueKey: 'moment_id'
	});

	let hasLoadedOnce = $state(false);
	let activeTab = $state('nearby');
	let searchVal = $state('');

	const returnTarget = $derived(page.url.searchParams.get('from') || '/search');

	function getAnchor() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	async function loadMoments(refresh = false) {
		await pagination.load(
			(cursor) =>
				momentsService.search({
					radius: mapAnchorStore.radius * 1000,
					reference_point: getAnchor(),
					pagination: { limit: 20, cursor }
				}),
			refresh
		);
		hasLoadedOnce = true;
	}

	function handleBack() {
		navigateToNearbyBackTarget(typeof window === 'undefined' ? undefined : window, returnTarget);
	}

	function handleRetry() {
		void loadMoments(true);
	}

	useLocationAwareLoad((refresh) => loadMoments(refresh));
</script>

<svelte:head>
	<title>{t('discover.title')} - HotSpot</title>
</svelte:head>

{#if !platformStore.isDesktop}
	<!-- Mobile / Tablet Discover -->
	<div class="flex flex-col">
		<!-- Sticky header -->
		<div
			class="sticky top-0 z-10 px-5 pt-3 pb-0"
			style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
		>
			<!-- Top bar -->
			<div class="mb-2.5 flex items-center justify-between">
				<div class="flex items-center gap-2">
					<button onclick={handleBack} class="outline-none" aria-label="返回">
						<ArrowLeft size={18} color="var(--neu-text-muted)" />
					</button>
					<h1 style="font-size: 20px; color: var(--neu-text); font-weight: 600;">{t('discover.title')}</h1>
				</div>
				<div
					class="flex items-center gap-1 rounded-full px-2.5 py-1"
					style="box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
				>
					<MapPin size={10} strokeWidth={1.5} color="var(--neu-accent-primary)" />
					<span style="font-size: 10px; color: var(--neu-text-muted);">{mapAnchorStore.displayName}</span>
				</div>
			</div>

			<!-- Search bar -->
			<div
				class="mb-2.5 flex items-center gap-3 rounded-[18px] px-4 py-2.5"
				style="box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
			>
				<Search size={15} strokeWidth={1.5} color="var(--neu-text-muted)" />
				<input
					type="text"
					bind:value={searchVal}
					placeholder="搜索地点、人物、关键词…"
					class="flex-1 bg-transparent outline-none"
					style="font-size: 13px; color: var(--neu-text);"
				/>
			</div>

			<!-- Tabs -->
			<div class="flex overflow-x-auto no-scrollbar" style="border-bottom: 1px solid var(--neu-border);">
				{#each TABS as tab (tab.id)}
					<button
						class="relative flex-shrink-0 pb-3 pt-1 outline-none"
						style="padding-left: 18px; padding-right: 18px; color: {activeTab === tab.id ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'}; font-weight: {activeTab === tab.id ? 500 : 400}; font-size: 13px; transition: color 0.2s;"
						onclick={() => { activeTab = tab.id; }}
					>
						{tab.label}
						{#if activeTab === tab.id}
							<div
								style="position: absolute; bottom: -1px; left: 12%; right: 12%; height: 2px; border-radius: 1px; background: var(--neu-accent-primary);"
							></div>
						{/if}
					</button>
				{/each}
			</div>
		</div>

		<!-- Content -->
		<div class="flex flex-col gap-4 px-5 pt-3 pb-6">
			{#if pagination.loading}
				{#each Array(4) as _}
					<div class="rounded-[24px]" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<div class="rounded-t-[22px]" style="height: 140px; background: var(--neu-border);"></div>
						<div class="p-4">
							<div class="mb-2 rounded" style="width: 80%; height: 12px; background: var(--neu-border);"></div>
							<div class="rounded" style="width: 60%; height: 10px; background: var(--neu-border);"></div>
						</div>
					</div>
				{/each}
			{:else if pagination.error}
				<EmptyState
					icon={AlertTriangle}
					title={t('discover.loadFailed')}
					description={pagination.error}
					actionText={t('common.retry')}
					onAction={handleRetry}
				/>
			{:else if pagination.items.length === 0 && hasLoadedOnce}
				<EmptyState icon={Image} title={t('discover.noMoments')} description={t('discover.noMomentsDesc')} />
			{:else}
				{#each pagination.items as moment (moment.moment_id)}
					<MomentCard {moment} />
				{/each}
				<LoadMoreButton
					loading={pagination.loadingMore}
					hasMore={pagination.hasMore}
					onLoadMore={() => loadMoments(false)}
				/>
			{/if}
		</div>
	</div>
{:else}
	<!-- Desktop Discover -->
	<div class="flex h-full flex-col">
		<!-- Sticky header -->
		<div
			class="flex-shrink-0 px-6 pt-5 pb-0"
			style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
		>
			<div class="mb-3 flex items-baseline justify-between">
				<div class="flex items-center gap-3">
					<button onclick={handleBack} class="outline-none" aria-label="返回">
						<ArrowLeft size={18} color="var(--neu-text-muted)" />
					</button>
					<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">{t('discover.title')}</h1>
				</div>
				<span style="font-size: 12px; color: var(--neu-text-muted);">
					{pagination.items.length} {t('discover.results')}
				</span>
			</div>

			<!-- Search bar -->
			<div
				class="mb-2.5 flex items-center gap-3 rounded-[18px] px-4 py-2.5"
				style="box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
			>
				<Search size={15} strokeWidth={1.5} color="var(--neu-text-muted)" />
				<input
					type="text"
					bind:value={searchVal}
					placeholder="搜索地点、人物、关键词…"
					class="flex-1 bg-transparent outline-none"
					style="font-size: 13px; color: var(--neu-text);"
				/>
			</div>

			<!-- Tabs -->
			<div class="flex overflow-x-auto no-scrollbar" style="border-bottom: 1px solid var(--neu-border);">
				{#each TABS as tab (tab.id)}
					<button
						class="relative flex-shrink-0 pb-3 pt-1 outline-none"
						style="padding-left: 18px; padding-right: 18px; color: {activeTab === tab.id ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'}; font-weight: {activeTab === tab.id ? 500 : 400}; font-size: 13px; transition: color 0.2s;"
						onclick={() => { activeTab = tab.id; }}
					>
						{tab.label}
						{#if activeTab === tab.id}
							<div
								style="position: absolute; bottom: -1px; left: 12%; right: 12%; height: 2px; border-radius: 1px; background: var(--neu-accent-primary);"
							></div>
						{/if}
					</button>
				{/each}
			</div>
		</div>

		<!-- Content: 3-col grid -->
		<div class="flex-1 overflow-y-auto no-scrollbar px-6 pt-5 pb-6">
			{#if pagination.loading}
				<div class="grid grid-cols-3 gap-4">
					{#each Array(6) as _}
						<div class="rounded-[24px]" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<div class="rounded-t-[22px]" style="height: 120px; background: var(--neu-border);"></div>
							<div class="p-4">
								<div class="mb-2 rounded" style="width: 80%; height: 12px; background: var(--neu-border);"></div>
								<div class="rounded" style="width: 60%; height: 10px; background: var(--neu-border);"></div>
							</div>
						</div>
					{/each}
				</div>
			{:else if pagination.error}
				<EmptyState
					icon={AlertTriangle}
					title={t('discover.loadFailed')}
					description={pagination.error}
					actionText={t('common.retry')}
					onAction={handleRetry}
				/>
			{:else if pagination.items.length === 0 && hasLoadedOnce}
				<EmptyState icon={Image} title={t('discover.noMoments')} description={t('discover.noMomentsDesc')} />
			{:else}
				<div class="columns-3 gap-4">
					{#each pagination.items as moment (moment.moment_id)}
						<div class="mb-4 break-inside-avoid">
							<MomentCard {moment} />
						</div>
					{/each}
				</div>
				<LoadMoreButton
					loading={pagination.loadingMore}
					hasMore={pagination.hasMore}
					onLoadMore={() => loadMoments(false)}
				/>
			{/if}
		</div>
	</div>
{/if}
