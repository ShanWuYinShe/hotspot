<script lang="ts">
  import { onMount } from 'svelte';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { adminAppConfigService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import type { AppConfig } from '$lib/types/api';
  import AdminPageHeader from '$lib/components/common/admin-page-header.svelte';
  import RefreshCw from 'lucide-svelte/icons/refresh-cw';
  import Save from 'lucide-svelte/icons/save';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import Lock from 'lucide-svelte/icons/lock';
  import FileText from 'lucide-svelte/icons/file-text';
  import Bell from 'lucide-svelte/icons/bell';

  let appConfig = $state<AppConfig | null>(null);
  let isLoading = $state(true);
  let isSaving = $state(false);

  let imageMaxSize = $state('');
  let videoMaxSize = $state('');
  let audioMaxSize = $state('');
  let fileMaxSize = $state('');
  let enablePublicRegistration = $state(true);
  let enableGuestLogin = $state(true);
  let enableMoments = $state(true);
  let enableNearbyPeople = $state(true);
  let maxMomentTags = $state('');
  let maxMomentImages = $state('');
  let momentDescriptionMaxLength = $state('');
  let commentMaxLength = $state('');
  let maxGroupMembers = $state('');
  let searchRadiusMax = $state('');

  async function loadConfig() {
    isLoading = true;
    try {
      const response = await withMinimumLoadingTime(() => adminAppConfigService.get());
      if (isSuccess(response) && response.data) {
        appConfig = response.data;
        imageMaxSize = String(appConfig.upload?.image_max_size || 5242880);
        videoMaxSize = String(appConfig.upload?.video_max_size || 52428800);
        audioMaxSize = String(appConfig.upload?.audio_max_size || 10485760);
        fileMaxSize = String(appConfig.upload?.file_max_size || 10485760);
        enablePublicRegistration = appConfig.features?.enable_public_registration ?? true;
        enableGuestLogin = appConfig.features?.enable_guest_login ?? true;
        enableMoments = appConfig.features?.enable_moments ?? true;
        enableNearbyPeople = appConfig.features?.enable_nearby_people ?? true;
        maxMomentTags = String(appConfig.limits?.max_moment_tags || 10);
        maxMomentImages = String(appConfig.limits?.max_moment_images || 9);
        momentDescriptionMaxLength = String(appConfig.limits?.moment_description_max_length || 300);
        commentMaxLength = String(appConfig.limits?.comment_max_length || 200);
        maxGroupMembers = String(appConfig.limits?.max_group_members || 500);
        searchRadiusMax = String(appConfig.limits?.search_radius_max || 50000);
      }
    } catch {
      toast.error('加载配置失败');
    } finally {
      isLoading = false;
    }
  }

  async function handleSaveUpload() {
    isSaving = true;
    try {
      const response = await adminAppConfigService.updateUpload({
        upload: {
          image_max_size: parseInt(imageMaxSize) || undefined,
          video_max_size: parseInt(videoMaxSize) || undefined,
          audio_max_size: parseInt(audioMaxSize) || undefined,
          file_max_size: parseInt(fileMaxSize) || undefined
        }
      });
      if (isSuccess(response)) {
        toast.success('上传配置已保存');
      } else {
        toast.error(response.message || '保存失败');
      }
    } catch {
      toast.error('保存失败');
    } finally {
      isSaving = false;
    }
  }

  async function handleSaveFeatures() {
    isSaving = true;
    try {
      const response = await adminAppConfigService.updateFeatures({
        enable_public_registration: enablePublicRegistration,
        enable_guest_login: enableGuestLogin,
        enable_moments: enableMoments,
        enable_nearby_people: enableNearbyPeople
      });
      if (isSuccess(response)) {
        toast.success('功能配置已保存');
      } else {
        toast.error(response.message || '保存失败');
      }
    } catch {
      toast.error('保存失败');
    } finally {
      isSaving = false;
    }
  }

  async function handleSaveLimits() {
    isSaving = true;
    try {
      const response = await adminAppConfigService.updateLimits({
        limits: {
          max_moment_tags: parseInt(maxMomentTags) || undefined,
          max_moment_images: parseInt(maxMomentImages) || undefined,
          moment_description_max_length: parseInt(momentDescriptionMaxLength) || undefined,
          comment_max_length: parseInt(commentMaxLength) || undefined,
          max_group_members: parseInt(maxGroupMembers) || undefined,
          search_radius_max: parseInt(searchRadiusMax) || undefined
        }
      });
      if (isSuccess(response)) {
        toast.success('限制配置已保存');
      } else {
        toast.error(response.message || '保存失败');
      }
    } catch {
      toast.error('保存失败');
    } finally {
      isSaving = false;
    }
  }

  onMount(() => {
    loadConfig();
  });
</script>

<svelte:head>
  <title>系统配置 - HotSpot 管理后台</title>
</svelte:head>

<AdminPageHeader title="系统配置" description="系统参数配置">
  {#snippet actions()}
    <button
      class="flex items-center gap-1.5 rounded-full px-4 py-2 outline-none"
      style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); font-size: 13px; color: var(--text-primary);"
      onclick={loadConfig}
      disabled={isLoading}
    >
      <RefreshCw size={14} strokeWidth={1.5} class={isLoading ? 'animate-spin' : ''} />
      刷新
    </button>
  {/snippet}
</AdminPageHeader>

<div class="page-pad">
  <div class="page-shell page-shell--wide space-y-6">
    {#if isLoading}
      <div class="grid gap-6 lg:grid-cols-2">
        {#each Array(3) as _, i (i)}
          <div
            class="rounded-[22px] p-4"
            style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
          >
            <div class="space-y-3">
              <div
                class="rounded"
                style="height: 14px; width: 120px; background: var(--border-theme);"
              ></div>
              {#each Array(3) as _, j (j)}
                <div class="space-y-1.5">
                  <div
                    class="rounded"
                    style="height: 10px; width: 80px; background: var(--border-theme);"
                  ></div>
                  <div
                    class="rounded-[14px]"
                    style="height: 36px; width: 100%; background: var(--border-theme);"
                  ></div>
                </div>
              {/each}
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="grid gap-6 lg:grid-cols-2">
        <!-- Upload settings -->
        <div
          class="rounded-[22px] overflow-hidden"
          style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
        >
          <div
            class="flex items-center gap-2 px-4 py-3"
            style="border-bottom: 1px solid var(--border-theme);"
          >
            <FileText size={16} strokeWidth={1.5} color="var(--brand)" />
            <span style="font-size: 14px; font-weight: 500; color: var(--text-primary);"
              >上传设置</span
            >
          </div>
          <div class="space-y-4 p-4">
            {#each [{ id: 'image-max-size', label: '图片最大大小 (字节)', value: imageMaxSize, placeholder: '5242880', set: (value: string) => (imageMaxSize = value) }, { id: 'video-max-size', label: '视频最大大小 (字节)', value: videoMaxSize, placeholder: '52428800', set: (value: string) => (videoMaxSize = value) }, { id: 'audio-max-size', label: '音频最大大小 (字节)', value: audioMaxSize, placeholder: '10485760', set: (value: string) => (audioMaxSize = value) }, { id: 'file-max-size', label: '文件最大大小 (字节)', value: fileMaxSize, placeholder: '10485760', set: (value: string) => (fileMaxSize = value) }] as field (field.id)}
              <div>
                <label for={field.id} style="font-size: 12px; color: var(--text-primary);"
                  >{field.label}</label
                >
                <input
                  id={field.id}
                  type="number"
                  value={field.value}
                  placeholder={field.placeholder}
                  class="field-inset mt-1.5 w-full rounded-[14px] px-4 py-2.5 outline-none"
                  style="background: var(--surface-bg); color: var(--text-primary); font-size: 13px;"
                  oninput={(event) => field.set(event.currentTarget.value)}
                />
                <p style="font-size: 11px; color: var(--text-secondary); margin-top: 2px;">
                  当前: {(Number(field.value || 0) / 1024 / 1024).toFixed(1)} MB
                </p>
              </div>
            {/each}
            <button
              class="flex items-center gap-1.5 rounded-full px-5 py-2 outline-none"
              style="background: var(--brand); color: var(--text-on-accent); font-size: 13px; box-shadow: 0 3px 10px var(--glow);"
              onclick={handleSaveUpload}
              disabled={isSaving}
            >
              {#if isSaving}<Loader2 size={14} strokeWidth={1.5} class="animate-spin" />{:else}<Save
                  size={14}
                  strokeWidth={1.5}
                />{/if}
              保存上传设置
            </button>
          </div>
        </div>

        <!-- Feature toggles -->
        <div
          class="rounded-[22px] overflow-hidden"
          style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
        >
          <div
            class="flex items-center gap-2 px-4 py-3"
            style="border-bottom: 1px solid var(--border-theme);"
          >
            <Lock size={16} strokeWidth={1.5} color="var(--brand)" />
            <span style="font-size: 14px; font-weight: 500; color: var(--text-primary);"
              >功能开关</span
            >
          </div>
          <div class="space-y-3 p-4">
            {#each [{ label: '允许新用户注册', desc: '关闭后新用户无法注册', val: enablePublicRegistration, key: 'reg' }, { label: '允许访客模式', desc: '允许用户以访客身份使用', val: enableGuestLogin, key: 'guest' }, { label: '启用动态功能', desc: '控制是否开放生活瞬间', val: enableMoments, key: 'moments' }, { label: '启用附近的人', desc: '是否展示附近用户', val: enableNearbyPeople, key: 'nearby' }] as toggle (toggle.key)}
              <div class="flex items-center justify-between gap-4">
                <div>
                  <p style="font-size: 13px; color: var(--text-primary);">{toggle.label}</p>
                  <p style="font-size: 11px; color: var(--text-secondary); margin-top: 1px;">
                    {toggle.desc}
                  </p>
                </div>
                <button
                  type="button"
                  role="switch"
                  aria-checked={toggle.val}
                  aria-label={toggle.label}
                  class="relative rounded-full outline-none"
                  style="width: 44px; height: 24px; transition: background 0.2s; {toggle.val
                    ? 'background: var(--brand);'
                    : 'background: var(--border-theme);'}"
                  onclick={() => {
                    if (toggle.key === 'reg') enablePublicRegistration = !enablePublicRegistration;
                    else if (toggle.key === 'guest') enableGuestLogin = !enableGuestLogin;
                    else if (toggle.key === 'moments') enableMoments = !enableMoments;
                    else enableNearbyPeople = !enableNearbyPeople;
                  }}
                >
                  <div
                    class="absolute top-1 rounded-full"
                    style="width: 16px; height: 16px; background: var(--surface-bg); transition: left 0.2s; left: {toggle.val
                      ? '23px'
                      : '4px'}; box-shadow: var(--theme-switch-shadow);"
                  ></div>
                </button>
              </div>
            {/each}
            <button
              class="flex items-center gap-1.5 rounded-full px-5 py-2 outline-none"
              style="background: var(--brand); color: var(--text-on-accent); font-size: 13px; box-shadow: 0 3px 10px var(--glow);"
              onclick={handleSaveFeatures}
              disabled={isSaving}
            >
              {#if isSaving}<Loader2 size={14} strokeWidth={1.5} class="animate-spin" />{:else}<Save
                  size={14}
                  strokeWidth={1.5}
                />{/if}
              保存功能设置
            </button>
          </div>
        </div>

        <!-- Limits -->
        <div
          class="rounded-[22px] overflow-hidden"
          style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
        >
          <div
            class="flex items-center gap-2 px-4 py-3"
            style="border-bottom: 1px solid var(--border-theme);"
          >
            <Bell size={16} strokeWidth={1.5} color="var(--brand)" />
            <span style="font-size: 14px; font-weight: 500; color: var(--text-primary);"
              >限制设置</span
            >
          </div>
          <div class="space-y-4 p-4">
            {#each [{ id: 'max-tags', label: '动态最大标签数', value: maxMomentTags, ph: '10', set: (value: string) => (maxMomentTags = value) }, { id: 'max-images', label: '动态最大图片数', value: maxMomentImages, ph: '9', set: (value: string) => (maxMomentImages = value) }, { id: 'desc-max', label: '动态描述最大长度', value: momentDescriptionMaxLength, ph: '300', set: (value: string) => (momentDescriptionMaxLength = value) }, { id: 'comment-max', label: '评论最大长度', value: commentMaxLength, ph: '200', set: (value: string) => (commentMaxLength = value) }, { id: 'group-max', label: '群组最大成员数', value: maxGroupMembers, ph: '500', set: (value: string) => (maxGroupMembers = value) }, { id: 'radius-max', label: '搜索最大半径 (km)', value: searchRadiusMax, ph: '50', set: (value: string) => (searchRadiusMax = value) }] as f (f.id)}
              <div>
                <label for={f.id} style="font-size: 12px; color: var(--text-primary);"
                  >{f.label}</label
                >
                <input
                  id={f.id}
                  type="number"
                  value={f.value}
                  placeholder={f.ph}
                  class="field-inset mt-1.5 w-full rounded-[14px] px-4 py-2.5 outline-none"
                  style="background: var(--surface-bg); color: var(--text-primary); font-size: 13px;"
                  oninput={(event) => f.set(event.currentTarget.value)}
                />
              </div>
            {/each}
            <button
              class="flex items-center gap-1.5 rounded-full px-5 py-2 outline-none"
              style="background: var(--brand); color: var(--text-on-accent); font-size: 13px; box-shadow: 0 3px 10px var(--glow);"
              onclick={handleSaveLimits}
              disabled={isSaving}
            >
              {#if isSaving}<Loader2 size={14} strokeWidth={1.5} class="animate-spin" />{:else}<Save
                  size={14}
                  strokeWidth={1.5}
                />{/if}
              保存限制设置
            </button>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>
