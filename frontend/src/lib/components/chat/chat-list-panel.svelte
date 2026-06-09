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
  import Users from 'lucide-svelte/icons/users';
  import Search from 'lucide-svelte/icons/search';
  import UserPlus from 'lucide-svelte/icons/user-plus';
  import Plus from 'lucide-svelte/icons/plus';

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
      <h1 style="font-size: 18px; color: var(--text-primary);">消息</h1>
      <p style="font-size: 11px; color: var(--text-secondary); margin-top: 1px;">
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
  <div
    class="flex items-center gap-2 rounded-[14px] px-3 py-2"
    style="box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
  >
    <Search size={14} strokeWidth={1.5} color="var(--text-secondary)" />
    <input
      type="text"
      bind:value={searchVal}
      placeholder="搜索…"
      class="flex-1 bg-transparent outline-none"
      style="font-size: 12px; color: var(--text-primary);"
    />
  </div>
</div>

<!-- Online strip -->
{#if onlineChats.length > 0}
  <div class="flex-shrink-0 px-4 py-2.5" style="border-bottom: 1px solid var(--border-theme);">
    <p
      class="uppercase mb-2"
      style="font-size: 9px; letter-spacing: 0.12em; color: var(--text-secondary);"
    >
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
              <img
                src={chat.other_user.avatar_url || ''}
                alt={chat.other_user.nickname}
                class="h-full w-full rounded-full object-cover"
              />
            </div>
            <span
              class="absolute bottom-0 right-0 rounded-full"
              style="width: 8px; height: 8px; background: var(--semantic-online); border: 1.5px solid var(--surface-bg);"
            ></span>
          </div>
          <span style="font-size: 10px; color: var(--text-secondary);"
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
      class="rounded-full px-2 py-0.5 outline-none transition-all"
      style={currentTab === tab.id
        ? 'background: var(--brand); color: white; font-size: 11px; box-shadow: 0 2px 8px var(--glow);'
        : 'background: var(--surface-bg); color: var(--text-secondary); font-size: 11px; border: 1px solid var(--border-theme);'}
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
          style="padding: 10px 16px; border-radius: 18px; background: {selected
            ? 'var(--brand)'
            : 'transparent'}; box-shadow: {selected
            ? '0 4px 16px var(--glow)'
            : 'none'}; transition: background 0.16s ease;"
        >
          <div class="relative flex-shrink-0">
            <div
              class="overflow-hidden rounded-full"
              style="width: 40px; height: 40px; border: {selected
                ? '1.5px solid rgba(255,255,255,0.35)'
                : '1.5px solid var(--border-theme)'};"
            >
              <img
                src={chat.other_user.avatar_url || ''}
                alt={chat.other_user.nickname}
                class="h-full w-full object-cover"
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
                style="font-size: 14px; color: {selected
                  ? 'white'
                  : 'var(--text-primary)'}; font-weight: {chat.unread_count > 0 ? 600 : 400};"
                >{chat.other_user.nickname}</span
              >
              <span
                style="font-size: 11px; flex-shrink: 0; color: {selected
                  ? 'rgba(255,255,255,0.7)'
                  : 'var(--text-secondary)'};">{formatRelativeTime(chat.last_activity_at)}</span
              >
            </div>
            <p
              class="truncate pr-4"
              style="font-size: 12px; color: {selected
                ? 'rgba(255,255,255,0.8)'
                : 'var(--text-secondary)'};"
            >
              {#if chat.last_message}{chat.last_message.content || '[媒体消息]'}{:else}暂无消息{/if}
            </p>
          </div>
          {#if chat.unread_count > 0 && !selected}
            <div
              class="flex flex-shrink-0 items-center justify-center rounded-full"
              style="width: 20px; height: 20px; background: var(--brand); color: white; font-size: 11px; font-weight: 600; box-shadow: 0 2px 8px var(--glow);"
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
        style="padding: 10px 16px; border-radius: 18px; background: {selected
          ? 'var(--brand)'
          : 'transparent'}; box-shadow: {selected
          ? '0 4px 16px var(--glow)'
          : 'none'}; transition: background 0.16s ease;"
      >
        <div
          class="flex-shrink-0 overflow-hidden rounded-full"
          style="width: 40px; height: 40px; border: {selected
            ? '1.5px solid rgba(255,255,255,0.35)'
            : '1.5px solid var(--border-theme)'};"
        >
          {#if group.group.avatar_url}
            <img
              src={group.group.avatar_url}
              alt={group.group.name}
              class="h-full w-full object-cover"
            />
          {:else}
            <div
              class="flex h-full w-full items-center justify-center"
              style="background: var(--surface-secondary);"
            >
              <Users size={16} color="var(--text-tertiary)" />
            </div>
          {/if}
        </div>
        <div class="min-w-0 flex-1">
          <div class="mb-0.5 flex items-center justify-between">
            <span
              style="font-size: 14px; font-weight: 500; color: {selected
                ? 'white'
                : 'var(--text-primary)'};">{group.group.name}</span
            >
            <span
              style="font-size: 11px; flex-shrink: 0; color: {selected
                ? 'rgba(255,255,255,0.7)'
                : 'var(--text-secondary)'};">{group.group.members_count}人</span
            >
          </div>
          <p
            class="truncate pr-4"
            style="font-size: 12px; color: {selected
              ? 'rgba(255,255,255,0.8)'
              : 'var(--text-secondary)'};"
          >
            {formatRelativeTime(group.last_activity_at ?? group.my_state?.joined_at ?? '')}
          </p>
        </div>
        {#if (group.my_state?.unread_count ?? 0) > 0 && !selected}
          <div
            class="flex flex-shrink-0 items-center justify-center rounded-full"
            style="width: 20px; height: 20px; background: var(--brand); color: white; font-size: 11px; font-weight: 600; box-shadow: 0 2px 8px var(--glow);"
          >
            {(group.my_state?.unread_count ?? 0) > 99 ? '99+' : (group.my_state?.unread_count ?? 0)}
          </div>
        {/if}
      </a>
    {/each}
  {/if}
</div>
