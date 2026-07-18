<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { formatRelativeTime } from '$lib/utils/format';
  import { privateChatsService, groupsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import { t } from '$lib/i18n';
  import { presenceStore } from '$lib/stores/presence.svelte';
  import { messageStore } from '$lib/stores/message.svelte';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import type { PrivateChatItem, GroupItem } from '$lib/types/api';
  import { PAGE_SIZE } from '$lib/config/pagination';
  import Search from 'lucide-svelte/icons/search';
  import UserPlus from 'lucide-svelte/icons/user-plus';
  import Plus from 'lucide-svelte/icons/plus';
  import Bell from 'lucide-svelte/icons/bell';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import Avatar from '$lib/components/common/avatar.svelte';
  import OnlineDot from '$lib/components/common/online-dot.svelte';

  let currentTab = $state<'private' | 'group'>('private');
  let privateChats = $state.raw<PrivateChatItem[]>([]);
  let groupChats = $state.raw<GroupItem[]>([]);
  let isLoadingPrivate = $state(true);
  let isLoadingGroup = $state(false);
  let groupsLoaded = $state(false);
  let searchVal = $state('');

  async function loadPrivateChats() {
    try {
      const response = await withMinimumLoadingTime(() =>
        privateChatsService.getChats({ pagination: { limit: PAGE_SIZE.chat } })
      );
      if (isSuccess(response) && response.data) {
        privateChats = response.data.items;
        messageStore.syncFromChatList(privateChats, []);
      }
    } catch {
      toast.error(t('chat.loadFailed'));
    } finally {
      isLoadingPrivate = false;
    }
  }

  async function loadGroupChats() {
    if (groupsLoaded) return;
    isLoadingGroup = true;
    try {
      const response = await withMinimumLoadingTime(() =>
        groupsService.getMyGroups({ pagination: { limit: PAGE_SIZE.chat } })
      );
      if (isSuccess(response) && response.data) {
        groupChats = response.data.items;
        groupsLoaded = true;
        messageStore.syncFromChatList(privateChats, groupChats);
      }
    } catch {
      toast.error(t('chat.loadGroupsFailed'));
    } finally {
      isLoadingGroup = false;
    }
  }

  function handleTabChange(value: string) {
    currentTab = value as 'private' | 'group';
    if (value === 'group' && !groupsLoaded) loadGroupChats();
  }

  const totalUnread = $derived(messageStore.totalUnreadCount);

  const onlineChats = $derived(
    privateChats.filter(
      (c) => presenceStore.isOnline(c.other_user.user_id) || c.presence_status === 'online'
    )
  );

  const filteredPrivateChats = $derived(
    searchVal.trim()
      ? privateChats.filter((c) =>
          c.other_user.nickname.toLowerCase().includes(searchVal.toLowerCase())
        )
      : privateChats
  );

  const filteredGroupChats = $derived(
    searchVal.trim()
      ? groupChats.filter((g) => g.group.name.toLowerCase().includes(searchVal.toLowerCase()))
      : groupChats
  );

  onMount(() => {
    // wide 模式由 ChatListPanel(layout 渲染)负责数据加载,这里跳过避免重复 onMount
    // 导致 messageStore.syncFromChatList 被调用两次、未读计数瞬时错乱。
    if (platformStore.useMasterDetail) return;
    loadPrivateChats();
  });
</script>

<svelte:head>
  <title>{t('chat.pageTitle')}</title>
</svelte:head>

{#if platformStore.useMasterDetail}
  <!-- Wide: layout 负责 split-pane 并在 /chat 根渲染空态，本页无需再渲染内容。 -->
  <div class="hidden"></div>
{:else}
  <!-- Mobile: single-column layout -->
  <div class="flex flex-col">
    <!-- Sticky header -->
    <div
      class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <div class="page-shell page-shell--narrow">
        <div class="mb-3 flex items-center justify-between">
          <div>
            <h1 class="page-title" style="color: var(--text-primary);">{t('chat.title')}</h1>
            <p class="text-meta" style="color: var(--text-secondary); margin-top: 2px">
              {totalUnread > 0 ? t('chat.unreadCount', { count: totalUnread }) : t('chat.allRead')}
            </p>
          </div>
          <div class="flex gap-2.5">
            <button
              class="relative flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.9] surface-flat"
              style="width: 38px; height: 38px;"
              aria-label={t('chat.notificationsAria')}
              onclick={() => goto('/notifications')}
            >
              <Bell size={16} strokeWidth={1.5} color="var(--text-secondary)" />
              {#if notificationStore.unreadCount > 0}
                <span
                  class="text-xs absolute flex items-center justify-center rounded-full"
                  style="top: -2px; right: -2px; height: 16px; min-width: 16px; padding: 0 4px; background: var(--brand); color: var(--text-on-accent); font-weight: 600; line-height: 16px; box-shadow: var(--shadow-brand-btn)"
                >
                  {notificationStore.unreadCount > 99 ? '99+' : notificationStore.unreadCount}
                </span>
              {/if}
            </button>
            <button
              class="flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.9] surface-flat"
              style="width: 38px; height: 38px;"
              aria-label={t('chat.startPrivateChatAria')}
              onclick={() => goto('/users/nearby')}
            >
              <UserPlus size={16} strokeWidth={1.5} color="var(--text-secondary)" />
            </button>
            <button
              class="flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.9] surface-flat"
              style="width: 38px; height: 38px;"
              aria-label={t('chat.discoverGroupsAria')}
              onclick={() => goto('/groups/nearby')}
            >
              <Plus size={16} strokeWidth={1.5} color="var(--text-secondary)" />
            </button>
          </div>
        </div>

        <!-- Search bar -->
        <div class="input-surface flex items-center gap-3 rounded-[var(--radius-md)] px-4 py-2.5">
          <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
          <input
            type="text"
            name="chat-search"
            aria-label={t('chat.searchAria')}
            bind:value={searchVal}
            placeholder={t('chat.searchPlaceholder')}
            class="text-body flex-1 bg-transparent outline-none"
            style="color: var(--text-primary)"
          />
        </div>
      </div>
    </div>

    <!-- Content -->
    <div class="page-pad">
      <div class="page-shell page-shell--narrow flex flex-col">
        <!-- Online avatars strip -->
        {#if onlineChats.length > 0}
          <div class="mb-4">
            <p
              class="text-xs uppercase mb-2.5"
              style="letter-spacing: 0.14em; color: var(--text-secondary)"
            >
              {t('common.online')}
            </p>
            <div class="flex gap-4 overflow-x-auto no-scrollbar pb-1 h-scroll">
              {#each onlineChats as chat (chat.other_user.user_id)}
                <a
                  href="/chat/users/{chat.other_user.user_id}"
                  class="flex flex-col items-center gap-1.5 flex-shrink-0 cursor-pointer"
                >
                  <div class="relative">
                    <div
                      class="neu-flat overflow-hidden rounded-full"
                      style="width: 52px; height: 52px; padding: 2.5px; background: var(--surface-bg); border: 1.5px solid var(--brand);"
                    >
                      <Avatar
                        src={chat.other_user.avatar_url}
                        name={chat.other_user.nickname}
                        alt={chat.other_user.nickname}
                        size={48}
                        border={false}
                      />
                    </div>
                    <OnlineDot size="lg" />
                  </div>
                  <span class="text-caption" style="color: var(--text-secondary)"
                    >{chat.other_user.nickname}</span
                  >
                </a>
              {/each}
            </div>
          </div>
        {/if}

        <!-- Tab switcher -->
        <div class="mb-2.5 flex items-center gap-2">
          <p class="text-xs uppercase" style="letter-spacing: 0.14em; color: var(--text-secondary)">
            {t('chat.recent')}
          </p>
          <div class="flex-1"></div>
          {#each [{ id: 'private', label: t('chat.privateChats') }, { id: 'group', label: t('chat.groupChats') }] as tab (tab.id)}
            <button
              class="filter-chip"
              class:filter-chip--active={currentTab === tab.id}
              onclick={() => handleTabChange(tab.id)}
            >
              {tab.label}
            </button>
          {/each}
        </div>

        <!-- Chat list card -->
        <div class="surface-card overflow-hidden rounded-[var(--radius-lg)]">
          {#if currentTab === 'private'}
            {#if isLoadingPrivate}
              {#each Array(5) as _}
                <div class="flex items-center gap-3.5 px-4 py-3">
                  <div
                    class="rounded-full"
                    style="width: 44px; height: 44px; background: var(--border-theme);"
                  ></div>
                  <div class="flex-1">
                    <div
                      class="mb-1.5 rounded"
                      style="width: 70px; height: 12px; background: var(--border-theme);"
                    ></div>
                    <div
                      class="rounded"
                      style="width: 120px; height: 10px; background: var(--border-theme);"
                    ></div>
                  </div>
                </div>
              {/each}
            {:else if filteredPrivateChats.length === 0}
              <div class="flex flex-col items-center justify-center gap-3 py-12">
                <MessageCircle
                  size={36}
                  strokeWidth={1.25}
                  color="var(--text-secondary)"
                  style="opacity: 0.5;"
                />
                <p class="text-body-lg" style="color: var(--text-secondary)">{t('chat.noChatsFound')}</p>
              </div>
            {:else}
              {#each filteredPrivateChats as chat, i (chat.other_user.user_id)}
                {@const isOnline =
                  presenceStore.isOnline(chat.other_user.user_id) ||
                  chat.presence_status === 'online'}
                <div class="px-2 py-1">
                  <a
                    href="/chat/users/{chat.other_user.user_id}"
                    class="flex items-center gap-3.5 rounded-[var(--radius-md)] px-4 py-3 transition-all list-row-hover active:scale-[0.98]"
                  >
                    <div class="relative flex-shrink-0">
                      <div
                        class="overflow-hidden rounded-full"
                        style="width: 44px; height: 44px; border: 1.5px solid var(--border-theme);"
                      >
                        <Avatar
                          src={chat.other_user.avatar_url}
                          name={chat.other_user.nickname}
                          alt={chat.other_user.nickname}
                          size={40}
                          border={false}
                        />
                      </div>
                      {#if isOnline}
                        <OnlineDot size="sm" />
                      {/if}
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="mb-0.5 flex items-center justify-between">
                        <span
                          class="text-body-lg"
                          style="color: var(--text-primary); font-weight: {chat.unread_count > 0
                            ? 600
                            : 400}">{chat.other_user.nickname}</span
                        >
                        <span class="text-caption" style="color: var(--text-secondary)"
                          >{formatRelativeTime(chat.last_activity_at)}</span
                        >
                      </div>
                      <p class="text-meta truncate pr-4" style="color: var(--text-secondary)">
                        {#if chat.last_message}{chat.last_message.content ||
                            t('chat.mediaMessage')}{:else}{t('chat.noMessages')}{/if}
                      </p>
                    </div>
                    {#if chat.unread_count > 0}
                      <div
                        class="text-caption flex flex-shrink-0 items-center justify-center rounded-full"
                        style="width: 20px; height: 20px; background: var(--brand); color: var(--text-on-accent); font-weight: 600; box-shadow: var(--shadow-brand-btn)"
                      >
                        {chat.unread_count > 99 ? '99+' : chat.unread_count}
                      </div>
                    {/if}
                  </a>
                </div>
                {#if i < filteredPrivateChats.length - 1}
                  <div class="mx-4" style="height: 1px; background: var(--border-theme);"></div>
                {/if}
              {/each}
            {/if}
          {:else if isLoadingGroup}
            {#each Array(5) as _}
              <div class="flex items-center gap-3.5 px-4 py-3">
                <div
                  class="rounded-full"
                  style="width: 44px; height: 44px; background: var(--border-theme);"
                ></div>
                <div class="flex-1">
                  <div
                    class="mb-1.5 rounded"
                    style="width: 80px; height: 12px; background: var(--border-theme);"
                  ></div>
                  <div
                    class="rounded"
                    style="width: 100px; height: 10px; background: var(--border-theme);"
                  ></div>
                </div>
              </div>
            {/each}
          {:else if filteredGroupChats.length === 0}
            <div class="flex flex-col items-center justify-center gap-3 py-12">
              <MessageCircle
                size={36}
                strokeWidth={1.25}
                color="var(--text-secondary)"
                style="opacity: 0.5;"
              />
              <p class="text-body-lg" style="color: var(--text-secondary)">{t('chat.noGroupsFound')}</p>
            </div>
          {:else}
            {#each filteredGroupChats as group, i (group.group.group_id)}
              <div class="px-2 py-1">
                <a
                  href="/chat/groups/{group.group.group_id}"
                  class="flex items-center gap-3.5 rounded-[var(--radius-md)] px-4 py-3 transition-all list-row-hover active:scale-[0.98]"
                >
                  <div
                    class="flex-shrink-0 overflow-hidden rounded-full"
                    style="width: 44px; height: 44px; border: 1.5px solid var(--border-theme);"
                  >
                    <Avatar
                      src={group.group.avatar_url}
                      name={group.group.name}
                      alt={group.group.name}
                      size={40}
                      type="group"
                      border={false}
                    />
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="mb-0.5 flex items-center justify-between">
                      <span
                        class="text-body-lg"
                        style="color: var(--text-primary); font-weight: 500"
                        >{group.group.name}</span
                      >
                      <span class="text-caption" style="color: var(--text-secondary)"
                        >{t('chat.membersCount', { count: group.group.members_count })}</span
                      >
                    </div>
                    <p class="text-meta" style="color: var(--text-secondary)">
                      {formatRelativeTime(
                        group.last_activity_at ?? group.my_state?.joined_at ?? ''
                      )}
                    </p>
                  </div>
                  {#if (group.my_state?.unread_count ?? 0) > 0}
                    <div
                      class="text-caption flex flex-shrink-0 items-center justify-center rounded-full"
                      style="width: 20px; height: 20px; background: var(--brand); color: var(--text-on-accent); font-weight: 600; box-shadow: var(--shadow-brand-btn)"
                    >
                      {(group.my_state?.unread_count ?? 0) > 99
                        ? '99+'
                        : (group.my_state?.unread_count ?? 0)}
                    </div>
                  {/if}
                </a>
              </div>
              {#if i < filteredGroupChats.length - 1}
                <div class="mx-4" style="height: 1px; background: var(--border-theme);"></div>
              {/if}
            {/each}
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}
