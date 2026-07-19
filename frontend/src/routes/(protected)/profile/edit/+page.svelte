<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { usersService } from '$lib/api/services';
  import { mediaService } from '$lib/api/services/media';
  import { isSuccess } from '$lib/api/client';
  import { formatHandle } from '$lib/utils/format';
  import { authStore } from '$lib/stores/auth.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import type { MyProfile } from '$lib/types/api';
  import { t } from '$lib/i18n';
  import BackButton from '$lib/components/common/back-button.svelte';
  import Camera from 'lucide-svelte/icons/camera';
  import User from 'lucide-svelte/icons/user';
  import FileText from 'lucide-svelte/icons/file-text';
  import Check from 'lucide-svelte/icons/check';
  import Spinner from '$lib/components/common/spinner.svelte';
  import { Textarea } from '$lib/components/ui/textarea';

  let profile = $state<MyProfile | null>(null);
  let isLoading = $state(true);
  let isSaving = $state(false);
  let saved = $state(false);

  let nickname = $state('');
  let bio = $state('');
  let avatarUrl = $state('');
  let avatarFileId = $state<string | null>(null);

  let avatarInput: HTMLInputElement | undefined = $state();

  onMount(async () => {
    try {
      const response = await usersService.getMyProfile();
      if (isSuccess(response) && response.data) {
        profile = response.data;
        nickname = response.data.nickname ?? '';
        bio = response.data.bio ?? '';
        avatarUrl = response.data.avatar_url ?? '';
        authStore.setProfile(response.data);
      } else {
        toast.error(response.message || t('profileEdit.loadFailed'));
        void goto('/profile');
      }
    } catch {
      toast.error(t('profileEdit.loadFailed'));
      void goto('/profile');
    } finally {
      isLoading = false;
    }
  });

  function handleAvatarClick() {
    avatarInput?.click();
  }

  async function handleAvatarChange(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    if (!file.type.startsWith('image/')) {
      toast.error(t('profileEdit.avatarInvalidType'));
      return;
    }

    if (file.size > 10 * 1024 * 1024) {
      toast.error(t('profileEdit.avatarTooLarge'));
      return;
    }

    try {
      const uploaded = await mediaService.uploadFile(file, 'avatar');
      avatarUrl = uploaded.thumbnail_url ?? '';
      avatarFileId = uploaded.file_id;
      toast.success(t('profileEdit.avatarUpdated'));
    } catch {
      toast.error(t('profileEdit.avatarUploadFailed'));
    }

    input.value = '';
  }

  async function saveProfile() {
    const trimmedNickname = nickname.trim();
    const trimmedBio = bio.trim();

    if (!trimmedNickname) {
      toast.error(t('profileEdit.nicknameEmpty'));
      return;
    }
    if (trimmedNickname.length > 30) {
      toast.error(t('profileEdit.nicknameLengthError'));
      return;
    }
    if (trimmedBio.length > 500) {
      toast.error(t('profileEdit.bioTooLong'));
      return;
    }

    isSaving = true;
    try {
      const data: Parameters<typeof usersService.updateMyProfile>[0] = {
        nickname: trimmedNickname,
        bio: trimmedBio || undefined
      };

      if (avatarFileId) {
        data.avatar_file_id = avatarFileId;
      }

      const response = await usersService.updateMyProfile(data);
      if (isSuccess(response)) {
        const profileRes = await usersService.getMyProfile();
        if (isSuccess(profileRes) && profileRes.data) {
          profile = profileRes.data;
          authStore.setProfile(profile);
        }
        saved = true;
        setTimeout(() => {
          saved = false;
          void goto('/profile');
        }, 800);
      } else {
        toast.error(response.message || t('profileEdit.saveFailed'));
      }
    } catch {
      toast.error(t('profileEdit.saveFailed'));
    } finally {
      isSaving = false;
    }
  }

  const canSave = $derived(nickname.trim().length > 0 && !isSaving);
</script>

<svelte:head>
  <title>{t('profileEdit.title')} - HotSpot</title>
</svelte:head>

{#if isLoading}
  <div class="page-pad">
    <div class="page-shell page-shell--form flex flex-col items-center justify-center py-24">
      <Spinner size="xl" color="var(--brand)" />
    </div>
  </div>
{:else if profile}
  <!-- Header -->
  <div
    class="sticky top-0 z-20 flex items-center justify-between px-5 py-3"
    style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
  >
    <div class="flex items-center gap-3">
      <BackButton onclick={() => history.back()} size={36} />
      <h1 class="page-title" style="color: var(--text-primary);">{t('profileEdit.title')}</h1>
    </div>
    <button
      class="flex items-center gap-1.5 rounded-full px-4 py-2 outline-none transition-all"
      style={saved
        ? 'background: var(--theme-success); color: var(--text-on-accent); font-size: var(--text-body); box-shadow: 0 4px 14px color-mix(in srgb, var(--theme-success) 35%, transparent); border: none;'
        : canSave
          ? 'background: var(--brand); color: var(--text-on-accent); font-size: var(--text-body); box-shadow: var(--shadow-brand-btn); border: none;'
          : 'background: var(--surface-secondary); color: var(--text-secondary); font-size: var(--text-body); border: none; opacity: 0.5;'}
      onclick={saveProfile}
      disabled={!canSave}
    >
      {#if saved}
        <Check size={14} color="var(--text-on-accent)" strokeWidth={2.5} />
      {:else}
        {t('profileEdit.save')}
      {/if}
    </button>
  </div>

  <div class="page-pad">
    <div class="page-shell page-shell--form flex flex-col gap-5">
      <!-- Banner -->
      <div
        class="relative overflow-hidden rounded-[var(--radius-xl)]"
        style="height: clamp(120px, 18vw, 160px);"
      >
        <div
          class="absolute inset-0"
          style="background: linear-gradient(135deg, var(--brand), var(--glow)); opacity: 0.3;"
        ></div>
        <div
          class="absolute inset-0"
          style="background: linear-gradient(to bottom, transparent 40%, rgba(0,0,0,0.5));"
        ></div>
        <button
          class="text-meta absolute top-3 right-3 flex items-center gap-1.5 rounded-full outline-none"
          style="padding: 6px 12px; background: var(--overlay); backdrop-filter: blur(10px); border: 1px solid var(--glass-border); color: var(--text-on-accent)"
        >
          <Camera size={13} strokeWidth={1.75} />
          {t('profileEdit.changeBanner')}
        </button>
      </div>

      <!-- Avatar edit -->
      <div class="flex items-center gap-4 -mt-8 ml-2">
        <div class="relative flex-shrink-0">
          <div
            class="overflow-hidden rounded-[var(--radius-lg)]"
            style="width: 72px; height: 72px; border: 3px solid var(--surface-bg);"
          >
            {#if avatarUrl}
              <img src={avatarUrl} alt="avatar" class="w-full h-full object-cover" />
            {:else}
              <div
                class="text-h1 flex h-full w-full items-center justify-center"
                style="background: var(--surface-bg); color: var(--text-primary)"
              >
                {nickname.slice(0, 2)}
              </div>
            {/if}
          </div>
          <button
            class="absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none"
            style="width: 26px; height: 26px; background: var(--brand); border: 2 solid var(--surface-bg); box-shadow: var(--shadow-brand-btn);"
            title={t('profileEdit.changeAvatar')}
            onclick={handleAvatarClick}
          >
            <Camera size={12} color="var(--text-on-accent)" strokeWidth={2} />
          </button>
          <input
            type="file"
            accept="image/*"
            bind:this={avatarInput}
            class="hidden"
            onchange={handleAvatarChange}
          />
        </div>
        <div>
          <p class="text-title" style="color: var(--text-primary)">{profile.nickname}</p>
          <p class="text-meta" style="color: var(--text-secondary)">
            @{formatHandle(profile)}
          </p>
        </div>
      </div>

      <!-- Nickname field -->
      <div class="flex flex-col gap-1.5">
        <div class="flex items-center gap-1.5">
          <User style="width: 13px; height: 13px; color: var(--brand);" strokeWidth={1.75} />
          <span class="eyebrow">{t('profileEdit.nicknameLabel')}</span>
        </div>
        <div class="input-surface rounded-[var(--radius-md)] px-4 py-3">
          <input
            id="nickname"
            bind:value={nickname}
            placeholder={t('profileEdit.nicknamePlaceholder')}
            maxlength={30}
            class="text-body-lg w-full bg-transparent outline-none"
            style="color: var(--text-primary)"
          />
        </div>
      </div>

      <!-- Bio field -->
      <div class="flex flex-col gap-1.5">
        <div class="flex items-center gap-1.5">
          <FileText style="width: 13px; height: 13px; color: var(--brand);" strokeWidth={1.75} />
          <span class="eyebrow">{t('profileEdit.bioLabel')}</span>
        </div>
        <div class="input-surface rounded-[var(--radius-md)] px-4 py-3">
          <Textarea
            id="bio"
            variant="transparent"
            size="md"
            class="text-body-lg"
            bind:value={bio}
            placeholder={t('profileEdit.bioPlaceholder')}
            maxlength={150}
          />
          <div class="flex justify-end mt-1">
            <span
              class="text-caption"
              style="color: {bio.length > 150 * 0.85 ? 'var(--brand)' : 'var(--text-secondary)'}"
              >{bio.length}/150</span
            >
          </div>
        </div>
      </div>

      <!-- Save button (bottom) -->
      <button
        class="flex w-full items-center justify-center gap-2 rounded-[var(--radius-lg)] px-4 py-3.5 outline-none transition-all"
        style={saved
          ? 'background: var(--theme-success); color: var(--text-on-accent); font-size: var(--text-body-lg); box-shadow: 0 4px 14px color-mix(in srgb, var(--theme-success) 35%, transparent); border: none;'
          : canSave
            ? 'background: var(--brand); color: var(--text-on-accent); font-size: var(--text-body-lg); box-shadow: var(--shadow-brand-btn); border: none;'
            : 'background: var(--surface-secondary); color: var(--text-secondary); font-size: var(--text-body-lg); border: none; opacity: 0.5;'}
        onclick={saveProfile}
        disabled={!canSave}
      >
        {#if saved}
          <Check size={16} color="var(--text-on-accent)" strokeWidth={2.5} />
          {t('profileEdit.saved')}
        {:else if isSaving}
          <Spinner size="md" />
          {t('profileEdit.saving')}
        {:else}
          {t('profileEdit.save')}
        {/if}
      </button>
    </div>
  </div>
{:else}
  <!-- Error state -->
  <div class="page-pad">
    <div class="page-shell page-shell--form flex flex-col items-center justify-center py-16">
      <p class="text-body-lg" style="color: var(--text-secondary)">{t('profileEdit.loadFailed')}</p>
      <button
        class="text-body-lg rounded-[var(--radius-lg)] px-5 py-3.5 outline-none surface-flat mt-4"
        style="color: var(--brand)"
        onclick={() => void goto('/profile')}
      >
        {t('profileEdit.backToProfile')}
      </button>
    </div>
  </div>
{/if}
