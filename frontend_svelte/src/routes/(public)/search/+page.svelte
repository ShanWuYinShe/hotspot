<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { usersService, momentsService, groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import type { UserSearchItem, MomentSearchItem, GroupSearchItem } from '$lib/types/api';
	import Search from 'lucide-svelte/icons/search';
	import Users from 'lucide-svelte/icons/users';
	import Image from 'lucide-svelte/icons/image';
	import UsersRound from 'lucide-svelte/icons/users-round';
	import X from 'lucide-svelte/icons/x';

	let keyword = $state($page.url.searchParams.get('q') || '');
	let tagQuery = $state('');
	let lastQuery = $state<string | null>(null);
	let lastTag = $state<string | null>(null);
	let currentTab = $state<'users' | 'moments' | 'groups'>('users');
	let isLoading = $state(false);

	let users = $state<UserSearchItem[]>([]);
	let moments = $state<MomentSearchItem[]>([]);
	let groups = $state<GroupSearchItem[]>([]);

	function getReferencePoint() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	async function handleSearch() {
		if (!keyword.trim()) return;

		isLoading = true;
		lastQuery = keyword.trim();
		tagQuery = '';
		goto(`/search?q=${encodeURIComponent(keyword.trim())}`, { replaceState: true });

		try {
			const [usersRes, momentsRes, groupsRes] = await Promise.all([
				usersService.searchByKeyword({ reference_point: getReferencePoint(), kw: keyword.trim(), limit: 20 }),
				momentsService.searchByKeyword({ reference_point: getReferencePoint(), kw: keyword.trim(), limit: 20 }),
				groupsService.searchByKeyword({ reference_point: getReferencePoint(), kw: keyword.trim(), limit: 20 })
			]);

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

	async function handleTagSearch(nextTag: string) {
		if (!nextTag) return;

		isLoading = true;
		lastTag = nextTag;
		keyword = nextTag;
		tagQuery = nextTag;
		goto(`/search?tag=${encodeURIComponent(nextTag)}`, { replaceState: true });

		try {
			const [usersRes, momentsRes, groupsRes] = await Promise.all([
				usersService.searchByTag({ reference_point: getReferencePoint(), tag: nextTag, limit: 20 }),
				momentsService.searchByTag({ reference_point: getReferencePoint(), tag: nextTag, limit: 20 }),
				groupsService.searchByTag({ reference_point: getReferencePoint(), tag: nextTag, limit: 20 })
			]);

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

	function clearSearch() {
		keyword = '';
		tagQuery = '';
		users = [];
		moments = [];
		groups = [];
		goto('/search', { replaceState: true });
	}

	// 页面加载时如果有 query 参数则自动搜索
	$effect(() => {
		const q = $page.url.searchParams.get('q');
		const tag = $page.url.searchParams.get('tag');
		if (tag && tag !== lastTag) {
			handleTagSearch(tag);
			return;
		}
		if (q && q !== lastQuery) {
			keyword = q;
			tagQuery = '';
			handleSearch();
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
					<div class="space-y-2">
						{#each Array(3) as _}
							<Card>
								<CardContent class="flex items-center gap-4 p-4">
									<Skeleton class="h-12 w-12 rounded-full" />
									<div class="flex-1 space-y-2">
										<Skeleton class="h-4 w-24" />
										<Skeleton class="h-3 w-48" />
									</div>
								</CardContent>
							</Card>
						{/each}
					</div>
				{:else if users.length === 0}
					<Card>
						<CardContent class="py-8 text-center text-muted-foreground">
							未找到相关用户
						</CardContent>
					</Card>
				{:else}
					<div class="columns-1 gap-3 sm:columns-2 xl:columns-3">
						{#each users as user (user.user_id)}
							<div class="mb-3 break-inside-avoid">
								<a href="/users/{user.user_id}">
									<Card class="cursor-pointer transition-colors hover:bg-accent/50">
										<CardContent class="flex items-center gap-4 p-4">
											<Avatar class="h-12 w-12">
												<AvatarImage src={user.avatar_url} alt={user.nickname} />
												<AvatarFallback>{user.nickname.slice(0, 2)}</AvatarFallback>
											</Avatar>
											<div class="min-w-0 flex-1">
												<p class="font-medium">{user.nickname}</p>
												{#if user.bio}
													<p class="line-clamp-3 text-sm text-muted-foreground">{user.bio}</p>
												{/if}
												{#if user.tags && user.tags.length > 0}
													<div class="mt-1 flex flex-wrap gap-1">
														{#each user.tags.slice(0, 3) as tag}
															<Badge variant="secondary" class="text-xs">#{tag}</Badge>
														{/each}
													</div>
												{/if}
											</div>
										</CardContent>
									</Card>
								</a>
							</div>
						{/each}
					</div>
				{/if}
			</TabsContent>

			<!-- Moments -->
			<TabsContent value="moments" class="mt-6">
				{#if isLoading}
					<div class="space-y-2">
						{#each Array(3) as _}
							<Card>
								<CardContent class="p-4">
									<div class="flex gap-3">
										<Skeleton class="h-10 w-10 rounded-full" />
										<div class="flex-1 space-y-2">
											<Skeleton class="h-4 w-24" />
											<Skeleton class="h-4 w-full" />
										</div>
									</div>
								</CardContent>
							</Card>
						{/each}
					</div>
				{:else if moments.length === 0}
					<Card>
						<CardContent class="py-8 text-center text-muted-foreground">
							未找到相关瞬间
						</CardContent>
					</Card>
				{:else}
					<div class="columns-1 gap-3 sm:columns-2 lg:columns-3">
						{#each moments as moment (moment.moment_id)}
							<div class="mb-3 break-inside-avoid">
								<MomentCard {moment} />
							</div>
						{/each}
					</div>
				{/if}
			</TabsContent>

			<!-- Groups -->
			<TabsContent value="groups" class="mt-6">
				{#if isLoading}
					<div class="space-y-2">
						{#each Array(3) as _}
							<Card>
								<CardContent class="flex items-center gap-4 p-4">
									<Skeleton class="h-12 w-12 rounded-full" />
									<div class="flex-1 space-y-2">
										<Skeleton class="h-4 w-24" />
										<Skeleton class="h-3 w-16" />
									</div>
								</CardContent>
							</Card>
						{/each}
					</div>
				{:else if groups.length === 0}
					<Card>
						<CardContent class="py-8 text-center text-muted-foreground">
							未找到相关群组
						</CardContent>
					</Card>
				{:else}
					<div class="columns-1 gap-3 sm:columns-2 xl:columns-3">
						{#each groups as group (group.group_chat_id)}
							<div class="mb-3 break-inside-avoid">
								<a href="/groups/{group.group_chat_id}">
									<Card class="cursor-pointer transition-colors hover:bg-accent/50">
										<CardContent class="flex items-center gap-4 p-4">
											<Avatar class="h-12 w-12">
												<AvatarImage src={group.avatar_url} alt={group.name} />
												<AvatarFallback>{group.name.slice(0, 2)}</AvatarFallback>
											</Avatar>
											<div class="min-w-0 flex-1">
												<p class="font-medium">{group.name}</p>
												{#if group.description}
													<p class="line-clamp-3 text-sm text-muted-foreground">{group.description}</p>
												{/if}
												<p class="text-xs text-muted-foreground">{group.members_count} 成员</p>
											</div>
										</CardContent>
									</Card>
								</a>
							</div>
						{/each}
					</div>
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
