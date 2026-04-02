<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { searchService, feedService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { t } from '$lib/i18n';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { authStore } from '$lib/stores/auth.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Badge } from '$lib/components/ui/badge';
	import { Card, CardContent } from '$lib/components/ui/card';
	import type { SearchSuggestItem, TagItem } from '$lib/types/api';
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
	let trendingTags = $state.raw<TagItem[]>([]);
	let hasLoadedTrendingTags = $state(false);
	let trendingLoadVersion = 0;
	let suggestions = $state.raw<SearchSuggestItem[]>([]);
	let recentSearches = $state.raw<string[]>([]);
	const trimmedKeyword = $derived(keyword.trim());
	const recentMatches = $derived(
		trimmedKeyword ? recentSearches.filter((item) => item.includes(trimmedKeyword)).slice(0, 6) : []
	);
	const hasSuggestions = $derived(recentMatches.length > 0 || suggestions.length > 0);
	let suggestTimer: ReturnType<typeof setTimeout> | null = null;
	// 用于取消进行中的搜索建议请求，防止竞态条件
	let suggestAbortController: AbortController | null = null;

	/** 获取搜索建议项的唯一 key */
	function getSuggestionKey(item: SearchSuggestItem): string {
		switch (item.type) {
			case 'user': return `user_${item.payload.user_id}`;
			case 'group': return `group_${item.payload.group_id}`;
			case 'moment': return `moment_${item.payload.moment_id}`;
			case 'tag': return `tag_${item.payload.name}`;
		}
	}

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

	function removeRecent(term: string) {
		if (!browser) return;
		const next = recentSearches.filter((item) => item !== term);
		recentSearches = next;
		try {
			localStorage.setItem(RECENT_KEY, JSON.stringify(next));
		} catch {
			// ignore
		}
	}

	function clearAllRecent() {
		if (!browser) return;
		recentSearches = [];
		try {
			localStorage.removeItem(RECENT_KEY);
		} catch {
			// ignore
		}
	}

	function clearSearch() {
		keyword = '';
		suggestions = [];
	}

	function setOpenState(nextOpen: boolean) {
		if (open === nextOpen) {
			return;
		}
		open = nextOpen;
		onOpenChange?.(nextOpen);
	}

	function handleSearchNavigate(term?: string) {
		const next = (term ?? keyword).trim();
		if (!next) return;
		saveRecent(next);
		setOpenState(false);
		goto(`/search?q=${encodeURIComponent(next)}`);
	}

	function handleSearch() {
		const term = keyword.trim();
		if (!term) return;
		handleSearchNavigate(term);
	}

	async function loadTrendingTags() {
		if (isLoadingTags || hasLoadedTrendingTags) return;

		const version = ++trendingLoadVersion;
		isLoadingTags = true;
		try {
			const response = await feedService.getTrending();
			if (version !== trendingLoadVersion) return;
			if (isSuccess(response) && response.data) {
				trendingTags = response.data.trending_tags;
				hasLoadedTrendingTags = true;
			}
		} catch {
			if (version !== trendingLoadVersion) return;
		} finally {
			if (version === trendingLoadVersion) {
				isLoadingTags = false;
			}
		}
	}

	async function loadSuggestions(prefix: string) {
		if (!prefix) {
			suggestions = [];
			return;
		}

		// 取消上一次进行中的建议请求
		suggestAbortController?.abort();
		const controller = new AbortController();
		suggestAbortController = controller;

		isSuggesting = true;
		try {
			const response = await searchService.suggest({
				prefix,
				reference_point: {
					latitude: mapAnchorStore.latitude,
					longitude: mapAnchorStore.longitude
				}
			}, { signal: controller.signal });

			if (controller.signal.aborted) return;
			const rawItems = isSuccess(response) && response.data ? response.data.items : [];
			const currentUserId = authStore.currentUserId;
			suggestions = currentUserId
				? rawItems.filter((item) =>
					item.type !== 'user' || item.payload.user_id !== currentUserId
				  )
				: rawItems;
		} catch {
			if (!controller.signal.aborted) {
				suggestions = [];
			}
		} finally {
			if (!controller.signal.aborted) {
				isSuggesting = false;
			}
		}
	}

	function handleTagClick(tag: string) {
		handleSearchNavigate(tag);
	}

	function handleRecentClick(term: string) {
		handleSearchNavigate(term);
	}

	function getRemoveRecentLabel(term: string): string {
		return t('searchModal.removeRecent', { term });
	}

	function closeModal() {
		setOpenState(false);
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

		if (item.type === 'group' && item.payload.group_id) {
			goto(`/groups/${item.payload.group_id}`);
			return;
		}

		if (item.type === 'moment' && item.payload.moment_id) {
			goto(`/moments/${item.payload.moment_id}`);
		}
	}

	function getSuggestionPresentation(item: SearchSuggestItem): {
		title: string;
		subtitle: string | null;
		badge: string;
		avatarAlt: string | null;
		avatarFallback: string | null;
	} {
		const unknown = t('searchModal.unknown');
		const unknownUser = t('searchModal.unknownUser');
		const unknownGroup = t('searchModal.unknownGroup');

		switch (item.type) {
			case 'tag': {
				const payload = item.payload;
				return {
					title: `#${payload.name || ''}`,
					subtitle: t('searchModal.usedCount', { count: payload.usage_count || 0 }),
					badge: t('searchModal.tagBadge'),
					avatarAlt: null,
					avatarFallback: null
				};
			}
			case 'user': {
				const payload = item.payload;
				const name = payload.nickname || unknownUser;
				return {
					title: name,
					subtitle: payload.bio || null,
					badge: t('searchModal.userBadge'),
					avatarAlt: name,
					avatarFallback: name.slice(0, 2)
				};
			}
			case 'group': {
				const payload = item.payload;
				const name = payload.name || unknownGroup;
				return {
					title: name,
					subtitle: payload.description || t('searchModal.memberCount', { count: payload.members_count || 0 }),
					badge: t('searchModal.groupBadge'),
					avatarAlt: name,
					avatarFallback: name.slice(0, 2)
				};
			}
			case 'moment': {
				const payload = item.payload;
				const authorName = payload.author?.nickname || unknownUser;
				return {
					title: payload.description_preview || t('searchModal.momentFallback'),
					subtitle: payload.author?.nickname
						? t('searchModal.fromAuthor', { nickname: payload.author.nickname })
						: null,
					badge: t('searchModal.momentBadge'),
					avatarAlt: authorName,
					avatarFallback: authorName.slice(0, 2)
				};
			}
			default:
				return {
					title: unknown,
					subtitle: null,
					badge: unknown,
					avatarAlt: null,
					avatarFallback: null
				};
		}
	}

	$effect(() => {
		if (open) {
			loadRecent();
			loadTrendingTags();
		} else {
			keyword = '';
			suggestions = [];
			if (suggestTimer) {
				clearTimeout(suggestTimer);
				suggestTimer = null;
			}
			// 取消进行中的建议请求，防止关闭后仍更新状态
			suggestAbortController?.abort();
			suggestAbortController = null;
		}
	});

	$effect(() => {
		if (!open) {
			return;
		}
		if (suggestTimer) clearTimeout(suggestTimer);
		if (!trimmedKeyword) {
			suggestions = [];
			return;
		}
		suggestTimer = setTimeout(() => {
			loadSuggestions(trimmedKeyword);
		}, 250);

		return () => {
			if (suggestTimer) {
				clearTimeout(suggestTimer);
				suggestTimer = null;
			}
		};
	});
</script>

<Dialog.Root bind:open onOpenChange={setOpenState}>
	<Dialog.Content class="flex max-h-[85vh] flex-col rounded-3xl border border-border/60 bg-card/95 shadow-xl backdrop-blur-md sm:max-w-3xl">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-2xl border border-border/50 bg-primary/10 text-primary">
					<Search class="h-5 w-5" />
				</div>
				<div>{t('searchModal.title')}</div>
			</Dialog.Title>
			<Dialog.Description>{t('searchModal.description')}</Dialog.Description>
		</Dialog.Header>

		<div class="min-h-0 flex-1 space-y-4 overflow-y-auto px-1 py-4">
			<form
				class="flex items-center gap-2 rounded-2xl border border-border/60 bg-background/65 p-2.5 shadow-sm"
				onsubmit={(event) => {
					event.preventDefault();
					handleSearch();
				}}
			>
				<div class="relative flex-1">
					<Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						type="text"
						placeholder={t('searchModal.placeholder')}
						class="h-11 rounded-2xl border-border/60 bg-background/75 pl-9 pr-9"
						bind:value={keyword}
						autocomplete="off"
						aria-label={t('searchModal.inputAria')}
					/>
					{#if keyword}
						<Button
							type="button"
							variant="ghost"
							size="icon"
							class="absolute right-1 top-1/2 h-9 w-9 -translate-y-1/2 rounded-xl text-muted-foreground hover:text-foreground"
							onclick={clearSearch}
							aria-label={t('searchModal.clearInput')}
						>
							<X class="h-4 w-4" />
						</Button>
					{/if}
				</div>
				<Button size="icon" class="h-11 w-11 rounded-2xl" aria-label={t('searchModal.submit')} type="submit" disabled={!keyword.trim()}>
					<Search class="h-4 w-4" />
				</Button>
			</form>

			{#if trimmedKeyword}
				<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
					<CardContent class="space-y-3 p-4">
						{#if isSuggesting}
							<div class="space-y-2">
								{#each Array(3) as _, i (i)}
									<div class="h-8 w-full animate-pulse rounded-2xl bg-muted"></div>
								{/each}
							</div>
						{:else if !hasSuggestions}
							<p class="text-sm text-muted-foreground">{t('searchModal.noSuggestions')}</p>
						{:else}
							<div class="space-y-4">
								{#if recentMatches.length > 0}
									<div class="space-y-2">
										<div class="flex items-center justify-between">
											<div class="flex items-center gap-2 text-xs font-medium text-muted-foreground">
												<Clock class="h-3 w-3" />
												{t('searchModal.matchedHistory')}
											</div>
											<Button
												variant="ghost"
												size="sm"
												class="h-6 px-2 text-xs text-muted-foreground hover:text-destructive hover:scale-105 active:scale-95 transition-all"
												onclick={clearAllRecent}
											>
												{t('searchModal.clearAll')}
											</Button>
										</div>
										<div class="flex flex-wrap gap-2">
											{#each recentMatches as term (term)}
												<div class="group flex items-center gap-1 rounded-2xl bg-secondary/80 py-1 pl-1 pr-1 shadow-sm transition-colors hover:bg-secondary hover:shadow-md focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2">
													<Button
														type="button"
														variant="ghost"
														size="sm"
														class="h-9 rounded-2xl px-3 text-sm font-medium text-foreground hover:text-primary"
														onclick={() => handleSearchNavigate(term)}
													>
														{term}
													</Button>
													<Button
														type="button"
														variant="ghost"
														size="icon"
														class="h-9 w-9 rounded-xl text-muted-foreground opacity-80 transition-all hover:bg-destructive/20 hover:text-destructive [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-hover:opacity-100 [@media(hover:hover)]:group-focus-within:opacity-100"
														onclick={() => removeRecent(term)}
														aria-label={getRemoveRecentLabel(term)}
													>
														<X class="h-3.5 w-3.5" />
													</Button>
												</div>
											{/each}
										</div>
									</div>
								{/if}

								{#if suggestions.length > 0}
									<div class="space-y-2">
										<div class="flex items-center gap-2 text-xs font-medium text-muted-foreground">
											<TrendingUp class="h-3 w-3" />
											{t('searchModal.suggestions')}
										</div>
										<div class="space-y-1">
											{#each suggestions as item (getSuggestionKey(item))}
												{@const presentation = getSuggestionPresentation(item)}
												<button
													type="button"
													class="flex w-full items-center gap-3 rounded-xl px-2 py-2 text-left transition-colors hover:bg-accent/50"
													onclick={() => handleSuggestionClick(item)}
												>
													{#if item.type === 'tag'}
														<div class="flex h-9 w-9 items-center justify-center rounded-2xl bg-primary/10 text-primary">
															<Hash class="h-4 w-4" />
														</div>
													{:else if item.type === 'user' && item.payload}
														<Avatar class="h-9 w-9">
															<AvatarImage src={item.payload.avatar_url} alt={presentation.avatarAlt ?? ''} />
															<AvatarFallback>{presentation.avatarFallback ?? '?'}</AvatarFallback>
														</Avatar>
													{:else if item.type === 'group' && item.payload}
														<Avatar class="h-9 w-9">
															<AvatarImage src={item.payload.avatar_url} alt={presentation.avatarAlt ?? ''} />
															<AvatarFallback>{presentation.avatarFallback ?? '?'}</AvatarFallback>
														</Avatar>
													{:else if item.type === 'moment' && item.payload}
														<Avatar class="h-9 w-9">
															<AvatarImage src={item.payload.author?.avatar_url} alt={presentation.avatarAlt ?? ''} />
															<AvatarFallback>{presentation.avatarFallback ?? '?'}</AvatarFallback>
														</Avatar>
													{:else}
														<Avatar class="h-9 w-9">
															<AvatarFallback>?</AvatarFallback>
														</Avatar>
													{/if}
													<div class="min-w-0 flex-1">
														<div class="flex items-center gap-2">
															<div class="truncate text-sm font-medium">{presentation.title}</div>
															<Badge variant="secondary" class="rounded-2xl px-2 py-0.5 text-[10px] font-medium">
																{presentation.badge}
															</Badge>
														</div>
														{#if presentation.subtitle}
															<div class="truncate text-xs text-muted-foreground">{presentation.subtitle}</div>
														{/if}
													</div>
												</button>
											{/each}
										</div>
									</div>
								{/if}
							</div>
						{/if}
						<div class="text-xs text-muted-foreground">{t('searchModal.clickSearchIcon')}</div>
					</CardContent>
				</Card>
			{:else}
				<div class="grid gap-4 sm:grid-cols-2">
					<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
						<CardContent class="space-y-3 p-4">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-2 text-sm font-medium">
									<Clock class="h-4 w-4 text-muted-foreground" />
									{t('searchModal.recentSearches')}
								</div>
								{#if recentSearches.length > 0}
									<Button
										variant="ghost"
										size="sm"
										class="h-7 px-2 text-xs text-muted-foreground hover:text-destructive"
										onclick={clearAllRecent}
									>
										{t('searchModal.clearAll')}
									</Button>
								{/if}
							</div>
							{#if recentSearches.length === 0}
								<p class="text-sm text-muted-foreground">{t('searchModal.noRecentSearches')}</p>
							{:else}
								<div class="flex flex-wrap gap-2">
									{#each recentSearches as term (term)}
										<div class="group flex items-center gap-1 rounded-2xl bg-secondary/80 py-1 pl-1 pr-1 shadow-sm transition-colors hover:bg-secondary hover:shadow-md focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2">
											<Button
												type="button"
												variant="ghost"
												size="sm"
												class="h-9 rounded-2xl px-3 text-sm font-medium text-foreground hover:text-primary"
												onclick={() => handleRecentClick(term)}
											>
												{term}
											</Button>
											<Button
												type="button"
												variant="ghost"
												size="icon"
												class="h-9 w-9 rounded-xl text-muted-foreground opacity-80 transition-all hover:bg-destructive/20 hover:text-destructive [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-hover:opacity-100 [@media(hover:hover)]:group-focus-within:opacity-100"
												onclick={() => removeRecent(term)}
												aria-label={getRemoveRecentLabel(term)}
											>
												<X class="h-3.5 w-3.5" />
											</Button>
										</div>
									{/each}
								</div>
							{/if}
						</CardContent>
					</Card>

					<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
						<CardContent class="space-y-3 p-4">
							<div class="flex items-center gap-2 text-sm font-medium">
								<TrendingUp class="h-4 w-4 text-muted-foreground" />
								{t('searchModal.trendingTags')}
							</div>
							{#if isLoadingTags}
								<div class="space-y-2">
									{#each Array(4) as _, i (i)}
										<div class="h-8 w-full animate-pulse rounded-2xl bg-muted"></div>
									{/each}
								</div>
							{:else if trendingTags.length === 0}
								<p class="text-sm text-muted-foreground">{t('searchModal.noTrendingTags')}</p>
							{:else}
								<div class="flex flex-wrap gap-2">
									{#each trendingTags as tag (tag.name)}
										<Button
											type="button"
											variant="ghost"
											class="rounded-2xl p-0 hover:bg-transparent"
											onclick={() => handleTagClick(tag.name)}
											aria-label={t('searchModal.searchHotTag', { name: tag.name })}
										>
											<Badge
												variant="secondary"
												class="rounded-2xl px-3 py-1.5 hover:bg-primary/10 hover:text-primary"
											>
												<Hash class="mr-1 h-3 w-3" />
												{tag.name}
											</Badge>
										</Button>
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
