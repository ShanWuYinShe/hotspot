<script lang="ts">
  import { onMount } from 'svelte';
  import { withMinimumLoadingTime } from '$lib/utils/loading';
  import { usersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import PageHeader from '$lib/components/layout/page-header.svelte';
  import Loader2 from 'lucide-svelte/icons/loader-2';
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
      toast.error('加载隐私设置失败');
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
        toast.success('隐私设置已保存');
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
    loadSettings();
  });
</script>

<svelte:head>
  <title>隐私设置 - HotSpot</title>
</svelte:head>

<div class="page-pad">
  <div class="page-shell page-shell--form space-y-5">
    <PageHeader
      title="隐私设置"
      description="控制你在 HotSpot 的可见范围与互动权限"
      backHref="/settings"
      backLabel="返回设置"
    />

    {#if isLoading}
      <div
        class="rounded-[22px] p-5"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
      >
        <div class="space-y-4">
          {#each Array(5) as _, i (i)}
            <div
              class="h-11 w-full rounded-[18px]"
              style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
            ></div>
          {/each}
        </div>
      </div>
    {:else}
      <!-- Basic Visibility Card -->
      <div
        class="rounded-[22px] p-5"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
      >
        <p
          style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 4px;"
        >
          基础可见性
        </p>
        <p style="font-size: 12px; color: var(--text-secondary); margin-bottom: 16px;">
          控制大部分信息默认可见范围
        </p>

        <div class="space-y-3">
          <div
            class="rounded-[18px] p-4"
            style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
          >
            <label
              for="privacy-level"
              style="font-size: 13px; color: var(--text-secondary); display: block; margin-bottom: 8px;"
              >隐私级别</label
            >
            <select
              id="privacy-level"
              class="input-base w-full rounded-[18px] px-4 py-2.5 outline-none"
              style="border: 1px solid var(--border-theme); background: var(--surface-bg); color: var(--text-primary);"
              bind:value={privacyLevel}
            >
              <option value="public">公开</option>
              <option value="private">仅自己</option>
            </select>
          </div>

          <div
            class="flex items-center justify-between rounded-[18px] px-4 py-3.5"
            style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
          >
            <div>
              <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">位置可见</p>
              <p style="font-size: 12px; color: var(--text-secondary);">允许展示你的地理位置</p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={locationVisible}
              aria-label="位置可见"
              class="relative flex h-7 w-12 shrink-0 items-center rounded-full transition-colors"
              style={locationVisible
                ? 'background: var(--brand); box-shadow: 0 2px 8px var(--glow);'
                : 'background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);'}
              onclick={() => (locationVisible = !locationVisible)}
            >
              <span
                class="block h-5 w-5 rounded-full transition-transform"
                style="background: {locationVisible
                  ? 'white'
                  : 'var(--text-secondary)'}; transform: translateX({locationVisible
                  ? '22px'
                  : '4px'}); box-shadow: 0 1px 3px var(--theme-switch-shadow);"
              ></span>
            </button>
          </div>

          <div
            class="flex items-center justify-between rounded-[18px] px-4 py-3.5"
            style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
          >
            <div>
              <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">
                在线状态可见
              </p>
              <p style="font-size: 12px; color: var(--text-secondary);">允许显示在线/离线状态</p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={presenceVisible}
              aria-label="在线状态可见"
              class="relative flex h-7 w-12 shrink-0 items-center rounded-full transition-colors"
              style={presenceVisible
                ? 'background: var(--brand); box-shadow: 0 2px 8px var(--glow);'
                : 'background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);'}
              onclick={() => (presenceVisible = !presenceVisible)}
            >
              <span
                class="block h-5 w-5 rounded-full transition-transform"
                style="background: {presenceVisible
                  ? 'white'
                  : 'var(--text-secondary)'}; transform: translateX({presenceVisible
                  ? '22px'
                  : '4px'}); box-shadow: 0 1px 3px var(--theme-switch-shadow);"
              ></span>
            </button>
          </div>

          <div
            class="flex items-center justify-between rounded-[18px] px-4 py-3.5"
            style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
          >
            <div>
              <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">
                好友/关注/粉丝列表可见
              </p>
              <p style="font-size: 12px; color: var(--text-secondary);">
                控制你的社交列表是否对他人可见
              </p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={socialListsVisible}
              aria-label="好友/关注/粉丝列表可见"
              class="relative flex h-7 w-12 shrink-0 items-center rounded-full transition-colors"
              style={socialListsVisible
                ? 'background: var(--brand); box-shadow: 0 2px 8px var(--glow);'
                : 'background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);'}
              onclick={() => (socialListsVisible = !socialListsVisible)}
            >
              <span
                class="block h-5 w-5 rounded-full transition-transform"
                style="background: {socialListsVisible
                  ? 'white'
                  : 'var(--text-secondary)'}; transform: translateX({socialListsVisible
                  ? '22px'
                  : '4px'}); box-shadow: 0 1px 3px var(--theme-switch-shadow);"
              ></span>
            </button>
          </div>
        </div>
      </div>

      <!-- Social & Chat Card -->
      <div
        class="rounded-[22px] p-5"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
      >
        <p
          style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 4px;"
        >
          社交与私聊
        </p>
        <p style="font-size: 12px; color: var(--text-secondary); margin-bottom: 16px;">
          管理关注与私聊权限
        </p>

        <div class="space-y-3">
          <div
            class="flex items-center justify-between rounded-[18px] px-4 py-3.5"
            style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
          >
            <div>
              <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">允许关注</p>
              <p style="font-size: 12px; color: var(--text-secondary);">关闭后他人关注需审批</p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={allowFollow}
              aria-label="允许关注"
              class="relative flex h-7 w-12 shrink-0 items-center rounded-full transition-colors"
              style={allowFollow
                ? 'background: var(--brand); box-shadow: 0 2px 8px var(--glow);'
                : 'background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);'}
              onclick={() => (allowFollow = !allowFollow)}
            >
              <span
                class="block h-5 w-5 rounded-full transition-transform"
                style="background: {allowFollow
                  ? 'white'
                  : 'var(--text-secondary)'}; transform: translateX({allowFollow
                  ? '22px'
                  : '4px'}); box-shadow: 0 1px 3px var(--theme-switch-shadow);"
              ></span>
            </button>
          </div>

          <div
            class="rounded-[18px] p-4"
            style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
          >
            <label
              for="private-chat"
              style="font-size: 13px; color: var(--text-secondary); display: block; margin-bottom: 8px;"
              >私聊权限</label
            >
            <select
              id="private-chat"
              class="field-inset w-full rounded-[18px] px-4 py-2.5 outline-none"
              style="background: var(--surface-bg); color: var(--text-primary);"
              bind:value={allowPrivateChat}
            >
              <option value="everyone">所有人可私聊</option>
              <option value="friends">仅好友可私聊</option>
              <option value="nobody">不允许私聊</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Search & Discovery Card -->
      <div
        class="rounded-[22px] p-5"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
      >
        <p
          style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 4px;"
        >
          搜索与发现
        </p>
        <p style="font-size: 12px; color: var(--text-secondary); margin-bottom: 16px;">
          控制你在搜索/附近中的展示
        </p>

        <div class="space-y-3">
          <div
            class="flex items-center justify-between rounded-[18px] px-4 py-3.5"
            style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
          >
            <div>
              <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">
                允许被搜索
              </p>
              <p style="font-size: 12px; color: var(--text-secondary);">关闭后不会出现在搜索结果</p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={searchable}
              aria-label="允许被搜索"
              class="relative flex h-7 w-12 shrink-0 items-center rounded-full transition-colors"
              style={searchable
                ? 'background: var(--brand); box-shadow: 0 2px 8px var(--glow);'
                : 'background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);'}
              onclick={() => (searchable = !searchable)}
            >
              <span
                class="block h-5 w-5 rounded-full transition-transform"
                style="background: {searchable
                  ? 'white'
                  : 'var(--text-secondary)'}; transform: translateX({searchable
                  ? '22px'
                  : '4px'}); box-shadow: 0 1px 3px var(--theme-switch-shadow);"
              ></span>
            </button>
          </div>

          <div
            class="flex items-center justify-between rounded-[18px] px-4 py-3.5"
            style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
          >
            <div>
              <p style="font-size: 14px; font-weight: 500; color: var(--text-primary);">附近可见</p>
              <p style="font-size: 12px; color: var(--text-secondary);">允许出现在附近的人</p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={locationSearchable}
              aria-label="附近可见"
              class="relative flex h-7 w-12 shrink-0 items-center rounded-full transition-colors"
              style={locationSearchable
                ? 'background: var(--brand); box-shadow: 0 2px 8px var(--glow);'
                : 'background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);'}
              onclick={() => (locationSearchable = !locationSearchable)}
            >
              <span
                class="block h-5 w-5 rounded-full transition-transform"
                style="background: {locationSearchable
                  ? 'white'
                  : 'var(--text-secondary)'}; transform: translateX({locationSearchable
                  ? '22px'
                  : '4px'}); box-shadow: 0 1px 3px var(--theme-switch-shadow);"
              ></span>
            </button>
          </div>
        </div>
      </div>

      <button
        class="flex h-11 w-full items-center justify-center rounded-full text-sm font-medium transition-opacity hover:opacity-90 disabled:opacity-50"
        style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 4px 14px var(--glow);"
        onclick={saveSettings}
        disabled={isSaving}
      >
        {#if isSaving}
          <Loader2 size={16} class="mr-2 animate-spin" />
        {/if}
        保存设置
      </button>
    {/if}
  </div>
</div>
