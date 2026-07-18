<script lang="ts">
  import { browser } from '$app/environment';
  import { goto } from '$app/navigation';
  import { searchService, feedService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { t } from '$lib/i18n';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { authStore } from '$lib/stores/auth.svelte';
  import * as Dialog from '$lib/components/ui/dialog';
  import Avatar from '$lib/components/common/avatar.svelte';
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
  let suggestAbortController: AbortController | null = null;

  function getSuggestionKey(item: SearchSuggestItem): string {
    switch (item.type) {
      case 'user':
        return `user_${item.payload.user.user_id}`;
      case 'group':
        return `group_${item.payload.group_id}`;
      case 'moment':
        return `moment_${item.payload.moment_id}`;
      case 'tag':
        return `tag_${item.payload.name}`;
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
    if (open === nextOpen) return;
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
      if (version === trendingLoadVersion) isLoadingTags = false;
    }
  }

  async function loadSuggestions(prefix: string) {
    if (!prefix) {
      suggestions = [];
      return;
    }
    suggestAbortController?.abort();
    const controller = new AbortController();
    suggestAbortController = controller;
    isSuggesting = true;
    try {
      const response = await searchService.suggest(
        {
          prefix,
          reference_point: {
            latitude: mapAnchorStore.latitude,
            longitude: mapAnchorStore.longitude
          }
        },
        { signal: controller.signal }
      );
      if (controller.signal.aborted) return;
      const rawItems = isSuccess(response) && response.data ? response.data.items : [];
      const currentUserId = authStore.currentUserId;
      suggestions = currentUserId
        ? rawItems.filter((item) => item.type !== 'user' || item.payload.user.user_id !== currentUserId)
        : rawItems;
    } catch {
      if (!controller.signal.aborted) suggestions = [];
    } finally {
      if (!controller.signal.aborted) isSuggesting = false;
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
    if (item.type === 'user' && item.payload.user?.user_id) {
      goto(`/users/${item.payload.user.user_id}`);
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

  function getSuggestionPresentation(item: SearchSuggestItem) {
    const unknown = t('searchModal.unknown');
    const unknownUser = t('searchModal.unknownUser');
    const unknownGroup = t('searchModal.unknownGroup');
    switch (item.type) {
      case 'tag':
        return {
          title: `#${item.payload.name || ''}`,
          subtitle: t('searchModal.usedCount', { count: item.payload.usage_count || 0 }),
          badge: t('searchModal.tagBadge'),
          avatarUrl: null,
          avatarFallback: null
        };
      case 'user': {
        const name = item.payload.user?.nickname || unknownUser;
        return {
          title: name,
          subtitle: item.payload.bio || null,
          badge: t('searchModal.userBadge'),
          avatarUrl: item.payload.user?.avatar_url,
          avatarFallback: name.slice(0, 2)
        };
      }
      case 'group': {
        const name = item.payload.name || unknownGroup;
        return {
          title: name,
          subtitle:
            item.payload.description ||
            t('searchModal.memberCount', { count: item.payload.members_count || 0 }),
          badge: t('searchModal.groupBadge'),
          avatarUrl: item.payload.avatar_url,
          avatarFallback: name.slice(0, 2)
        };
      }
      case 'moment': {
        const authorName = item.payload.author?.nickname || unknownUser;
        return {
          title: item.payload.description_preview || t('searchModal.momentFallback'),
          subtitle: item.payload.author?.nickname
            ? t('searchModal.fromAuthor', { nickname: item.payload.author.nickname })
            : null,
          badge: t('searchModal.momentBadge'),
          avatarUrl: item.payload.author?.avatar_url,
          avatarFallback: authorName.slice(0, 2)
        };
      }
      default:
        return {
          title: unknown,
          subtitle: null,
          badge: unknown,
          avatarUrl: null,
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
      suggestAbortController?.abort();
      suggestAbortController = null;
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
    return () => {
      if (suggestTimer) {
        clearTimeout(suggestTimer);
        suggestTimer = null;
      }
    };
  });
</script>

<Dialog.Root bind:open onOpenChange={setOpenState}>
  <Dialog.Content
    class="flex max-h-[85dvh] flex-col sm:max-w-3xl"
    style="background: var(--surface-bg); border: 1px solid var(--border-theme); box-shadow: var(--shadow-lg);"
  >
    <!-- Header -->
    <div class="flex items-center gap-3 px-5 pt-5 pb-0">
      <div
        class="flex items-center justify-center rounded-full"
        style="width: 40px; height: 40px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
      >
        <Search size={18} strokeWidth={1.5} color="var(--brand)" />
      </div>
      <div>
        <h2 class="text-title" style="font-weight: 600; color: var(--text-primary)">
          {t('searchModal.title')}
        </h2>
        <p class="text-meta" style="color: var(--text-secondary)">{t('searchModal.description')}</p>
      </div>
    </div>

    <!-- Body -->
    <div class="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
      <!-- Search bar -->
      <form
        class="input-surface flex items-center gap-3 rounded-[var(--radius-md)] px-4 py-2.5"
        onsubmit={(event) => {
          event.preventDefault();
          handleSearch();
        }}
      >
        <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
        <input
          type="text"
          name="global-modal-search"
          placeholder={t('searchModal.placeholder')}
          bind:value={keyword}
          autocomplete="off"
          aria-label={t('searchModal.inputAria')}
          class="text-body flex-1 bg-transparent outline-none"
          style="color: var(--text-primary)"
        />
        {#if keyword}
          <button
            type="button"
            class="flex items-center justify-center outline-none"
            onclick={clearSearch}
            aria-label={t('searchModal.clearInput')}
          >
            <X size={14} strokeWidth={1.5} color="var(--text-secondary)" />
          </button>
        {/if}
        <button
          type="submit"
          class="flex items-center justify-center rounded-full outline-none"
          style="width: 36px; height: 36px; background: var(--brand); box-shadow: var(--shadow-brand-btn);"
          disabled={!keyword.trim()}
          aria-label={t('searchModal.submit')}
        >
          <Search size={15} strokeWidth={1.5} color="var(--text-on-accent)" />
        </button>
      </form>

      {#if trimmedKeyword}
        <!-- Search suggestions -->
        <div
          class="rounded-[var(--radius-md)] p-4"
          style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
        >
          {#if isSuggesting}
            <div class="space-y-2">
              {#each Array(3) as _, i (i)}
                <div
                  class="rounded-[var(--radius-md)]"
                  style="height: 32px; background: var(--border-theme);"
                ></div>
              {/each}
            </div>
          {:else if !hasSuggestions}
            <p class="text-body" style="color: var(--text-secondary)">
              {t('searchModal.noSuggestions')}
            </p>
          {:else}
            <div class="space-y-4">
              {#if recentMatches.length > 0}
                <div class="space-y-2">
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-1.5">
                      <Clock size={12} strokeWidth={1.5} color="var(--text-secondary)" />
                      <span class="text-caption" style="color: var(--text-secondary)"
                        >{t('searchModal.matchedHistory')}</span
                      >
                    </div>
                    <button
                      class="text-caption outline-none"
                      style="color: var(--text-secondary)"
                      onclick={clearAllRecent}
                    >
                      {t('searchModal.clearAll')}
                    </button>
                  </div>
                  <div class="flex flex-wrap gap-2">
                    {#each recentMatches as term (term)}
                      <div class="flex items-center gap-1">
                        <button
                          class="text-meta rounded-full px-3 py-1 outline-none hover:opacity-80 active:scale-95 transition-all duration-150"
                          style="color: var(--text-primary); background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme)"
                          onclick={() => handleSearchNavigate(term)}
                        >
                          {term}
                        </button>
                        <button
                          class="flex items-center justify-center rounded-full outline-none"
                          style="width: 22px; height: 22px; background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
                          onclick={() => removeRecent(term)}
                          aria-label={getRemoveRecentLabel(term)}
                        >
                          <X size={10} strokeWidth={1.5} color="var(--text-secondary)" />
                        </button>
                      </div>
                    {/each}
                  </div>
                </div>
              {/if}

              {#if suggestions.length > 0}
                <div class="space-y-1">
                  <div class="flex items-center gap-1.5 mb-2">
                    <TrendingUp size={12} strokeWidth={1.5} color="var(--text-secondary)" />
                    <span class="text-caption" style="color: var(--text-secondary)"
                      >{t('searchModal.suggestions')}</span
                    >
                  </div>
                  {#each suggestions as item (getSuggestionKey(item))}
                    {@const p = getSuggestionPresentation(item)}
                    <button
                      type="button"
                      class="flex w-full items-center gap-3 rounded-[var(--radius-md)] px-3 py-2.5 text-left outline-none hover:bg-[var(--surface-secondary)] transition-colors duration-150"
                      style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
                      onclick={() => handleSuggestionClick(item)}
                    >
                      {#if item.type === 'tag'}
                        <div
                          class="flex flex-shrink-0 items-center justify-center rounded-full"
                          style="width: 34px; height: 34px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
                        >
                          <Hash size={14} strokeWidth={1.5} color="var(--brand)" />
                        </div>
                      {:else}
                        <Avatar
                          class="flex-shrink-0"
                          src={p.avatarUrl}
                          name={p.avatarFallback ?? ''}
                          size={32}
                        />
                      {/if}
                      <div class="min-w-0 flex-1">
                        <div class="flex items-center gap-2">
                          <span
                            class="text-body truncate"
                            style="font-weight: 500; color: var(--text-primary)">{p.title}</span
                          >
                          <span
                            class="text-xs flex-shrink-0 rounded-full px-1.5 py-0.5"
                            style="font-weight: 500; background: var(--brand); color: var(--text-on-accent)"
                            >{p.badge}</span
                          >
                        </div>
                        {#if p.subtitle}
                          <p
                            class="text-caption truncate"
                            style="color: var(--text-secondary); margin-top: 1px"
                          >
                            {p.subtitle}
                          </p>
                        {/if}
                      </div>
                    </button>
                  {/each}
                </div>
              {/if}
            </div>
          {/if}
          <p class="text-caption" style="color: var(--text-secondary); margin-top: 8px">
            {t('searchModal.clickSearchIcon')}
          </p>
        </div>
      {:else}
        <!-- Default: recent + trending -->
        <div class="grid gap-4 sm:grid-cols-2">
          <!-- Recent searches -->
          <div
            class="rounded-[var(--radius-md)] p-4"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-1.5">
                <Clock size={14} strokeWidth={1.5} color="var(--text-secondary)" />
                <span class="text-body" style="font-weight: 500; color: var(--text-primary)"
                  >{t('searchModal.recentSearches')}</span
                >
              </div>
              {#if recentSearches.length > 0}
                <button
                  class="text-caption outline-none"
                  style="color: var(--text-secondary)"
                  onclick={clearAllRecent}
                >
                  {t('searchModal.clearAll')}
                </button>
              {/if}
            </div>
            {#if recentSearches.length === 0}
              <p class="text-meta" style="color: var(--text-secondary)">
                {t('searchModal.noRecentSearches')}
              </p>
            {:else}
              <div class="flex flex-wrap gap-2">
                {#each recentSearches as term (term)}
                  <div class="flex items-center gap-1">
                    <button
                      class="text-meta rounded-full px-3 py-1 outline-none hover:opacity-80 active:scale-95 transition-all duration-150"
                      style="color: var(--text-primary); background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme)"
                      onclick={() => handleRecentClick(term)}
                    >
                      {term}
                    </button>
                    <button
                      class="flex items-center justify-center rounded-full outline-none"
                      style="width: 22px; height: 22px; background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
                      onclick={() => removeRecent(term)}
                      aria-label={getRemoveRecentLabel(term)}
                    >
                      <X size={10} strokeWidth={1.5} color="var(--text-secondary)" />
                    </button>
                  </div>
                {/each}
              </div>
            {/if}
          </div>

          <!-- Trending tags -->
          <div
            class="rounded-[var(--radius-md)] p-4"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div class="flex items-center gap-1.5 mb-3">
              <TrendingUp size={14} strokeWidth={1.5} color="var(--text-secondary)" />
              <span class="text-body" style="font-weight: 500; color: var(--text-primary)"
                >{t('searchModal.trendingTags')}</span
              >
            </div>
            {#if isLoadingTags}
              <div class="space-y-2">
                {#each Array(4) as _, i (i)}
                  <div
                    class="rounded-[var(--radius-md)]"
                    style="height: 28px; background: var(--border-theme);"
                  ></div>
                {/each}
              </div>
            {:else if trendingTags.length === 0}
              <p class="text-meta" style="color: var(--text-secondary)">
                {t('searchModal.noTrendingTags')}
              </p>
            {:else}
              <div class="flex flex-wrap gap-2">
                {#each trendingTags as tag (tag.name)}
                  <button
                    class="text-meta flex items-center gap-1 rounded-full px-3 py-1 outline-none hover:opacity-80 active:scale-95 transition-all duration-150"
                    style="color: var(--brand); background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme)"
                    onclick={() => handleTagClick(tag.name)}
                    aria-label={t('searchModal.searchHotTag', { name: tag.name })}
                  >
                    <Hash size={11} strokeWidth={1.5} />
                    {tag.name}
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  </Dialog.Content>
</Dialog.Root>
