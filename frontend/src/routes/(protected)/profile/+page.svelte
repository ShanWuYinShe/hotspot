<script lang="ts">
  import { onMount } from 'svelte';
  import { usersService, momentsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { formatHandle } from '$lib/utils/format';
  import type { MomentItem } from '$lib/types/api';
  import { authStore } from '$lib/stores/auth.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import type { MyProfile } from '$lib/types/api';
  import Camera from 'lucide-svelte/icons/camera';
  import TriangleAlert from 'lucide-svelte/icons/triangle-alert';
  import Spinner from '$lib/components/common/spinner.svelte';
  import { Textarea } from '$lib/components/ui/textarea';
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
  import { t } from '$lib/i18n';
  import { goto } from '$app/navigation';
  import Avatar from '$lib/components/common/avatar.svelte';
  import SmartImage from '$lib/components/common/smart-image.svelte';
  import { mediaService } from '$lib/api/services';

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
  let likedMoments = $state<MomentItem[]>([]);
  let savedMoments = $state<MomentItem[]>([]);
  let loadingLiked = $state(false);
  let loadingSaved = $state(false);
  let bannerUploading = $state(false);
  let bannerInputElement: HTMLInputElement | undefined = $state();
  let avatarInputElement: HTMLInputElement | undefined = $state();

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

  async function loadLikedMoments() {
    if (!profile) return;
    loadingLiked = true;
    try {
      const resp = await usersService.getLikedMoments({
        user_id: profile.user_id,
        pagination: { limit: 9 }
      });
      if (isSuccess(resp) && resp.data) {
        likedMoments = resp.data.items ?? [];
      }
    } catch (_) {
      // 静默忽略
    } finally {
      loadingLiked = false;
    }
  }

  async function loadSavedMoments() {
    loadingSaved = true;
    try {
      const resp = await momentsService.getMyFavorites({
        pagination: { limit: 9 }
      });
      if (isSuccess(resp) && resp.data) {
        savedMoments = resp.data.items ?? [];
      }
    } catch (_) {
      // 静默忽略
    } finally {
      loadingSaved = false;
    }
  }

  function handleTabChange(tab: 'posts' | 'liked' | 'saved') {
    activeTab = tab;
    if (tab === 'liked' && likedMoments.length === 0) {
      loadLikedMoments();
    } else if (tab === 'saved' && savedMoments.length === 0) {
      loadSavedMoments();
    }
  }

  async function saveProfile() {
    const trimmedNickname = nickname.trim();
    const trimmedBio = bio.trim();
    const uniqueTags = [...new Set(Array.from(selectedInterestTags))];

    if (!trimmedNickname) {
      toast.error(t('profile.nicknameEmpty'));
      return;
    }
    if (trimmedNickname.length > 30) {
      toast.error(t('profile.nicknameLengthError'));
      return;
    }
    if (trimmedBio.length > 500) {
      toast.error(t('profile.bioTooLong'));
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
        toast.error(response.message || t('profile.saveFailed'));
      }
    } catch {
      toast.error(t('profile.saveFailed'));
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

  async function handleBannerUpload(file: File) {
    if (file.size > 10 * 1024 * 1024) {
      toast.error(t('profile.bannerTooLarge'));
      return;
    }
    bannerUploading = true;
    try {
      await mediaService.uploadFile(file, 'avatar');
      toast.success(t('profile.bannerUpdated'));
      const profileRes = await usersService.getMyProfile();
      if (isSuccess(profileRes) && profileRes.data) {
        profile = profileRes.data;
        authStore.setProfile(profile);
      }
    } catch {
      toast.error(t('profile.bannerUploadFailed'));
    } finally {
      bannerUploading = false;
    }
  }

  async function handleAvatarUpload(file: File) {
    if (file.size > 5 * 1024 * 1024) {
      toast.error(t('profile.avatarTooLarge'));
      return;
    }
    try {
      const result = await mediaService.uploadFile(file, 'avatar');
      await usersService.updateMyProfile({ avatar_file_id: result.file_id });
      const profileRes = await usersService.getMyProfile();
      if (isSuccess(profileRes) && profileRes.data) {
        profile = profileRes.data;
        authStore.setProfile(profile);
      }
      toast.success(t('profile.avatarUpdated'));
    } catch {
      toast.error(t('profile.avatarUploadFailed'));
    }
  }

  const tabs = [
    { id: 'posts' as const, icon: Grid3X3, label: t('profile.tabPosts') },
    { id: 'liked' as const, icon: Heart, label: t('profile.tabLiked') },
    { id: 'saved' as const, icon: Bookmark, label: t('profile.tabSaved') }
  ];

  const statsRow = $derived([
    { label: t('profile.statPosts'), value: profile?.stats.moments_count ?? 0 },
    { label: t('profile.statFollowers'), value: profile?.stats.followers_count ?? 0 },
    { label: t('profile.statFootprint'), value: profile?.stats.friends_count ?? 0 }
  ]);

  const quickLinks = $derived([
    {
      icon: Image,
      label: t('profile.linkMoments'),
      href: '/moments',
      count: profile?.stats.moments_count ?? 0
    },
    {
      icon: UsersRound,
      label: t('profile.linkGroups'),
      href: '/groups',
      count: null
    },
    {
      icon: Heart,
      label: t('profile.linkRelations'),
      href: '/relations?tab=friends',
      count: profile?.stats.friends_count ?? 0
    },
    {
      icon: Bell,
      label: t('profile.linkNotifications'),
      href: '/notifications',
      count: null
    },
    {
      icon: Settings,
      label: t('profile.linkSettings'),
      href: '/settings',
      count: null
    },
    ...(adminPermissionsStore.isAdmin
      ? [{ icon: Shield, label: t('profile.linkAdmin'), href: '/admin', count: null }]
      : [])
  ]);
</script>

<svelte:head>
  <title>{t('profile.title')} - HotSpot</title>
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
        <Avatar src={p.avatar_url} name={p.nickname} alt={p.nickname} size={96} border={false} />
      </div>
      <button
        class="absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none"
        style="width: 26px; height: 26px; background: var(--brand); border: 2.5px solid var(--surface-bg); box-shadow: var(--shadow-brand-btn);"
        title={t('profile.changeAvatar')}
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
          class="text-body relative flex flex-1 items-center justify-center gap-1.5 pb-3 pt-1.5 outline-none"
          style="color: {isActive
            ? 'var(--brand)'
            : 'var(--text-secondary)'}; font-weight: {isActive ? 500 : 400}"
          onclick={() => handleTabChange(tab.id)}
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
    <div class="w-full rounded-[var(--radius-md)] overflow-hidden surface-card">
      {#each quickLinks as link (link.href)}
        <a
          href={link.href}
          class="list-row flex items-center gap-3 px-4 py-3"
          style="border-bottom: 1px solid var(--border-theme);"
        >
          <link.icon size={16} strokeWidth={1.5} color="var(--text-secondary)" />
          <span class="text-body flex-1" style="color: var(--text-primary)">{link.label}</span>
          {#if link.count != null}
            <span class="text-meta" style="color: var(--text-secondary)">{link.count}</span>
          {/if}
          <ChevronRight size={15} strokeWidth={1.5} color="var(--text-secondary)" />
        </a>
      {/each}
    </div>
  {/snippet}

  <!-- ═══ Shared gallery grid ═══ -->
  {#snippet galleryGrid()}
    {#if activeTab === 'liked' && loadingLiked}
      <div class="flex items-center justify-center py-12">
        <Spinner size="lg" color="var(--text-secondary)" />
      </div>
    {:else if activeTab === 'saved' && loadingSaved}
      <div class="flex items-center justify-center py-12">
        <Spinner size="lg" color="var(--text-secondary)" />
      </div>
    {:else}
      {@const items = activeTab === 'liked' ? likedMoments : activeTab === 'saved' ? savedMoments : galleryMoments}
      {#if items.length === 0}
        <div class="flex flex-col items-center justify-center py-12">
          <div
            class="flex items-center justify-center rounded-full"
            style="width: 56px; height: 56px; border: 1px solid var(--border-theme);"
          >
            <Grid3X3
              size={22}
              color="var(--text-secondary)"
              strokeWidth={1.25}
              style="opacity: 0.5;"
            />
          </div>
          <p class="text-body-lg" style="color: var(--text-secondary); margin-top: 12px">
            {activeTab === 'liked' ? t('profile.emptyLiked') : activeTab === 'saved' ? t('profile.emptySaved') : t('profile.emptyPosts')}
          </p>
        </div>
      {:else}
        <div class="grid grid-cols-2 md:grid-cols-3" style="gap: 10px; width: 100%;">
          {#each items as moment (moment.moment_id)}
            {@const img = moment.images?.[0]}
            <a
              href="/moments/{moment.moment_id}"
              class="surface-flat relative block overflow-hidden rounded-[var(--radius-md)]"
              style="aspect-ratio: 1/1;"
            >
              {#if img}
                <SmartImage
                  src={img.url}
                  thumbnail={img.thumbnail_url}
                  alt={moment.description}
                  width={img.width}
                  height={img.height}
                  sizes="150px"
                  class="h-full w-full object-cover"
                />
                <div
                  class="absolute bottom-0 left-0 right-0 flex items-center gap-2 px-2 py-1.5"
                  style="background: linear-gradient(transparent, rgba(0,0,0,0.5));"
                >
                  <span class="text-xs" style="color: var(--text-on-overlay)">
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
            </a>
          {/each}
        </div>
      {/if}
    {/if}
  {/snippet}

  <!-- ═══ Shared stats row ═══ -->
  {#snippet statsRowDisplay()}
    <div
      class="flex overflow-hidden rounded-[var(--radius-lg)] surface-flat"
      style="border: 1px solid var(--border-theme);"
    >
      {#each statsRow as stat, i (stat.label)}
        <div class="relative flex flex-1 flex-col items-center justify-center py-4">
          {#if i > 0}
            <div
              class="absolute left-0 top-1/2 -translate-y-1/2"
              style="width: 1px; height: 32px; background: var(--border-theme);"
            ></div>
          {/if}
          <span
            style="font-size: var(--text-display); font-weight: 600; color: var(--text-primary);"
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
        <input
          type="file"
          accept="image/*"
          class="hidden"
          bind:this={bannerInputElement}
          onchange={() => {
            if (bannerInputElement?.files?.[0]) {
              handleBannerUpload(bannerInputElement.files[0]);
              bannerInputElement.value = '';
            }
          }}
        />
        <button
          class="text-meta absolute top-3 right-3 flex items-center gap-1.5 rounded-full outline-none"
          style="padding: 6px 12px; background: var(--overlay); backdrop-filter: blur(10px); border: 1px solid var(--glass-border); color: var(--text-on-accent)"
          onclick={() => bannerInputElement?.click()}
          disabled={bannerUploading}
        >
          <Camera size={13} strokeWidth={1.75} />
          {bannerUploading ? t('profile.uploading') : t('profile.changeBanner')}
        </button>
      </div>

      <!-- Avatar edit -->
      <div class="flex items-center gap-4 -mt-8 ml-2">
        <div class="relative flex-shrink-0">
          <div
            class="overflow-hidden rounded-[var(--radius-lg)]"
            style="width: 72px; height: 72px; border: 3px solid var(--surface-bg);"
          >
            <Avatar
              src={p.avatar_url}
              name={nickname}
              alt="avatar"
              size={80}
              shape="rounded"
              border={false}
            />
          </div>
          <input
            type="file"
            accept="image/*"
            class="hidden"
            bind:this={avatarInputElement}
            onchange={() => {
              if (avatarInputElement?.files?.[0]) {
                handleAvatarUpload(avatarInputElement.files[0]);
                avatarInputElement.value = '';
              }
            }}
          />
          <button
            class="absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none"
            style="width: 26px; height: 26px; background: var(--brand); border: 2 solid var(--surface-bg); box-shadow: var(--shadow-brand-btn);"
            title={t('profile.changeAvatar')}
            onclick={() => avatarInputElement?.click()}
          >
            <Camera size={12} color="var(--text-on-accent)" strokeWidth={2} />
          </button>
        </div>
        <div>
          <p class="text-title" style="color: var(--text-primary)">{p.nickname}</p>
          <p class="text-meta" style="color: var(--text-secondary)">
            @{formatHandle(p)}
          </p>
        </div>
      </div>

      <!-- Name field -->
      <div class="flex flex-col gap-1.5">
        <div class="flex items-center gap-1.5">
          <User style="width: 13px; height: 13px; color: var(--brand);" strokeWidth={1.75} />
          <span class="eyebrow">{t('profile.nameFieldLabel')}</span>
        </div>
        <div class="input-surface rounded-[var(--radius-md)] px-4 py-3">
          <input
            id="nickname"
            bind:value={nickname}
            placeholder={t('profile.nameFieldPlaceholder')}
            class="text-body-lg w-full bg-transparent outline-none"
            style="color: var(--text-primary)"
          />
        </div>
      </div>

      <!-- Bio field -->
      <div class="flex flex-col gap-1.5">
        <div class="flex items-center gap-1.5">
          <FileText style="width: 13px; height: 13px; color: var(--brand);" strokeWidth={1.75} />
          <span class="eyebrow">{t('profile.bioFieldLabel')}</span>
        </div>
        <div class="input-surface rounded-[var(--radius-md)] px-4 py-3">
          <Textarea
            id="bio"
            variant="transparent"
            size="md"
            class="text-body-lg"
            bind:value={bio}
            placeholder={t('profile.bioFieldPlaceholder')}
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

      <!-- Location field -->
      {#if p.location?.location_name}
        <div class="flex flex-col gap-1.5">
          <div class="flex items-center gap-1.5">
            <MapPin style="width: 13px; height: 13px; color: var(--brand);" strokeWidth={1.75} />
            <span class="eyebrow">{t('profile.locationFieldLabel')}</span>
          </div>
          <div class="input-surface rounded-[var(--radius-md)] px-4 py-3">
            <input
              value={p.location.location_name}
              disabled
              class="text-body-lg w-full bg-transparent outline-none"
              style="color: var(--text-secondary); opacity: 0.7"
            />
          </div>
        </div>
      {/if}

      <!-- Interest tags -->
      <div class="flex flex-col gap-2.5">
        <div class="flex items-center gap-1.5">
          <span class="eyebrow">{t('profile.interestTagsLabel')}</span>
          <span class="text-caption" style="color: var(--text-secondary); opacity: 0.6"
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
          class="flex flex-1 items-center justify-center gap-1.5 rounded-[var(--radius-lg)] px-4 py-3.5 outline-none transition-all"
          style={saved
            ? 'background: var(--theme-success); color: var(--text-on-accent); font-size: var(--text-body-lg); box-shadow: 0 4px 14px color-mix(in srgb, var(--theme-success) 35%, transparent); border: none;'
            : 'background: var(--brand); color: var(--text-on-accent); font-size: var(--text-body-lg); box-shadow: var(--shadow-brand-btn); border: none;'}
          onclick={saveProfile}
          disabled={isSaving}
        >
          {#if saved}
            <Check size={14} color="var(--text-on-accent)" strokeWidth={2.5} />
          {:else if isSaving}
            <Spinner size="md" />
            {t('profile.saving')}
          {:else}
            {t('profile.save')}
          {/if}
        </button>
        <button
          class="text-body-lg flex flex-1 items-center justify-center gap-1.5 rounded-[var(--radius-lg)] px-4 py-3.5 outline-none surface-flat"
          style="color: var(--text-primary)"
          onclick={cancelEdit}
          disabled={isSaving}
        >
          {t('profile.cancel')}
        </button>
      </div>
    </div>
  {/snippet}

  <!-- ═══════════════════════════════════════════════
       WIDE (landscape/large tablet + desktop): master-detail 双栏
       窄屏改用长滚动（见 {:else}）。信息架构级差异，容器查询无法消解。
  ══════════════════════════════════════════════════ -->
  {#if platformStore.useMasterDetail}
    <div class="flex flex-col h-full">
      <!-- Sticky header -->
      <div
        class="flex-shrink-0 page-pad-x py-4"
        style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
      >
        <div class="page-shell page-shell--form flex items-center justify-between">
          <div>
            <p class="text-h1" style="color: var(--text-primary)">{profile.nickname}</p>
            <p class="text-meta" style="color: var(--text-secondary); margin-top: 1px">
              @{formatHandle(profile)}
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
      </div>

      <!-- Two-column body -->
      <div class="flex flex-1 min-h-0 overflow-hidden">
        <!-- Left panel (260px) -->
        <div
          class="flex-shrink-0 flex flex-col items-center overflow-y-auto no-scrollbar"
          style="width: 260px; border-right: 1px solid var(--border-theme); padding: 24px 20px;"
        >
          {@render avatarBlock(profile)}
          <h2 class="text-h2" style="color: var(--text-primary)">{profile.nickname}</h2>
          {#if profile.bio}
            <p
              class="text-meta"
              style="color: var(--text-secondary); text-align: center; padding: 6px 8px; line-height: 1.5"
            >
              {profile.bio}
            </p>
          {/if}
          {#if profile.location?.location_name}
            <div class="flex items-center gap-1 mb-2">
              <MapPin size={11} strokeWidth={1.5} color="var(--brand)" />
              <span class="text-caption" style="color: var(--text-secondary)"
                >{profile.location.location_name}</span
              >
            </div>
          {/if}

          <button
            class="text-body w-full flex items-center justify-center gap-2 rounded-full py-2.5 mb-5 outline-none surface-flat"
            onclick={() => (isEditing = !isEditing)}
          >
            <Pencil size={13} strokeWidth={1.5} color="var(--text-secondary)" />
            <span style="color: var(--text-secondary);">{t('profile.editProfile')}</span>
          </button>

          <!-- Analytics card -->
          <div
            class="w-full overflow-hidden rounded-[var(--radius-md)] mb-4 surface-flat"
            style="border: 1px solid var(--border-theme);"
          >
            {#each statsRow as stat}
              <div
                class="flex items-center justify-between px-4 py-3"
                style="border-bottom: 1px solid var(--border-theme);"
              >
                <span class="text-body" style="color: var(--text-secondary)">{stat.label}</span>
                <span class="text-title" style="font-weight: 600; color: var(--text-primary)"
                  >{stat.value}</span
                >
              </div>
            {/each}
          </div>

          <!-- Personal hub — 仅宽屏平板（无侧栏直达）需要；桌面侧栏已直达各项，去重 -->
          {#if !platformStore.hideRedundantEntry}
            {@render hubCard()}
          {/if}
        </div>

        <!-- Right panel: tab bar + gallery -->
        <div class="flex-1 overflow-y-auto no-scrollbar page-pad">
          <div class="page-shell page-shell--narrow">
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
        class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
        style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
      >
        <div class="page-shell page-shell--form flex items-center justify-between">
          <div>
            <p class="text-h2" style="color: var(--text-primary)">{profile.nickname}</p>
            <p class="text-caption" style="color: var(--text-secondary); margin-top: 1px">
              @{formatHandle(profile)}
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
      </div>

      <div class="flex flex-col page-pad">
        <div class="page-shell page-shell--form">
          <!-- Avatar section -->
          <div class="flex flex-col items-center px-5 pt-5 pb-4">
            {@render avatarBlock(profile)}
            <h2 class="text-h1" style="color: var(--text-primary)">{profile.nickname}</h2>
            {#if profile.bio && !isEditing}
              <p
                class="text-meta"
                style="color: var(--text-secondary); text-align: center; padding: 8px 32px; line-height: 1.6"
              >
                {profile.bio}
              </p>
            {/if}
            {#if profile.location?.location_name}
              <div class="flex items-center gap-1 mb-2">
                <MapPin size={11} strokeWidth={1.5} color="var(--brand)" />
                <span class="text-caption" style="color: var(--text-secondary)"
                  >{profile.location.location_name}</span
                >
              </div>
            {/if}
            {#if !isEditing}
              <button
                class="text-body flex items-center gap-2 rounded-full px-10 py-2.5 outline-none surface-flat"
                onclick={() => (isEditing = true)}
              >
                <Pencil size={13} strokeWidth={1.5} color="var(--text-secondary)" />
                <span style="color: var(--text-secondary);">{t('profile.editProfile')}</span>
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
        <TriangleAlert size={28} strokeWidth={1.5} color="var(--text-secondary)" />
      </div>
      <p
        class="text-title"
        style="font-weight: 500; color: var(--text-primary); margin-bottom: 8px"
      >
        {t('profile.loadProfileFailed')}
      </p>
      <p class="text-body-lg" style="color: var(--text-secondary); margin-bottom: 16px">
        {data.error ?? t('profile.loadProfileError')}
      </p>
      <button
        class="text-body-lg rounded-[var(--radius-lg)] px-5 py-3.5 outline-none surface-flat"
        style="color: var(--brand)"
        onclick={() => location.reload()}>{t('profile.retry')}</button
      >
    </div>
  </div>
{/if}
