<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { formatRelativeTime } from '$lib/utils/format';
  import { privateChatsService, groupsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import { presenceStore } from '$lib/stores/presence.svelte';
  import { messageStore } from '$lib/stores/message.svelte';
  import type { PrivateChatItem, GroupItem } from '$lib/types/api';
  import Search from 'lucide-svelte/icons/search';
  import UserPlus from 'lucide-svelte/icons/user-plus';
  import Plus from 'lucide-svelte/icons/plus';
  import Avatar from '$lib/components/common/avatar.svelte';

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
        privateChatsService.getChats({ pagination: { limit: 50 } })
      );
      if (isSuccess(response) && response.data) {
        privateChats = response.data.items;
        messageStore.syncFromChatList(privateChats, []);
      }
    } catch {
      toast.error('加载失败');
    } finally {
      isLoadingPrivate = false;
    }
  }

  async function loadGroupChats() {
    if (groupsLoaded) return;
    isLoadingGroup = true;
    try {
      const response = await withMinimumLoadingTime(() =>
        groupsService.getMyGroups({ pagination: { limit: 50 } })
      );
      if (isSuccess(response) && response.data) {
        groupChats = response.data.items;
        groupsLoaded = true;
        messageStore.syncFromChatList(privateChats, groupChats);
      }
    } catch {
      toast.error('加载群聊失败');
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

  function isPrivateChatSelected(userId: string): boolean {
    const path = $page.url.pathname;
    return path === `/chat/users/${userId}`;
  }

  function isGroupSelected(groupId: string): boolean {
    const path = $page.url.pathname;
    return path === `/chat/groups/${groupId}`;
  }

  onMount(() => {
    loadPrivateChats();
  });
</script>

<!-- List header -->
<div class="flex-shrink-0 px-4 pt-4 pb-3" style="border-bottom: 1px solid var(--border-theme);">
  <div class="mb-3 flex items-center justify-between">
    <div>
      <h1 class="text-h2" style="color: var(--text-primary)">消息</h1>
      <p class="text-caption" style="color: var(--text-secondary); margin-top: 1px">
        {totalUnread > 0 ? `${totalUnread} 条未读` : '全部已读'}
      </p>
    </div>
    <div class="flex gap-1.5">
      <button
        class="flex items-center justify-center rounded-full outline-none"
        style="width: 32px; height: 32px; background: var(--surface-bg); box-shadow: 4px 4px 9px var(--shadow-dark), -4px -4px 9px var(--shadow-light); border: 1px solid var(--border-theme);"
      >
        <UserPlus size={14} strokeWidth={1.5} color="var(--text-secondary)" />
      </button>
      <button
        class="flex items-center justify-center rounded-full outline-none"
        style="width: 32px; height: 32px; background: var(--surface-bg); box-shadow: 4px 4px 9px var(--shadow-dark), -4px -4px 9px var(--shadow-light); border: 1px solid var(--border-theme);"
      >
        <Plus size={14} strokeWidth={1.5} color="var(--text-secondary)" />
      </button>
    </div>
  </div>
  <!-- Search -->
  <div class="input-surface flex items-center gap-3 rounded-[var(--radius-md)] px-4 py-2.5">
    <Search size={15} strokeWidth={1.5} color="var(--text-secondary)" class="flex-shrink-0" />
    <input
      type="text"
      name="chat-list-search"
      aria-label="搜索会话"
      bind:value={searchVal}
      placeholder="搜索…"
      class="text-body flex-1 bg-transparent outline-none"
      style="color: var(--text-primary)"
    />
  </div>
</div>

<!-- Online strip -->
{#if onlineChats.length > 0}
  <div class="flex-shrink-0 px-4 py-2.5" style="border-bottom: 1px solid var(--border-theme);">
    <p class="text-xs uppercase mb-2" style="letter-spacing: 0.12em; color: var(--text-secondary)">
      在线
    </p>
    <div class="flex gap-3 overflow-x-auto no-scrollbar">
      {#each onlineChats as chat (chat.other_user.user_id)}
        {@const selected = isPrivateChatSelected(chat.other_user.user_id)}
        <a
          href="/chat/users/{chat.other_user.user_id}"
          class="flex flex-col items-center gap-1 flex-shrink-0"
        >
          <div class="relative">
            <div
              class="overflow-hidden rounded-full"
              style="width: 36px; height: 36px; border: {selected
                ? '2px solid var(--brand)'
                : '1.5px solid var(--border-theme)'};"
            >
              <Avatar
                src={chat.other_user.avatar_url}
                name={chat.other_user.nickname}
                alt={chat.other_user.nickname}
                size={40}
                border={false}
              />
            </div>
            <span
              class="absolute bottom-0 right-0 rounded-full"
              style="width: 8px; height: 8px; background: var(--semantic-online); border: 1.5px solid var(--surface-bg);"
            ></span>
          </div>
          <span class="text-xs" style="color: var(--text-secondary)"
            >{chat.other_user.nickname}</span
          >
        </a>
      {/each}
    </div>
  </div>
{/if}

<!-- Tab pills -->
<div class="flex-shrink-0 flex items-center gap-2 px-4 pt-2 pb-1">
  {#each [{ id: 'private', label: '私聊' }, { id: 'group', label: '群聊' }] as tab (tab.id)}
    <button
      class="filter-chip"
      class:filter-chip--active={currentTab === tab.id}
      onclick={() => handleTabChange(tab.id)}
    >
      {tab.label}
    </button>
  {/each}
</div>

<!-- Chat list -->
<div class="flex-1 overflow-y-auto no-scrollbar px-2 py-2">
  {#if currentTab === 'private'}
    {#if isLoadingPrivate}
      {#each Array(5) as _}
        <div class="flex items-center gap-3 px-3 py-2.5">
          <div
            class="rounded-full"
            style="width: 40px; height: 40px; background: var(--border-theme);"
          ></div>
          <div class="flex-1">
            <div
              class="mb-1 rounded"
              style="width: 60px; height: 11px; background: var(--border-theme);"
            ></div>
            <div
              class="rounded"
              style="width: 100px; height: 9px; background: var(--border-theme);"
            ></div>
          </div>
        </div>
      {/each}
    {:else}
      {#each filteredPrivateChats as chat (chat.other_user.user_id)}
        {@const isOnline =
          presenceStore.isOnline(chat.other_user.user_id) || chat.presence_status === 'online'}
        {@const selected = isPrivateChatSelected(chat.other_user.user_id)}
        <a
          href="/chat/users/{chat.other_user.user_id}"
          class="flex items-center gap-3.5 cursor-pointer transition-all row-hover"
          style="padding: 10px 16px; border-radius: var(--radius-md); background: {selected
            ? 'var(--brand)'
            : 'transparent'}; box-shadow: {selected
            ? '0 4px 16px var(--glow)'
            : 'none'}; transition: background 0.16s ease;"
        >
          <div class="relative flex-shrink-0">
            <div
              class="overflow-hidden rounded-full"
              style="width: 40px; height: 40px; border: {selected
                ? '1.5px solid color-mix(in srgb, var(--text-on-accent) 35%, transparent)'
                : '1.5px solid var(--border-theme)'};"
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
              <span
                class="absolute bottom-0 right-0 rounded-full"
                style="width: 10px; height: 10px; background: var(--semantic-online); border: 2px solid var(--surface-bg);"
              ></span>
            {/if}
          </div>
          <div class="min-w-0 flex-1">
            <div class="mb-0.5 flex items-center justify-between">
              <span
                class="text-body-lg"
                style="color: {selected
                  ? 'var(--text-on-accent)'
                  : 'var(--text-primary)'}; font-weight: {chat.unread_count > 0 ? 600 : 400}"
                >{chat.other_user.nickname}</span
              >
              <span
                class="text-caption"
                style="flex-shrink: 0; color: {selected
                  ? 'color-mix(in srgb, var(--text-on-accent) 70%, transparent)'
                  : 'var(--text-secondary)'}">{formatRelativeTime(chat.last_activity_at)}</span
              >
            </div>
            <p
              class="text-meta truncate pr-4"
              style="color: {selected
                ? 'color-mix(in srgb, var(--text-on-accent) 80%, transparent)'
                : 'var(--text-secondary)'}"
            >
              {#if chat.last_message}{chat.last_message.content || '[媒体消息]'}{:else}暂无消息{/if}
            </p>
          </div>
          {#if chat.unread_count > 0 && !selected}
            <div
              class="text-caption flex flex-shrink-0 items-center justify-center rounded-full"
              style="width: 20px; height: 20px; background: var(--brand); color: var(--text-on-accent); font-weight: 600; box-shadow: var(--shadow-brand-btn)"
            >
              {chat.unread_count > 99 ? '99+' : chat.unread_count}
            </div>
          {/if}
        </a>
      {/each}
    {/if}
  {:else if isLoadingGroup}
    {#each Array(3) as _}
      <div class="flex items-center gap-3 px-3 py-2.5">
        <div
          class="rounded-full"
          style="width: 40px; height: 40px; background: var(--border-theme);"
        ></div>
        <div class="flex-1">
          <div
            class="mb-1 rounded"
            style="width: 70px; height: 11px; background: var(--border-theme);"
          ></div>
          <div
            class="rounded"
            style="width: 90px; height: 9px; background: var(--border-theme);"
          ></div>
        </div>
      </div>
    {/each}
  {:else}
    {#each filteredGroupChats as group (group.group.group_id)}
      {@const selected = isGroupSelected(group.group.group_id)}
      <a
        href="/chat/groups/{group.group.group_id}"
        class="flex items-center gap-3.5 cursor-pointer transition-all row-hover"
        style="padding: 10px 16px; border-radius: var(--radius-md); background: {selected
          ? 'var(--brand)'
          : 'transparent'}; box-shadow: {selected
          ? '0 4px 16px var(--glow)'
          : 'none'}; transition: background 0.16s ease;"
      >
        <div
          class="flex-shrink-0 overflow-hidden rounded-full"
          style="width: 40px; height: 40px; border: {selected
            ? '1.5px solid color-mix(in srgb, var(--text-on-accent) 35%, transparent)'
            : '1.5px solid var(--border-theme)'};"
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
              style="font-weight: 500; color: {selected
                ? 'var(--text-on-accent)'
                : 'var(--text-primary)'}">{group.group.name}</span
            >
            <span
              class="text-caption"
              style="flex-shrink: 0; color: {selected
                ? 'color-mix(in srgb, var(--text-on-accent) 70%, transparent)'
                : 'var(--text-secondary)'}">{group.group.members_count}人</span
            >
          </div>
          <p
            class="text-meta truncate pr-4"
            style="color: {selected
              ? 'color-mix(in srgb, var(--text-on-accent) 80%, transparent)'
              : 'var(--text-secondary)'}"
          >
            {formatRelativeTime(group.last_activity_at ?? group.my_state?.joined_at ?? '')}
          </p>
        </div>
        {#if (group.my_state?.unread_count ?? 0) > 0 && !selected}
          <div
            class="text-caption flex flex-shrink-0 items-center justify-center rounded-full"
            style="width: 20px; height: 20px; background: var(--brand); color: var(--text-on-accent); font-weight: 600; box-shadow: var(--shadow-brand-btn)"
          >
            {(group.my_state?.unread_count ?? 0) > 99 ? '99+' : (group.my_state?.unread_count ?? 0)}
          </div>
        {/if}
      </a>
    {/each}
  {/if}
</div>
