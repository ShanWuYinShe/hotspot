<script lang="ts">
  import { onMount } from 'svelte';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { usersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import { t } from '$lib/i18n';
  import PageHeader from '$lib/components/layout/page-header.svelte';
  import { Switch } from '$lib/components/ui/switch';
  import Spinner from '$lib/components/common/spinner.svelte';
  import type { PrivacyLevel, PrivateChatPermission, UserSettings } from '$lib/types/api';

  let settings = $state<UserSettings | null>(null);
  let isLoading = $state(true);
  let isSaving = $state(false);

  let privacyLevel = $state<PrivacyLevel>('public');
  let allowPrivateChat = $state<PrivateChatPermission>('everyone');
  let allowFollow = $state(true);
  let searchable = $state(true);
  let locationSearchable = $state(true);
  let locationVisible = $state(true);
  let presenceVisible = $state(true);
  let socialListsVisible = $state(true);

  async function loadSettings() {
    isLoading = true;
    try {
      const response = await withMinimumLoadingTime(() => usersService.getMySettings());
      if (isSuccess(response) && response.data) {
        settings = response.data;
        privacyLevel = settings.privacy_level;
        allowPrivateChat = settings.allow_private_chat;
        allowFollow = settings.allow_follow;
        searchable = settings.searchable;
        locationSearchable = settings.location_searchable;
        locationVisible = settings.location_visible;
        presenceVisible = settings.presence_visible;
        socialListsVisible = settings.social_lists_visible;
      }
    } catch {
      toast.error(t('settings.loadPrivacyFailed'));
    } finally {
      isLoading = false;
    }
  }

  async function saveSettings() {
    isSaving = true;
    try {
      const response = await usersService.updateMySettings({
        privacy_level: privacyLevel,
        allow_private_chat: allowPrivateChat,
        allow_follow: allowFollow,
        searchable,
        location_searchable: locationSearchable,
        location_visible: locationVisible,
        presence_visible: presenceVisible,
        social_lists_visible: socialListsVisible
      });
      if (isSuccess(response)) {
        toast.success(t('settings.privacySaveSuccess'));
      } else {
        toast.error(response.message || t('settings.privacySaveFailed'));
      }
    } catch {
      toast.error(t('settings.privacySaveFailed'));
    } finally {
      isSaving = false;
    }
  }

  onMount(() => {
    loadSettings();
  });
</script>

<svelte:head>
  <title>{t('settings.privacyPageTitle')}</title>
</svelte:head>

<div class="page-pad">
  <div class="page-shell page-shell--form space-y-5">
    <PageHeader
      title={t('settings.privacyHeader')}
      description={t('settings.privacyHeaderDesc')}
      backHref="/settings"
      backLabel={t('settings.backToSettings')}
    />

    {#if isLoading}
      <div
        class="rounded-[var(--radius-lg)] p-5"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <div class="space-y-4">
          {#each Array(5) as _, i (i)}
            <div
              class="h-11 w-full rounded-[var(--radius-md)]"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            ></div>
          {/each}
        </div>
      </div>
    {:else}
      <!-- Basic Visibility Card -->
      <div
        class="rounded-[var(--radius-lg)] p-5"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <p
          class="text-xs"
          style="letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 4px"
        >
          {t('settings.basicVisibility')}
        </p>
        <p class="text-meta" style="color: var(--text-secondary); margin-bottom: 16px">
          {t('settings.basicVisibilityDesc')}
        </p>

        <div class="space-y-3">
          <div
            class="rounded-[var(--radius-md)] p-4"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <label
              for="privacy-level"
              class="text-body"
              style="color: var(--text-secondary); display: block; margin-bottom: 8px"
              >{t('settings.privacyLevel')}</label
            >
            <select
              id="privacy-level"
              class="input-base w-full rounded-[var(--radius-md)] px-4 py-2.5 outline-none"
              style="border: 1px solid var(--border-theme); background: var(--surface-bg); color: var(--text-primary);"
              bind:value={privacyLevel}
            >
              <option value="public">{t('settings.privacyLevelPublic')}</option>
              <option value="private">{t('settings.privacyLevelPrivate')}</option>
            </select>
          </div>

          <div
            class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-3.5"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <div>
              <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
                {t('settings.locationVisible')}
              </p>
              <p class="text-meta" style="color: var(--text-secondary)">{t('settings.locationVisibleDesc')}</p>
            </div>
            <Switch bind:checked={locationVisible} aria-label={t('settings.locationVisibleAria')} />
          </div>

          <div
            class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-3.5"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <div>
              <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
                {t('settings.presenceVisible')}
              </p>
              <p class="text-meta" style="color: var(--text-secondary)">{t('settings.presenceVisibleDesc')}</p>
            </div>
            <Switch bind:checked={presenceVisible} aria-label={t('settings.presenceVisibleAria')} />
          </div>

          <div
            class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-3.5"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <div>
              <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
                {t('settings.socialListsVisible')}
              </p>
              <p class="text-meta" style="color: var(--text-secondary)">
                {t('settings.socialListsVisibleDesc')}
              </p>
            </div>
            <Switch bind:checked={socialListsVisible} aria-label={t('settings.socialListsVisibleAria')} />
          </div>
        </div>
      </div>

      <!-- Social & Chat Card -->
      <div
        class="rounded-[var(--radius-lg)] p-5"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <p
          class="text-xs"
          style="letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 4px"
        >
          {t('settings.socialAndChat')}
        </p>
        <p class="text-meta" style="color: var(--text-secondary); margin-bottom: 16px">
          {t('settings.socialAndChatDesc')}
        </p>

        <div class="space-y-3">
          <div
            class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-3.5"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <div>
              <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
                {t('settings.allowFollow')}
              </p>
              <p class="text-meta" style="color: var(--text-secondary)">{t('settings.allowFollowDesc')}</p>
            </div>
            <Switch bind:checked={allowFollow} aria-label={t('settings.allowFollowAria')} />
          </div>

          <div
            class="rounded-[var(--radius-md)] p-4"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <label
              for="private-chat"
              class="text-body"
              style="color: var(--text-secondary); display: block; margin-bottom: 8px"
              >{t('settings.privateChatPermission')}</label
            >
            <select
              id="private-chat"
              class="field-inset w-full rounded-[var(--radius-md)] px-4 py-2.5 outline-none"
              style="background: var(--surface-bg); color: var(--text-primary);"
              bind:value={allowPrivateChat}
            >
              <option value="everyone">{t('settings.privateChatEveryone')}</option>
              <option value="friends">{t('settings.privateChatFriends')}</option>
              <option value="nobody">{t('settings.privateChatNobody')}</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Search & Discovery Card -->
      <div
        class="rounded-[var(--radius-lg)] p-5"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <p
          class="text-xs"
          style="letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 4px"
        >
          {t('settings.searchAndDiscovery')}
        </p>
        <p class="text-meta" style="color: var(--text-secondary); margin-bottom: 16px">
          {t('settings.searchAndDiscoveryDesc')}
        </p>

        <div class="space-y-3">
          <div
            class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-3.5"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <div>
              <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
                {t('settings.searchable')}
              </p>
              <p class="text-meta" style="color: var(--text-secondary)">{t('settings.searchableDesc')}</p>
            </div>
            <Switch bind:checked={searchable} aria-label={t('settings.searchableAria')} />
          </div>

          <div
            class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-3.5"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            <div>
              <p class="text-body-lg" style="font-weight: 500; color: var(--text-primary)">
                {t('settings.nearbyVisible')}
              </p>
              <p class="text-meta" style="color: var(--text-secondary)">{t('settings.nearbyVisibleDesc')}</p>
            </div>
            <Switch bind:checked={locationSearchable} aria-label={t('settings.nearbyVisibleAria')} />
          </div>
        </div>
      </div>

      <button
        class="flex h-11 w-full items-center justify-center rounded-full text-sm font-medium transition-opacity hover:opacity-90 disabled:opacity-50"
        style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
        onclick={saveSettings}
        disabled={isSaving}
      >
        {#if isSaving}
          <Spinner size="md" class="mr-2" />
        {/if}
        {t('settings.saveSettings')}
      </button>
    {/if}
  </div>
</div>
