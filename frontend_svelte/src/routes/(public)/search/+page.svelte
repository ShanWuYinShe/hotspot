<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { usersService, momentsService, groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Input } from '$lib/components/ui/input';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import UserCard from '$lib/components/common/user-card.svelte';
	import GroupCard from '$lib/components/common/group-card.svelte';
	import MasonryLayout from '$lib/components/common/masonry-layout.svelte';
	import MasonryLayoutItem from '$lib/components/common/masonry-layout-item.svelte';
	import type { UserItem, MomentItem, GroupItem } from '$lib/types/api';
	import Search from 'lucide-svelte/icons/search';
	import Users from 'lucide-svelte/icons/users';
	import Image from 'lucide-svelte/icons/image';
	import UsersRound from 'lucide-svelte/icons/users-round';
	import X from 'lucide-svelte/icons/x';

	// 从 URL 获取初始搜索词（$derived 使 URL 变化时更新）
	const initialKeyword = $derived(page.url.searchParams.get('q') || '');

	let keyword = $state('');
	let currentTab = $state<'users' | 'moments' | 'groups'>('users');

	$effect(() => {
		keyword = initialKeyword;
	});
	let isLoading = $state(false);

	let users = $state.raw<UserItem[]>([]);
	let moments = $state.raw<MomentItem[]>([]);
	let groups = $state.raw<GroupItem[]>([]);
	
	// 用于防止重复触发搜索
	let lastSearchedQuery = $state<string>('');

	function getReferencePoint() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	/**
	 * 执行关键词搜索
	 */
	async function performKeywordSearch(searchTerm: string) {
		if (!searchTerm || searchTerm === lastSearchedQuery) return;
		
		isLoading = true;
		lastSearchedQuery = searchTerm;

		try {
			const [usersRes, momentsRes, groupsRes] = await withMinimumLoadingTime(() =>
				Promise.all([
					usersService.search({ keyword: searchTerm, reference_point: getReferencePoint() }),
					momentsService.search({ keyword: searchTerm, reference_point: getReferencePoint() }),
					groupsService.search({ keyword: searchTerm, reference_point: getReferencePoint() })
				])
			);

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
			toast.error('搜索失败');
		} finally {
			isLoading = false;
		}
	}

	/**
	 * 执行标签搜索
	 */
	async function performTagSearch(tag: string) {
		if (!tag || tag === lastSearchedQuery) return;

		isLoading = true;
		lastSearchedQuery = tag;

		try {
			const [usersRes, momentsRes, groupsRes] = await withMinimumLoadingTime(() =>
				Promise.all([
					usersService.search({ tag, reference_point: getReferencePoint() }),
					momentsService.search({ tag, reference_point: getReferencePoint() }),
					groupsService.search({ tag, reference_point: getReferencePoint() })
				])
			);

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
			toast.error('搜索失败');
		} finally {
			isLoading = false;
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
		lastSearchedQuery = '';
		goto('/search', { replaceState: true });
	}

	// 监听 URL 参数变化，执行搜索（只在首次加载或外部跳转时触发）
	$effect(() => {
		const q = page.url.searchParams.get('q');
		const tag = page.url.searchParams.get('tag');
		
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
	});
</script>

<svelte:head>
	<title>搜索 - HotSpot</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center gap-3">
		<div class="rounded-xl bg-linear-to-br from-primary/20 to-primary/5 p-2.5 text-primary">
			<Search class="h-5 w-5" />
		</div>
		<div>
			<h1 class="text-2xl font-bold">搜索</h1>
			<p class="text-sm text-muted-foreground">发现用户、群组与附近动态</p>
		</div>
	</div>

	<!-- Search input -->
	<div class="relative">
		<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
		<Input
			type="text"
			placeholder="搜索用户、瞬间、群组..."
			class="pl-9 pr-9"
			autocomplete="off"
			bind:value={keyword}
			onkeydown={(e) => e.key === 'Enter' && handleSearch()}
		/>
		{#if keyword}
			<button
				type="button"
				class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
				onclick={clearSearch}
			>
				<X class="h-4 w-4" />
			</button>
		{/if}
	</div>

	{#if keyword}
		<Tabs bind:value={currentTab}>
			<TabsList class="h-auto gap-1.5 rounded-full bg-card/80 p-1.5 shadow-md shadow-black/10 ring-1 ring-border/50 backdrop-blur-sm">
				<TabsTrigger value="users" class="gap-2">
					<Users class="h-4 w-4" />
					用户 ({users.length})
				</TabsTrigger>
				<TabsTrigger value="moments" class="gap-2">
					<Image class="h-4 w-4" />
					瞬间 ({moments.length})
				</TabsTrigger>
				<TabsTrigger value="groups" class="gap-2">
					<UsersRound class="h-4 w-4" />
					群组 ({groups.length})
				</TabsTrigger>
			</TabsList>

			<!-- Users -->
			<TabsContent value="users" class="mt-6">
				{#if isLoading}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
						{#each Array(6) as _}
							<MasonryLayoutItem>
								<div class="h-48 animate-pulse rounded-2xl bg-muted"></div>
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{:else if users.length === 0}
					<Card>
						<CardContent class="py-8 text-center text-muted-foreground">
							未找到相关用户
						</CardContent>
					</Card>
				{:else}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
						{#each users as user (user.user.user_id)}
							<MasonryLayoutItem>
								<UserCard {user} showBio showTags showDistance variant="square" />
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{/if}
			</TabsContent>

			<!-- Moments -->
			<TabsContent value="moments" class="mt-6">
				{#if isLoading}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={20}>
						{#each Array(6) as _}
							<MasonryLayoutItem>
								<div class="h-64 animate-pulse rounded-2xl bg-muted"></div>
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{:else if moments.length === 0}
					<Card>
						<CardContent class="py-8 text-center text-muted-foreground">
							未找到相关瞬间
						</CardContent>
					</Card>
				{:else}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={20}>
						{#each moments as moment (moment.moment_id)}
							<MasonryLayoutItem>
								<MomentCard {moment} />
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{/if}
			</TabsContent>

			<!-- Groups -->
			<TabsContent value="groups" class="mt-6">
				{#if isLoading}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
						{#each Array(6) as _}
							<MasonryLayoutItem>
								<div class="h-52 animate-pulse rounded-2xl bg-muted"></div>
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{:else if groups.length === 0}
					<Card>
						<CardContent class="py-8 text-center text-muted-foreground">
							未找到相关群组
						</CardContent>
					</Card>
				{:else}
					<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
						{#each groups as group (group.group.group_id)}
							<MasonryLayoutItem>
								<GroupCard {group} showMembers showDistance variant="square" />
							</MasonryLayoutItem>
						{/each}
					</MasonryLayout>
				{/if}
			</TabsContent>
		</Tabs>
	{:else}
		<Card class="border-0 bg-linear-to-br from-primary/5 via-background to-primary/10">
			<CardContent class="flex flex-col items-center justify-center py-12">
				<Search class="mb-4 h-12 w-12 text-muted-foreground" />
				<p class="text-muted-foreground">输入关键词开始搜索</p>
			</CardContent>
		</Card>
	{/if}
</div>
