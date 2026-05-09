<script lang="ts">
  import { onMount } from 'svelte';
  import { usersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import type { MyProfile } from '$lib/types/api';
  import Camera from 'lucide-svelte/icons/camera';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Heart from 'lucide-svelte/icons/heart';
  import Share2 from 'lucide-svelte/icons/share-2';
  import Settings from 'lucide-svelte/icons/settings';
  import Pencil from 'lucide-svelte/icons/pencil';
  import Grid3X3 from 'lucide-svelte/icons/grid-3x3';
  import Bookmark from 'lucide-svelte/icons/bookmark';
  import Check from 'lucide-svelte/icons/check';
  import X from 'lucide-svelte/icons/x';
  import Users from 'lucide-svelte/icons/users';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { goto } from '$app/navigation';

  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  let profile = $state<MyProfile | null>(null);
  let isSaving = $state(false);
  let isEditing = $state(false);
  let activeTab = $state<'posts' | 'liked' | 'saved'>('posts');

  let nickname = $state('');
  let bio = $state('');
  let tags = $state('');
  const MAX_TAGS = 10;

  $effect(() => {
    profile = data.profile;
    if (data.profile && !isEditing) {
      nickname = data.profile.nickname ?? '';
      bio = data.profile.bio ?? '';
      tags = data.profile.tags?.join(', ') ?? '';
    }
  });

  onMount(() => {
    if (profile) {
      authStore.setProfile(profile);
    }
  });

  async function saveProfile() {
    const trimmedNickname = nickname.trim();
    const trimmedBio = bio.trim();
    const parsedTags = tags
      .split(',')
      .map((t) => t.trim())
      .filter((t) => t);

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
    if (parsedTags.length > MAX_TAGS) {
      toast.error('标签最多 10 个');
      return;
    }

    isSaving = true;
    try {
      const response = await usersService.updateMyProfile({
        nickname: trimmedNickname,
        bio: trimmedBio || undefined,
        tags: parsedTags
      });

      if (isSuccess(response)) {
        const profileRes = await usersService.getMyProfile();
        if (isSuccess(profileRes) && profileRes.data) {
          profile = profileRes.data;
          authStore.setProfile(profile);
        }
        toast.success('保存成功');
        isEditing = false;
      } else {
        toast.error(response.message || '保存失败');
      }
    } catch {
      toast.error('保存失败');
    } finally {
      isSaving = false;
    }
  }

  function cancelEdit() {
    if (profile) {
      nickname = profile.nickname;
      bio = profile.bio ?? '';
      tags = profile.tags?.join(', ') ?? '';
    }
    isEditing = false;
  }

  const tabs = [
    { id: 'posts' as const, icon: Grid3X3, label: '帖子' },
    { id: 'liked' as const, icon: Heart, label: '喜欢' },
    { id: 'saved' as const, icon: Bookmark, label: '收藏' }
  ];
</script>

<svelte:head>
  <title>个人资料 - HotSpot</title>
</svelte:head>

{#if profile}
  {#if platformStore.isDesktop}
    tt<!-- Desktop: two-column layout (260px left + right content) -->
    tt
    <div class="flex flex-col h-full">
      <!-- Sticky header -->

      <div
        class="flex-shrink-0 flex items-center justify-between px-6 py-4"
        style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
      >
        <div>
          <p style="font-size: 20px; color: var(--neu-text);">{profile.nickname}</p>
          <p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 1px;">
            @{profile.nickname.toLowerCase().replace(/\s+/g, '.')}
          </p>
        </div>
        <div class="flex gap-2.5">
          <button
            class="flex items-center justify-center rounded-full outline-none"
            style="width: 36px; height: 36px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
          >
            t<Share2 size={15} strokeWidth={1.5} color="var(--neu-text-muted)" />
          </button>
          <button
            class="flex items-center justify-center rounded-full outline-none"
            style="width: 36px; height: 36px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
            onclick={() => void goto('/settings')}
          >
            t<Settings size={14} strokeWidth={1.5} color="var(--neu-text-muted)" />
          </button>
        </div>
      </div>
      <!-- Two-column body -->

      <div class="flex flex-1 min-h-0 overflow-hidden">
        <!-- Left panel (260px) -->
        <div
          class="flex-shrink-0 flex flex-col items-center overflow-y-auto no-scrollbar"
          style="width: 260px; border-right: 1px solid var(--neu-border); padding: 24px 20px;"
        >
          <!-- Avatar -->
          <div class="relative mb-4">
            t
            <div
              class="pointer-events-none absolute inset-0 rounded-full"
              style="background: var(--neu-glow); filter: blur(20px); transform: scale(1.18); opacity: 0.5;"
            ></div>
            t
            <div
              class="relative rounded-full"
              style="width: 96px; height: 96px; padding: 2px; background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1.5px solid var(--neu-border);"
            >
              tt{#if profile.avatar_url}
                <img
                  src={profile.avatar_url}
                  alt={profile.nickname}
                  class="h-full w-full rounded-full object-cover"
                />
                tt{:else}
                <div
                  class="flex h-full w-full items-center justify-center rounded-full"
                  style="font-size: 28px; font-weight: 700; color: var(--neu-text);"
                >
                  {profile.nickname.slice(0, 2)}
                </div>
                tt{/if}
              t
            </div>
            t<button
              class="absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none"
              style="width: 26px; height: 26px; background: var(--neu-accent-primary); border: 2.5px solid var(--neu-bg); box-shadow: 0 2px 8px var(--neu-glow);"
              title="更换头像"
            >
              tt<Camera size={12} color="white" strokeWidth={2} />
              t</button
            >
          </div>
          <h2 style="font-size: 18px; color: var(--neu-text);">{profile.nickname}</h2>
          {#if profile.bio}
            t
            <p
              style="font-size: 12px; color: var(--neu-text-muted); text-align: center; padding: 6px 8px; line-height: 1.5;"
            >
              {profile.bio}
            </p>
          {/if}
          {#if profile.location?.location_name}
            t
            <div class="flex items-center gap-1 mb-2">
              tt<MapPin size={11} strokeWidth={1.5} color="var(--neu-accent-primary)" />
              tt<span style="font-size: 11px; color: var(--neu-text-muted);"
                >{profile.location.location_name}</span
              >
              t
            </div>
          {/if}
          <!-- Edit button -->
          <button
            class="w-full flex items-center justify-center gap-2 rounded-full py-2.5 mb-5 outline-none"
            style="background: var(--neu-bg); color: var(--neu-text-muted); border: 1px solid var(--neu-border); box-shadow: 4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light); font-size: 13px;"
            onclick={() => (isEditing = !isEditing)}
          >
            t<Pencil size={13} strokeWidth={1.5} />
            t编辑资料
          </button>
          <!-- Stats card -->
          <div
            class="w-full rounded-[18px] mb-4"
            style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
          >
            t{#each [{ label: '帖子', value: profile.stats.moments_count }, { label: '关注者', value: profile.stats.followers_count }, { label: '足迹', value: profile.stats.friends_count }] as stat}
              tt
              <div class="flex items-center justify-between px-4 py-3">
                <span style="font-size: 13px; color: var(--neu-text-muted);">{stat.label}</span>
                <span style="font-size: 17px; color: var(--neu-text);">{stat.value}</span>
                tt
              </div>
              t{/each}
          </div>
          <!-- Quick links -->
          <div
            class="w-full rounded-[18px]"
            style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
          >
            t{#each [{ icon: Grid3X3, label: '我的瞬间', count: profile.stats.moments_count }, { icon: Users, label: '好友', count: profile.stats.friends_count }] as link}
              tt<a href="'/profile'" class="flex items-center gap-3 px-4 py-2.5 neu-row-hover">
                <link.icon size={14} strokeWidth={1.5} color="var(--neu-text-muted)" />
                <span class="flex-1" style="font-size: 13px; color: var(--neu-text-muted);"
                  >{link.label}</span
                >
                <span style="font-size: 12px; color: var(--neu-text-muted);">{link.count}</span>
                tt</a
              >
              t{/each}
          </div>
        </div>
        <!-- Right panel: gallery -->
        <div class="flex-1 overflow-y-auto no-scrollbar px-6 py-5">
          <div class="flex items-center justify-between mb-4">
            t
            <h3 style="font-size: 16px; color: var(--neu-text);">最近瞬间</h3>
          </div>
          {#if profile.stats.moments_count === 0}
            t
            <div class="flex flex-col items-center justify-center py-16">
              tt
              <div
                class="flex items-center justify-center rounded-full"
                style="width: 72px; height: 72px; box-shadow: 8px 8px 18px var(--neu-shadow-dark), -8px -8px 18px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
              >
                <Grid3X3
                  size={28}
                  color="var(--neu-text-muted)"
                  strokeWidth={1.25}
                  style="opacity: 0.5;"
                />
                tt
              </div>
              tt
              <p style="font-size: 14px; color: var(--neu-text-muted); margin-top: 12px;">
                还没有发布帖子
              </p>
              t
            </div>
          {:else}
            t
            <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px;">
              tt{#each Array(Math.min(profile.stats.moments_count, 9)) as _}
                <div
                  class="relative cursor-pointer overflow-hidden rounded-[16px]"
                  style="aspect-ratio: 1/1; background: var(--neu-fallback-bg); box-shadow: 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
                >
                  <div class="flex h-full w-full items-center justify-center">
                    <Grid3X3
                      size={20}
                      color="var(--neu-text-muted)"
                      strokeWidth={1.25}
                      style="opacity: 0.3;"
                    />
                  </div>
                </div>
                tt{/each}
              t
            </div>
          {/if}
        </div>
      </div>
      tt
    </div>
    t{:else}
    <div class="flex flex-col">
      <!-- Sticky header -->
      <div
        class="sticky top-0 z-20 flex items-center justify-between px-5 py-3"
        style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
      >
        <div>
          <p style="font-size: 18px; color: var(--neu-text);">{profile.nickname}</p>
          <p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 1px;">
            @{profile.nickname.toLowerCase().replace(/\s+/g, '.')}
          </p>
        </div>
        <div class="flex gap-2.5">
          <button
            class="flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.9]"
            style="width: 38px; height: 38px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
          >
            <Share2 size={15} strokeWidth={1.5} color="var(--neu-text-muted)" />
          </button>
          <button
            class="flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.9]"
            style="width: 38px; height: 38px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
            onclick={() => void goto('/settings')}
          >
            <Settings size={15} strokeWidth={1.5} color="var(--neu-text-muted)" />
          </button>
        </div>
      </div>

      <div class="flex flex-col pb-6">
        <!-- Avatar section (centered with glow) -->
        <div class="flex flex-col items-center px-5 pt-5 pb-4">
          <div class="relative mb-4">
            <div
              class="pointer-events-none absolute inset-0 rounded-full"
              style="background: var(--neu-glow); filter: blur(22px); transform: scale(1.18); opacity: 0.55;"
            ></div>
            <div
              class="relative rounded-full"
              style="width: 96px; height: 96px; padding: 2px; background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1.5px solid var(--neu-border);"
            >
              {#if profile.avatar_url}
                <img
                  src={profile.avatar_url}
                  alt={profile.nickname}
                  class="h-full w-full rounded-full object-cover"
                />
              {:else}
                <div
                  class="flex h-full w-full items-center justify-center rounded-full"
                  style="font-size: 28px; font-weight: 700; color: var(--neu-text);"
                >
                  {profile.nickname.slice(0, 2)}
                </div>
              {/if}
            </div>
            <button
              class="absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.9]"
              style="width: 26px; height: 26px; background: var(--neu-accent-primary); border: 2.5px solid var(--neu-bg); box-shadow: 0 2px 8px var(--neu-glow);"
              title="更换头像"
            >
              <Camera size={12} color="white" strokeWidth={2} />
            </button>
          </div>

          <h2 style="font-size: 22px; color: var(--neu-text);">{profile.nickname}</h2>
          {#if profile.bio && !isEditing}
            <p
              style="font-size: 12px; color: var(--neu-text-muted); text-align: center; padding: 8px 32px; line-height: 1.6;"
            >
              {profile.bio}
            </p>
          {/if}
          {#if profile.location?.location_name}
            <div class="flex items-center gap-1 mb-2">
              <MapPin size={11} strokeWidth={1.5} color="var(--neu-accent-primary)" />
              <span style="font-size: 11px; color: var(--neu-text-muted);"
                >{profile.location.location_name}</span
              >
            </div>
          {/if}
          {#if !isEditing}
            <button
              class="flex items-center gap-2 rounded-full px-10 py-2.5 outline-none transition-transform duration-150 active:scale-[0.96]"
              style="background: var(--neu-bg); color: var(--neu-text-muted); border: 1px solid var(--neu-border); box-shadow: 4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light); font-size: 13px;"
              onclick={() => (isEditing = true)}
            >
              <Pencil size={13} strokeWidth={1.5} />
              编辑资料
            </button>
          {/if}
        </div>

        <!-- Edit form (inline, hidden by default) -->
        {#if isEditing}
          <div
            class="rounded-[22px] p-5 mx-5 mb-4"
            style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
          >
            <div class="space-y-4">
              <div>
                <label
                  for="nickname"
                  style="font-size: 12px; font-weight: 500; color: var(--neu-text-muted); display: block; margin-bottom: 6px;"
                  >昵称 *</label
                >
                <input
                  id="nickname"
                  bind:value={nickname}
                  placeholder="输入你的昵称"
                  class="w-full rounded-[14px] px-4 py-2.5 outline-none"
                  style="background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border); color: var(--neu-text); font-size: 14px;"
                />
              </div>
              <div>
                <label
                  for="bio"
                  style="font-size: 12px; font-weight: 500; color: var(--neu-text-muted); display: block; margin-bottom: 6px;"
                  >个人简介</label
                >
                <textarea
                  id="bio"
                  bind:value={bio}
                  placeholder="介绍一下自己..."
                  rows="3"
                  class="w-full resize-none rounded-[14px] px-4 py-2.5 outline-none"
                  style="background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border); color: var(--neu-text); font-size: 14px;"
                ></textarea>
              </div>
              <div>
                <label
                  for="tags"
                  style="font-size: 12px; font-weight: 500; color: var(--neu-text-muted); display: block; margin-bottom: 6px;"
                  >兴趣标签</label
                >
                <input
                  id="tags"
                  bind:value={tags}
                  placeholder="摄影, 旅行, 美食..."
                  class="w-full rounded-[14px] px-4 py-2.5 outline-none"
                  style="background: var(--neu-bg); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border); color: var(--neu-text); font-size: 14px;"
                />
              </div>
              <div class="flex gap-3 pt-2">
                <button
                  class="flex flex-1 items-center justify-center gap-1.5 rounded-[22px] px-4 py-3.5 outline-none transition-transform duration-150 active:scale-[0.95]"
                  style="background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 14px; box-shadow: 0 2px 8px var(--neu-glow);"
                  onclick={saveProfile}
                  disabled={isSaving}
                >
                  {#if isSaving}
                    <Loader2 class="h-4 w-4 animate-spin" />
                    保存中...
                  {:else}
                    <Check size={15} />
                    保存更改
                  {/if}
                </button>
                <button
                  class="flex flex-1 items-center justify-center gap-1.5 rounded-[22px] px-4 py-3.5 outline-none transition-transform duration-150 active:scale-[0.97]"
                  style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text); font-size: 14px;"
                  onclick={cancelEdit}
                  disabled={isSaving}
                >
                  <X size={15} />
                  取消
                </button>
              </div>
            </div>
          </div>
        {/if}

        <!-- Stats row (single card with dividers) -->
        <div class="mb-5 px-5">
          <div
            class="flex overflow-hidden rounded-[20px]"
            style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
          >
            {#each [{ label: '帖子', value: profile.stats.moments_count }, { label: '关注者', value: profile.stats.followers_count }, { label: '足迹', value: profile.stats.friends_count }] as stat, i}
              <div class="relative flex flex-1 flex-col items-center justify-center py-4">
                {#if i > 0}
                  <div
                    class="absolute left-0 top-1/2 -translate-y-1/2"
                    style="width: 1px; height: 32px; background: var(--neu-border);"
                  ></div>
                {/if}
                <span style="font-size: 24px; color: var(--neu-text);">{stat.value}</span>
                <span
                  style="font-size: 10px; letter-spacing: 0.10em; text-transform: uppercase; color: var(--neu-accent-primary); opacity: 0.8; margin-top: 4px;"
                  >{stat.label}</span
                >
              </div>
            {/each}
          </div>
        </div>

        <!-- Tab bar (Figma Make: 帖子/喜欢/收藏) -->
        <div
          class="flex"
          style="border-bottom: 1px solid var(--neu-border); padding-left: 16px; padding-right: 16px;"
        >
          {#each tabs as tab (tab.id)}
            {@const isActive = activeTab === tab.id}
            <button
              class="relative flex flex-1 items-center justify-center gap-1.5 pb-3 pt-1.5 outline-none transition-transform duration-150 active:scale-[0.94]"
              style="color: {isActive
                ? 'var(--neu-accent-primary)'
                : 'var(--neu-text-muted)'}; font-size: 13px; font-weight: {isActive
                ? 500
                : 400}; transition: color 0.2s;"
              onclick={() => (activeTab = tab.id)}
            >
              <tab.icon size={14} strokeWidth={isActive ? 2 : 1.5} />
              <span>{tab.label}</span>
              {#if isActive}
                <div
                  style="position: absolute; bottom: -1px; left: 18%; right: 18%; height: 2px; border-radius: 1px; background: var(--neu-accent-primary);"
                ></div>
              {/if}
            </button>
          {/each}
        </div>

        <!-- Gallery placeholder (no moments gallery API yet) -->
        <div class="px-4 pt-3">
          {#if profile.stats.moments_count === 0}
            <div class="flex flex-col items-center justify-center py-12">
              <div
                class="flex items-center justify-center rounded-full"
                style="width: 56px; height: 56px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
              >
                <Grid3X3
                  size={22}
                  color="var(--neu-text-muted)"
                  strokeWidth={1.25}
                  style="opacity: 0.5;"
                />
              </div>
              <p style="font-size: 14px; color: var(--neu-text-muted); margin-top: 12px;">
                {activeTab === 'posts'
                  ? '还没有发布帖子'
                  : activeTab === 'liked'
                    ? '还没有喜欢的内容'
                    : '还没有收藏'}
              </p>
            </div>
          {:else}
            <div
              style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; width: 100%;"
            >
              {#each Array(Math.min(profile.stats.moments_count, 6)) as _}
                <div
                  class="relative cursor-pointer overflow-hidden rounded-[18px]"
                  style="aspect-ratio: 1/1; background: var(--neu-fallback-bg); box-shadow: 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
                >
                  <div class="flex h-full w-full items-center justify-center">
                    <Grid3X3
                      size={20}
                      color="var(--neu-text-muted)"
                      strokeWidth={1.25}
                      style="opacity: 0.3;"
                    />
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>
    t{/if}{:else}
  <div class="flex flex-col items-center justify-center px-4 py-16">
    <div
      class="mb-4 flex items-center justify-center rounded-full"
      style="width: 64px; height: 64px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="28"
        height="28"
        viewBox="0 0 24 24"
        fill="none"
        stroke="var(--neu-text-muted)"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
        ><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z" /><path
          d="M12 9v4"
        /><path d="M12 17h.01" /></svg
      >
    </div>
    <p style="font-size: 16px; font-weight: 500; color: var(--neu-text); margin-bottom: 8px;">
      加载个人资料失败
    </p>
    <p style="font-size: 14px; color: var(--neu-text-muted); margin-bottom: 16px;">
      {data.error ?? '无法获取个人资料，请稍后重试'}
    </p>
    <button
      class="rounded-[22px] px-5 py-3.5 outline-none transition-transform duration-150 active:scale-[0.97]"
      style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-accent-primary); font-size: 14px;"
      onclick={() => location.reload()}
    >
      重试
    </button>
  </div>
{/if}
