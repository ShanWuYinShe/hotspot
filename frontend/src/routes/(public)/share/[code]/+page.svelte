<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { toast } from '$lib/stores/toast.svelte';
  import { t } from '$lib/i18n';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Share2 from 'lucide-svelte/icons/share-2';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import type {
    ShareDetail,
    LocationSharePayload,
    MomentSharePayload,
    GroupSharePayload,
    UserSharePayload
  } from '$lib/types/api';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  const share = $derived<ShareDetail | null>(data.share);

  // 用于清理导航延迟定时器
  let navigationTimer: ReturnType<typeof setTimeout> | null = null;

  // 复制分享链接
  async function copyShareLink() {
    const url = window.location.href;
    try {
      await navigator.clipboard.writeText(url);
      toast.success(t('share.linkCopied'));
    } catch (err) {
      toast.error(t('share.copyFailed'));
    }
  }

  // 处理分享（客户端本地导航，不使用 apply 接口）
  function handleApply() {
    if (!share) return;

    const { share_type } = share.share;
    const payload = share.payload;

    if (share_type === 'location') {
      const loc = (payload as LocationSharePayload).shared_location;
      mapAnchorStore.setAnchor(loc.point.latitude, loc.point.longitude, loc.location_name);
      toast.success(t('share.anchorUpdated'));
      navigationTimer = setTimeout(() => goto('/'), 500);
    } else if (share_type === 'user') {
      goto(`/users/${(payload as UserSharePayload).shared_user_id}`);
    } else if (share_type === 'group') {
      goto(`/groups/${(payload as GroupSharePayload).shared_group_id}`);
    } else if (share_type === 'moment') {
      goto(`/moments/${(payload as MomentSharePayload).shared_moment_id}`);
    }
  }

  // 显示错误信息（一次性操作）
  onMount(() => {
    if (data.error) {
      toast.error(data.error);
    }

    return () => {
      if (navigationTimer) clearTimeout(navigationTimer);
    };
  });
</script>

<svelte:head>
  <title
    >{share?.share.share_type === 'location' &&
    (share.payload as LocationSharePayload).shared_location?.location_name
      ? (share.payload as LocationSharePayload).shared_location.location_name
      : t('share.titleExplore')} - HotSpot</title
  >
</svelte:head>

<div class="page-pad">
  <div class="page-shell page-shell--narrow">
    {#if !share}
      <div
        class="rounded-[var(--radius-lg)] p-6"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <h2 class="text-h2" style="font-weight: 600; color: var(--theme-error)">{t('share.loadFailed')}</h2>
        <p class="text-body-lg mt-2" style="color: var(--text-secondary)">
          {data.error || t('share.notExist')}
        </p>
        <button
          class="mt-4 rounded-full px-5 py-2.5 font-medium outline-none transition-all duration-150 hover:-translate-y-0.5 active:scale-[0.97]"
          style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
          onclick={() => goto('/')}
        >
          {t('share.backHome')}
        </button>
      </div>
    {:else if share.share.share_type === 'location'}
      {@const payload = share.payload as LocationSharePayload}
      <div
        class="overflow-hidden rounded-[var(--radius-lg)]"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <!-- Header with creator info -->
        <div
          class="flex items-center justify-between p-6"
          style="border-bottom: 1px solid var(--border-theme);"
        >
          <div>
            <h2 class="text-h1" style="font-weight: 700; color: var(--text-primary)">
              {payload.shared_location.location_name}
            </h2>
          </div>
          <div class="flex items-center gap-3">
            <Avatar
              src={share.share.creator.avatar_url}
              name={share.share.creator.nickname}
              alt={share.share.creator.nickname}
              size={32}
            />
            <span class="text-body" style="color: var(--text-secondary)">
              <span style="font-weight: 500; color: var(--text-primary);"
                >{share.share.creator.nickname}</span
              > {t('share.sharedLocation')}
            </span>
          </div>
        </div>

        <div class="space-y-6 p-6">
          <!-- Primary action button -->
          <button
            class="flex w-full items-center justify-center gap-2 rounded-full py-4 text-base font-semibold outline-none transition-all duration-150 hover:-translate-y-0.5 active:scale-[0.97]"
            style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
            onclick={handleApply}
          >
            <MapPin size={20} />
            {t('share.setAsAnchor')}
          </button>

          <!-- Info card -->
          <div
            class="rounded-[var(--radius-md)] p-4"
            style="border: 1px solid var(--border-theme);"
          >
            <p
              class="text-body-lg flex items-center gap-2"
              style="font-weight: 500; color: var(--text-primary)"
            >
              <UsersRound size={16} />
              {t('share.willSeeTitle')}
            </p>
            <ul class="text-body mt-3 space-y-2" style="color: var(--text-secondary)">
              <li class="flex items-start gap-2">
                <span style="color: var(--brand);">•</span>
                <span>{t('share.willSeePeople')}</span>
              </li>
              <li class="flex items-start gap-2">
                <span style="color: var(--brand);">•</span>
                <span>{t('share.willSeeGroups')}</span>
              </li>
              <li class="flex items-start gap-2">
                <span style="color: var(--brand);">•</span>
                <span>{t('share.willSeeMoments')}</span>
              </li>
            </ul>
          </div>

          <!-- Location note -->
          <div>
            <p
              class="text-xs uppercase mb-2"
              style="letter-spacing: 0.14em; color: var(--text-secondary);"
            >
              {t('share.locationNote')}
            </p>
            <div
              class="rounded-[var(--radius-md)] p-3"
              style="border: 1px solid var(--border-theme);"
            >
              <p class="text-meta" style="color: var(--text-secondary)">
                {t('share.locationNoteDesc', { name: payload.shared_location.location_name })}
              </p>
            </div>
          </div>

          <!-- Secondary actions -->
          <div class="flex items-center gap-2">
            <button
              class="flex flex-1 items-center justify-center gap-2 rounded-full py-2.5 font-medium outline-none"
              style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
              onclick={copyShareLink}
            >
              <Share2 size={16} />
              分享给朋友
            </button>
            <button
              class="rounded-full px-5 py-2.5 font-medium outline-none"
              style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
              onclick={() => goto('/')}
            >
              返回首页
            </button>
          </div>
        </div>
      </div>
    {:else}
      <!-- Other share types (user/group/moment) -->
      <div
        class="rounded-[var(--radius-lg)] p-6"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
      >
        <h2 class="text-h2" style="font-weight: 600; color: var(--text-primary)">{t('share.otherShareTitle')}</h2>
        <p class="text-body-lg mt-2" style="color: var(--text-secondary)">
          {t('share.shareTypeLabel', { type: share.share.share_type === 'user'
            ? t('share.typeUser')
            : share.share.share_type === 'group'
              ? t('share.typeGroup')
              : t('share.typeMoment') })}
        </p>
        <div class="mt-4 space-y-3">
          <button
            class="w-full rounded-full py-3 font-medium outline-none"
            style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
            onclick={handleApply}
          >
            {t('share.viewNow')}
          </button>
          <button
            class="w-full rounded-full py-3 font-medium outline-none"
            style="border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
            onclick={() => goto('/')}
          >
            返回首页
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>
