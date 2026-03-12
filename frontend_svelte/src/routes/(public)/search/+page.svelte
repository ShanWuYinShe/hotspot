<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { usersService, momentsService, groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import AnchorSummaryCard from '$lib/components/common/anchor-summary-card.svelte';
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
	import X from 'lucide-svelte/icons/x';

	let keyword = $state('');
	let currentTab = $state<'users' | 'moments' | 'groups'>('users');
	let isLoading = $state(false);

	let users = $state.raw<UserItem[]>([]);
	let moments = $state.raw<MomentItem[]>([]);
	let groups = $state.raw<GroupItem[]>([]);
	
	// 用于防止重复触发同条件搜索
	let lastSearchFingerprint = $state<string>('');
	// 用于取消进行中的搜索请求，防止竞态条件
	let searchAbortController: AbortController | null = null;

	function getReferencePoint() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	function buildSearchFingerprint(mode: 'keyword' | 'tag', value: string): string {
		const point = getReferencePoint();
		return `${mode}:${value}:${point.latitude.toFixed(6)}:${point.longitude.toFixed(6)}:${mapAnchorStore.radius}`;
	}

	/**
	 * 执行关键词搜索
	 */
	async function performKeywordSearch(searchTerm: string) {
		const normalized = searchTerm.trim();
		if (!normalized) return;
		const fingerprint = buildSearchFingerprint('keyword', normalized);
		if (fingerprint === lastSearchFingerprint) return;
		
		// 取消上一次进行中的请求
		searchAbortController?.abort();
		const controller = new AbortController();
		searchAbortController = controller;

		isLoading = true;
		lastSearchFingerprint = fingerprint;

		try {
			const signal = controller.signal;
			const [usersRes, momentsRes, groupsRes] = await withMinimumLoadingTime(() =>
				Promise.all([
					usersService.search({ keyword: normalized, reference_point: getReferencePoint() }, { signal }),
					momentsService.search({ keyword: normalized, reference_point: getReferencePoint() }, { signal }),
					groupsService.search({ keyword: normalized, reference_point: getReferencePoint() }, { signal })
				])
			);

			// 请求已被取消（新搜索已发起），不更新 UI
			if (controller.signal.aborted) return;

			if (isSuccess(usersRes) && usersRes.data) {
				users = usersRes.data.items;
			}
			if (isSuccess(momentsRes) && momentsRes.data) {
				moments = momentsRes.data.items;
			}
			if (isSuccess(groupsRes) && groupsRes.data) {
				groups = groupsRes.data.items;
			}
		} catch {
			if (!controller.signal.aborted) {
				toast.error('搜索失败');
			}
		} finally {
			if (!controller.signal.aborted) {
				isLoading = false;
			}
		}
	}

	/**
	 * 执行标签搜索
	 */
	async function performTagSearch(tag: string) {
		const normalized = tag.trim();
		if (!normalized) return;
		const fingerprint = buildSearchFingerprint('tag', normalized);
		if (fingerprint === lastSearchFingerprint) return;

		// 取消上一次进行中的请求
		searchAbortController?.abort();
		const controller = new AbortController();
		searchAbortController = controller;

		isLoading = true;
		lastSearchFingerprint = fingerprint;

		try {
			const signal = controller.signal;
			const [usersRes, momentsRes, groupsRes] = await withMinimumLoadingTime(() =>
				Promise.all([
					usersService.search({ tag: normalized, reference_point: getReferencePoint() }, { signal }),
					momentsService.search({ tag: normalized, reference_point: getReferencePoint() }, { signal }),
					groupsService.search({ tag: normalized, reference_point: getReferencePoint() }, { signal })
				])
			);

			if (controller.signal.aborted) return;

			if (isSuccess(usersRes) && usersRes.data) {
				users = usersRes.data.items;
			}
			if (isSuccess(momentsRes) && momentsRes.data) {
				moments = momentsRes.data.items;
			}
			if (isSuccess(groupsRes) && groupsRes.data) {
				groups = groupsRes.data.items;
			}
		} catch {
			if (!controller.signal.aborted) {
				toast.error('搜索失败');
			}
		} finally {
			if (!controller.signal.aborted) {
				isLoading = false;
			}
		}
	}

	/**
	 * 处理输入框的搜索（按回车或点击搜索按钮）
	 */
	async function handleSearch() {
		const searchTerm = keyword.trim();
		if (!searchTerm) return;

		// 更新 URL（不触发 effect）
		goto(`/search?q=${encodeURIComponent(searchTerm)}`, { replaceState: true, noScroll: true });
		
		// 执行搜索
		await performKeywordSearch(searchTerm);
	}

	function clearSearch() {
		keyword = '';
		users = [];
		moments = [];
		groups = [];
		lastSearchFingerprint = '';
		goto('/search', { replaceState: true });
	}

	// 监听 URL 参数与浏览锚点变化，保持搜索结果与当前位置一致
	$effect(() => {
		const q = page.url.searchParams.get('q');
		const tag = page.url.searchParams.get('tag');
		mapAnchorStore.anchor.updatedAt;
		
		// 优先处理标签搜索
		if (tag) {
			keyword = tag;
			performTagSearch(tag);
			return;
		}
		
		// 处理关键词搜索
		if (q) {
			keyword = q;
			performKeywordSearch(q);
		}

		// 组件卸载时取消进行中的请求
		return () => {
			searchAbortController?.abort();
		};
	});
</script>

<svelte:head>
	<title>搜索 - HotSpot</title>
</svelte:head>

<div class="space-y-5">
	<PageHeader
		title="搜索"
		description="发现用户、群组与附近动态"
	/>

	<AnchorSummaryCard
		label="搜索参考锚点"
		description={`当前按 ${mapAnchorStore.radius}km 范围参与距离排序`}
	>
		{#snippet actions()}
			<div class="flex items-center">
				<Button variant="outline" size="sm" class="gap-2" onclick={() => goto('/moments/nearby')}>
					<Compass class="h-4 w-4" />
					附近探索
				</Button>
			</div>
		{/snippet}
	</AnchorSummaryCard>

	<!-- Search input -->
	<form
		class="rounded-3xl border border-border/60 bg-card/85 p-3 shadow-sm backdrop-blur-md"
		onsubmit={(event) => {
			event.preventDefault();
			handleSearch();
		}}
	>
		<div class="flex items-center gap-2">
			<div class="relative flex-1">
				<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
				<Input
					type="text"
					placeholder="搜索用户、瞬间、群组..."
					class="h-11 rounded-2xl border-border/60 bg-background/70 pl-9 pr-9"
					autocomplete="off"
					bind:value={keyword}
				/>
				{#if keyword}
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
			<Button type="submit" class="h-11 rounded-2xl border border-border/60 bg-background/70 px-5 text-foreground hover:bg-accent/70" disabled={!keyword.trim()}>
				<Search class="h-4 w-4" />
				搜索
			</Button>
		</div>
	</form>

	{#if keyword}
		<Tabs bind:value={currentTab}>
			<TabsList class="h-auto gap-1.5 rounded-3xl bg-card/80 p-1.5 shadow-md shadow-black/10 dark:shadow-black/30 ring-1 ring-border/50 backdrop-blur-sm">
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

			<!-- Users -->
			<TabsContent value="users" class="mt-6">
				{#if isLoading}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
						{#each Array(6) as _, i (i)}
							<MasonryLayoutItem>
								<div class="h-48 animate-pulse rounded-2xl bg-muted"></div>
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{:else if users.length === 0}
					<Card class="rounded-3xl border border-border/60 bg-card/85 shadow-sm backdrop-blur-md">
						<CardContent class="py-8 text-center text-muted-foreground">
							未找到相关用户
						</CardContent>
					</Card>
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

			<!-- Moments -->
			<TabsContent value="moments" class="mt-6">
				{#if isLoading}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={20}>
						{#each Array(6) as _, i (i)}
							<MasonryLayoutItem>
								<div class="h-64 animate-pulse rounded-2xl bg-muted"></div>
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{:else if moments.length === 0}
					<Card class="rounded-3xl border border-border/60 bg-card/85 shadow-sm backdrop-blur-md">
						<CardContent class="py-8 text-center text-muted-foreground">
							未找到相关瞬间
						</CardContent>
					</Card>
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

			<!-- Groups -->
			<TabsContent value="groups" class="mt-6">
				{#if isLoading}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
						{#each Array(6) as _, i (i)}
							<MasonryLayoutItem>
								<div class="h-52 animate-pulse rounded-2xl bg-muted"></div>
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{:else if groups.length === 0}
					<Card class="rounded-3xl border border-border/60 bg-card/85 shadow-sm backdrop-blur-md">
						<CardContent class="py-8 text-center text-muted-foreground">
							未找到相关群组
						</CardContent>
					</Card>
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
			<CardContent class="flex flex-col items-center justify-center py-12">
				<Search class="mb-4 h-12 w-12 text-muted-foreground" />
				<p class="text-muted-foreground">输入关键词开始搜索</p>
			</CardContent>
		</Card>
	{/if}
</div>
