<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Label } from '$lib/components/ui/label';
	import { Badge } from '$lib/components/ui/badge';
	import { momentsService, mediaService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { parseTags } from '$lib/utils/format';
	import { authStore } from '$lib/stores/auth.svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import LocationPickerModal from './location-picker-modal.svelte';
	import AnchorSummaryCard from '$lib/components/common/anchor-summary-card.svelte';
	import type { PrivacyLevel } from '$lib/types/api';
	import { untrack } from 'svelte';
	import Image from 'lucide-svelte/icons/image';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Send from 'lucide-svelte/icons/send';
	import X from 'lucide-svelte/icons/x';

	interface Props {
		open: boolean;
		onOpenChange?: (open: boolean) => void;
		onCreated?: () => void;
	}

	interface SelectedLocation {
		latitude: number;
		longitude: number;
		locationName: string;
		radius: number;
	}

	let { open = $bindable(), onOpenChange, onCreated }: Props = $props();

	let description = $state('');
	let tagsInput = $state('');
	let privacy = $state<PrivacyLevel>('public');
	let isSubmitting = $state(false);
	let isUploadingImages = $state(false);
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

	const tags = $derived(parseTags(tagsInput));

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
		description = '';
		tagsInput = '';
		privacy = 'public';
		const images = untrack(() => selectedImages);
		images.forEach((image) => URL.revokeObjectURL(image.previewUrl));
		selectedImages = [];
		selectedLocation = getDefaultSelectedLocation();
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
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			open = false;
			onOpenChange?.(false);
			return;
		}

		if (!description.trim()) {
			toast.error('请输入动态内容');
			return;
		}

		isSubmitting = true;
		try {
			const imageIds = await uploadImages();
			const locationName = selectedLocation.locationName.trim();
			if (!locationName) {
				toast.error('请选择发布位置');
				return;
			}
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
		} catch (error) {
			if (error instanceof Error && error.message === '图片上传失败') {
				toast.error('图片上传失败，请重试');
			} else {
				toast.error('发布失败');
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
</script>

<Dialog.Root bind:open onOpenChange={(value) => { open = value; onOpenChange?.(value); }}>
	<Dialog.Content class="flex max-h-[85vh] flex-col sm:max-w-2xl">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-linear-to-br from-primary/20 to-primary/10">
					<Send class="h-5 w-5 text-primary" />
				</div>
				<div>
					<div>发布动态</div>
					<div class="text-sm font-normal text-muted-foreground">分享你此刻的生活瞬间</div>
				</div>
			</Dialog.Title>
		</Dialog.Header>

		<div class="flex-1 overflow-y-auto min-h-0 py-4 px-1 space-y-4">
			<div class="space-y-2">
				<Label for="moment-description">内容</Label>
				<Textarea
					id="moment-description"
					placeholder="写点什么吧..."
					class="min-h-[120px] resize-none"
					bind:value={description}
				/>
			</div>

			<div class="space-y-2">
				<Label for="moment-tags">标签</Label>
				<Input
					id="moment-tags"
					placeholder="用空格或逗号分隔，例如：城市 旅行 生活"
					bind:value={tagsInput}
				/>
				{#if tags.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each tags as tag (tag)}
							<Badge variant="secondary" class="rounded-2xl text-xs">#{tag}</Badge>
						{/each}
					</div>
				{/if}
			</div>

			<div class="space-y-2">
				<Label>图片</Label>
				<div class="flex flex-wrap items-center gap-2">
					<label class="inline-flex cursor-pointer items-center gap-2 rounded-2xl border border-dashed border-border px-4 py-2 text-sm text-muted-foreground hover:bg-accent">
						<Image class="h-4 w-4" />
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
					<span class="text-xs text-muted-foreground">最多 {MAX_IMAGES} 张</span>
				</div>

				{#if selectedImages.length > 0}
					<div class="grid gap-3 sm:grid-cols-3">
						{#each selectedImages as image, index (image.previewUrl)}
							<div class="group relative overflow-hidden rounded-xl border bg-muted/30">
								<img src={image.previewUrl} alt="预览图" class="h-28 w-full object-cover" />
								{#if image.isUploading}
									<div class="absolute inset-0 flex items-center justify-center bg-background/70 text-xs text-muted-foreground">
										上传中 {Math.round(image.progress)}%
									</div>
								{/if}
								<button
									type="button"
									class="absolute right-2 top-2 rounded-2xl bg-background/80 p-1 opacity-0 shadow transition-opacity group-hover:opacity-100"
									onclick={() => removeImage(index)}
								>
									<X class="h-3.5 w-3.5" />
								</button>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<div class="space-y-2">
				<Label for="moment-privacy">可见范围</Label>
				<select
					id="moment-privacy"
					class="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
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
				overrideLatitude={selectedLocation.latitude}
				overrideLongitude={selectedLocation.longitude}
				onAdjust={() => (showLocationPicker = true)}
			/>
		</div>

		<Dialog.Footer class="pt-2">
			<Button variant="outline" onclick={() => { open = false; onOpenChange?.(false); }}>
				取消
			</Button>
			<Button onclick={handleSubmit} disabled={isSubmitting || isUploadingImages}>
				{#if isSubmitting}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				发布
			</Button>
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
