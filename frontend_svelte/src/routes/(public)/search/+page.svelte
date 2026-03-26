<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { usersService, momentsService, groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import AnchorSummaryCard from '$lib/components/common/anchor-summary-card.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Card, CardContent } from '$lib/components/ui/card';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
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
				searchError = '搜索失败';
				toast.error('搜索失败');
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

<div class="space-y-5">
	<PageHeader title="搜索" description="发现用户、群组与附近动态" />

	<AnchorSummaryCard
		label="搜索参考锚点"
		description={`当前按 ${mapAnchorStore.radius}km 范围参与距离排序`}
		subtle
	>
		{#snippet actions()}
			<div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center">
				<Button variant="outline" size="sm" class="gap-2" onclick={() => goto(nearbyMomentsUrl)}>
					<Compass class="h-4 w-4" />
					附近探索
				</Button>
			</div>
		{/snippet}
	</AnchorSummaryCard>

	<form
		class="rounded-3xl border border-border/60 bg-card/85 p-3 shadow-sm backdrop-blur-md"
		onsubmit={(event) => {
			event.preventDefault();
			handleSearch();
		}}
	>
		<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
			<div class="relative flex-1">
				<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
				<Input
					type="text"
					placeholder="搜索用户、瞬间、群组..."
					class="h-11 rounded-2xl border-border/60 bg-background/70 pl-9 pr-9"
					autocomplete="off"
					bind:value={keywordInput}
				/>
				{#if keywordInput}
					<Button
						type="button"
						variant="ghost"
						size="icon"
						class="absolute right-3 top-1/2 h-6 w-6 -translate-y-1/2 rounded-full border-none bg-transparent text-muted-foreground shadow-none hover:text-foreground"
						onclick={clearSearch}
						aria-label="清除搜索"
					>
						<X class="h-3 w-3" />
					</Button>
				{/if}
			</div>
			<Button
				type="submit"
				class="h-11 rounded-2xl border border-border/60 bg-background/70 px-5 text-foreground hover:bg-accent/70 sm:min-w-28"
				disabled={!keywordInput.trim()}
			>
				<Search class="h-4 w-4" />
				搜索
			</Button>
		</div>
	</form>

	{#if activeTerm}
		<Card class="rounded-3xl border border-border/60 bg-card/85 shadow-sm backdrop-blur-md">
			<CardContent class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
				<div class="min-w-0 space-y-1">
					<div class="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
						{#if activeMode === 'tag'}
							<span class="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2.5 py-1 font-medium text-primary">
								<Hash class="h-3.5 w-3.5" />
								标签搜索
							</span>
						{:else}
							<span class="inline-flex items-center rounded-full bg-muted px-2.5 py-1 font-medium text-foreground">
								关键词搜索
							</span>
						{/if}
						<span>搜索范围 {mapAnchorStore.radius}km</span>
					</div>
					<div class="truncate text-lg font-semibold">“{activeLabel}” 的搜索结果</div>
				</div>
				<div class="grid grid-cols-3 gap-2 text-center text-sm sm:min-w-72">
					<div class="rounded-2xl bg-muted/70 px-3 py-2">
						<div class="font-semibold text-foreground">{users.length}</div>
						<div class="text-xs text-muted-foreground">用户</div>
					</div>
					<div class="rounded-2xl bg-muted/70 px-3 py-2">
						<div class="font-semibold text-foreground">{moments.length}</div>
						<div class="text-xs text-muted-foreground">瞬间</div>
					</div>
					<div class="rounded-2xl bg-muted/70 px-3 py-2">
						<div class="font-semibold text-foreground">{groups.length}</div>
						<div class="text-xs text-muted-foreground">群组</div>
					</div>
				</div>
			</CardContent>
		</Card>

		<Tabs value={currentTab} onValueChange={handleTabChange}>
			<TabsList class="h-auto flex-wrap gap-1.5 rounded-3xl bg-card/80 p-1.5 shadow-md shadow-black/10 ring-1 ring-border/50 backdrop-blur-sm dark:shadow-black/30">
				<TabsTrigger value="users" class="gap-2 rounded-2xl">
					<Users class="h-4 w-4" />
					用户 ({users.length})
				</TabsTrigger>
				<TabsTrigger value="moments" class="gap-2 rounded-2xl">
					<Image class="h-4 w-4" />
					瞬间 ({moments.length})
				</TabsTrigger>
				<TabsTrigger value="groups" class="gap-2 rounded-2xl">
					<UsersRound class="h-4 w-4" />
					群组 ({groups.length})
				</TabsTrigger>
			</TabsList>

			<TabsContent value="users" class="mt-6">
				{#if isLoading}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
						{#each Array(6) as _, i (i)}
							<MasonryLayoutItem>
								<div class="h-48 animate-pulse rounded-2xl bg-muted"></div>
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
						description={`换个关键词，或调整 “${activeLabel}” 的搜索方式后再试试。`}
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
									class="border-border/60 bg-card/95 shadow-md shadow-black/5 dark:shadow-black/30"
								/>
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{/if}
			</TabsContent>

			<TabsContent value="moments" class="mt-6">
				{#if isLoading}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={20}>
						{#each Array(6) as _, i (i)}
							<MasonryLayoutItem>
								<div class="h-64 animate-pulse rounded-2xl bg-muted"></div>
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
						description={`当前没有与 “${activeLabel}” 匹配的动态内容。`}
					/>
				{:else}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={20}>
						{#each moments as moment (moment.moment_id)}
							<MasonryLayoutItem>
								<MomentCard
									{moment}
									class="border-border/60 bg-card/95 shadow-md shadow-black/5 dark:shadow-black/30"
								/>
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{/if}
			</TabsContent>

			<TabsContent value="groups" class="mt-6">
				{#if isLoading}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
						{#each Array(6) as _, i (i)}
							<MasonryLayoutItem>
								<div class="h-52 animate-pulse rounded-2xl bg-muted"></div>
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
						description={`附近暂时没有与 “${activeLabel}” 相关的群组。`}
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
									class="border-border/60 bg-card/95 shadow-md shadow-black/5 dark:shadow-black/30"
								/>
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{/if}
			</TabsContent>
		</Tabs>
	{:else}
		<Card class="rounded-3xl border border-border/60 bg-linear-to-br from-primary/5 via-background to-primary/10 shadow-sm">
			<CardContent class="flex flex-col items-center justify-center gap-3 py-12 text-center">
				<Search class="h-12 w-12 text-muted-foreground" />
				<div class="space-y-1">
					<p class="font-medium">输入关键词开始搜索</p>
					<p class="text-sm text-muted-foreground">支持关键词搜索，也可以从标签链接直接进入对应结果页。</p>
				</div>
			</CardContent>
		</Card>
	{/if}
</div>
