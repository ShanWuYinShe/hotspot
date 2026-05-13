<script lang="ts">
  import { browser } from '$app/environment';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import type { NotificationType } from '$lib/types/api';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import PageHeader from '$lib/components/layout/page-header.svelte';
  import Bell from 'lucide-svelte/icons/bell';
  import CheckCheck from 'lucide-svelte/icons/check-check';
  import ExternalLink from 'lucide-svelte/icons/external-link';
  import Loader2 from 'lucide-svelte/icons/loader-2';

  type NotificationPreferenceGroup = 'social' | 'interaction' | 'group' | 'system';
  type NotificationPreferenceMap = Record<NotificationType, boolean>;

  const STORAGE_KEY = 'hotspot_notification_preferences_v2';
  const DEFAULT_PREFERENCES: NotificationPreferenceMap = {
    new_follower: true,
    mutual_follow: true,
    moment_liked: true,
    moment_commented: true,
    comment_liked: true,
    comment_reply: true,
    mentioned: true,
    group_invite_received: true,
    group_invite_accepted: true,
    group_application_received: true,
    group_application_accepted: true,
    system_announcement: true
  };

  const GROUP_TO_TYPES: Record<NotificationPreferenceGroup, NotificationType[]> = {
    social: ['new_follower', 'mutual_follow'],
    interaction: [
      'moment_liked',
      'moment_commented',
      'comment_liked',
      'comment_reply',
      'mentioned'
    ],
    group: [
      'group_invite_received',
      'group_invite_accepted',
      'group_application_received',
      'group_application_accepted'
    ],
    system: ['system_announcement']
  };

  const preferenceItems: Array<{
    key: NotificationPreferenceGroup;
    title: string;
    description: string;
  }> = [
    { key: 'social', title: '社交关系提醒', description: '新关注、互相关注等社交关系变动' },
    { key: 'interaction', title: '互动提醒', description: '点赞、评论、回复、提及等互动通知' },
    { key: 'group', title: '群组提醒', description: '群邀请、入群申请和群组相关通知' },
    { key: 'system', title: '系统公告', description: '平台公告、维护提醒和系统消息' }
  ];

  let preferences = $state<NotificationPreferenceMap>({ ...DEFAULT_PREFERENCES });
  let isSaving = $state(false);
  let isMarkingAllRead = $state(false);

  function loadPreferences() {
    if (!browser) return;
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return;
      const parsed = JSON.parse(raw) as Partial<NotificationPreferenceMap>;
      preferences = { ...DEFAULT_PREFERENCES, ...parsed };
    } catch {
      preferences = { ...DEFAULT_PREFERENCES };
    }
  }

  function isGroupEnabled(group: NotificationPreferenceGroup): boolean {
    return GROUP_TO_TYPES[group].every((type) => preferences[type]);
  }

  function togglePreference(group: NotificationPreferenceGroup, checked: boolean) {
    const next = { ...preferences };
    for (const type of GROUP_TO_TYPES[group]) {
      next[type] = checked;
    }
    preferences = next;
  }

  async function savePreferences() {
    if (!browser) return;
    isSaving = true;
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences));
      toast.success('通知偏好已保存到当前浏览器');
    } catch {
      toast.error('保存通知偏好失败');
    } finally {
      isSaving = false;
    }
  }

  async function markAllRead() {
    isMarkingAllRead = true;
    try {
      const ok = await notificationStore.markAllAsRead();
      if (ok) {
        toast.success('已将全部通知标记为已读');
      } else {
        toast.error('标记全部已读失败');
      }
    } catch {
      toast.error('标记全部已读失败');
    } finally {
      isMarkingAllRead = false;
    }
  }

  onMount(() => {
    loadPreferences();
    notificationStore.fetchUnreadCount();
  });
</script>

<svelte:head>
  <title>通知设置 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-3xl space-y-5">
  <PageHeader
    title="通知设置"
    description="管理通知偏好、未读状态和通知中心入口"
    backHref="/settings"
    backLabel="返回设置"
  />

  <!-- Notification Status Card -->
  <div
    class="rounded-[22px] p-5"
    style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
  >
    <div class="mb-4 flex items-center gap-2">
      <div
        class="flex items-center justify-center rounded-full"
        style="width: 36px; height: 36px; box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
      >
        <Bell size={15} strokeWidth={1.5} color="var(--brand)" />
      </div>
      <div>
        <p style="font-size: 14px; font-weight: 600; color: var(--text-primary);">通知状态</p>
        <p style="font-size: 12px; color: var(--text-secondary);">
          快速查看当前未读情况，并进入通知中心处理通知。
        </p>
      </div>
    </div>

    <div
      class="flex items-center justify-between rounded-[18px] px-4 py-4"
      style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
    >
      <div>
        <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">当前未读通知</p>
        <p style="font-size: 12px; color: var(--text-secondary);">
          未读数量会和通知下拉、通知列表保持一致
        </p>
      </div>
      <div style="font-size: 24px; font-weight: 600; color: var(--brand);" class="tabular-nums">
        {notificationStore.unreadCount}
      </div>
    </div>

    <div class="mt-4 flex flex-col gap-3 sm:flex-row">
      <button
        class="flex h-11 flex-1 items-center justify-center rounded-full text-sm font-medium transition-opacity hover:opacity-90"
        style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 4px 14px var(--glow);"
        onclick={() => goto('/notifications')}
      >
        <ExternalLink size={16} class="mr-2" />
        进入通知中心
      </button>
      <button
        class="flex h-11 flex-1 items-center justify-center rounded-[18px] text-sm font-medium transition-opacity hover:opacity-90 disabled:opacity-50"
        style="background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme); color: var(--text-primary);"
        onclick={markAllRead}
        disabled={isMarkingAllRead || notificationStore.unreadCount === 0}
      >
        {#if isMarkingAllRead}
          <Loader2 size={16} class="mr-2 animate-spin" />
        {:else}
          <CheckCheck size={16} class="mr-2" />
        {/if}
        全部标记为已读
      </button>
    </div>
  </div>

  <!-- Preference Explanation Card -->
  <div
    class="rounded-[22px] p-5"
    style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
  >
    <div class="mb-3">
      <p style="font-size: 14px; font-weight: 600; color: var(--text-primary);">偏好说明</p>
      <p style="font-size: 12px; color: var(--text-secondary);">
        当前仓库尚未接入服务端通知偏好协议，所以这里先用本地持久化保存一个与 NotificationType
        对齐的偏好模型。
      </p>
    </div>
    <div
      class="rounded-[18px] px-4 py-3"
      style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
    >
      <p style="font-size: 13px; color: var(--text-secondary); line-height: 1.6;">
        这些开关用于保留你在当前浏览器的通知偏好，并已按真实 NotificationType
        存储，后续若接入服务端协议，可直接映射到服务端字段。真正收到的通知仍以服务端与 WebSocket
        推送为准。
      </p>
    </div>
  </div>

  <!-- Notification Preferences Card -->
  <div
    class="rounded-[22px] p-5"
    style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
  >
    <div class="mb-4">
      <p style="font-size: 14px; font-weight: 600; color: var(--text-primary);">
        当前浏览器通知偏好
      </p>
      <p style="font-size: 12px; color: var(--text-secondary);">
        本页设置仅保存在当前浏览器，用于保留你的本地通知偏好；不会影响其他设备。
      </p>
    </div>

    <div class="space-y-3">
      {#each preferenceItems as item (item.key)}
        <div
          class="flex items-center justify-between rounded-[18px] px-4 py-3.5"
          style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
        >
          <div>
            <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">
              {item.title}
            </p>
            <p style="font-size: 12px; color: var(--text-secondary);">{item.description}</p>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={isGroupEnabled(item.key)}
            class="relative flex h-7 w-12 shrink-0 items-center rounded-full transition-colors"
            style={isGroupEnabled(item.key)
              ? 'background: var(--brand); box-shadow: 0 2px 8px var(--glow);'
              : 'background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);'}
            onclick={() => togglePreference(item.key, !isGroupEnabled(item.key))}
          >
            <span
              class="block h-5 w-5 rounded-full transition-transform"
              style="background: {isGroupEnabled(item.key)
                ? 'white'
                : 'var(--text-secondary)'}; transform: translateX({isGroupEnabled(item.key)
                ? '22px'
                : '4px'}); box-shadow: 0 1px 3px var(--neu-switch-shadow);"
            ></span>
          </button>
        </div>
      {/each}
    </div>

    <button
      class="mt-4 flex h-11 w-full items-center justify-center rounded-full text-sm font-medium transition-opacity hover:opacity-90 disabled:opacity-50"
      style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 4px 14px var(--glow);"
      onclick={savePreferences}
      disabled={isSaving}
    >
      {#if isSaving}
        <Loader2 size={16} class="mr-2 animate-spin" />
      {/if}
      保存通知偏好
    </button>
  </div>
</div>
