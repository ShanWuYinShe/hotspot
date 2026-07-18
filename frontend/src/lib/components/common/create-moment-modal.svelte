<script lang="ts">
  import * as Dialog from '$lib/components/ui/dialog';
  import { momentsService, mediaService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { parseTags } from '$lib/utils/format';
  import { validateField } from '$lib/utils/form-validation';
  import type { FieldRule } from '$lib/utils/form-validation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { configStore } from '$lib/stores/config.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { t } from '$lib/i18n';
  import LocationPickerModal from './location-picker-modal.svelte';
  import AnchorSummaryCard from '$lib/components/common/anchor-summary-card.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';
  import SmartImage from '$lib/components/common/smart-image.svelte';
  import type { PrivacyLevel, MomentItem } from '$lib/types/api';
  import { untrack } from 'svelte';
  import ImagePlus from 'lucide-svelte/icons/image-plus';
  import Spinner from '$lib/components/common/spinner.svelte';
  import Send from 'lucide-svelte/icons/send';
  import X from 'lucide-svelte/icons/x';
  import MapPin from 'lucide-svelte/icons/map-pin';

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
  const dialogTitle = $derived(isEditMode ? t('createMoment.titleEdit') : t('createMoment.titleCreate'));
  const dialogSubtitle = $derived(isEditMode ? t('createMoment.subtitleEdit') : t('createMoment.subtitleCreate'));

  const VIBE_TAGS = $derived([
    t('createMoment.vibeDaily'),
    t('createMoment.vibeStore'),
    t('createMoment.vibeFood'),
    t('createMoment.vibeTravel'),
    t('createMoment.vibeOutdoor'),
    t('createMoment.vibeCity')
  ]);

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

  // 上传/输入限制：优先使用服务端 ClientConfig 派生值，未加载时回退到默认
  const MAX_IMAGES = $derived(configStore.maxMomentImages);
  const DESCRIPTION_MAX = $derived(configStore.momentDescriptionMaxLength);
  const MAX_TAGS = $derived(configStore.maxMomentTags);

  const tags = $derived(parseTags(tagsInput));

  const canPost = $derived(description.trim().length > 0);

  const descriptionRules = $derived<FieldRule[]>([
    { required: true, requiredMessage: t('createMoment.descriptionRequired') },
    { maxLength: DESCRIPTION_MAX, maxLengthMessage: t('createMoment.descriptionMaxError', { count: DESCRIPTION_MAX }) }
  ]);
  let descriptionError = $derived(
    touched['description'] ? validateField(description, descriptionRules) : undefined
  );

  function toggleVibe(tag: string) {
    const next = new Set(selectedVibes);
    if (next.has(tag)) {
      next.delete(tag);
    } else {
      // 标签总数（解析标签 + 氛围标签）不超过服务端配置上限
      if (tags.length + next.size >= MAX_TAGS) {
        toast.error(t('createMoment.tagsMaxError', { count: MAX_TAGS }));
        return;
      }
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
      toast.error(t('createMoment.imagesMaxError', { count: MAX_IMAGES }));
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

    // 并行上传：每张图片独立走 presign+PUT+complete，互不阻塞
    const tasks = selectedImages.map((image, i) => {
      if (image.uploadedId) {
        return Promise.resolve({ index: i, fileId: image.uploadedId });
      }

      selectedImages = selectedImages.map((item, idx) =>
        idx === i ? { ...item, isUploading: true, progress: 0 } : item
      );

      return mediaService
        .uploadFile(image.file, 'moment_image', (progress) => {
          selectedImages = selectedImages.map((item, idx) =>
            idx === i ? { ...item, progress } : item
          );
        })
        .then((uploaded) => {
          selectedImages = selectedImages.map((item, idx) =>
            idx === i
              ? { ...item, uploadedId: uploaded.file_id, isUploading: false, progress: 100 }
              : item
          );
          return { index: i, fileId: uploaded.file_id };
        })
        .catch((err) => {
          selectedImages = selectedImages.map((item, idx) =>
            idx === i ? { ...item, isUploading: false } : item
          );
          throw err;
        });
    });

    try {
      const results = await Promise.all(tasks);
      // 按原始顺序返回 file_id
      const uploadedIds = results.sort((a, b) => a.index - b.index).map((r) => r.fileId);
      isUploadingImages = false;
      return uploadedIds;
    } catch {
      isUploadingImages = false;
      throw new Error(t('createMoment.imageUploadFailed'));
    }
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
        toast.error(t('createMoment.locationRequired'));
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
          toast.success(t('createMoment.momentUpdated'));
          open = false;
          onOpenChange?.(false);
          onEdited?.();
        } else {
          toast.error(response.message || t('createMoment.updateFailed'));
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
          toast.success(t('createMoment.momentPublished'));
          resetForm();
          open = false;
          onOpenChange?.(false);
          onCreated?.();
        } else {
          toast.error(response.message || t('createMoment.publishFailed'));
        }
      }
    } catch (error) {
      if (error instanceof Error && error.message === t('createMoment.imageUploadFailed')) {
        toast.error(t('createMoment.imageUploadRetry'));
      } else {
        toast.error(isEditMode ? t('createMoment.updateFailed') : t('createMoment.publishFailed'));
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
    hideClose
    class="compose compose-dialog flex max-h-[85dvh] flex-col"
  >
    <!-- Header: title + close (fixed, no drag handle) -->
    <div class="compose__header">
      <div class="compose__titles">
        <h1 class="compose__title">{dialogTitle}</h1>
        <p class="compose__subtitle">{dialogSubtitle}</p>
      </div>
      <button class="compose__close" onclick={() => { open = false; onOpenChange?.(false); }} aria-label={t('createMoment.closeAria')}>
        <X size={18} color="var(--text-secondary)" strokeWidth={1.75} />
      </button>
    </div>

    <!-- Scrollable Body -->
    <div class="compose__body no-scrollbar">
      <!-- Author + location context -->
      <div class="compose__author">
        <Avatar
          src={authStore.currentAccount?.avatar_url}
          name={authStore.currentAccount?.nickname}
          alt="You"
          size={36}
        />
        <div class="compose__author-info">
          <span class="compose__author-name">{authStore.currentAccount?.nickname || t('createMoment.authorFallback')}</span>
          <span class="compose__author-loc">
            <MapPin size={11} color="var(--brand)" strokeWidth={1.5} />
            {selectedLocation.locationName || t('createMoment.locating')}
          </span>
        </div>
      </div>

      <!-- Textarea -->
      <div class="compose__textarea-wrap input-surface" class:compose__textarea-wrap--error={descriptionError}>
        <textarea
          id="moment-description"
          name="moment-description"
          placeholder={t('createMoment.placeholder')}
          class="compose__textarea"
          bind:value={description}
          onblur={() => markTouched('description')}
        ></textarea>
        <div class="compose__textarea-footer">
          {#if descriptionError}
            <span class="compose__error">{descriptionError}</span>
          {/if}
          <span class="compose__counter" class:compose__counter--warn={description.length > DESCRIPTION_MAX * 0.9}>
            {description.length}/{DESCRIPTION_MAX}
          </span>
        </div>
      </div>

      <!-- Parsed tag chips -->
      {#if tags.length > 0}
        <div class="compose__tag-chips">
          {#each tags as tag (tag)}
            <span class="compose__tag-chip">#{tag}</span>
          {/each}
        </div>
      {/if}

      <!-- Image upload grid -->
      <div class="compose__media">
        <div class="compose__section-label">
          <span>{t('createMoment.photos')}</span>
          <span class="compose__section-count">{selectedImages.length}/{MAX_IMAGES}</span>
        </div>
        {#if selectedImages.length > 0}
          <div class="compose__image-grid">
            {#each selectedImages as image, index (image.previewUrl)}
              <div class="compose__image-tile surface-flat">
                <img src={image.previewUrl} alt={t('createMoment.previewAlt')} class="compose__image-img" />
                {#if image.isUploading}
                  <div class="compose__image-overlay">
                    {t('createMoment.uploading', { percent: Math.round(image.progress) })}
                  </div>
                {/if}
                <button type="button" class="compose__image-remove" onclick={() => removeImage(index)} aria-label={t('createMoment.removeImageAria')}>
                  <X size={14} color="var(--text-on-overlay)" strokeWidth={2} />
                </button>
              </div>
            {/each}
            {#if selectedImages.length < MAX_IMAGES}
              <label class="compose__image-add surface-flat">
                <ImagePlus size={20} color="var(--text-secondary)" strokeWidth={1.25} />
                <input type="file" accept="image/*" multiple class="hidden" onchange={handleImageChange} />
              </label>
            {/if}
          </div>
        {:else}
          <label class="compose__image-empty surface-flat">
            <ImagePlus size={22} color="var(--text-secondary)" strokeWidth={1.25} />
            <span>{t('createMoment.addPhotos')}</span>
            <input type="file" accept="image/*" multiple class="hidden" onchange={handleImageChange} disabled={selectedImages.length >= MAX_IMAGES} />
          </label>
        {/if}
        {#if isEditMode && editMoment?.images && editMoment.images.length > 0}
          <div class="compose__image-grid compose__image-grid--existing">
            {#each editMoment.images as image (image.thumbnail_url ?? image.url)}
              <div class="compose__image-tile compose__image-tile--existing surface-flat">
                <SmartImage src={image.url} thumbnail={image.thumbnail_url} alt={t('createMoment.existingImageAlt')} width={image.width} height={image.height} class="compose__image-img" />
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Vibe tags -->
      <div class="compose__vibes">
        <span class="compose__section-label"><span>{t('createMoment.vibe')}</span></span>
        <div class="compose__vibe-list">
          {#each VIBE_TAGS as tag (tag)}
            <button class="filter-chip" class:filter-chip--active={selectedVibes.has(tag)} onclick={() => toggleVibe(tag)}>
              {tag}
            </button>
          {/each}
        </div>
      </div>

      <!-- Tags input -->
      <div class="compose__tags">
        <span class="compose__section-label"><span>{t('createMoment.tags')}</span></span>
        <input
          name="moment-tags"
          placeholder={t('createMoment.tagsPlaceholder')}
          class="compose__tags-input input-surface"
          bind:value={tagsInput}
        />
      </div>

      <!-- Location -->
      <AnchorSummaryCard
        label={t('createMoment.publishLocation')}
        description={t('createMoment.publishLocationDesc')}
        overrideName={selectedLocation.locationName}
        onAdjust={() => (showLocationPicker = true)}
      />
    </div>

    <!-- Footer: privacy + submit (fixed, not scrolling) -->
    <div class="compose__footer">
      <select class="compose__privacy" bind:value={privacy}>
        <option value="public">{t('createMoment.privacyPublic')}</option>
        <option value="private">{t('createMoment.privacyPrivate')}</option>
      </select>
      <button
        class="compose__submit"
        class:compose__submit--disabled={!canPost || isSubmitting}
        onclick={handleSubmit}
        disabled={!canPost || isSubmitting || isUploadingImages}
      >
        {#if isSubmitting}
          <Spinner size="md" />
        {:else}
          <Send size={16} strokeWidth={1.75} />
        {/if}
        <span>{isEditMode ? t('createMoment.submitEdit') : t('createMoment.submitCreate')}</span>
      </button>
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

<svelte:head>
  <style>
  /* Injected via <svelte:head> to bypass Svelte's CSS tree-shaking, which
     marks portaled Dialog.Content selectors as "(unused)" and strips them.
     All selectors are qualified under .compose to avoid leaking globally. */

  /* ── Dialog width override (desktop wider than the 32rem default) ── */
  .dialog-responsive-position.compose-dialog {
    max-width: 640px;
  }
  @media (min-width: 1024px) {
    .dialog-responsive-position.compose-dialog {
      max-width: 720px;
    }
  }

  .compose {
    background: var(--surface-bg);
    border: 1px solid var(--border-theme);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-lg);
    color: var(--text-primary);
    gap: 0;
    padding: 0;
  }

  /* ── Header (fixed) ─────────────────────────────────────── */
  .compose .compose__header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: 18px 20px 14px;
    border-bottom: 1px solid var(--border-theme);
    flex-shrink: 0;
  }
  .compose .compose__titles { min-width: 0; }
  .compose .compose__title {
    font-size: 18px;
    font-weight: 600;
    letter-spacing: -0.01em;
    color: var(--text-primary);
    margin: 0;
  }
  .compose .compose__subtitle {
    font-size: 12px;
    color: var(--text-secondary);
    margin: 2px 0 0;
  }
  .compose .compose__close {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: var(--radius-sm);
    border: 1px solid transparent;
    background: transparent;
    cursor: pointer;
    outline: none;
    transition: background 0.16s ease, border-color 0.16s ease;
  }
  .compose .compose__close:hover {
    background: var(--border-theme);
    border-color: var(--border-theme);
  }

  /* ── Body (scrollable) ──────────────────────────────────── */
  .compose .compose__body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 16px 20px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  /* Author row */
  .compose .compose__author {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .compose .compose__author-info {
    display: flex;
    flex-direction: column;
    gap: 1px;
    min-width: 0;
  }
  .compose .compose__author-name {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary);
  }
  .compose .compose__author-loc {
    display: flex;
    align-items: center;
    gap: 3px;
    font-size: 11px;
    color: var(--text-secondary);
  }

  /* Textarea */
  .compose .compose__textarea-wrap {
    border-radius: var(--radius-md);
    padding: 12px 14px;
    min-height: 120px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .compose .compose__textarea-wrap--error {
    border-color: var(--error);
  }
  .compose .compose__textarea {
    flex: 1;
    width: 100%;
    min-height: 80px;
    resize: none;
    border: none;
    outline: none;
    background: transparent;
    color: var(--text-primary);
    font-size: 13px;
    line-height: 1.6;
    font-family: inherit;
  }
  .compose .compose__textarea::placeholder { color: var(--text-secondary); }
  .compose .compose__textarea-footer {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 8px;
  }
  .compose .compose__error {
    font-size: 11px;
    color: var(--error);
    margin-right: auto;
  }
  .compose .compose__counter {
    font-size: 11px;
    color: var(--text-secondary);
    font-variant-numeric: tabular-nums;
  }
  .compose .compose__counter--warn { color: var(--brand); }

  /* Parsed tag chips */
  .compose .compose__tag-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: -8px;
  }
  .compose .compose__tag-chip {
    display: inline-flex;
    padding: 2px 10px;
    font-size: 11px;
    font-weight: 500;
    border-radius: var(--radius-pill);
    background: var(--brand);
    color: var(--text-on-accent);
  }

  /* Section label */
  .compose .compose__section-label {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }
  .compose .compose__section-count {
    text-transform: none;
    letter-spacing: 0;
    font-variant-numeric: tabular-nums;
  }

  /* Image grid */
  .compose .compose__image-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }
  .compose .compose__image-grid--existing { margin-top: 8px; }
  .compose .compose__image-tile {
    position: relative;
    aspect-ratio: 1 / 1;
    overflow: hidden;
    border-radius: var(--radius-sm);
  }
  .compose .compose__image-tile--existing { opacity: 0.75; }
  .compose .compose__image-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  .compose .compose__image-overlay {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--overlay);
    color: var(--text-on-overlay);
    font-size: 11px;
    font-variant-numeric: tabular-nums;
  }
  .compose .compose__image-remove {
    position: absolute;
    top: 6px;
    right: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: var(--radius-pill);
    border: none;
    background: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(4px);
    cursor: pointer;
    outline: none;
  }
  .compose .compose__image-add {
    aspect-ratio: 1 / 1;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-sm);
    border: 1.5px dashed var(--border-active) !important;
    cursor: pointer;
    transition: border-color 0.16s ease;
  }
  .compose .compose__image-add:hover { border-color: var(--brand) !important; }
  .compose .compose__image-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    height: 72px;
    border-radius: var(--radius-md);
    border: 1.5px dashed var(--border-active) !important;
    cursor: pointer;
    color: var(--text-secondary);
    font-size: 12px;
    transition: border-color 0.16s ease;
  }
  .compose .compose__image-empty:hover { border-color: var(--brand) !important; }

  /* Vibe tags */
  .compose .compose__vibes { display: flex; flex-direction: column; }
  .compose .compose__vibe-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  /* Tags input */
  .compose .compose__tags { display: flex; flex-direction: column; }
  .compose .compose__tags-input {
    border-radius: var(--radius-md);
    padding: 8px 12px;
    font-size: 12px;
    color: var(--text-primary);
    outline: none;
    width: 100%;
    font-family: inherit;
  }
  .compose .compose__tags-input::placeholder { color: var(--text-secondary); }

  /* ── Footer (fixed) ─────────────────────────────────────── */
  .compose .compose__footer {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 20px 16px;
    border-top: 1px solid var(--border-theme);
    flex-shrink: 0;
  }
  .compose .compose__privacy {
    padding: 7px 12px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border-theme);
    background: var(--surface-bg);
    color: var(--text-primary);
    font-size: 12px;
    font-family: inherit;
    outline: none;
    cursor: pointer;
  }
  .compose .compose__submit {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 10px 20px;
    border-radius: var(--radius-md);
    border: 1px solid transparent;
    background: var(--brand);
    color: var(--text-on-accent);
    font-size: 14px;
    font-weight: 500;
    font-family: inherit;
    cursor: pointer;
    outline: none;
    box-shadow: var(--shadow-brand-btn);
    transition: transform 0.15s ease, opacity 0.15s ease;
  }
  .compose .compose__submit:active { transform: scale(0.97); }
  .compose .compose__submit--disabled {
    background: var(--surface-bg);
    color: var(--text-secondary);
    border-color: var(--border-theme);
    box-shadow: none;
    opacity: 0.6;
    cursor: not-allowed;
  }

  /* Mobile: tighter padding */
  @media (max-width: 639px) {
    .compose .compose__body { padding: 14px 16px; }
    .compose .compose__header { padding: 14px 16px 12px; }
    .compose .compose__footer { padding: 10px 16px 14px; }
  }
  </style>
</svelte:head>
