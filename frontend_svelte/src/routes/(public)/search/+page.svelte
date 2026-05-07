<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { t } from '$lib/i18n';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { usersService, momentsService, groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import AnchorSummaryCard from '$lib/components/common/anchor-summary-card.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import UserCard from '$lib/components/common/user-card.svelte';
	import GroupCard from '$lib/components/common/group-card.svelte';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import type { UserItem, MomentItem, GroupItem } from '$lib/types/api';
	import Search from 'lucide-svelte/icons/search';
	import Compass from 'lucide-svelte/icons/compass';
	import Users from 'lucide-svelte/icons/users';
	import Image from 'lucide-svelte/icons/image';
	import UsersRound from 'lucide-svelte/icons/users-round';
	import Hash from 'lucide-svelte/icons/hash';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';
	import LogIn from 'lucide-svelte/icons/log-in';
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

<div class="flex flex-col" style="height: 100%;">
	<!-- Search header -->
	<div
		class="flex-shrink-0 flex items-center gap-3 px-5 py-3"
		style="border-bottom: 1px solid var(--neu-border); background: var(--neu-bg);"
	>
		<div class="min-w-0 flex-1">
			<div class="flex items-center justify-between">
				<h1 style="font-size: 18px; font-weight: 600; color: var(--neu-text);">{t('searchPage.title')}</h1>
				<button
					class="flex items-center gap-1.5 rounded-full px-3.5 py-1.5 outline-none"
					style="box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg); font-size: 12px;"
					onclick={() => goto(nearbyMomentsUrl)}
				>
					<Compass size={13} />
					{t('searchPage.nearbyExplore')}
				</button>
			</div>
		</div>
	</div>

	<!-- Scrollable body -->
	<div class="flex-1 overflow-y-auto no-scrollbar">
		<div style="max-width: 640px; margin: 0 auto; width: 100%;">
			<!-- Search form -->
			<form
				class="px-5 pt-4 pb-3"
				onsubmit={(event) => {
					event.preventDefault();
					handleSearch();
				}}
			>
				<div class="flex items-center gap-2.5">
					<div class="relative flex-1 flex items-center gap-2.5 rounded-[18px] px-4"
						style="height: 38px; background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
					>
						<Search size={14} style="flex-shrink: 0; color: var(--neu-text-muted);" strokeWidth={1.5} />
						<input
							type="text"
							placeholder={t('searchPage.placeholder')}
							class="flex-1 bg-transparent outline-none"
							style="font-size: 13px; color: var(--neu-text);"
							autocomplete="off"
							bind:value={keywordInput}
						/>
						{#if keywordInput}
							<button
								type="button"
								class="flex-shrink-0 outline-none"
								onclick={clearSearch}
								aria-label={t('searchPage.clear')}
							>
								<X size={13} style="color: var(--neu-text-muted);" strokeWidth={2} />
							</button>
						{/if}
					</div>
					<button
						type="submit"
						class="flex items-center justify-center gap-2 rounded-full px-5 py-2 font-medium outline-none"
						style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 3px 10px var(--neu-glow); font-size: 13px; height: 38px;"
						disabled={!keywordInput.trim()}
					>
						{t('searchPage.submit')}
					</button>
				</div>
			</form>

			{#if activeTerm}
				<!-- Result summary -->
				<div class="px-5 pb-3">
					<div class="flex items-center gap-2" style="font-size: 13px; color: var(--neu-text-muted);">
						{#if activeMode === 'tag'}
							<span
								class="inline-flex items-center gap-1 rounded-full px-2.5 py-1"
								style="font-weight: 500; color: var(--neu-accent-primary); background: var(--neu-bg); box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border); font-size: 11px;"
							>
								<Hash size={12} />
								{t('searchPage.tagMode')}
							</span>
						{:else}
							<span
								class="inline-flex items-center rounded-full px-2.5 py-1"
								style="font-weight: 500; color: var(--neu-text); background: var(--neu-bg); box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border); font-size: 11px;"
							>
								{t('searchPage.keywordMode')}
							</span>
						{/if}
						<span>{t('searchPage.range', { radius: mapAnchorStore.radius })}</span>
					</div>
					<p class="mt-1 truncate" style="font-size: 16px; font-weight: 600; color: var(--neu-text);">{t('searchPage.resultTitle', { term: activeLabel })}</p>
				</div>

				<!-- Tab bar -->
				<div class="flex gap-2 px-5 pt-2 pb-0">
					{#each [
						{ value: 'users', label: `用户 (${users.length})`, icon: Users },
						{ value: 'moments', label: `瞬间 (${moments.length})`, icon: Image },
						{ value: 'groups', label: `群组 (${groups.length})`, icon: UsersRound }
					] as tab (tab.value)}
						<button
							class="flex-shrink-0 rounded-full px-3.5 py-1.5 outline-none"
							style={currentTab === tab.value
								? 'background: var(--neu-accent-primary); color: white; font-size: 12px; box-shadow: 0 3px 10px var(--neu-glow);'
								: 'background: var(--neu-bg); color: var(--neu-text-muted); font-size: 12px; border: 1px solid var(--neu-border); box-shadow: var(--neu-shadow-sm);'}
							onclick={() => handleTabChange(tab.value)}
						>
							{tab.label}
						</button>
					{/each}
				</div>

				<!-- Results -->
				<div class="px-5 pt-4 pb-6 flex flex-col gap-3">
					{#if currentTab === 'users'}
						{#if isLoading}
							{#each Array(4) as _, i (i)}
								<div class="h-16 animate-pulse rounded-[20px]" style="background: var(--neu-border);"></div>
							{/each}
						{:else if searchError && !hasAnyResults}
							{#if !authStore.isAuthenticated}
								<EmptyState
									icon={LogIn}
									title="登录后查看"
									description="登录后即可使用完整搜索功能"
									actionText="立即登录"
									onAction={() => uiStore.openLoginModalWithRedirect({})}
								/>
							{:else}
								<EmptyState
									icon={AlertTriangle}
									title="搜索失败"
									description={searchError}
									actionText="重试"
									onAction={retryActiveSearch}
								/>
							{/if}
						{:else if users.length === 0}
							<EmptyState
								icon={Users}
								title="未找到相关用户"
								description={`换个关键词，或调整 "${activeLabel}" 的搜索方式后再试试。`}
							/>
						{:else}
							{#each users as user (user.user.user_id)}
								<UserCard {user} showBio showTags showDistance variant="compact" />
							{/each}
						{/if}
					{/if}

					{#if currentTab === 'moments'}
						{#if isLoading}
							{#each Array(4) as _, i (i)}
								<div class="h-20 animate-pulse rounded-[20px]" style="background: var(--neu-border);"></div>
							{/each}
						{:else if searchError && !hasAnyResults}
							{#if !authStore.isAuthenticated}
								<EmptyState
									icon={LogIn}
									title="登录后查看"
									description="登录后即可使用完整搜索功能"
									actionText="立即登录"
									onAction={() => uiStore.openLoginModalWithRedirect({})}
								/>
							{:else}
								<EmptyState
									icon={AlertTriangle}
									title="搜索失败"
									description={searchError}
									actionText="重试"
									onAction={retryActiveSearch}
								/>
							{/if}
						{:else if moments.length === 0}
							<EmptyState
								icon={Image}
								title="未找到相关瞬间"
								description={`当前没有与 "${activeLabel}" 匹配的动态内容。`}
							/>
						{:else}
							{#each moments as moment (moment.moment_id)}
								<MomentCard {moment} />
							{/each}
						{/if}
					{/if}

					{#if currentTab === 'groups'}
						{#if isLoading}
							{#each Array(4) as _, i (i)}
								<div class="h-16 animate-pulse rounded-[20px]" style="background: var(--neu-border);"></div>
							{/each}
						{:else if searchError && !hasAnyResults}
							{#if !authStore.isAuthenticated}
								<EmptyState
									icon={LogIn}
									title="登录后查看"
									description="登录后即可使用完整搜索功能"
									actionText="立即登录"
									onAction={() => uiStore.openLoginModalWithRedirect({})}
								/>
							{:else}
								<EmptyState
									icon={AlertTriangle}
									title="搜索失败"
									description={searchError}
									actionText="重试"
									onAction={retryActiveSearch}
								/>
							{/if}
						{:else if groups.length === 0}
							<EmptyState
								icon={UsersRound}
								title="未找到相关群组"
								description={`附近暂时没有与 "${activeLabel}" 相关的群组。`}
							/>
						{:else}
							{#each groups as group (group.group.group_id)}
								<GroupCard {group} showMembers showDistance variant="compact" />
							{/each}
						{/if}
					{/if}
				</div>
			{:else}
				<!-- Empty search state -->
				<div class="flex flex-col items-center justify-center gap-3 px-5 py-16 text-center">
					<Search size={36} style="color: var(--neu-text-muted); opacity: 0.4;" strokeWidth={1.25} />
					<p style="font-size: 15px; color: var(--neu-text);">输入关键词开始搜索</p>
					<p style="font-size: 13px; color: var(--neu-text-muted);">支持关键词搜索，也可以从标签链接直接进入对应结果页。</p>
				</div>

				<!-- Anchor info when no search -->
				<div class="px-5 pb-6">
					<AnchorSummaryCard
						label={t('searchPage.anchorLabel')}
						description={t('searchPage.anchorDescription', { radius: mapAnchorStore.radius })}
						subtle
					/>
				</div>
			{/if}
		</div>
	</div>
</div>
