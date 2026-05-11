<script lang="ts">
  import { onMount } from 'svelte';
  import { usersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import type { MomentItem } from '$lib/types/api';
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

  import Users from 'lucide-svelte/icons/users';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { goto } from '$app/navigation';

  import User from 'lucide-svelte/icons/user';
  import FileText from 'lucide-svelte/icons/file-text';

  const TAG_OPTIONS = [
    '摄影',
    '旅行',
    '音乐',
    '美食',
    '运动',
    '阅读',
    '艺术',
    '科技',
    '游戏',
    '户外',
    '时尚',
    '电影'
  ];
  const MAX_INTEREST_TAGS = 5;

  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  let profile = $state<MyProfile | null>(null);
  let isSaving = $state(false);
  let saved = $state(false);
  let isEditing = $state(false);
  let activeTab = $state<'posts' | 'liked' | 'saved'>('posts');

  let nickname = $state('');
  let bio = $state('');
  let tags = $state('');
  let selectedInterestTags = $state<Set<string>>(new Set());
  let galleryMoments = $state<MomentItem[]>([]);

  function toggleInterestTag(t: string) {
    const next = new Set(selectedInterestTags);
    if (next.has(t)) {
      next.delete(t);
    } else if (next.size < MAX_INTEREST_TAGS) {
      next.add(t);
    }
    selectedInterestTags = next;
  }

  $effect(() => {
    profile = data.profile;
    if (data.profile && !isEditing) {
      nickname = data.profile.nickname ?? '';
      bio = data.profile.bio ?? '';
      tags = data.profile.tags?.join(', ') ?? '';
      selectedInterestTags = new Set(data.profile.tags?.slice(0, MAX_INTEREST_TAGS) ?? []);
    }
  });

  onMount(() => {
    if (profile) {
      authStore.setProfile(profile);
      loadGallery();
    }
  });

  async function loadGallery() {
    if (!profile) return;
    try {
      const resp = await usersService.getMoments({
        user_id: profile.user_id,
        pagination: { limit: 9 }
      });
      if (isSuccess(resp) && resp.data) {
        galleryMoments = resp.data.items ?? [];
      }
    } catch (_) {
      // 静默忽略
    }
  }

  async function saveProfile() {
    const trimmedNickname = nickname.trim();
    const trimmedBio = bio.trim();
    const allTags = [
      ...tags
        .split(',')
        .map((t) => t.trim())
        .filter((t) => t),
      ...Array.from(selectedInterestTags)
    ];
    const uniqueTags = [...new Set(allTags)];

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
      const response = await usersService.updateMyProfile({
        nickname: trimmedNickname,
        bio: trimmedBio || undefined,
        tags: uniqueTags.length > 0 ? uniqueTags : undefined
      });

      if (isSuccess(response)) {
        const profileRes = await usersService.getMyProfile();
        if (isSuccess(profileRes) && profileRes.data) {
          profile = profileRes.data;
          authStore.setProfile(profile);
        }
        saved = true;
        setTimeout(() => {
          saved = false;
          isEditing = false;
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
    <!-- Desktop: two-column layout (260px left + right content) -->
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
            <Share2 size={15} strokeWidth={1.5} color="var(--neu-text-muted)" />
          </button>
          <button
            class="flex items-center justify-center rounded-full outline-none"
            style="width: 36px; height: 36px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
            onclick={() => void goto('/settings')}
          >
            <Settings size={14} strokeWidth={1.5} color="var(--neu-text-muted)" />
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
            <div
              class="pointer-events-none absolute inset-0 rounded-full"
              style="background: var(--neu-glow); filter: blur(20px); transform: scale(1.2); opacity: 0.5;"
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
              class="absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none"
              style="width: 26px; height: 26px; background: var(--neu-accent-primary); border: 2.5px solid var(--neu-bg); box-shadow: 0 2px 8px var(--neu-glow);"
              title="更换头像"
            >
              <Camera size={12} color="white" strokeWidth={2} />
            </button>
          </div>
          <h2 style="font-size: 18px; color: var(--neu-text);">{profile.nickname}</h2>
          {#if profile.bio}
            <p
              style="font-size: 12px; color: var(--neu-text-muted); text-align: center; padding: 6px 8px; line-height: 1.5;"
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
          <!-- Edit button -->
          <button
            class="w-full flex items-center justify-center gap-2 rounded-full py-2.5 mb-5 outline-none"
            style="background: var(--neu-bg); color: var(--neu-text-muted); border: 1px solid var(--neu-border); box-shadow: 4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light); font-size: 13px;"
            onclick={() => (isEditing = !isEditing)}
          >
            <Pencil size={13} strokeWidth={1.5} />
            编辑资料
          </button>
          <!-- Stats card -->
          <div
            class="w-full overflow-hidden rounded-[18px] mb-4"
            style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
          >
            {#each [{ label: '帖子', value: profile.stats.moments_count }, { label: '关注者', value: profile.stats.followers_count }, { label: '足迹', value: profile.stats.friends_count }] as stat}
              <div class="flex items-center justify-between px-4 py-3">
                <span style="font-size: 13px; color: var(--neu-text-muted);">{stat.label}</span>
                <span style="font-size: 17px; color: var(--neu-text);">{stat.value}</span>
              </div>
            {/each}
          </div>
          <!-- Quick links -->
          <div
            class="w-full rounded-[18px]"
            style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
          >
            {#each [{ icon: Grid3X3, label: '我的瞬间', count: profile.stats.moments_count }, { icon: Users, label: '好友', count: profile.stats.friends_count }] as link}
              <a href="'/profile'" class="flex items-center gap-3 px-4 py-2.5 neu-row-hover">
                <link.icon size={14} strokeWidth={1.5} color="var(--neu-text-muted)" />
                <span class="flex-1" style="font-size: 13px; color: var(--neu-text-muted);"
                  >{link.label}</span
                >
                <span style="font-size: 12px; color: var(--neu-text-muted);">{link.count}</span>
              </a>
            {/each}
          </div>
        </div>
        <!-- Right panel: gallery -->
        <div class="flex-1 overflow-y-auto no-scrollbar px-6 py-5">
          <div class="flex items-center justify-between mb-4">
            <h3 style="font-size: 16px; color: var(--neu-text);">最近瞬间</h3>
          </div>
          {#if galleryMoments.length === 0}
            <div class="flex flex-col items-center justify-center py-16">
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
              </div>
              <p style="font-size: 14px; color: var(--neu-text-muted); margin-top: 12px;">
                还没有发布帖子
              </p>
            </div>
          {:else}
            <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px;">
              {#each galleryMoments as moment}
                {@const img = moment.images?.[0]}
                <div
                  class="relative cursor-pointer overflow-hidden rounded-[16px]"
                  style="aspect-ratio: 1/1; box-shadow: 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
                >
                  {#if img}
                    <img
                      src={img.thumbnail_url || img.url}
                      alt={moment.description}
                      class="h-full w-full object-cover"
                    />
                    <div
                      class="absolute bottom-0 left-0 right-0 flex items-center gap-2 px-2 py-1.5"
                      style="background: linear-gradient(transparent, rgba(0,0,0,0.5));"
                    >
                      <span style="font-size: 10px; color: white;">
                        <Heart
                          size={10}
                          color="white"
                          style="display: inline; vertical-align: middle;"
                        />
                        {moment.likes_count}
                      </span>
                    </div>
                  {:else}
                    <div
                      class="flex h-full w-full items-center justify-center"
                      style="background: var(--neu-fallback-bg);"
                    >
                      <Grid3X3
                        size={20}
                        color="var(--neu-text-muted)"
                        strokeWidth={1.25}
                        style="opacity: 0.3;"
                      />
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>
  {:else}
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
            class="flex flex-col px-5 pt-4 pb-8 gap-5"
            style="max-width: 560px; margin-left: auto; margin-right: auto; width: 100%;"
          >
            <!-- Banner image -->
            <div
              class="relative rounded-[24px] overflow-hidden"
              style="height: clamp(120px, 18vw, 160px);"
            >
              <div
                class="absolute inset-0"
                style="background: linear-gradient(135deg, var(--neu-accent-primary), var(--neu-glow)); opacity: 0.3;"
              ></div>
              <div
                class="absolute inset-0"
                style="background: linear-gradient(to bottom, transparent 40%, rgba(0,0,0,0.5));"
              ></div>
              <button
                class="absolute top-3 right-3 flex items-center gap-1.5 rounded-full outline-none"
                style="padding: 6px 12px; background: rgba(0,0,0,0.45); backdrop-filter: blur(10px); border: 1px solid rgba(255,255,255,0.25); font-size: 12px; color: white;"
              >
                <Camera size={13} strokeWidth={1.75} />
                更换横幅
              </button>
            </div>

            <!-- Avatar edit -->
            <div class="flex items-center gap-4 -mt-8 ml-2">
              <div class="relative flex-shrink-0">
                <div
                  class="rounded-[22px] overflow-hidden"
                  style="width: 72px; height: 72px; border: 3px solid var(--neu-bg); box-shadow: 0 6px 20px rgba(0,0,0,0.22);"
                >
                  {#if profile.avatar_url}
                    <img src={profile.avatar_url} alt="avatar" class="w-full h-full object-cover" />
                  {:else}
                    <div
                      class="flex h-full w-full items-center justify-center"
                      style="background: var(--neu-bg); font-size: 20px; color: var(--neu-text);"
                    >
                      {nickname.slice(0, 2)}
                    </div>
                  {/if}
                </div>
                <button
                  class="absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none"
                  style="width: 26px; height: 26px; background: var(--neu-accent-primary); border: 2px solid var(--neu-bg); box-shadow: 0 2px 8px var(--neu-glow);"
                  title="更换头像"
                >
                  <Camera size={12} color="white" strokeWidth={2} />
                </button>
              </div>
              <div>
                <p style="font-size: 16px; color: var(--neu-text);">{profile.nickname}</p>
                <p style="font-size: 12px; color: var(--neu-text-muted);">
                  @{profile.nickname.toLowerCase().replace(/\s+/g, '.')}
                </p>
              </div>
            </div>

            <!-- Name field -->
            <div class="flex flex-col gap-1.5">
              <div class="flex items-center gap-1.5">
                <User
                  style="width: 13px; height: 13px; color: var(--neu-accent-primary);"
                  strokeWidth={1.75}
                />
                <label
                  style="font-size: 11px; color: var(--neu-text-muted); text-transform: uppercase; letter-spacing: 0.1em;"
                  >姓名</label
                >
              </div>
              <div
                class="px-4 py-3 rounded-[18px]"
                style="box-shadow: 4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light); border: 1px solid var(--neu-border); background: var(--neu-bg);"
              >
                <input
                  id="nickname"
                  bind:value={nickname}
                  placeholder="你的名字"
                  class="w-full bg-transparent outline-none"
                  style="color: var(--neu-text); font-size: 14px;"
                />
              </div>
            </div>

            <!-- Bio field -->
            <div class="flex flex-col gap-1.5">
              <div class="flex items-center gap-1.5">
                <FileText
                  style="width: 13px; height: 13px; color: var(--neu-accent-primary);"
                  strokeWidth={1.75}
                />
                <label
                  style="font-size: 11px; color: var(--neu-text-muted); text-transform: uppercase; letter-spacing: 0.1em;"
                  >个人简介</label
                >
              </div>
              <div
                class="px-4 py-3 rounded-[18px]"
                style="box-shadow: 4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light); border: 1px solid var(--neu-border); background: var(--neu-bg);"
              >
                <textarea
                  id="bio"
                  bind:value={bio}
                  placeholder="介绍一下自己..."
                  rows="3"
                  class="w-full bg-transparent outline-none resize-none"
                  style="color: var(--neu-text); font-size: 14px; line-height: 1.65;"
                ></textarea>
                <div class="flex justify-end mt-1">
                  <span
                    style="font-size: 11px; color: {bio.length > 150 * 0.85
                      ? 'var(--neu-accent-primary)'
                      : 'var(--neu-text-muted)'};"
                  >
                    {bio.length}/150
                  </span>
                </div>
              </div>
            </div>

            <!-- Location field -->
            {#if profile.location?.location_name}
              <div class="flex flex-col gap-1.5">
                <div class="flex items-center gap-1.5">
                  <MapPin
                    style="width: 13px; height: 13px; color: var(--neu-accent-primary);"
                    strokeWidth={1.75}
                  />
                  <label
                    style="font-size: 11px; color: var(--neu-text-muted); text-transform: uppercase; letter-spacing: 0.1em;"
                    >所在地</label
                  >
                </div>
                <div
                  class="px-4 py-3 rounded-[18px]"
                  style="box-shadow: 4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light); border: 1px solid var(--neu-border); background: var(--neu-bg);"
                >
                  <input
                    value={profile.location.location_name}
                    disabled
                    class="w-full bg-transparent outline-none"
                    style="color: var(--neu-text-muted); font-size: 14px; opacity: 0.7;"
                  />
                </div>
              </div>
            {/if}

            <!-- Interest tags -->
            <div class="flex flex-col gap-2.5">
              <div class="flex items-center gap-1.5">
                <label
                  style="font-size: 11px; color: var(--neu-text-muted); text-transform: uppercase; letter-spacing: 0.1em;"
                  >兴趣标签</label
                >
                <span style="font-size: 11px; color: var(--neu-text-muted); opacity: 0.6;"
                  >({selectedInterestTags.size}/{MAX_INTEREST_TAGS})</span
                >
              </div>
              <div class="flex flex-wrap gap-2">
                {#each TAG_OPTIONS as t (t)}
                  {@const active = selectedInterestTags.has(t)}
                  {@const disabled = !active && selectedInterestTags.size >= MAX_INTEREST_TAGS}
                  <button
                    class="px-3.5 py-1.5 rounded-full outline-none transition-all duration-200 active:scale-[0.92]"
                    style={active
                      ? 'background: var(--neu-accent-primary); color: white; font-size: 12px; box-shadow: 0 3px 10px var(--neu-glow); border: none;'
                      : 'background: var(--neu-bg); color: var(--neu-text-muted); font-size: 12px; border: 1px solid var(--neu-border); opacity: {disabled ? 0.4 : 1}; box-shadow: 3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light);'}
                    onclick={() => !disabled && toggleInterestTag(t)}
                    {disabled}
                  >
                    #{t}
                  </button>
                {/each}
              </div>
            </div>

            <!-- Action buttons -->
            <div class="flex gap-3 pt-2">
              <button
                class="flex flex-1 items-center justify-center gap-1.5 rounded-[22px] px-4 py-3.5 outline-none transition-all duration-300 active:scale-[0.95]"
                style={saved
                  ? 'background: #22c55e; color: white; font-size: 14px; box-shadow: 0 4px 14px rgba(34,197,94,0.35); border: none;'
                  : 'background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 14px; box-shadow: 0 4px 14px var(--neu-glow); border: none;'}
                onclick={saveProfile}
                disabled={isSaving}
              >
                {#if saved}
                  <Check size={14} color="white" strokeWidth={2.5} />
                {:else if isSaving}
                  <Loader2 class="h-4 w-4 animate-spin" />
                  保存中...
                {:else}
                  保存
                {/if}
              </button>
              <button
                class="flex flex-1 items-center justify-center gap-1.5 rounded-[22px] px-4 py-3.5 outline-none transition-transform duration-150 active:scale-[0.97]"
                style="background: var(--neu-bg); box-shadow: 4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text); font-size: 14px;"
                onclick={cancelEdit}
                disabled={isSaving}
              >
                取消
              </button>
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

        <!-- Gallery -->
        <div class="px-4 pt-3">
          {#if galleryMoments.length === 0}
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
                还没有发布帖子
              </p>
            </div>
          {:else}
            <div
              style="display: grid; grid-template-columns: repeat({platformStore.isTablet
                ? 3
                : 2}, 1fr); gap: 10px; width: 100%;"
            >
              {#each galleryMoments as moment}
                {@const img = moment.images?.[0]}
                <div
                  class="relative cursor-pointer overflow-hidden rounded-[18px]"
                  style="aspect-ratio: 1/1; box-shadow: 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
                >
                  {#if img}
                    <img
                      src={img.thumbnail_url || img.url}
                      alt={moment.description}
                      class="h-full w-full object-cover"
                    />
                    <div
                      class="absolute bottom-0 left-0 right-0 flex items-center gap-2 px-2 py-1.5"
                      style="background: linear-gradient(transparent, rgba(0,0,0,0.5));"
                    >
                      <span style="font-size: 10px; color: white;">
                        <Heart
                          size={10}
                          color="white"
                          style="display: inline; vertical-align: middle;"
                        />
                        {moment.likes_count}
                      </span>
                    </div>
                  {:else}
                    <div
                      class="flex h-full w-full items-center justify-center"
                      style="background: var(--neu-fallback-bg);"
                    >
                      <Grid3X3
                        size={20}
                        color="var(--neu-text-muted)"
                        strokeWidth={1.25}
                        style="opacity: 0.3;"
                      />
                    </div>
                  {/if}
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
