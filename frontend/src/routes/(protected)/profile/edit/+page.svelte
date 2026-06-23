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
  import BackButton from '$lib/components/common/back-button.svelte';
  import Camera from 'lucide-svelte/icons/camera';
  import User from 'lucide-svelte/icons/user';
  import FileText from 'lucide-svelte/icons/file-text';
  import Check from 'lucide-svelte/icons/check';
  import Loader2 from 'lucide-svelte/icons/loader-2';

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
        toast.error(response.message || '加载失败');
        void goto('/profile');
      }
    } catch {
      toast.error('加载失败');
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
      toast.error('请选择图片文件');
      return;
    }

    if (file.size > 10 * 1024 * 1024) {
      toast.error('图片大小不能超过 10MB');
      return;
    }

    try {
      const uploaded = await mediaService.uploadFile(file, 'avatar');
      avatarUrl = uploaded.thumbnail_url ?? '';
      avatarFileId = uploaded.file_id;
      toast.success('头像已更新，保存后生效');
    } catch {
      toast.error('头像上传失败');
    }

    input.value = '';
  }

  async function saveProfile() {
    const trimmedNickname = nickname.trim();
    const trimmedBio = bio.trim();

    if (!trimmedNickname) {
      toast.error('昵称不能为空');
      return;
    }
    if (trimmedNickname.length > 30) {
      toast.error('昵称长度需在 1-30 字符之间');
      return;
    }
    if (trimmedBio.length > 500) {
      toast.error('个人简介不能超过 500 字符');
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
        toast.error(response.message || '保存失败');
      }
    } catch {
      toast.error('保存失败');
    } finally {
      isSaving = false;
    }
  }

  const canSave = $derived(nickname.trim().length > 0 && !isSaving);
</script>

<svelte:head>
  <title>编辑资料 - HotSpot</title>
</svelte:head>

{#if isLoading}
  <div class="page-pad">
    <div class="page-shell page-shell--form flex flex-col items-center justify-center py-24">
      <Loader2 class="h-8 w-8 animate-spin" style="color: var(--brand);" />
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
      <h1 style="font-size: 20px; font-weight: 600; color: var(--text-primary);">编辑资料</h1>
    </div>
    <button
      class="flex items-center gap-1.5 rounded-full px-4 py-2 outline-none transition-all"
      style={saved
        ? 'background: var(--theme-success); color: var(--text-on-accent); font-size: 13px; box-shadow: 0 4px 14px color-mix(in srgb, var(--theme-success) 35%, transparent); border: none;'
        : canSave
          ? 'background: var(--brand); color: var(--text-on-accent); font-size: 13px; box-shadow: var(--shadow-brand-btn); border: none;'
          : 'background: var(--surface-secondary); color: var(--text-secondary); font-size: 13px; border: none; opacity: 0.5;'}
      onclick={saveProfile}
      disabled={!canSave}
    >
      {#if saved}
        <Check size={14} color="var(--text-on-accent)" strokeWidth={2.5} />
      {:else}
        保存
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
          class="absolute top-3 right-3 flex items-center gap-1.5 rounded-full outline-none"
          style="padding: 6px 12px; background: var(--overlay); backdrop-filter: blur(10px); border: 1px solid var(--glass-border); font-size: 12px; color: var(--text-on-accent);"
        >
          <Camera size={13} strokeWidth={1.75} />
          更换横幅
        </button>
      </div>

      <!-- Avatar edit -->
      <div class="flex items-center gap-4 -mt-8 ml-2">
        <div class="relative flex-shrink-0">
          <div
            class="overflow-hidden rounded-[var(--radius-lg)]"
            style="width: 72px; height: 72px; border: 3px solid var(--surface-bg); box-shadow: var(--shadow-lg);"
          >
            {#if avatarUrl}
              <img src={avatarUrl} alt="avatar" class="w-full h-full object-cover" />
            {:else}
              <div
                class="flex h-full w-full items-center justify-center"
                style="background: var(--surface-bg); font-size: 20px; color: var(--text-primary);"
              >
                {nickname.slice(0, 2)}
              </div>
            {/if}
          </div>
          <button
            class="absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none"
            style="width: 26px; height: 26px; background: var(--brand); border: 2px solid var(--surface-bg); box-shadow: var(--shadow-brand-btn);"
            title="更换头像"
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
          <p style="font-size: 16px; color: var(--text-primary);">{profile.nickname}</p>
          <p style="font-size: 12px; color: var(--text-secondary);">
            @{formatHandle(profile)}
          </p>
        </div>
      </div>

      <!-- Nickname field -->
      <div class="flex flex-col gap-1.5">
        <div class="flex items-center gap-1.5">
          <User style="width: 13px; height: 13px; color: var(--brand);" strokeWidth={1.75} />
          <span class="eyebrow">昵称</span>
        </div>
        <div class="input-surface rounded-[var(--radius-md)] px-4 py-3">
          <input
            id="nickname"
            bind:value={nickname}
            placeholder="你的名字"
            maxlength={30}
            class="w-full bg-transparent outline-none"
            style="color: var(--text-primary); font-size: 14px;"
          />
        </div>
      </div>

      <!-- Bio field -->
      <div class="flex flex-col gap-1.5">
        <div class="flex items-center gap-1.5">
          <FileText style="width: 13px; height: 13px; color: var(--brand);" strokeWidth={1.75} />
          <span class="eyebrow">简介</span>
        </div>
        <div class="input-surface rounded-[var(--radius-md)] px-4 py-3">
          <textarea
            id="bio"
            bind:value={bio}
            placeholder="介绍一下自己..."
            rows="3"
            maxlength={150}
            class="w-full bg-transparent outline-none resize-none"
            style="color: var(--text-primary); font-size: 14px; line-height: 1.65;"
          ></textarea>
          <div class="flex justify-end mt-1">
            <span
              style="font-size: 11px; color: {bio.length > 150 * 0.85
                ? 'var(--brand)'
                : 'var(--text-secondary)'};">{bio.length}/150</span
            >
          </div>
        </div>
      </div>

      <!-- Save button (bottom) -->
      <button
        class="flex w-full items-center justify-center gap-2 rounded-[var(--radius-lg)] px-4 py-3.5 outline-none transition-all"
        style={saved
          ? 'background: var(--theme-success); color: var(--text-on-accent); font-size: 14px; box-shadow: 0 4px 14px color-mix(in srgb, var(--theme-success) 35%, transparent); border: none;'
          : canSave
            ? 'background: var(--brand); color: var(--text-on-accent); font-size: 14px; box-shadow: var(--shadow-brand-btn); border: none;'
            : 'background: var(--surface-secondary); color: var(--text-secondary); font-size: 14px; border: none; opacity: 0.5;'}
        onclick={saveProfile}
        disabled={!canSave}
      >
        {#if saved}
          <Check size={16} color="var(--text-on-accent)" strokeWidth={2.5} />
          已保存
        {:else if isSaving}
          <Loader2 class="h-4 w-4 animate-spin" />
          保存中...
        {:else}
          保存
        {/if}
      </button>
    </div>
  </div>
{:else}
  <!-- Error state -->
  <div class="page-pad">
    <div class="page-shell page-shell--form flex flex-col items-center justify-center py-16">
      <p style="font-size: 14px; color: var(--text-secondary);">加载失败</p>
      <button
        class="rounded-[var(--radius-lg)] px-5 py-3.5 outline-none surface-flat mt-4"
        style="color: var(--brand); font-size: 14px;"
        onclick={() => void goto('/profile')}
      >
        返回个人资料
      </button>
    </div>
  </div>
{/if}
