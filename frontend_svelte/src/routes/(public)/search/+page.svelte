<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { t } from '$lib/i18n';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { usersService, momentsService, groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import AnchorSummaryCard from '$lib/components/common/anchor-summary-card.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import UserCard from '$lib/components/common/user-card.svelte';
	import GroupCard from '$lib/components/common/group-card.svelte';
	import MasonryLayout from '$lib/components/common/masonry-layout.svelte';
	import MasonryLayoutItem from '$lib/components/common/masonry-layout-item.svelte';
	import type { UserItem, MomentItem, GroupItem } from '$lib/types/api';
	import Search from 'lucide-svelte/icons/search';
	import Compass from 'lucide-svelte/icons/compass';
	import Users from 'lucide-svelte/icons/users';
	import Image from 'lucide-svelte/icons/image';
	import UsersRound from 'lucide-svelte/icons/users-round';
	import Hash from 'lucide-svelte/icons/hash';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';
	import X from 'lucide-svelte/icons/x';

	type SearchMode = 'keyword' | 'tag';
	type SearchTab = 'users' | 'moments' | 'groups';

	const SEARCH_TABS: SearchTab[] = ['users', 'moments', 'groups'];

	let keywordInput = $state('');
	let isLoading = $state(false);
	let searchError = $state<string | null>(null);

	let users = $state.raw<UserItem[]>([]);
	let moments = $state.raw<MomentItem[]>([]);
	let groups = $state.raw<GroupItem[]>([]);

	let lastSearchFingerprint = $state('');
	let searchAbortController: AbortController | null = null;
	let lastAnchorUpdatedAt = mapAnchorStore.anchor.updatedAt;

	const currentTab = $derived(parseSearchTab(page.url.searchParams.get('tab')));
	const activeMode = $derived.by<SearchMode | null>(() => {
		const tag = page.url.searchParams.get('tag')?.trim() ?? '';
		const q = page.url.searchParams.get('q')?.trim() ?? '';
		return tag ? 'tag' : q ? 'keyword' : null;
	});
	const activeTerm = $derived(page.url.searchParams.get('tag')?.trim() ?? page.url.searchParams.get('q')?.trim() ?? '');
	const activeLabel = $derived(activeMode === 'tag' ? `#${activeTerm}` : activeTerm);
	const hasAnyResults = $derived(users.length + moments.length + groups.length > 0);
	const nearbyMomentsUrl = $derived(
		`/moments/nearby?from=${encodeURIComponent(`${page.url.pathname}${page.url.search}` || '/search')}`
	);

	function getReferencePoint() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	function parseSearchTab(value: string | null): SearchTab {
		return SEARCH_TABS.includes(value as SearchTab) ? (value as SearchTab) : 'users';
	}

	function buildSearchFingerprint(mode: SearchMode, value: string): string {
		const point = getReferencePoint();
		return `${mode}:${value}:${point.latitude.toFixed(6)}:${point.longitude.toFixed(6)}:${mapAnchorStore.radius}`;
	}

	function buildSearchUrl(mode: SearchMode, value: string, tab: SearchTab): string {
		const params = new URLSearchParams();
		params.set(mode === 'tag' ? 'tag' : 'q', value);
		if (tab !== 'users') {
			params.set('tab', tab);
		}
		return `/search?${params.toString()}`;
	}

	function resetResults() {
		searchAbortController?.abort();
		searchAbortController = null;
		users = [];
		moments = [];
		groups = [];
		isLoading = false;
		searchError = null;
		lastSearchFingerprint = '';
	}

	async function performSearch(mode: SearchMode, value: string) {
		const normalized = value.trim();
		if (!normalized) {
			resetResults();
			return;
		}

		const fingerprint = buildSearchFingerprint(mode, normalized);
		if (fingerprint === lastSearchFingerprint) {
			searchError = null;
			return;
		}

		searchAbortController?.abort();
		const controller = new AbortController();
		searchAbortController = controller;

		lastSearchFingerprint = fingerprint;
		searchError = null;
		isLoading = true;
		users = [];
		moments = [];
		groups = [];

		try {
			const signal = controller.signal;
			const request = mode === 'tag'
				? {
					tag: normalized,
					reference_point: getReferencePoint()
				}
				: {
					keyword: normalized,
					reference_point: getReferencePoint()
				};

			const [usersRes, momentsRes, groupsRes] = await withMinimumLoadingTime(() =>
				Promise.all([
					usersService.search(request, { signal }),
					momentsService.search(request, { signal }),
					groupsService.search(request, { signal })
				])
			);

			if (signal.aborted) {
				return;
			}

			const nextUsers = isSuccess(usersRes) && usersRes.data ? usersRes.data.items : [];
			const nextMoments = isSuccess(momentsRes) && momentsRes.data ? momentsRes.data.items : [];
			const nextGroups = isSuccess(groupsRes) && groupsRes.data ? groupsRes.data.items : [];
			const nextError = !isSuccess(usersRes)
				? (usersRes.message ?? null)
				: !isSuccess(momentsRes)
					? (momentsRes.message ?? null)
					: !isSuccess(groupsRes)
						? (groupsRes.message ?? null)
						: null;

			users = nextUsers;
			moments = nextMoments;
			groups = nextGroups;
			searchError = nextError;

			if (nextError && nextUsers.length === 0 && nextMoments.length === 0 && nextGroups.length === 0) {
				toast.error(nextError);
			}
		} catch {
			if (!controller.signal.aborted) {
				searchError = t('searchPage.searchFailed');
				toast.error(t('searchPage.searchFailed'));
			}
		} finally {
			if (!controller.signal.aborted) {
				isLoading = false;
				searchAbortController = null;
			}
		}
	}

	async function handleSearch() {
		const searchTerm = keywordInput.trim();
		if (!searchTerm) {
			clearSearch();
			return;
		}

		await goto(buildSearchUrl('keyword', searchTerm, currentTab), {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
	}

	function clearSearch() {
		keywordInput = '';
		resetResults();
		void goto('/search', {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
	}

	async function handleTabChange(value: string) {
		const nextTab = parseSearchTab(value);
		if (nextTab === currentTab || !activeMode || !activeTerm) {
			return;
		}

		await goto(buildSearchUrl(activeMode, activeTerm, nextTab), {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
	}

	function retryActiveSearch() {
		if (!activeMode || !activeTerm) {
			return;
		}

		lastSearchFingerprint = '';
		void performSearch(activeMode, activeTerm);
	}

	$effect(() => {
		if (keywordInput !== activeTerm) {
			keywordInput = activeTerm;
		}
	});

	$effect(() => {
		if (!activeMode || !activeTerm) {
			resetResults();
			return;
		}

		void performSearch(activeMode, activeTerm);
	});

	$effect(() => {
		const anchorUpdatedAt = mapAnchorStore.anchor.updatedAt;
		if (!activeMode || !activeTerm) {
			lastAnchorUpdatedAt = anchorUpdatedAt;
			return;
		}
		if (anchorUpdatedAt === lastAnchorUpdatedAt) {
			return;
		}

		lastAnchorUpdatedAt = anchorUpdatedAt;
		lastSearchFingerprint = '';
		void performSearch(activeMode, activeTerm);
	});

	$effect(() => {
		return () => {
			searchAbortController?.abort();
		};
	});
</script>

<svelte:head>
	<title>搜索 - HotSpot</title>
</svelte:head>

<div class="space-y-4 sm:space-y-5">
	<!-- Sticky header -->
	<div
		class="sticky top-0 z-10 flex items-center justify-between px-4 py-3"
		style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
	>
		<div>
			<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">{t('searchPage.title')}</h1>
			<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">{t('searchPage.description')}</p>
		</div>
		<button
			class="flex items-center gap-2 rounded-full px-4 py-2 font-medium outline-none"
			style="box-shadow: 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
			onclick={() => goto(nearbyMomentsUrl)}
		>
			<Compass size={16} />
			{t('searchPage.nearbyExplore')}
		</button>
	</div>

	<AnchorSummaryCard
		label={t('searchPage.anchorLabel')}
		description={t('searchPage.anchorDescription', { radius: mapAnchorStore.radius })}
		subtle
	/>

	<!-- Search form -->
	<form
		class="rounded-[22px] p-3"
		style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
		onsubmit={(event) => {
			event.preventDefault();
			handleSearch();
		}}
	>
		<div class="flex flex-col gap-2.5 sm:flex-row sm:items-center">
			<div class="relative flex-1">
				<Search size={16} style="position: absolute; left: 12px; top: 50%; transform: translateY(-50%); color: var(--neu-text-muted);" />
				<input
					type="text"
					placeholder={t('searchPage.placeholder')}
					class="w-full rounded-full py-2.5 pl-9 pr-9 outline-none"
					style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text); font-size: 14px;"
					autocomplete="off"
					bind:value={keywordInput}
				/>
				{#if keywordInput}
					<button
						type="button"
						class="absolute right-3 top-1/2 flex items-center justify-center rounded-full outline-none"
						style="width: 24px; height: 24px; transform: translateY(-50%); background: transparent; color: var(--neu-text-muted);"
						onclick={clearSearch}
						aria-label={t('searchPage.clear')}
					>
						<X size={12} />
					</button>
				{/if}
			</div>
			<button
				type="submit"
				class="flex items-center justify-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none"
				style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);"
				disabled={!keywordInput.trim()}
			>
				<Search size={16} />
				{t('searchPage.submit')}
			</button>
		</div>
	</form>

	{#if activeTerm}
		<!-- Result summary card -->
		<div
			class="rounded-[22px] p-4"
			style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
		>
			<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div class="min-w-0 space-y-1">
					<div class="flex flex-wrap items-center gap-2" style="font-size: 14px; color: var(--neu-text-muted);">
						{#if activeMode === 'tag'}
							<span
								class="inline-flex items-center gap-1 rounded-full px-2.5 py-1"
								style="font-weight: 500; color: var(--neu-accent-primary); background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
							>
								<Hash size={14} />
								{t('searchPage.tagMode')}
							</span>
						{:else}
							<span
								class="inline-flex items-center rounded-full px-2.5 py-1"
								style="font-weight: 500; color: var(--neu-text); background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
							>
								{t('searchPage.keywordMode')}
							</span>
						{/if}
						<span>{t('searchPage.range', { radius: mapAnchorStore.radius })}</span>
					</div>
					<div class="truncate" style="font-size: 18px; font-weight: 600; color: var(--neu-text);">{t('searchPage.resultTitle', { term: activeLabel })}</div>
				</div>
				<div class="grid grid-cols-3 gap-2 text-center sm:min-w-72">
					<div
						class="rounded-[14px] px-3 py-2"
						style="box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
					>
						<div style="font-weight: 600; color: var(--neu-text);">{users.length}</div>
						<div style="font-size: 12px; color: var(--neu-text-muted);">用户</div>
					</div>
					<div
						class="rounded-[14px] px-3 py-2"
						style="box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
					>
						<div style="font-weight: 600; color: var(--neu-text);">{moments.length}</div>
						<div style="font-size: 12px; color: var(--neu-text-muted);">瞬间</div>
					</div>
					<div
						class="rounded-[14px] px-3 py-2"
						style="box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
					>
						<div style="font-weight: 600; color: var(--neu-text);">{groups.length}</div>
						<div style="font-size: 12px; color: var(--neu-text-muted);">群组</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Tab bar -->
		<div class="flex gap-2">
			{#each [
				{ value: 'users', label: `用户 (${users.length})`, icon: Users },
				{ value: 'moments', label: `瞬间 (${moments.length})`, icon: Image },
				{ value: 'groups', label: `群组 (${groups.length})`, icon: UsersRound }
			] as tab (tab.value)}
				<button
					class="flex items-center gap-2 rounded-full px-4 py-2 font-medium outline-none transition-all"
					style={currentTab === tab.value
						? 'background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 12px var(--neu-glow);'
						: 'background: var(--neu-bg); color: var(--neu-text-muted); box-shadow: 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);'}
					onclick={() => handleTabChange(tab.value)}
				>
					<tab.icon size={16} />
					<span style="font-size: 13px;">{tab.label}</span>
				</button>
			{/each}
		</div>

		<!-- Tab content: Users -->
		{#if currentTab === 'users'}
			{#if isLoading}
				<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
					{#each Array(6) as _, i (i)}
						<MasonryLayoutItem>
							<div class="h-48 animate-pulse rounded-[22px]" style="background: var(--neu-border);"></div>
						</MasonryLayoutItem>
					{/each}
				</MasonryLayout>
			{:else if searchError && !hasAnyResults}
				<EmptyState
					icon={AlertTriangle}
					title="搜索失败"
					description={searchError}
					actionText="重试"
					onAction={retryActiveSearch}
				/>
			{:else if users.length === 0}
				<EmptyState
					icon={Users}
					title="未找到相关用户"
					description={`换个关键词，或调整 "${activeLabel}" 的搜索方式后再试试。`}
				/>
			{:else}
				<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
					{#each users as user (user.user.user_id)}
						<MasonryLayoutItem>
							<UserCard
								{user}
								showBio
								showTags
								showDistance
								variant="square"
								class="var(--neu-border) var(--neu-bg)"
							/>
						</MasonryLayoutItem>
					{/each}
				</MasonryLayout>
			{/if}
		{/if}

		<!-- Tab content: Moments -->
		{#if currentTab === 'moments'}
			{#if isLoading}
				<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={20}>
					{#each Array(6) as _, i (i)}
						<MasonryLayoutItem>
							<div class="h-64 animate-pulse rounded-[22px]" style="background: var(--neu-border);"></div>
						</MasonryLayoutItem>
					{/each}
				</MasonryLayout>
			{:else if searchError && !hasAnyResults}
				<EmptyState
					icon={AlertTriangle}
					title="搜索失败"
					description={searchError}
					actionText="重试"
					onAction={retryActiveSearch}
				/>
			{:else if moments.length === 0}
				<EmptyState
					icon={Image}
					title="未找到相关瞬间"
					description={`当前没有与 "${activeLabel}" 匹配的动态内容。`}
				/>
			{:else}
				<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={20}>
					{#each moments as moment (moment.moment_id)}
						<MasonryLayoutItem>
							<MomentCard
								{moment}
								class="var(--neu-border) var(--neu-bg)"
							/>
						</MasonryLayoutItem>
					{/each}
				</MasonryLayout>
			{/if}
		{/if}

		<!-- Tab content: Groups -->
		{#if currentTab === 'groups'}
			{#if isLoading}
				<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
					{#each Array(6) as _, i (i)}
						<MasonryLayoutItem>
							<div class="h-52 animate-pulse rounded-[22px]" style="background: var(--neu-border);"></div>
						</MasonryLayoutItem>
					{/each}
				</MasonryLayout>
			{:else if searchError && !hasAnyResults}
				<EmptyState
					icon={AlertTriangle}
					title="搜索失败"
					description={searchError}
					actionText="重试"
					onAction={retryActiveSearch}
				/>
			{:else if groups.length === 0}
				<EmptyState
					icon={UsersRound}
					title="未找到相关群组"
					description={`附近暂时没有与 "${activeLabel}" 相关的群组。`}
				/>
			{:else}
				<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
					{#each groups as group (group.group.group_id)}
						<MasonryLayoutItem>
							<GroupCard
								{group}
								showMembers
								showDistance
								variant="square"
								class="var(--neu-border) var(--neu-bg)"
							/>
						</MasonryLayoutItem>
					{/each}
				</MasonryLayout>
			{/if}
		{/if}
	{:else}
		<!-- Empty search state -->
		<div
			class="flex flex-col items-center justify-center gap-3 rounded-[22px] py-12 text-center"
			style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
		>
			<Search size={48} color="var(--neu-text-muted)" />
			<div class="space-y-1">
				<p style="font-weight: 500; color: var(--neu-text);">输入关键词开始搜索</p>
				<p style="font-size: 14px; color: var(--neu-text-muted);">支持关键词搜索，也可以从标签链接直接进入对应结果页。</p>
			</div>
		</div>
	{/if}
</div>
