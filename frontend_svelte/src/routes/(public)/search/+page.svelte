<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { browser } from '$app/environment';
	import { t } from '$lib/i18n';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { usersService, momentsService, groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import UserCard from '$lib/components/common/user-card.svelte';
	import GroupCard from '$lib/components/common/group-card.svelte';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import type { UserItem, MomentItem, GroupItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import Search from 'lucide-svelte/icons/search';
	import Clock from 'lucide-svelte/icons/clock';
	import TrendingUp from 'lucide-svelte/icons/trending-up';
	import Hash from 'lucide-svelte/icons/hash';
	import X from 'lucide-svelte/icons/x';

	type SearchMode = 'keyword' | 'tag';
	type SearchTab = 'all' | 'users' | 'moments' | 'groups';

	const SEARCH_TABS: SearchTab[] = ['all', 'users', 'moments', 'groups'];
	const HOT_SEARCHES = ['植物园', '云龙湖', '周末市集', '咖啡馆', '摄影'];
	const SUGGESTED_TAGS = ['摄影', '城市探索', '周末去哪', '隐藏咖啡馆', '徐州美食', '本地生活', '户外运动', '街头摄影'];

	const STORAGE_KEY_RECENT = 'hotspot_recent_searches';
	const MAX_RECENT = 6;

	let keywordInput = $state('');
	let isLoading = $state(false);
	let searchError = $state<string | null>(null);
	let recentSearches = $state<string[]>(loadRecent());

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
	const hasAnyResults = $derived(users.length + moments.length + groups.length > 0);

	function loadRecent(): string[] {
		if (!browser) return [];
		try {
			const raw = localStorage.getItem(STORAGE_KEY_RECENT);
			return raw ? JSON.parse(raw) : [];
		} catch {
			return [];
		}
	}

	function saveRecent(items: string[]) {
		if (!browser) return;
		try { localStorage.setItem(STORAGE_KEY_RECENT, JSON.stringify(items)); } catch { /* ignore */ }
	}

	function addRecent(term: string) {
		const next = [term, ...recentSearches.filter(r => r !== term)].slice(0, MAX_RECENT);
		recentSearches = next;
		saveRecent(next);
	}

	function clearRecent() {
		recentSearches = [];
		saveRecent([]);
	}

	function getReferencePoint() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	function parseSearchTab(value: string | null): SearchTab {
		return SEARCH_TABS.includes(value as SearchTab) ? (value as SearchTab) : 'all';
	}

	function buildSearchFingerprint(mode: SearchMode, value: string): string {
		const point = getReferencePoint();
		return `${mode}:${value}:${point.latitude.toFixed(6)}:${point.longitude.toFixed(6)}:${mapAnchorStore.radius}`;
	}

	function buildSearchUrl(mode: SearchMode, value: string, tab: SearchTab): string {
		const params = new URLSearchParams();
		params.set(mode === 'tag' ? 'tag' : 'q', value);
		if (tab !== 'all') {
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
				? { tag: normalized, reference_point: getReferencePoint() }
				: { keyword: normalized, reference_point: getReferencePoint() };

			const [usersRes, momentsRes, groupsRes] = await withMinimumLoadingTime(() =>
				Promise.all([
					usersService.search(request, { signal }),
					momentsService.search(request, { signal }),
					groupsService.search(request, { signal })
				])
			);

			if (signal.aborted) return;

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

		addRecent(searchTerm);
		await goto(buildSearchUrl('keyword', searchTerm, currentTab), {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
	}

	function searchFor(term: string) {
		keywordInput = term;
		addRecent(term);
		void goto(buildSearchUrl('keyword', term, 'all'), {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
	}

	function searchTag(tag: string) {
		addRecent(tag);
		void goto(buildSearchUrl('tag', tag, 'all'), {
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
		if (nextTab === currentTab || !activeMode || !activeTerm) return;

		await goto(buildSearchUrl(activeMode, activeTerm, nextTab), {
			replaceState: true,
			noScroll: true,
			keepFocus: true
		});
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
		if (anchorUpdatedAt === lastAnchorUpdatedAt) return;

		lastAnchorUpdatedAt = anchorUpdatedAt;
		lastSearchFingerprint = '';
		void performSearch(activeMode, activeTerm);
	});

	$effect(() => {
		return () => { searchAbortController?.abort(); };
	});
</script>

<svelte:head>
	<title>搜索 - HotSpot</title>
</svelte:head>

<div class="flex flex-col" style="height: 100%; background-color: var(--neu-bg);">
	<!-- Search header — back button + search input -->
	<div
		class="flex-shrink-0 flex items-center gap-3 px-5 py-3"
		style="border-bottom: 1px solid var(--neu-border); background-color: var(--neu-bg);"
	>
		<button
			class="flex items-center justify-center rounded-full outline-none flex-shrink-0 transition-transform duration-150 active:scale-[0.88]"
			style="width: 36px; height: 36px; background: var(--neu-bg); box-shadow: 4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
			onclick={() => history.back()}
			aria-label="返回"
		>
			<ArrowLeft size={15} style="color: var(--neu-text-muted);" strokeWidth={1.5} />
		</button>

		<div
			class="flex-1 flex items-center gap-2.5 px-4 rounded-[18px] neu-input-surface"
			style="height: 38px;"
		>
			<Search size={14} style="flex-shrink: 0; color: var(--neu-text-muted);" strokeWidth={1.5} />
			<input
				type="text"
				placeholder="搜索用户、地点、话题…"
				class="flex-1 bg-transparent outline-none"
				style="font-size: 13px; color: var(--neu-text);"
				autocomplete="off"
				bind:value={keywordInput}
				onkeydown={(e) => { if (e.key === 'Enter' && keywordInput.trim()) handleSearch(); }}
			/>
			{#if keywordInput}
				<button
					type="button"
					class="flex-shrink-0 outline-none transition-transform duration-150 active:scale-[0.85]"
					onclick={clearSearch}
					aria-label={t('searchPage.clear')}
				>
					<X size={13} style="color: var(--neu-text-muted);" strokeWidth={2} />
				</button>
			{/if}
		</div>
	</div>

	<!-- Scrollable body -->
	<div class="flex-1 overflow-y-auto no-scrollbar">
		{#if !activeTerm}
			<!-- No query: discovery state -->
			<div class="px-5 pt-4 pb-6" style="max-width: 640px; margin-left: auto; margin-right: auto; width: 100%;">
				<!-- Recent searches -->
				{#if recentSearches.length > 0}
					<div class="mb-5">
						<div class="flex items-center justify-between mb-2.5">
							<div class="flex items-center gap-1.5">
								<Clock size={13} style="color: var(--neu-text-muted);" strokeWidth={1.5} />
								<p class="uppercase" style="font-size: 10px; letter-spacing: 0.12em; color: var(--neu-text-muted);">最近搜索</p>
							</div>
							<button class="outline-none transition-transform duration-150 active:scale-[0.93]" style="font-size: 12px; color: var(--neu-accent-primary);" onclick={clearRecent}>清空</button>
						</div>
						<div class="flex flex-wrap gap-2">
							{#each recentSearches as r (r)}
								<button
									class="flex items-center gap-1.5 px-3.5 py-1.5 rounded-full outline-none transition-transform duration-150 active:scale-[0.93]"
									style="background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: 3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light); font-size: 12px; color: var(--neu-text-muted);"
									onclick={() => searchFor(r)}
								>
									<Clock size={10} strokeWidth={1.5} />
									{r}
								</button>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Hot searches -->
				<div class="mb-5">
					<div class="flex items-center gap-1.5 mb-2.5">
						<TrendingUp size={13} style="color: var(--neu-accent-primary);" strokeWidth={2} />
						<p class="uppercase" style="font-size: 10px; letter-spacing: 0.12em; color: var(--neu-text-muted);">热门搜索</p>
					</div>
					{#each HOT_SEARCHES as h, i (h)}
						<button
							class="flex items-center gap-3 w-full py-2.5 outline-none transition-transform duration-150 active:scale-[0.98]"
							style="border-bottom: {i < HOT_SEARCHES.length - 1 ? '1px solid var(--neu-border)' : 'none'};"
							onclick={() => searchFor(h)}
						>
							<span style="font-size: 14px; color: var(--neu-accent-primary); font-weight: 700; width: 20px; text-align: center;">{i + 1}</span>
							<span style="font-size: 14px; color: var(--neu-text);">{h}</span>
							{#if i < 3}
								<span class="ml-auto px-2 py-0.5 rounded-full" style="font-size: 10px; background: rgba(239,68,68,0.12); color: #EF4444;">热</span>
							{/if}
						</button>
					{/each}
				</div>

				<!-- Tag suggestions -->
				<div>
					<div class="flex items-center gap-1.5 mb-2.5">
						<Hash size={13} style="color: var(--neu-accent-primary);" strokeWidth={2} />
						<p class="uppercase" style="font-size: 10px; letter-spacing: 0.12em; color: var(--neu-text-muted);">热门话题</p>
					</div>
					<div class="flex flex-wrap gap-2">
						{#each SUGGESTED_TAGS as tag (tag)}
							<button
								class="px-3.5 py-1.5 rounded-full outline-none transition-transform duration-150 active:scale-[0.93]"
								style="background: var(--neu-bg); color: var(--neu-accent-primary); font-size: 12px; box-shadow: 3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
								onclick={() => searchTag(tag)}
							>
								#{tag}
							</button>
						{/each}
					</div>
				</div>
			</div>
		{:else}
			<!-- Has query: results -->
			<div style="max-width: 640px; margin-left: auto; margin-right: auto; width: 100%;">
				<!-- Result tabs -->
				<div class="flex overflow-x-auto no-scrollbar px-5 pt-3 pb-0 gap-2">
					{#each [
						{ value: 'all', label: '全部' },
						{ value: 'users', label: `用户${users.length ? ` (${users.length})` : ''}` },
						{ value: 'moments', label: `动态${moments.length ? ` (${moments.length})` : ''}` },
						{ value: 'groups', label: `群组${groups.length ? ` (${groups.length})` : ''}` }
					] as tab (tab.value)}
						<button
							class="flex-shrink-0 px-3.5 py-1.5 rounded-full outline-none transition-transform duration-150 active:scale-[0.92]"
							style={currentTab === tab.value
								? 'background: var(--neu-accent-primary); color: white; font-size: 12px; box-shadow: 0 3px 10px var(--neu-glow);'
								: 'background: var(--neu-bg); color: var(--neu-text-muted); font-size: 12px; border: 1px solid var(--neu-border); box-shadow: 3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light);'}
							onclick={() => handleTabChange(tab.value)}
						>
							{tab.label}
						</button>
					{/each}
				</div>

				<!-- Results content -->
				<div class="px-5 pt-4 pb-6 flex flex-col gap-4">
					<!-- No results -->
					{#if !isLoading && !searchError && !hasAnyResults}
						<div class="flex flex-col items-center justify-center py-16 gap-3">
							<Search size={36} style="color: var(--neu-text-muted); opacity: 0.4;" strokeWidth={1.25} />
							<p style="font-size: 15px; color: var(--neu-text);">未找到相关内容</p>
							<p style="font-size: 13px; color: var(--neu-text-muted);">尝试其他关键词</p>
						</div>
					{/if}

					<!-- Users section -->
					{#if (currentTab === 'all' || currentTab === 'users') && (isLoading || users.length > 0)}
						<div>
							{#if currentTab === 'all' && users.length > 0}
								<p class="uppercase mb-2.5" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);">用户</p>
							{/if}
							<div class="flex flex-col gap-2.5">
								{#if isLoading && users.length === 0}
									{#each Array(3) as _, i (i)}
										<div class="h-16 animate-pulse rounded-[20px]" style="background: var(--neu-border);"></div>
									{/each}
								{:else}
									{#each users as user (user.user.user_id)}
										<UserCard {user} showBio showTags showDistance variant="compact" />
									{/each}
								{/if}
							</div>
						</div>
					{/if}

					<!-- Moments section -->
					{#if (currentTab === 'all' || currentTab === 'moments') && (isLoading || moments.length > 0)}
						<div>
							{#if currentTab === 'all' && moments.length > 0}
								<p class="uppercase mb-2.5" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);">动态</p>
							{/if}
							<div class="flex flex-col gap-2.5">
								{#if isLoading && moments.length === 0}
									{#each Array(3) as _, i (i)}
										<div class="h-20 animate-pulse rounded-[20px]" style="background: var(--neu-border);"></div>
									{/each}
								{:else}
									{#each moments as moment (moment.moment_id)}
										<MomentCard {moment} />
									{/each}
								{/if}
							</div>
						</div>
					{/if}

					<!-- Groups section -->
					{#if (currentTab === 'all' || currentTab === 'groups') && (isLoading || groups.length > 0)}
						<div>
							{#if currentTab === 'all' && groups.length > 0}
								<p class="uppercase mb-2.5" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);">群组</p>
							{/if}
							<div class="flex flex-col gap-2.5">
								{#if isLoading && groups.length === 0}
									{#each Array(3) as _, i (i)}
										<div class="h-16 animate-pulse rounded-[20px]" style="background: var(--neu-border);"></div>
									{/each}
								{:else}
									{#each groups as group (group.group.group_id)}
										<GroupCard {group} showMembers showDistance variant="compact" />
									{/each}
								{/if}
							</div>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</div>
</div>
