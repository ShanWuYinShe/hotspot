<script lang="ts">
  import * as Dialog from '$lib/components/ui/dialog';
  import { groupsService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { showApiError } from '$lib/utils/api-error';
  import { parseTags } from '$lib/utils/format';
  import { validateField } from '$lib/utils/form-validation';
  import type { FieldRule } from '$lib/utils/form-validation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import LocationPickerModal from './location-picker-modal.svelte';
  import AnchorSummaryCard from '$lib/components/common/anchor-summary-card.svelte';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import UsersRound from 'lucide-svelte/icons/users-round';

  type JoinMethod = 'open' | 'approval' | 'invite_only';

  interface SelectedLocation {
    latitude: number;
    longitude: number;
    locationName: string;
    radius: number;
  }

  interface Props {
    open: boolean;
    onOpenChange?: (open: boolean) => void;
    onCreated?: () => void;
  }

  let { open = $bindable(), onOpenChange, onCreated }: Props = $props();

  let name = $state('');
  let description = $state('');
  let tagsInput = $state('');
  let joinMethod = $state<JoinMethod>('open');
  let password = $state('');
  let maxMembers = $state('');
  let isSearchable = $state(true);
  let isSubmitting = $state(false);
  let touched = $state<Record<string, boolean>>({});
  let showLocationPicker = $state(false);
  const GROUP_NAME_MAX = 50;
  const GROUP_DESCRIPTION_MAX = 500;
  const GROUP_PASSWORD_MIN = 4;
  const GROUP_PASSWORD_MAX = 32;
  const GROUP_TAG_MAX = 10;
  const TAG_MAX_LENGTH = 30;
  const GROUP_MEMBERS_MIN = 2;
  const GROUP_MEMBERS_MAX = 2000;
  const LOCATION_NAME_MAX = 100;

  const tags = $derived(parseTags(tagsInput));

  const nameRules: FieldRule[] = [
    { required: true, requiredMessage: '请输入群组名称' },
    { maxLength: GROUP_NAME_MAX, maxLengthMessage: `群组名称最多 ${GROUP_NAME_MAX} 字符` }
  ];
  const groupDescriptionRules: FieldRule[] = [
    {
      maxLength: GROUP_DESCRIPTION_MAX,
      maxLengthMessage: `群组介绍最多 ${GROUP_DESCRIPTION_MAX} 字符`
    }
  ];
  const groupPasswordRules: FieldRule[] = [
    {
      minLength: GROUP_PASSWORD_MIN,
      minLengthMessage: `群组口令至少 ${GROUP_PASSWORD_MIN} 字符`
    },
    {
      maxLength: GROUP_PASSWORD_MAX,
      maxLengthMessage: `群组口令最多 ${GROUP_PASSWORD_MAX} 字符`
    }
  ];
  const maxMembersRules: FieldRule[] = [
    {
      custom: (v) => {
        if (!v.trim()) return undefined;
        const n = Number.parseInt(v, 10);
        if (!Number.isInteger(n) || n < GROUP_MEMBERS_MIN || n > GROUP_MEMBERS_MAX) {
          return `最大成员数需在 ${GROUP_MEMBERS_MIN}-${GROUP_MEMBERS_MAX} 之间`;
        }
        return undefined;
      }
    }
  ];

  let nameError = $derived(touched['name'] ? validateField(name, nameRules) : undefined);
  let groupDescriptionError = $derived(
    touched['description'] ? validateField(description, groupDescriptionRules) : undefined
  );
  let groupPasswordError = $derived(
    touched['password'] ? validateField(password, groupPasswordRules) : undefined
  );
  let maxMembersError = $derived(
    touched['maxMembers'] ? validateField(maxMembers, maxMembersRules) : undefined
  );

  function markTouched(field: string) {
    touched = { ...touched, [field]: true };
  }

  function getDefaultSelectedLocation(): SelectedLocation {
    const rawName = mapAnchorStore.name || mapAnchorStore.displayName;
    const normalizedName = typeof rawName === 'string' ? rawName.trim() : '';
    return {
      latitude: mapAnchorStore.latitude,
      longitude: mapAnchorStore.longitude,
      locationName: normalizedName,
      radius: mapAnchorStore.radius
    };
  }

  let selectedLocation = $state.raw<SelectedLocation>(getDefaultSelectedLocation());

  function resetForm() {
    name = '';
    description = '';
    tagsInput = '';
    joinMethod = 'open';
    password = '';
    maxMembers = '';
    isSearchable = true;
    selectedLocation = getDefaultSelectedLocation();
    touched = {};
  }

  async function handleSubmit() {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({});
      open = false;
      onOpenChange?.(false);
      return;
    }

    const trimmedName = typeof name === 'string' ? name.trim() : '';
    const trimmedDescription = typeof description === 'string' ? description.trim() : '';
    const trimmedPassword = typeof password === 'string' ? password.trim() : '';
    const rawLocationName = selectedLocation.locationName;
    const locationName = typeof rawLocationName === 'string' ? rawLocationName.trim() : '';
    const rawMaxMembers = typeof maxMembers === 'string' ? maxMembers.trim() : '';
    const parsedMaxMembers = rawMaxMembers ? Number.parseInt(rawMaxMembers, 10) : undefined;

    touched = { name: true, description: true, password: true, maxMembers: true };

    if (nameError || groupDescriptionError || groupPasswordError || maxMembersError) {
      return;
    }
    if (tags.length > GROUP_TAG_MAX) {
      toast.error(`标签最多 ${GROUP_TAG_MAX} 个`);
      return;
    }
    if (tags.some((tag) => tag.length < 1 || tag.length > TAG_MAX_LENGTH)) {
      toast.error(`标签长度需在 1-${TAG_MAX_LENGTH} 字符之间`);
      return;
    }
    if (!locationName) {
      toast.error('请选择群组位置');
      return;
    }
    if (locationName.length > LOCATION_NAME_MAX) {
      toast.error(`位置名称最多 ${LOCATION_NAME_MAX} 字符`);
      return;
    }

    isSubmitting = true;
    try {
      const payload: {
        name: string;
        location: { point: { latitude: number; longitude: number }; location_name: string };
        description?: string;
        tags?: string[];
        password?: string;
        join_method?: JoinMethod;
        is_searchable?: boolean;
        max_members?: number;
      } = {
        name: trimmedName,
        location: {
          point: {
            latitude: selectedLocation.latitude,
            longitude: selectedLocation.longitude
          },
          location_name: locationName
        },
        join_method: joinMethod,
        is_searchable: isSearchable
      };

      if (trimmedDescription) payload.description = trimmedDescription;
      if (tags.length > 0) payload.tags = tags;
      if (trimmedPassword) payload.password = trimmedPassword;
      if (parsedMaxMembers !== undefined) payload.max_members = parsedMaxMembers;

      const response = await groupsService.create(payload);
      if (isSuccess(response)) {
        toast.success('群组已创建');
        resetForm();
        open = false;
        onOpenChange?.(false);
        onCreated?.();
      } else {
        showApiError(toast, response, '创建失败');
      }
    } catch {
      toast.error('创建失败');
    } finally {
      isSubmitting = false;
    }
  }

  $effect(() => {
    if (!open) {
      resetForm();
    }
  });
</script>

<Dialog.Root
  bind:open
  onOpenChange={(value) => {
    open = value;
    onOpenChange?.(value);
  }}
>
  <Dialog.Content
    class="flex max-h-[85vh] flex-col sm:max-w-2xl"
    style="background: var(--surface-bg); border: 1px solid var(--border-theme); box-shadow: var(--shadow-lg);"
  >
    <Dialog.Header>
      <Dialog.Title class="flex items-center gap-3">
        <div
          class="flex items-center justify-center rounded-full"
          style="width: 40px; height: 40px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
        >
          <UsersRound style="width: 20px; height: 20px; color: var(--brand);" />
        </div>
        <div>
          <div style="color: var(--text-primary); font-weight: 600;">创建群组</div>
          <div style="font-size: 13px; font-weight: 400; color: var(--text-secondary);">
            邀请伙伴一起交流
          </div>
        </div>
      </Dialog.Title>
    </Dialog.Header>

    <div class="min-h-0 flex-1 space-y-4 overflow-y-auto px-1 py-4">
      <!-- 群组名称 -->
      <div
        class="rounded-[var(--radius-lg)] p-4 space-y-2"
        style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
      >
        <label for="group-name" style="font-size: 12px; color: var(--text-primary);">群组名称</label
        >
        <input
          id="group-name"
          class="field-inset rounded-[var(--radius-md)] px-4 py-2.5 outline-none transition-all duration-150 w-full"
          class:field-error={nameError}
          style="background: var(--surface-bg); color: var(--text-primary); font-size: 13px;"
          placeholder="例如：城市漫步俱乐部"
          bind:value={name}
          onblur={() => markTouched('name')}
        />
        <div class="flex items-center justify-between">
          {#if nameError}
            <p style="font-size: 12px; color: var(--error);">{nameError}</p>
          {:else}
            <span></span>
          {/if}
          <span style="font-size: 12px; color: var(--text-secondary);"
            >{name.length}/{GROUP_NAME_MAX}</span
          >
        </div>
      </div>

      <!-- 群组介绍 -->
      <div
        class="rounded-[var(--radius-lg)] p-4 space-y-2"
        style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
      >
        <label for="group-desc" style="font-size: 12px; color: var(--text-primary);">群组介绍</label
        >
        <textarea
          id="group-desc"
          placeholder="简单描述群组用途与氛围"
          class="field-inset rounded-[var(--radius-md)] px-4 py-2.5 outline-none transition-all duration-150 w-full resize-none"
          class:field-error={groupDescriptionError}
          style="background: var(--surface-bg); color: var(--text-primary); font-size: 13px; min-height: 100px;"
          bind:value={description}
          onblur={() => markTouched('description')}
        ></textarea>
        <div class="flex items-center justify-between">
          {#if groupDescriptionError}
            <p style="font-size: 12px; color: var(--error);">{groupDescriptionError}</p>
          {:else}
            <span></span>
          {/if}
          <span style="font-size: 12px; color: var(--text-secondary);"
            >{description.length}/{GROUP_DESCRIPTION_MAX}</span
          >
        </div>
      </div>

      <!-- 群组标签 -->
      <div
        class="rounded-[var(--radius-lg)] p-4 space-y-2"
        style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
      >
        <label for="group-tags" style="font-size: 12px; color: var(--text-primary);">群组标签</label
        >
        <input
          id="group-tags"
          class="field-inset rounded-[var(--radius-md)] px-4 py-2.5 outline-none transition-all duration-150 w-full"
          style="background: var(--surface-bg); color: var(--text-primary); font-size: 13px;"
          placeholder="兴趣、城市、活动"
          bind:value={tagsInput}
        />
        {#if tags.length > 0}
          <div class="flex flex-wrap gap-2">
            {#each tags as tag (tag)}
              <span
                class="rounded-full px-2.5 py-0.5"
                style="font-size: 11px; background: var(--brand); color: var(--text-on-accent);"
                >#{tag}</span
              >
            {/each}
          </div>
        {/if}
      </div>

      <!-- 加入方式 & 口令 -->
      <div
        class="grid gap-4 rounded-[var(--radius-lg)] p-4 sm:grid-cols-2"
        style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
      >
        <div class="space-y-2">
          <label for="join-method" style="font-size: 12px; color: var(--text-primary);"
            >加入方式</label
          >
          <select
            id="join-method"
            class="field-inset rounded-[var(--radius-md)] px-4 py-2.5 outline-none transition-all duration-150 w-full"
            style="background: var(--surface-bg); color: var(--text-primary); font-size: 13px;"
            bind:value={joinMethod}
          >
            <option value="open">自由加入</option>
            <option value="approval">需要审批</option>
            <option value="invite_only">仅邀请</option>
          </select>
        </div>
        <div class="space-y-2">
          <label for="group-password" style="font-size: 12px; color: var(--text-primary);"
            >入群口令 (可选)</label
          >
          <input
            id="group-password"
            class="field-inset rounded-[var(--radius-md)] px-4 py-2.5 outline-none transition-all duration-150 w-full"
            class:field-error={groupPasswordError}
            style="background: var(--surface-bg); color: var(--text-primary); font-size: 13px;"
            type="password"
            placeholder="设置后需口令加入"
            bind:value={password}
            onblur={() => markTouched('password')}
          />
          {#if groupPasswordError}
            <p style="font-size: 12px; color: var(--error);">{groupPasswordError}</p>
          {/if}
        </div>
      </div>

      <!-- 最大成员数 & 搜索 -->
      <div
        class="grid gap-4 rounded-[var(--radius-lg)] p-4 sm:grid-cols-2"
        style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
      >
        <div class="space-y-2">
          <label for="max-members" style="font-size: 12px; color: var(--text-primary);"
            >最大成员数 (可选)</label
          >
          <input
            id="max-members"
            class="field-inset rounded-[var(--radius-md)] px-4 py-2.5 outline-none transition-all duration-150 w-full"
            class:field-error={maxMembersError}
            style="background: var(--surface-bg); color: var(--text-primary); font-size: 13px;"
            type="number"
            min="2"
            max="2000"
            placeholder="例如 200"
            bind:value={maxMembers}
            onblur={() => markTouched('maxMembers')}
          />
          {#if maxMembersError}
            <p style="font-size: 12px; color: var(--error);">{maxMembersError}</p>
          {/if}
        </div>
        <div
          class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-2.5"
          style="background: var(--surface-bg); border: 1px solid var(--border-theme); font-size: 13px; color: var(--text-primary);"
        >
          <span>允许被搜索</span>
          <input type="checkbox" class="h-4 w-4" bind:checked={isSearchable} />
        </div>
      </div>

      <AnchorSummaryCard
        label="群组位置"
        description="影响附近推荐与搜索，可单独选点且不修改浏览锚点"
        overrideName={selectedLocation.locationName}
        onAdjust={() => (showLocationPicker = true)}
      />
    </div>

    <Dialog.Footer class="pt-2 flex gap-3 justify-end">
      <button
        class="rounded-full px-5 py-2 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="background: var(--surface-bg); color: var(--text-primary); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme); font-size: 13px;"
        onclick={() => {
          open = false;
          onOpenChange?.(false);
        }}
      >
        取消
      </button>
      <button
        class="rounded-full px-5 py-2 outline-none transition-all duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.97]"
        style="background: var(--brand); color: var(--text-on-accent); font-size: 13px; box-shadow: var(--shadow-brand-btn);"
        onclick={handleSubmit}
        disabled={isSubmitting}
      >
        {#if isSubmitting}
          <Loader2 class="mr-2 inline-block" style="width: 16px; height: 16px;" />
        {/if}
        创建群组
      </button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<LocationPickerModal
  bind:open={showLocationPicker}
  mode="location_pick"
  initialLocation={{
    latitude: selectedLocation.latitude,
    longitude: selectedLocation.longitude,
    locationName: selectedLocation.locationName,
    radius: selectedLocation.radius
  }}
  onConfirm={(value) => {
    selectedLocation = {
      latitude: value.latitude,
      longitude: value.longitude,
      locationName: value.locationName,
      radius: value.radius
    };
  }}
/>
