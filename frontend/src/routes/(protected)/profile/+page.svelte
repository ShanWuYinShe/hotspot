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
  import UsersRound from 'lucide-svelte/icons/users-round';
  import Bell from 'lucide-svelte/icons/bell';
  import Shield from 'lucide-svelte/icons/shield';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Image from 'lucide-svelte/icons/image';
  import User from 'lucide-svelte/icons/user';
  import FileText from 'lucide-svelte/icons/file-text';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { adminPermissionsStore } from '$lib/stores/admin-permissions.svelte';
  import { goto } from '$app/navigation';
  import Avatar from '$lib/components/common/avatar.svelte';

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
    const uniqueTags = [...new Set(Array.from(selectedInterestTags))];

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
    }
    isEditing = false;
  }

  const tabs = [
    { id: 'posts' as const, icon: Grid3X3, label: '帖子' },
    { id: 'liked' as const, icon: Heart, label: '喜欢' },
    { id: 'saved' as const, icon: Bookmark, label: '收藏' }
  ];

  const statsRow = $derived([
    { label: '帖子', value: profile?.stats.moments_count ?? 0 },
    { label: '关注者', value: profile?.stats.followers_count ?? 0 },
    { label: '足迹', value: profile?.stats.friends_count ?? 0 }
  ]);

  const quickLinks = $derived([
    {
      icon: Image,
      label: '我的瞬间',
      href: '/moments',
      count: profile?.stats.moments_count ?? 0
    },
    {
      icon: UsersRound,
      label: '我的群组',
      href: '/groups',
      count: null
    },
    {
      icon: Heart,
      label: '关系',
      href: '/relations?tab=friends',
      count: profile?.stats.friends_count ?? 0
    },
    {
      icon: Bell,
      label: '通知',
      href: '/notifications',
      count: null
    },
    {
      icon: Settings,
      label: '设置',
      href: '/settings',
      count: null
    },
    ...(adminPermissionsStore.isAdmin
      ? [{ icon: Shield, label: '管理后台', href: '/admin', count: null }]
      : [])
  ]);
</script>

<svelte:head>
  <title>个人资料 - HotSpot</title>
</svelte:head>

{#if profile}
  <!-- ═══ Shared avatar block ═══ -->
  {#snippet avatarBlock(p: MyProfile)}
    <div class="relative mb-4">
      <div
        class="pointer-events-none absolute inset-0 rounded-full"
        style="background: var(--glow); filter: blur(20px); transform: scale(1.2); opacity: 0.5;"
      ></div>
      <div
        class="surface-hero relative rounded-full"
        style="width: 96px; height: 96px; padding: 2px;"
      >
        <Avatar src={p.avatar_url} name={p.nickname} alt={p.nickname} size={92} border={false} />
      </div>
      <button
        class="absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none"
        style="width: 26px; height: 26px; background: var(--brand); border: 2.5px solid var(--surface-bg); box-shadow: 0 2px 8px var(--glow);"
        title="更换头像"
      >
        <Camera size={12} color="var(--text-on-accent)" strokeWidth={2} />
      </button>
    </div>
  {/snippet}

  <!-- ═══ Shared tab bar ═══ -->
  {#snippet tabBar()}
    <div class="flex" style="border-bottom: 1px solid var(--border-theme);">
      {#each tabs as tab (tab.id)}
        {@const isActive = activeTab === tab.id}
        <button
          class="relative flex flex-1 items-center justify-center gap-1.5 pb-3 pt-1.5 outline-none"
          style="color: {isActive
            ? 'var(--brand)'
            : 'var(--text-secondary)'}; font-size: 13px; font-weight: {isActive ? 500 : 400};"
          onclick={() => (activeTab = tab.id)}
        >
          <tab.icon size={14} strokeWidth={isActive ? 2 : 1.5} />
          <span>{tab.label}</span>
          {#if isActive}
            <div
              style="position: absolute; bottom: -1px; left: 18%; right: 18%; height: 2px; border-radius: 1px; background: var(--brand);"
            ></div>
          {/if}
        </button>
      {/each}
    </div>
  {/snippet}

  <!-- ═══ Shared personal hub (replaces the removed nav drawer on mobile/tablet) ═══ -->
  {#snippet hubCard()}
    <div class="w-full rounded-[18px] overflow-hidden surface-card">
      {#each quickLinks as link (link.href)}
        <a
          href={link.href}
          class="flex items-center gap-3 px-4 py-3 row-hover"
          style="border-bottom: 1px solid var(--border-theme);"
        >
          <link.icon size={16} strokeWidth={1.5} color="var(--text-secondary)" />
          <span class="flex-1" style="font-size: 13px; color: var(--text-primary);"
            >{link.label}</span
          >
          {#if link.count != null}
            <span style="font-size: 12px; color: var(--text-secondary);">{link.count}</span>
          {/if}
          <ChevronRight size={15} strokeWidth={1.5} color="var(--text-secondary)" />
        </a>
      {/each}
    </div>
  {/snippet}

  <!-- ═══ Shared gallery grid ═══ -->
  {#snippet galleryGrid()}
    {#if galleryMoments.length === 0}
      <div class="flex flex-col items-center justify-center py-12">
        <div
          class="flex items-center justify-center rounded-full"
          style="width: 56px; height: 56px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
        >
          <Grid3X3
            size={22}
            color="var(--text-secondary)"
            strokeWidth={1.25}
            style="opacity: 0.5;"
          />
        </div>
        <p style="font-size: 14px; color: var(--text-secondary); margin-top: 12px;">
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
            class="surface-flat relative cursor-pointer overflow-hidden rounded-[16px]"
            style="aspect-ratio: 1/1;"
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
                <span style="font-size: 10px; color: var(--text-on-overlay);">
                  <Heart
                    size={10}
                    color="var(--text-on-overlay)"
                    style="display: inline; vertical-align: middle;"
                  />
                  {moment.likes_count}
                </span>
              </div>
            {:else}
              <div
                class="flex h-full w-full items-center justify-center"
                style="background: var(--surface-secondary);"
              >
                <Grid3X3
                  size={20}
                  color="var(--text-secondary)"
                  strokeWidth={1.25}
                  style="opacity: 0.3;"
                />
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  {/snippet}

  <!-- ═══ Shared stats row ═══ -->
  {#snippet statsRowDisplay()}
    <div
      class="flex overflow-hidden rounded-[20px] surface-flat"
      style="border: 1px solid var(--border-theme);"
    >
      {#each statsRow as stat, i}
        <div class="relative flex flex-1 flex-col items-center justify-center py-4">
          {#if i > 0}
            <div
              class="absolute left-0 top-1/2 -translate-y-1/2"
              style="width: 1px; height: 32px; background: var(--border-theme);"
            ></div>
          {/if}
          <span style="font-size: 24px; font-weight: 600; color: var(--text-primary);"
            >{stat.value}</span
          >
          <span class="eyebrow" style="margin-top: 4px;">{stat.label}</span>
        </div>
      {/each}
    </div>
  {/snippet}

  <!-- ═══ Shared edit form ═══ -->
  {#snippet editForm(p: MyProfile)}
    <div class="flex flex-col gap-5 pt-4 pb-8">
      <!-- Banner -->
      <div
        class="relative overflow-hidden rounded-[24px]"
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
          style="padding: 6px 12px; background: rgba(0,0,0,0.45); backdrop-filter: blur(10px); border: 1px solid rgba(255,255,255,0.25); font-size: 12px; color: var(--text-on-overlay);"
        >
          <Camera size={13} strokeWidth={1.75} />
          更换横幅
        </button>
      </div>

      <!-- Avatar edit -->
      <div class="flex items-center gap-4 -mt-8 ml-2">
        <div class="relative flex-shrink-0">
          <div
            class="overflow-hidden rounded-[22px]"
            style="width: 72px; height: 72px; border: 3px solid var(--surface-bg); box-shadow: var(--shadow-lg);"
          >
            <Avatar
              src={p.avatar_url}
              name={nickname}
              alt="avatar"
              size={72}
              shape="rounded"
              border={false}
            />
          </div>
          <button
            class="absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none"
            style="width: 26px; height: 26px; background: var(--brand); border: 2px solid var(--surface-bg); box-shadow: 0 2px 8px var(--glow);"
            title="更换头像"
          >
            <Camera size={12} color="var(--text-on-accent)" strokeWidth={2} />
          </button>
        </div>
        <div>
          <p style="font-size: 16px; color: var(--text-primary);">{p.nickname}</p>
          <p style="font-size: 12px; color: var(--text-secondary);">
            @{p.nickname.toLowerCase().replace(/\s+/g, '.')}
          </p>
        </div>
      </div>

      <!-- Name field -->
      <div class="flex flex-col gap-1.5">
        <div class="flex items-center gap-1.5">
          <User style="width: 13px; height: 13px; color: var(--brand);" strokeWidth={1.75} />
          <span class="eyebrow">姓名</span>
        </div>
        <div class="input-surface rounded-[14px] px-4 py-3">
          <input
            id="nickname"
            bind:value={nickname}
            placeholder="你的名字"
            class="w-full bg-transparent outline-none"
            style="color: var(--text-primary); font-size: 14px;"
          />
        </div>
      </div>

      <!-- Bio field -->
      <div class="flex flex-col gap-1.5">
        <div class="flex items-center gap-1.5">
          <FileText style="width: 13px; height: 13px; color: var(--brand);" strokeWidth={1.75} />
          <span class="eyebrow">个人简介</span>
        </div>
        <div class="input-surface rounded-[14px] px-4 py-3">
          <textarea
            id="bio"
            bind:value={bio}
            placeholder="介绍一下自己..."
            rows="3"
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

      <!-- Location field -->
      {#if p.location?.location_name}
        <div class="flex flex-col gap-1.5">
          <div class="flex items-center gap-1.5">
            <MapPin style="width: 13px; height: 13px; color: var(--brand);" strokeWidth={1.75} />
            <span class="eyebrow">所在地</span>
          </div>
          <div class="input-surface rounded-[14px] px-4 py-3">
            <input
              value={p.location.location_name}
              disabled
              class="w-full bg-transparent outline-none"
              style="color: var(--text-secondary); font-size: 14px; opacity: 0.7;"
            />
          </div>
        </div>
      {/if}

      <!-- Interest tags -->
      <div class="flex flex-col gap-2.5">
        <div class="flex items-center gap-1.5">
          <span class="eyebrow">兴趣标签</span>
          <span style="font-size: 11px; color: var(--text-secondary); opacity: 0.6;"
            >({selectedInterestTags.size}/{MAX_INTEREST_TAGS})</span
          >
        </div>
        <div class="flex flex-wrap gap-2">
          {#each TAG_OPTIONS as t (t)}
            {@const active = selectedInterestTags.has(t)}
            {@const disabled = !active && selectedInterestTags.size >= MAX_INTEREST_TAGS}
            <button
              class="filter-chip disabled:opacity-40"
              class:filter-chip--active={active}
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
          class="flex flex-1 items-center justify-center gap-1.5 rounded-[22px] px-4 py-3.5 outline-none transition-all"
          style={saved
            ? 'background: var(--theme-success); color: var(--text-on-accent); font-size: 14px; box-shadow: 0 4px 14px color-mix(in srgb, var(--theme-success) 35%, transparent); border: none;'
            : 'background: var(--brand); color: var(--text-on-accent); font-size: 14px; box-shadow: 0 4px 14px var(--glow); border: none;'}
          onclick={saveProfile}
          disabled={isSaving}
        >
          {#if saved}
            <Check size={14} color="var(--text-on-accent)" strokeWidth={2.5} />
          {:else if isSaving}
            <Loader2 class="h-4 w-4 animate-spin" />
            保存中...
          {:else}
            保存
          {/if}
        </button>
        <button
          class="flex flex-1 items-center justify-center gap-1.5 rounded-[22px] px-4 py-3.5 outline-none surface-flat"
          style="color: var(--text-primary); font-size: 14px;"
          onclick={cancelEdit}
          disabled={isSaving}
        >
          取消
        </button>
      </div>
    </div>
  {/snippet}

  <!-- ═══════════════════════════════════════════════
       WIDE (landscape/large tablet + desktop)
  ══════════════════════════════════════════════════ -->
  {#if platformStore.isWide}
    <div class="flex flex-col h-full">
      <!-- Sticky header -->
      <div
        class="flex-shrink-0 flex items-center justify-between px-6 py-4"
        style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
      >
        <div>
          <p style="font-size: 20px; color: var(--text-primary);">{profile.nickname}</p>
          <p style="font-size: 12px; color: var(--text-secondary); margin-top: 1px;">
            @{profile.nickname.toLowerCase().replace(/\s+/g, '.')}
          </p>
        </div>
        <div class="flex gap-2.5">
          <button
            class="flex items-center justify-center rounded-full outline-none surface-flat"
            style="width: 36px; height: 36px;"
          >
            <Share2 size={15} strokeWidth={1.5} color="var(--text-secondary)" />
          </button>
          <button
            class="flex items-center justify-center rounded-full outline-none surface-flat"
            style="width: 36px; height: 36px;"
            onclick={() => void goto('/settings')}
          >
            <Settings size={14} strokeWidth={1.5} color="var(--text-secondary)" />
          </button>
        </div>
      </div>

      <!-- Two-column body -->
      <div class="flex flex-1 min-h-0 overflow-hidden">
        <!-- Left panel (260px) -->
        <div
          class="flex-shrink-0 flex flex-col items-center overflow-y-auto no-scrollbar"
          style="width: 260px; border-right: 1px solid var(--border-theme); padding: 24px 20px;"
        >
          {@render avatarBlock(profile)}
          <h2 style="font-size: 18px; color: var(--text-primary);">{profile.nickname}</h2>
          {#if profile.bio}
            <p
              style="font-size: 12px; color: var(--text-secondary); text-align: center; padding: 6px 8px; line-height: 1.5;"
            >
              {profile.bio}
            </p>
          {/if}
          {#if profile.location?.location_name}
            <div class="flex items-center gap-1 mb-2">
              <MapPin size={11} strokeWidth={1.5} color="var(--brand)" />
              <span style="font-size: 11px; color: var(--text-secondary);"
                >{profile.location.location_name}</span
              >
            </div>
          {/if}

          <button
            class="w-full flex items-center justify-center gap-2 rounded-full py-2.5 mb-5 outline-none surface-flat"
            style="font-size: 13px;"
            onclick={() => (isEditing = !isEditing)}
          >
            <Pencil size={13} strokeWidth={1.5} color="var(--text-secondary)" />
            <span style="color: var(--text-secondary);">编辑资料</span>
          </button>

          <!-- Analytics card -->
          <div
            class="w-full overflow-hidden rounded-[18px] mb-4 surface-flat"
            style="border: 1px solid var(--border-theme);"
          >
            {#each statsRow as stat}
              <div
                class="flex items-center justify-between px-4 py-3"
                style="border-bottom: 1px solid var(--border-theme);"
              >
                <span style="font-size: 13px; color: var(--text-secondary);">{stat.label}</span>
                <span style="font-size: 17px; font-weight: 600; color: var(--text-primary);"
                  >{stat.value}</span
                >
              </div>
            {/each}
          </div>

          <!-- Personal hub -->
          {@render hubCard()}
        </div>

        <!-- Right panel: tab bar + gallery -->
        <div class="flex-1 overflow-y-auto no-scrollbar page-pad">
          <div class="page-shell page-shell--form">
            <div class="pt-2">
              {@render tabBar()}
            </div>
            <div class="py-4">
              {@render galleryGrid()}
            </div>
          </div>
        </div>
      </div>
    </div>
  {:else}
    <!-- ═══════════════════════════════════════════════
         NARROW (mobile + portrait/small tablet)
    ══════════════════════════════════════════════════ -->
    <div class="flex flex-col">
      <!-- Sticky header -->
      <div
        class="sticky top-0 z-20 flex items-center justify-between px-5 py-3"
        style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
      >
        <div>
          <p style="font-size: 18px; color: var(--text-primary);">{profile.nickname}</p>
          <p style="font-size: 11px; color: var(--text-secondary); margin-top: 1px;">
            @{profile.nickname.toLowerCase().replace(/\s+/g, '.')}
          </p>
        </div>
        <div class="flex gap-2.5">
          <button
            class="flex items-center justify-center rounded-full outline-none surface-flat"
            style="width: 38px; height: 38px;"
          >
            <Share2 size={15} strokeWidth={1.5} color="var(--text-secondary)" />
          </button>
          <button
            class="flex items-center justify-center rounded-full outline-none surface-flat"
            style="width: 38px; height: 38px;"
            onclick={() => void goto('/settings')}
          >
            <Settings size={15} strokeWidth={1.5} color="var(--text-secondary)" />
          </button>
        </div>
      </div>

      <div class="flex flex-col page-pad">
        <div class="page-shell page-shell--form">
          <!-- Avatar section -->
          <div class="flex flex-col items-center px-5 pt-5 pb-4">
            {@render avatarBlock(profile)}
            <h2 style="font-size: 22px; color: var(--text-primary);">{profile.nickname}</h2>
            {#if profile.bio && !isEditing}
              <p
                style="font-size: 12px; color: var(--text-secondary); text-align: center; padding: 8px 32px; line-height: 1.6;"
              >
                {profile.bio}
              </p>
            {/if}
            {#if profile.location?.location_name}
              <div class="flex items-center gap-1 mb-2">
                <MapPin size={11} strokeWidth={1.5} color="var(--brand)" />
                <span style="font-size: 11px; color: var(--text-secondary);"
                  >{profile.location.location_name}</span
                >
              </div>
            {/if}
            {#if !isEditing}
              <button
                class="flex items-center gap-2 rounded-full px-10 py-2.5 outline-none surface-flat"
                style="font-size: 13px;"
                onclick={() => (isEditing = true)}
              >
                <Pencil size={13} strokeWidth={1.5} color="var(--text-secondary)" />
                <span style="color: var(--text-secondary);">编辑资料</span>
              </button>
            {/if}
          </div>

          <!-- Edit form -->
          {#if isEditing}
            {@render editForm(profile)}
          {/if}

          <!-- Stats row -->
          <div class="mb-5 px-5">
            {@render statsRowDisplay()}
          </div>

          <!-- Personal hub -->
          {#if !isEditing}
            <div class="mb-5 px-5">
              {@render hubCard()}
            </div>
          {/if}

          <!-- Tab bar -->
          {@render tabBar()}

          <!-- Gallery -->
          <div class="pt-3">
            {@render galleryGrid()}
          </div>
        </div>
      </div>
    </div>
  {/if}
{:else}
  <!-- Error state -->
  <div class="page-pad">
    <div class="page-shell page-shell--form flex flex-col items-center justify-center py-16">
      <div
        class="mb-4 flex items-center justify-center rounded-full surface-flat"
        style="width: 64px; height: 64px;"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="28"
          height="28"
          viewBox="0 0 24 24"
          fill="none"
          stroke="var(--text-secondary)"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z" />
          <path d="M12 9v4" />
          <path d="M12 17h.01" />
        </svg>
      </div>
      <p style="font-size: 16px; font-weight: 500; color: var(--text-primary); margin-bottom: 8px;">
        加载个人资料失败
      </p>
      <p style="font-size: 14px; color: var(--text-secondary); margin-bottom: 16px;">
        {data.error ?? '无法获取个人资料，请稍后重试'}
      </p>
      <button
        class="rounded-[22px] px-5 py-3.5 outline-none surface-flat"
        style="color: var(--brand); font-size: 14px;"
        onclick={() => location.reload()}>重试</button
      >
    </div>
  </div>
{/if}
