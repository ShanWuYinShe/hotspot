<script lang="ts">
  import { browser } from '$app/environment';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import type { NotificationType } from '$lib/types/api';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { t } from '$lib/i18n';
  import PageHeader from '$lib/components/layout/page-header.svelte';
  import { Switch } from '$lib/components/ui/switch';
  import Bell from 'lucide-svelte/icons/bell';
  import CheckCheck from 'lucide-svelte/icons/check-check';
  import ExternalLink from 'lucide-svelte/icons/external-link';
  import Spinner from '$lib/components/common/spinner.svelte';

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

  const preferenceItems = $derived<
    Array<{ key: NotificationPreferenceGroup; title: string; description: string }>
  >([
    {
      key: 'social',
      title: t('settings.preferenceGroupSocial'),
      description: t('settings.preferenceGroupSocialDesc')
    },
    {
      key: 'interaction',
      title: t('settings.preferenceGroupInteraction'),
      description: t('settings.preferenceGroupInteractionDesc')
    },
    {
      key: 'group',
      title: t('settings.preferenceGroupGroup'),
      description: t('settings.preferenceGroupGroupDesc')
    },
    {
      key: 'system',
      title: t('settings.preferenceGroupSystem'),
      description: t('settings.preferenceGroupSystemDesc')
    }
  ]);

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
      toast.success(t('settings.savePreferencesSuccess'));
    } catch {
      toast.error(t('settings.savePreferencesFailed'));
    } finally {
      isSaving = false;
    }
  }

  async function markAllRead() {
    isMarkingAllRead = true;
    try {
      const ok = await notificationStore.markAllAsRead();
      if (ok) {
        toast.success(t('settings.markAllReadSuccess'));
      } else {
        toast.error(t('settings.markAllReadFailed'));
      }
    } catch {
      toast.error(t('settings.markAllReadFailed'));
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
  <title>{t('settings.notificationsPageHead')}</title>
</svelte:head>

<div class="page-pad">
  <div class="page-shell page-shell--form space-y-5">
    <PageHeader
      title={t('settings.notificationsPageTitle')}
      description={t('settings.notificationsPageDesc')}
      backHref="/settings"
      backLabel={t('settings.backToSettings')}
    />

    <!-- Notification Status Card -->
    <div
      class="rounded-[var(--radius-lg)] p-5"
      style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
    >
      <div class="mb-4 flex items-center gap-2">
        <div
          class="flex items-center justify-center rounded-full"
          style="width: 36px; height: 36px; border: 1px solid var(--border-theme);"
        >
          <Bell size={15} strokeWidth={1.5} color="var(--brand)" />
        </div>
        <div>
          <p class="text-body-lg" style="font-weight: 600; color: var(--text-primary)">{t('settings.notificationsStatus')}</p>
          <p class="text-meta" style="color: var(--text-secondary)">
            {t('settings.notificationsStatusDesc')}
          </p>
        </div>
      </div>

      <div
        class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-4"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <div>
          <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
            {t('settings.currentUnread')}
          </p>
          <p class="text-meta" style="color: var(--text-secondary)">
            {t('settings.currentUnreadDesc')}
          </p>
        </div>
        <div
          class="tabular-nums"
          style="font-size: var(--text-display); font-weight: 600; color: var(--brand);"
        >
          {notificationStore.unreadCount}
        </div>
      </div>

      <div class="mt-4 flex flex-col gap-3 sm:flex-row">
        <button
          class="flex h-11 flex-1 items-center justify-center rounded-full text-sm font-medium transition-opacity hover:opacity-90"
          style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
          onclick={() => goto('/notifications')}
        >
          <ExternalLink size={16} class="mr-2" />
          {t('settings.enterNotificationCenter')}
        </button>
        <button
          class="flex h-11 flex-1 items-center justify-center rounded-[var(--radius-md)] text-sm font-medium transition-opacity hover:opacity-90 disabled:opacity-50"
          style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-primary);"
          onclick={markAllRead}
          disabled={isMarkingAllRead || notificationStore.unreadCount === 0}
        >
          {#if isMarkingAllRead}
            <Spinner size="md" class="mr-2" />
          {:else}
            <CheckCheck size={16} class="mr-2" />
          {/if}
          {t('settings.markAllRead')}
        </button>
      </div>
    </div>

    <!-- Preference Explanation Card -->
    <div
      class="rounded-[var(--radius-lg)] p-5"
      style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
    >
      <div class="mb-3">
        <p class="text-body-lg" style="font-weight: 600; color: var(--text-primary)">{t('settings.preferenceExplanation')}</p>
        <p class="text-meta" style="color: var(--text-secondary)">
          {t('settings.preferenceExplanationDesc')}
        </p>
      </div>
      <div
        class="rounded-[var(--radius-md)] px-4 py-3"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <p class="text-body" style="color: var(--text-secondary); line-height: 1.6">
          {t('settings.preferenceExplanationDetail')}
        </p>
      </div>
    </div>

    <!-- Notification Preferences Card -->
    <div
      class="rounded-[var(--radius-lg)] p-5"
      style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
    >
      <div class="mb-4">
        <p class="text-body-lg" style="font-weight: 600; color: var(--text-primary)">
          {t('settings.browserPreference')}
        </p>
        <p class="text-meta" style="color: var(--text-secondary)">
          {t('settings.browserPreferenceDesc')}
        </p>
      </div>

      <div class="space-y-3">
        {#each preferenceItems as item (item.key)}
          <div
            class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-3.5"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <div>
              <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
                {item.title}
              </p>
              <p class="text-meta" style="color: var(--text-secondary)">{item.description}</p>
            </div>
            <Switch
              checked={isGroupEnabled(item.key)}
              onchange={(checked) => togglePreference(item.key, checked)}
              aria-label={item.title}
            />
          </div>
        {/each}
      </div>

      <button
        class="mt-4 flex h-11 w-full items-center justify-center rounded-full text-sm font-medium transition-opacity hover:opacity-90 disabled:opacity-50"
        style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
        onclick={savePreferences}
        disabled={isSaving}
      >
        {#if isSaving}
          <Spinner size="md" class="mr-2" />
        {/if}
        {t('settings.savePreferences')}
      </button>
    </div>
  </div>
</div>
