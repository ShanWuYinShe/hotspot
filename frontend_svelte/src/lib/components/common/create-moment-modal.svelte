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
	import Image from 'lucide-svelte/icons/image';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Send from 'lucide-svelte/icons/send';
	import X from 'lucide-svelte/icons/x';

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

	let description = $state('');
	let tagsInput = $state('');
	let privacy = $state<PrivacyLevel>('public');
	let isSubmitting = $state(false);
	let isUploadingImages = $state(false);
	let touched = $state<Record<string, boolean>>({});
	let showLocationPicker = $state(false);
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

	const descriptionRules: FieldRule[] = [
		{ required: true, requiredMessage: '请输入动态内容' },
		{ maxLength: DESCRIPTION_MAX, maxLengthMessage: `动态内容最多 ${DESCRIPTION_MAX} 字符` }
	];
	let descriptionError = $derived(
		touched['description'] ? validateField(description, descriptionRules) : undefined
	);

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
					idx === i ? { ...item, uploadedId: uploaded.file_id, isUploading: false, progress: 100 } : item
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
					tags: tags.length > 0 ? tags : undefined,
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
					tags: tags.length > 0 ? tags : undefined,
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

<Dialog.Root bind:open onOpenChange={(value) => { open = value; onOpenChange?.(value); }}>
	<Dialog.Content class="flex max-h-[85vh] flex-col sm:max-w-2xl" style="background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: var(--neu-shadow-lg); border-radius: var(--neu-radius-lg);">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<div class="flex items-center justify-center rounded-full" style="width: 40px; height: 40px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
					<Send style="width: 20px; height: 20px; color: var(--neu-accent-primary);" />
				</div>
				<div>
					<div style="color: var(--neu-text); font-weight: 600;">{dialogTitle}</div>
					<div style="font-size: 13px; font-weight: 400; color: var(--neu-text-muted);">{dialogSubtitle}</div>
				</div>
			</Dialog.Title>
		</Dialog.Header>

		<div class="flex-1 overflow-y-auto min-h-0 py-4 px-1 space-y-4">
			<!-- 内容 -->
			<div class="space-y-2">
				<label for="moment-description" style="font-size: 12px; color: var(--neu-text);">内容</label>
				<textarea
					id="moment-description"
					placeholder="写点什么吧..."
					class="rounded-[14px] px-4 py-2.5 outline-none w-full resize-none"
					style="background: var(--neu-bg); color: var(--neu-text); box-shadow: var(--neu-shadow-inset); border: 1px solid {descriptionError ? 'var(--neu-error)' : 'var(--neu-border)'}; font-size: 13px; min-height: 120px;"
					bind:value={description}
					onblur={() => markTouched('description')}
				></textarea>
				<div class="flex items-center justify-between">
					{#if descriptionError}
						<p style="font-size: 12px; color: var(--neu-error);">{descriptionError}</p>
					{:else}
						<span></span>
					{/if}
					<span style="font-size: 12px; color: var(--neu-text-muted);">{description.length}/{DESCRIPTION_MAX}</span>
				</div>
			</div>

			<!-- 标签 -->
			<div class="space-y-2">
				<label for="moment-tags" style="font-size: 12px; color: var(--neu-text);">标签</label>
				<input
					id="moment-tags"
					placeholder="用空格或逗号分隔，例如：城市 旅行 生活"
					class="rounded-[14px] px-4 py-2.5 outline-none w-full"
					style="background: var(--neu-bg); color: var(--neu-text); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border); font-size: 13px;"
					bind:value={tagsInput}
				/>
				{#if tags.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each tags as tag (tag)}
							<span class="rounded-full px-2.5 py-0.5" style="font-size: 11px; background: var(--neu-accent-primary); color: var(--neu-active-text);">#{tag}</span>
						{/each}
					</div>
				{/if}
			</div>

			<!-- 图片 -->
			<div class="space-y-2">
				<label style="font-size: 12px; color: var(--neu-text);">图片</label>
				<div class="flex flex-wrap items-center gap-2">
					<label class="inline-flex cursor-pointer items-center gap-2 rounded-[14px] px-4 py-2" style="border: 1px dashed var(--neu-border); font-size: 13px; color: var(--neu-text-muted); background: var(--neu-bg); box-shadow: var(--neu-shadow-md);">
						<Image style="width: 16px; height: 16px;" />
						<span>选择图片</span>
						<input
							type="file"
							accept="image/*"
							multiple
							class="hidden"
							onchange={handleImageChange}
							disabled={selectedImages.length >= MAX_IMAGES}
						/>
					</label>
					<span style="font-size: 12px; color: var(--neu-text-muted);">最多 {MAX_IMAGES} 张</span>
				</div>

				{#if selectedImages.length > 0}
					<div class="grid gap-3 sm:grid-cols-3">
						{#each selectedImages as image, index (image.previewUrl)}
							<div class="group relative overflow-hidden rounded-[12px]" style="border: 1px solid var(--neu-border); background: var(--neu-bg);">
								<img src={image.previewUrl} alt="预览图" class="h-28 w-full object-cover" />
								{#if image.isUploading}
									<div class="absolute inset-0 flex items-center justify-center" style="background: var(--neu-overlay); font-size: 12px; color: white;">
										上传中 {Math.round(image.progress)}%
									</div>
								{/if}
								<button
									type="button"
									class="absolute right-2 top-2 flex h-6 w-6 items-center justify-center rounded-full opacity-0 transition-opacity group-hover:opacity-100"
									style="background: var(--neu-overlay-loading);"
									onclick={() => removeImage(index)}
								>
									<X style="width: 14px; height: 14px;" />
								</button>
							</div>
						{/each}
					</div>
				{/if}

				{#if isEditMode && editMoment?.images && editMoment.images.length > 0}
					<div class="grid gap-3 sm:grid-cols-3">
						{#each editMoment.images as image (image.thumbnail_url ?? image.url)}
							<div class="overflow-hidden rounded-[12px]" style="border: 1px solid var(--neu-border); background: var(--neu-bg);">
								<img src={image.thumbnail_url ?? image.url} alt="已有图片" class="h-28 w-full object-cover" style="opacity: 0.75;" />
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- 可见范围 -->
			<div class="space-y-2">
				<label for="moment-privacy" style="font-size: 12px; color: var(--neu-text);">可见范围</label>
				<select
					id="moment-privacy"
					class="rounded-[14px] px-4 py-2.5 outline-none w-full"
					style="background: var(--neu-bg); color: var(--neu-text); box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border); font-size: 13px;"
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
		</div>

		<Dialog.Footer class="pt-2 flex gap-3 justify-end">
			<button class="rounded-full px-5 py-2 outline-none" style="background: var(--neu-bg); color: var(--neu-text); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); font-size: 13px;" onclick={() => { open = false; onOpenChange?.(false); }}>
				取消
			</button>
			<button class="rounded-full px-5 py-2 outline-none" style="background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 13px; box-shadow: 0 3px 10px var(--neu-glow);" onclick={handleSubmit} disabled={isSubmitting || isUploadingImages}>
				{#if isSubmitting}
					<Loader2 class="mr-2 inline-block" style="width: 16px; height: 16px;" />
				{/if}
				{isEditMode ? '保存' : '发布'}
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
