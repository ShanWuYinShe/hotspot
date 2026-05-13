<script lang="ts">
  import * as Dialog from '$lib/components/ui/dialog';
  import { momentsService, mediaService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { parseTags } from '$lib/utils/format';
  import { validateField } from '$lib/utils/form-validation';
  import type { FieldRule } from '$lib/utils/form-validation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import LocationPickerModal from './location-picker-modal.svelte';
  import AnchorSummaryCard from '$lib/components/common/anchor-summary-card.svelte';
  import type { PrivacyLevel, MomentItem } from '$lib/types/api';
  import { untrack } from 'svelte';
  import ImagePlus from 'lucide-svelte/icons/image-plus';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import Send from 'lucide-svelte/icons/send';
  import X from 'lucide-svelte/icons/x';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Smile from 'lucide-svelte/icons/smile';
  import Hash from 'lucide-svelte/icons/hash';

  interface Props {
    open: boolean;
    onOpenChange?: (open: boolean) => void;
    onCreated?: () => void;
    editMoment?: MomentItem;
    onEdited?: () => void;
  }

  interface SelectedLocation {
    latitude: number;
    longitude: number;
    locationName: string;
    radius: number;
  }

  let { open = $bindable(), onOpenChange, onCreated, editMoment, onEdited }: Props = $props();

  const isEditMode = $derived(editMoment !== undefined);
  const dialogTitle = $derived(isEditMode ? '编辑动态' : '发布动态');
  const dialogSubtitle = $derived(isEditMode ? '修改你的生活瞬间' : '分享你此刻的生活瞬间');

  const VIBE_TAGS = ['日常', '探店', '美食', '旅行', '户外', '城市'];

  let description = $state('');
  let tagsInput = $state('');
  let privacy = $state<PrivacyLevel>('public');
  let isSubmitting = $state(false);
  let isUploadingImages = $state(false);
  let touched = $state<Record<string, boolean>>({});
  let showLocationPicker = $state(false);
  let selectedVibes = $state<Set<string>>(new Set());
  let selectedImages = $state.raw<
    {
      file: File;
      previewUrl: string;
      progress: number;
      uploadedId?: string;
      isUploading: boolean;
    }[]
  >([]);

  const MAX_IMAGES = 9;
  const DESCRIPTION_MAX = 2000;

  const tags = $derived(parseTags(tagsInput));

  const canPost = $derived(description.trim().length > 0);

  const descriptionRules: FieldRule[] = [
    { required: true, requiredMessage: '请输入动态内容' },
    { maxLength: DESCRIPTION_MAX, maxLengthMessage: `动态内容最多 ${DESCRIPTION_MAX} 字符` }
  ];
  let descriptionError = $derived(
    touched['description'] ? validateField(description, descriptionRules) : undefined
  );

  function toggleVibe(tag: string) {
    const next = new Set(selectedVibes);
    if (next.has(tag)) {
      next.delete(tag);
    } else {
      next.add(tag);
    }
    selectedVibes = next;
  }

  function markTouched(field: string) {
    touched = { ...touched, [field]: true };
  }

  function getDefaultSelectedLocation(): SelectedLocation {
    return {
      latitude: mapAnchorStore.latitude,
      longitude: mapAnchorStore.longitude,
      locationName: (mapAnchorStore.name || mapAnchorStore.displayName).trim(),
      radius: mapAnchorStore.radius
    };
  }

  let selectedLocation = $state.raw<SelectedLocation>(getDefaultSelectedLocation());

  function resetForm() {
    if (!isEditMode) {
      description = '';
      tagsInput = '';
      privacy = 'public';
    }
    const images = untrack(() => selectedImages);
    images.forEach((image) => URL.revokeObjectURL(image.previewUrl));
    selectedImages = [];
    selectedVibes = new Set();
    if (!isEditMode) {
      selectedLocation = getDefaultSelectedLocation();
    }
    touched = {};
  }

  function handleImageChange(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const files = Array.from(input.files ?? []);
    if (files.length === 0) return;

    const remaining = MAX_IMAGES - selectedImages.length;
    const nextFiles = files.slice(0, remaining);
    if (nextFiles.length < files.length) {
      toast.error(`最多上传 ${MAX_IMAGES} 张图片`);
    }

    const nextImages = nextFiles.map((file) => ({
      file,
      previewUrl: URL.createObjectURL(file),
      progress: 0,
      isUploading: false
    }));
    selectedImages = [...selectedImages, ...nextImages];
    input.value = '';
  }

  function removeImage(index: number) {
    const target = selectedImages[index];
    if (target) {
      URL.revokeObjectURL(target.previewUrl);
    }
    selectedImages = selectedImages.filter((_, i) => i !== index);
  }

  async function uploadImages(): Promise<string[]> {
    if (selectedImages.length === 0) return [];
    isUploadingImages = true;
    const uploadedIds: string[] = [];

    for (let i = 0; i < selectedImages.length; i += 1) {
      const image = selectedImages[i]!;
      if (image.uploadedId) {
        uploadedIds.push(image.uploadedId);
        continue;
      }

      selectedImages = selectedImages.map((item, idx) =>
        idx === i ? { ...item, isUploading: true, progress: 0 } : item
      );

      try {
        const uploaded = await mediaService.uploadFile(image.file, 'moment_image', (progress) => {
          selectedImages = selectedImages.map((item, idx) =>
            idx === i ? { ...item, progress } : item
          );
        });
        selectedImages = selectedImages.map((item, idx) =>
          idx === i
            ? { ...item, uploadedId: uploaded.file_id, isUploading: false, progress: 100 }
            : item
        );
        uploadedIds.push(uploaded.file_id);
      } catch {
        selectedImages = selectedImages.map((item, idx) =>
          idx === i ? { ...item, isUploading: false } : item
        );
        isUploadingImages = false;
        throw new Error('图片上传失败');
      }
    }

    isUploadingImages = false;
    return uploadedIds;
  }

  async function handleSubmit() {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({});
      open = false;
      onOpenChange?.(false);
      return;
    }

    touched = { description: true };
    if (descriptionError) return;

    isSubmitting = true;
    try {
      const imageIds = await uploadImages();
      const locationName = selectedLocation.locationName.trim();
      if (!locationName) {
        toast.error('请选择发布位置');
        return;
      }

      const allTags = [...tags, ...Array.from(selectedVibes)];

      if (isEditMode && editMoment) {
        const response = await momentsService.update({
          moment_id: editMoment.moment_id,
          description: description.trim(),
          location: {
            point: {
              latitude: selectedLocation.latitude,
              longitude: selectedLocation.longitude
            },
            location_name: locationName
          },
          image_file_ids: imageIds.length > 0 ? imageIds : undefined,
          tags: allTags.length > 0 ? allTags : undefined,
          privacy_level: privacy
        });
        if (isSuccess(response)) {
          toast.success('动态已更新');
          open = false;
          onOpenChange?.(false);
          onEdited?.();
        } else {
          toast.error(response.message || '更新失败');
        }
      } else {
        const response = await momentsService.create({
          description: description.trim(),
          location: {
            point: {
              latitude: selectedLocation.latitude,
              longitude: selectedLocation.longitude
            },
            location_name: locationName
          },
          image_file_ids: imageIds.length > 0 ? imageIds : undefined,
          tags: allTags.length > 0 ? allTags : undefined,
          privacy_level: privacy
        });

        if (isSuccess(response)) {
          toast.success('动态已发布');
          resetForm();
          open = false;
          onOpenChange?.(false);
          onCreated?.();
        } else {
          toast.error(response.message || '发布失败');
        }
      }
    } catch (error) {
      if (error instanceof Error && error.message === '图片上传失败') {
        toast.error('图片上传失败，请重试');
      } else {
        toast.error(isEditMode ? '更新失败' : '发布失败');
      }
    } finally {
      isSubmitting = false;
    }
  }

  $effect(() => {
    if (!open) {
      resetForm();
    }
  });

  $effect(() => {
    if (open && editMoment) {
      description = editMoment.description;
      tagsInput = editMoment.tags.join(' ');
      privacy = editMoment.privacy_level ?? 'public';
      selectedLocation = {
        latitude: editMoment.location.point.latitude,
        longitude: editMoment.location.point.longitude,
        locationName: editMoment.location.location_name,
        radius: mapAnchorStore.radius
      };
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
    style="background: var(--surface-bg); border: 1px solid var(--border-theme); box-shadow: var(--shadow-lg); border-radius: var(--radius-lg);"
  >
    <!-- Header: drag handle + title + close -->
    <div class="flex-shrink-0 px-5 pt-4 pb-3" style="border-bottom: 1px solid var(--border-theme);">
      <div class="flex justify-center mb-4">
        <div
          class="rounded-full"
          style="width: 36px; height: 4px; background: var(--border-active);"
        ></div>
      </div>
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-[20px] tracking-tight" style="color: var(--text-primary);">
            {dialogTitle}
          </h1>
          <p class="text-[12px] mt-0.5" style="color: var(--text-secondary);">
            {dialogSubtitle}
          </p>
        </div>
        <button
          class="flex items-center justify-center rounded-full outline-none"
          style="width: 38px; height: 38px; background: var(--surface-bg); box-shadow: 5px 5px 10px var(--shadow-dark), -5px -5px 10px var(--shadow-light); border: 1px solid var(--border-theme);"
          onclick={() => {
            open = false;
            onOpenChange?.(false);
          }}
          aria-label="关闭"
        >
          <X style="width: 16px; height: 16px; color: var(--text-secondary);" strokeWidth={1.75} />
        </button>
      </div>
    </div>

    <!-- Scrollable Body -->
    <div class="flex-1 overflow-y-auto no-scrollbar px-5 py-4 flex flex-col gap-4 min-h-0">
      <!-- Author Row -->
      <div class="flex items-center gap-3">
        <div
          class="rounded-full overflow-hidden flex-shrink-0"
          style="width: 40px; height: 40px; border: 1.5px solid var(--border-theme); box-shadow: 4px 4px 8px var(--shadow-dark), -4px -4px 8px var(--shadow-light);"
        >
          <img
            src={authStore.currentAccount?.avatar_url || ''}
            alt="You"
            class="w-full h-full object-cover"
          />
        </div>
        <div>
          <p class="text-[13px]" style="color: var(--text-primary);">
            {authStore.currentAccount?.nickname || '我'}
          </p>
          <div class="flex items-center gap-1">
            <MapPin style="width: 11px; height: 11px; color: var(--brand);" strokeWidth={1.5} />
            <span class="text-[11px]" style="color: var(--text-secondary);">
              {selectedLocation.locationName || '定位中...'}
            </span>
          </div>
        </div>
      </div>

      <!-- Text Area -->
      <div
        class="rounded-[22px] p-4"
        style="box-shadow: inset 6px 6px 12px var(--shadow-dark), inset -6px -6px 12px var(--shadow-light); border: 1px solid {descriptionError
          ? 'var(--error)'
          : 'var(--border-active)'}; min-height: 130px; display: flex; flex-direction: column;"
      >
        <textarea
          id="moment-description"
          placeholder="写点什么吧..."
          class="w-full flex-1 bg-transparent outline-none resize-none text-[13px]"
          style="color: var(--text-primary); min-height: 90px; line-height: 1.6;"
          bind:value={description}
          onblur={() => markTouched('description')}
        ></textarea>
        <div class="flex justify-between items-center mt-2">
          <div class="flex gap-2">
            <button
              class="flex items-center justify-center rounded-full outline-none"
              style="width: 30px; height: 30px; background: var(--surface-bg); box-shadow: 3px 3px 6px var(--shadow-dark), -3px -3px 6px var(--shadow-light); border: 1px solid var(--border-theme); color: var(--text-secondary);"
              aria-label="表情"
            >
              <Smile style="width: 15px; height: 15px;" strokeWidth={1.5} />
            </button>
            <button
              class="flex items-center justify-center rounded-full outline-none"
              style="width: 30px; height: 30px; background: var(--surface-bg); box-shadow: 3px 3px 6px var(--shadow-dark), -3px -3px 6px var(--shadow-light); border: 1px solid var(--border-theme); color: var(--text-secondary);"
              aria-label="标签"
            >
              <Hash style="width: 15px; height: 15px;" strokeWidth={1.5} />
            </button>
          </div>
          <span
            class="text-[11px]"
            style="color: {description.length > DESCRIPTION_MAX * 0.9
              ? 'var(--brand)'
              : 'var(--text-secondary)'};"
          >
            {description.length}/{DESCRIPTION_MAX}
          </span>
        </div>
        {#if descriptionError}
          <p class="text-[11px] mt-1" style="color: var(--error);">{descriptionError}</p>
        {/if}
      </div>

      <!-- Tags input (inline with textarea area) -->
      {#if tags.length > 0}
        <div class="flex flex-wrap gap-2">
          {#each tags as tag (tag)}
            <span
              class="rounded-full px-2.5 py-0.5 text-[11px]"
              style="background: var(--brand); color: var(--text-on-accent);">#{tag}</span
            >
          {/each}
        </div>
      {/if}

      <!-- Image Upload -->
      {#if selectedImages.length > 0}
        <div class="grid gap-3 sm:grid-cols-3">
          {#each selectedImages as image, index (image.previewUrl)}
            <div
              class="group relative overflow-hidden rounded-[22px]"
              style="height: 120px; box-shadow: 6px 6px 14px var(--shadow-dark), -6px -6px 14px var(--shadow-light); border: 1px solid var(--border-theme);"
            >
              <img src={image.previewUrl} alt="预览图" class="w-full h-full object-cover" />
              {#if image.isUploading}
                <div
                  class="absolute inset-0 flex items-center justify-center"
                  style="background: var(--overlay); font-size: 12px; color: white;"
                >
                  上传中 {Math.round(image.progress)}%
                </div>
              {/if}
              <button
                type="button"
                class="absolute top-2.5 right-2.5 rounded-full flex items-center justify-center outline-none"
                style="width: 26px; height: 26px; background: rgba(0,0,0,0.45); backdrop-filter: blur(4px);"
                onclick={() => removeImage(index)}
              >
                <X style="width: 14px; height: 14px; color: white;" strokeWidth={2} />
              </button>
            </div>
          {/each}

          <!-- Add more button -->
          {#if selectedImages.length < MAX_IMAGES}
            <label
              class="flex flex-col items-center justify-center gap-1.5 rounded-[22px] outline-none cursor-pointer"
              style="height: 120px; box-shadow: inset 4px 4px 8px var(--shadow-dark), inset -4px -4px 8px var(--shadow-light); border: 1.5px dashed var(--border-active);"
            >
              <ImagePlus
                style="width: 22px; height: 22px; color: var(--text-secondary);"
                strokeWidth={1.25}
              />
              <span class="text-[12px]" style="color: var(--text-secondary);">添加更多</span>
              <input
                type="file"
                accept="image/*"
                multiple
                class="hidden"
                onchange={handleImageChange}
              />
            </label>
          {/if}
        </div>
      {:else}
        <label
          class="flex flex-col items-center justify-center gap-1.5 rounded-[22px] outline-none cursor-pointer w-full"
          style="height: 80px; box-shadow: inset 4px 4px 8px var(--shadow-dark), inset -4px -4px 8px var(--shadow-light); border: 1.5px dashed var(--border-active);"
        >
          <ImagePlus
            style="width: 22px; height: 22px; color: var(--text-secondary);"
            strokeWidth={1.25}
          />
          <span class="text-[12px]" style="color: var(--text-secondary);">添加照片</span>
          <input
            type="file"
            accept="image/*"
            multiple
            class="hidden"
            onchange={handleImageChange}
            disabled={selectedImages.length >= MAX_IMAGES}
          />
        </label>
      {/if}

      {#if isEditMode && editMoment?.images && editMoment.images.length > 0}
        <div class="grid gap-3 sm:grid-cols-3">
          {#each editMoment.images as image (image.thumbnail_url ?? image.url)}
            <div
              class="overflow-hidden rounded-[22px]"
              style="border: 1px solid var(--border-theme); background: var(--surface-bg); height: 120px;"
            >
              <img
                src={image.thumbnail_url ?? image.url}
                alt="已有图片"
                class="w-full h-full object-cover"
                style="opacity: 0.75;"
              />
            </div>
          {/each}
        </div>
      {/if}

      <!-- Vibe Tags -->
      <div>
        <p
          class="text-[11px] uppercase tracking-widest mb-2.5"
          style="color: var(--text-secondary);"
        >
          氛围
        </p>
        <div class="flex flex-wrap gap-2">
          {#each VIBE_TAGS as tag (tag)}
            {@const active = selectedVibes.has(tag)}
            <button
              class="px-4 py-1.5 rounded-full text-[12px] outline-none transition-all duration-200 active:scale-[0.92]"
              style={active
                ? 'background: var(--brand); color: white; box-shadow: 0 4px 12px rgba(0,0,0,0.15); border: 1px solid transparent;'
                : 'background: var(--surface-bg); box-shadow: 4px 4px 8px var(--shadow-dark), -4px -4px 8px var(--shadow-light); border: 1px solid var(--border-theme); color: var(--text-secondary);'}
              onclick={() => toggleVibe(tag)}
            >
              {tag}
            </button>
          {/each}
        </div>
      </div>

      <!-- Tags input + Privacy (compact row) -->
      <div class="flex gap-3">
        <input
          placeholder="标签（空格分隔）"
          class="rounded-full px-4 py-2 outline-none flex-1"
          style="background: var(--surface-bg); color: var(--text-primary); box-shadow: inset 3px 3px 6px var(--shadow-dark), inset -3px -3px 6px var(--shadow-light); border: 1px solid var(--border-active); font-size: 12px;"
          bind:value={tagsInput}
        />
        <select
          class="rounded-full px-3 py-2 outline-none"
          style="background: var(--surface-bg); color: var(--text-primary); box-shadow: 4px 4px 8px var(--shadow-dark), -4px -4px 8px var(--shadow-light); border: 1px solid var(--border-theme); font-size: 12px;"
          bind:value={privacy}
        >
          <option value="public">公开</option>
          <option value="private">仅自己</option>
        </select>
      </div>

      <AnchorSummaryCard
        label="发布位置"
        description="发布将使用该位置，可单独选点且不修改浏览锚点"
        overrideName={selectedLocation.locationName}
        onAdjust={() => (showLocationPicker = true)}
      />

      <!-- Post Button -->
      <button
        class="flex items-center justify-center gap-2 w-full py-3.5 rounded-[22px] text-[14px] outline-none transition-all duration-300 active:scale-[0.97]"
        style={canPost && !isSubmitting
          ? 'background: var(--brand); color: white; box-shadow: 0 8px 24px rgba(0,0,0,0.18); border: 1px solid transparent;'
          : 'background: var(--surface-bg); box-shadow: 4px 4px 8px var(--shadow-dark), -4px -4px 8px var(--shadow-light); border: 1px solid var(--border-theme); color: var(--text-secondary); opacity: 0.6; cursor: not-allowed;'}
        onclick={handleSubmit}
        disabled={!canPost || isSubmitting || isUploadingImages}
      >
        {#if isSubmitting}
          <Loader2 style="width: 16px; height: 16px;" class="inline-block" />
        {:else}
          <Send style="width: 16px; height: 16px;" strokeWidth={1.75} />
        {/if}
        <span>{isEditMode ? '保存' : '发布'}</span>
      </button>

      <div style="height: 8px;"></div>
    </div>
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
