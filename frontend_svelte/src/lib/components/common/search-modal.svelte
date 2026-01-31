<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { searchService, tagsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Badge } from '$lib/components/ui/badge';
	import { Card, CardContent } from '$lib/components/ui/card';
	import type { SearchSuggestItem, TrendingTag } from '$lib/types/api';
	import Search from 'lucide-svelte/icons/search';
	import Hash from 'lucide-svelte/icons/hash';
	import Clock from 'lucide-svelte/icons/clock';
	import TrendingUp from 'lucide-svelte/icons/trending-up';
	import X from 'lucide-svelte/icons/x';

	interface Props {
		open: boolean;
		onOpenChange?: (open: boolean) => void;
	}

	let { open = $bindable(), onOpenChange }: Props = $props();

	let keyword = $state('');
	let isLoadingTags = $state(false);
	let isSuggesting = $state(false);
	let trendingTags = $state<TrendingTag[]>([]);
	let suggestions = $state<SearchSuggestItem[]>([]);
	let recentSearches = $state<string[]>([]);
	let trimmedKeyword = $derived(keyword.trim());
	let recentMatches = $derived(
		trimmedKeyword ? recentSearches.filter((item) => item.includes(trimmedKeyword)).slice(0, 6) : []
	);
	let hasSuggestions = $derived(recentMatches.length > 0 || suggestions.length > 0);
	let suggestTimer: ReturnType<typeof setTimeout> | null = null;

	const RECENT_KEY = 'hotspot_recent_searches';

	function loadRecent() {
		if (!browser) return;
		try {
			const stored = localStorage.getItem(RECENT_KEY);
			if (stored) {
				recentSearches = JSON.parse(stored);
			}
		} catch {
			// ignore
		}
	}

	function saveRecent(term: string) {
		if (!browser) return;
		const next = [term, ...recentSearches.filter((item) => item !== term)].slice(0, 8);
		recentSearches = next;
		try {
			localStorage.setItem(RECENT_KEY, JSON.stringify(next));
		} catch {
			// ignore
		}
	}

	function clearSearch() {
		keyword = '';
		suggestions = [];
	}

	function handleSearchNavigate(term?: string) {
		const next = (term ?? keyword).trim();
		if (!next) return;
		saveRecent(next);
		open = false;
		onOpenChange?.(false);
		goto(`/search?q=${encodeURIComponent(next)}`);
	}

	function handleSearch() {
		const term = keyword.trim();
		if (!term) return;
		handleSearchNavigate(term);
	}

	async function loadTrendingTags() {
		isLoadingTags = true;
		try {
			const response = await tagsService.getTrending({ limit: 10 });
			if (isSuccess(response) && response.data) {
				trendingTags = response.data.items;
			}
		} catch {
			// ignore
		} finally {
			isLoadingTags = false;
		}
	}

	async function loadSuggestions(prefix: string) {
		if (!prefix) {
			suggestions = [];
			return;
		}
		isSuggesting = true;
		try {
			const response = await searchService.suggest({
				prefix,
				limit: 10,
				location: {
					latitude: mapAnchorStore.latitude,
					longitude: mapAnchorStore.longitude
				}
			});
			suggestions = isSuccess(response) && response.data ? response.data.items : [];
		} catch {
			suggestions = [];
		} finally {
			isSuggesting = false;
		}
	}

	function handleTagClick(tag: string) {
		handleSearchNavigate(tag);
	}

	function handleRecentClick(term: string) {
		handleSearchNavigate(term);
	}

	function closeModal() {
		open = false;
		onOpenChange?.(false);
	}

	function handleSuggestionClick(item: SearchSuggestItem) {
		if (!item?.payload) return;
		
		if (item.type === 'tag') {
			const tag = item.payload.name;
			if (tag) {
				saveRecent(tag);
				closeModal();
				goto(`/search?tag=${encodeURIComponent(tag)}`);
			}
			return;
		}

		closeModal();

		if (item.type === 'user' && item.payload.user_id) {
			goto(`/users/${item.payload.user_id}`);
			return;
		}

		if (item.type === 'group' && item.payload.group_chat_id) {
			goto(`/groups/${item.payload.group_chat_id}`);
			return;
		}

		if (item.type === 'moment' && item.payload.moment_id) {
			goto(`/moments/${item.payload.moment_id}`);
		}
	}

	function getSuggestionTitle(item: SearchSuggestItem): string {
		if (!item?.payload) return '未知';
		
		switch (item.type) {
			case 'tag':
				return `#${item.payload.name || ''}`;
			case 'user':
				return item.payload.nickname || '未知用户';
			case 'group':
				return item.payload.name || '未知群组';
			case 'moment':
				return item.payload.description || '瞬间';
			default:
				return '未知';
		}
	}

	function getSuggestionSubtitle(item: SearchSuggestItem): string | null {
		if (!item?.payload) return null;
		
		switch (item.type) {
			case 'tag':
				return `使用 ${item.payload.usage_count || 0} 次`;
			case 'user':
				return item.payload.bio || null;
			case 'group':
				return item.payload.description || `${item.payload.members_count || 0} 位成员`;
			case 'moment': {
				const momentPayload = item.payload;
				return momentPayload.user?.nickname ? `来自 ${momentPayload.user.nickname}` : null;
			}
			default:
				return null;
		}
	}

	function getSuggestionBadge(item: SearchSuggestItem): string {
		switch (item.type) {
			case 'tag':
				return '标签';
			case 'user':
				return '用户';
			case 'group':
				return '群组';
			case 'moment':
				return '瞬间';
		}
	}

	$effect(() => {
		if (open) {
			loadRecent();
			loadTrendingTags();
		}
	});

	$effect(() => {
		if (!open) return;
		if (suggestTimer) clearTimeout(suggestTimer);
		if (!trimmedKeyword) {
			suggestions = [];
			return;
		}
		suggestTimer = setTimeout(() => {
			loadSuggestions(trimmedKeyword);
		}, 250);
	});

	onMount(() => {
		loadRecent();
	});
</script>

<Dialog.Root bind:open onOpenChange={(value) => { open = value; onOpenChange?.(value); }}>
	<Dialog.Content class="flex max-h-[85vh] flex-col sm:max-w-3xl">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<Search class="h-5 w-5 text-primary" />
				搜索
			</Dialog.Title>
			<Dialog.Description>输入关键词，点击搜索进入结果页</Dialog.Description>
		</Dialog.Header>

		<div class="flex-1 overflow-y-auto min-h-0 py-4 px-1 space-y-4">
			<form
				class="flex items-center gap-2"
				onsubmit={(event) => {
					event.preventDefault();
					handleSearch();
				}}
			>
				<div class="relative flex-1">
					<Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						type="text"
						placeholder="输入关键词开始搜索"
						class="pl-9 pr-9"
						bind:value={keyword}
						autocomplete="off"
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
				<Button size="icon" aria-label="搜索" type="submit" disabled={!keyword.trim()}>
					<Search class="h-4 w-4" />
				</Button>
			</form>

			{#if trimmedKeyword}
				<Card class="border-0 bg-card/50">
					<CardContent class="space-y-3 p-4">
						<div class="flex items-center gap-2 text-sm font-medium">
							<TrendingUp class="h-4 w-4 text-muted-foreground" />
							搜索建议
						</div>
						{#if isSuggesting}
							<div class="space-y-2">
								{#each Array(3) as _}
									<div class="h-8 w-full animate-pulse rounded-full bg-muted"></div>
								{/each}
							</div>
						{:else if !hasSuggestions}
							<p class="text-sm text-muted-foreground">暂无相关建议，按回车进入结果页</p>
						{:else}
							<div class="space-y-4">
								{#if recentMatches.length > 0}
									<div class="space-y-2">
										<div class="flex items-center gap-2 text-xs font-medium text-muted-foreground">
											<Clock class="h-3 w-3" />
											最近搜索
										</div>
										<div class="flex flex-wrap gap-2">
											{#each recentMatches as term}
												<Button variant="secondary" size="sm" class="rounded-full" onclick={() => handleSearchNavigate(term)}>
													{term}
												</Button>
											{/each}
										</div>
									</div>
								{/if}

								{#if suggestions.length > 0}
									<div class="space-y-2">
										<div class="flex items-center gap-2 text-xs font-medium text-muted-foreground">
											<TrendingUp class="h-3 w-3" />
											综合建议
										</div>
										<div class="space-y-1">
											{#each suggestions as item}
												<button
													type="button"
													class="flex w-full items-center gap-3 rounded-xl px-2 py-2 text-left transition-colors hover:bg-accent/50"
													onclick={() => handleSuggestionClick(item)}
												>
													{#if item.type === 'tag'}
														<div class="flex h-9 w-9 items-center justify-center rounded-full bg-primary/10 text-primary">
															<Hash class="h-4 w-4" />
														</div>
													{:else if item.type === 'user' && item.payload}
														<Avatar class="h-9 w-9">
															<AvatarImage src={item.payload.avatar_url} alt={item.payload.nickname || '用户'} />
															<AvatarFallback>{(item.payload.nickname || '用户').slice(0, 2)}</AvatarFallback>
														</Avatar>
													{:else if item.type === 'group' && item.payload}
														<Avatar class="h-9 w-9">
															<AvatarImage src={item.payload.avatar_url} alt={item.payload.name || '群组'} />
															<AvatarFallback>{(item.payload.name || '群组').slice(0, 2)}</AvatarFallback>
														</Avatar>
													{:else if item.type === 'moment' && item.payload?.user}
														<Avatar class="h-9 w-9">
															<AvatarImage src={item.payload.user.avatar_url} alt={item.payload.user.nickname || '用户'} />
															<AvatarFallback>{(item.payload.user.nickname || '用户').slice(0, 2)}</AvatarFallback>
														</Avatar>
													{:else}
														<Avatar class="h-9 w-9">
															<AvatarFallback>?</AvatarFallback>
														</Avatar>
													{/if}
													<div class="min-w-0 flex-1">
														<div class="flex items-center gap-2">
															<div class="truncate text-sm font-medium">{getSuggestionTitle(item)}</div>
															<Badge variant="secondary" class="rounded-full px-2 py-0.5 text-[10px] font-medium">
																{getSuggestionBadge(item)}
															</Badge>
														</div>
														{#if getSuggestionSubtitle(item)}
															<div class="truncate text-xs text-muted-foreground">{getSuggestionSubtitle(item)}</div>
														{/if}
													</div>
												</button>
											{/each}
										</div>
									</div>
								{/if}
							</div>
						{/if}
						<div class="text-xs text-muted-foreground">点击搜索图标进入结果页</div>
					</CardContent>
				</Card>
			{:else}
				<div class="grid gap-4 sm:grid-cols-2">
					<Card class="border-0 bg-card/50">
						<CardContent class="space-y-3 p-4">
							<div class="flex items-center gap-2 text-sm font-medium">
								<Clock class="h-4 w-4 text-muted-foreground" />
								最近搜索
							</div>
							{#if recentSearches.length === 0}
								<p class="text-sm text-muted-foreground">暂无搜索记录</p>
							{:else}
								<div class="flex flex-wrap gap-2">
									{#each recentSearches as term}
										<Button variant="secondary" size="sm" class="rounded-full" onclick={() => handleRecentClick(term)}>
											{term}
										</Button>
									{/each}
								</div>
							{/if}
						</CardContent>
					</Card>

					<Card class="border-0 bg-card/50">
						<CardContent class="space-y-3 p-4">
							<div class="flex items-center gap-2 text-sm font-medium">
								<TrendingUp class="h-4 w-4 text-muted-foreground" />
								热门标签
							</div>
							{#if isLoadingTags}
								<div class="space-y-2">
									{#each Array(4) as _}
										<div class="h-8 w-full animate-pulse rounded-full bg-muted"></div>
									{/each}
								</div>
							{:else if trendingTags.length === 0}
								<p class="text-sm text-muted-foreground">暂无热门标签</p>
					{:else}
						<div class="flex flex-wrap gap-2">
							{#each trendingTags as tag}
								<Badge
									variant="secondary"
									class="cursor-pointer rounded-full px-3 py-1.5 hover:bg-primary/10 hover:text-primary"
									onclick={() => handleTagClick(tag.name)}
								>
									<Hash class="mr-1 h-3 w-3" />
									{tag.name}
								</Badge>
							{/each}
						</div>
					{/if}
						</CardContent>
					</Card>
				</div>
			{/if}
		</div>
	</Dialog.Content>
</Dialog.Root>
