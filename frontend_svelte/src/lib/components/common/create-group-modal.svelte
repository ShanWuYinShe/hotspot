<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Label } from '$lib/components/ui/label';
	import { Badge } from '$lib/components/ui/badge';
	import { groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { parseTags } from '$lib/utils/format';
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
	}

	async function handleSubmit() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
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

		if (!trimmedName) {
			toast.error('请输入群组名称');
			return;
		}
		if (trimmedName.length > GROUP_NAME_MAX) {
			toast.error(`群组名称最多 ${GROUP_NAME_MAX} 字符`);
			return;
		}
		if (trimmedDescription.length > GROUP_DESCRIPTION_MAX) {
			toast.error(`群组介绍最多 ${GROUP_DESCRIPTION_MAX} 字符`);
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
		if (trimmedPassword && (trimmedPassword.length < GROUP_PASSWORD_MIN || trimmedPassword.length > GROUP_PASSWORD_MAX)) {
			toast.error(`群组口令长度需在 ${GROUP_PASSWORD_MIN}-${GROUP_PASSWORD_MAX} 字符之间`);
			return;
		}
		if (
			parsedMaxMembers !== undefined &&
			(!Number.isInteger(parsedMaxMembers) ||
				parsedMaxMembers < GROUP_MEMBERS_MIN ||
				parsedMaxMembers > GROUP_MEMBERS_MAX)
		) {
			toast.error(`最大成员数需在 ${GROUP_MEMBERS_MIN}-${GROUP_MEMBERS_MAX} 之间`);
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
				toast.error(response.message || '创建失败');
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

<Dialog.Root bind:open onOpenChange={(value) => { open = value; onOpenChange?.(value); }}>
	<Dialog.Content class="flex max-h-[85vh] flex-col rounded-3xl border border-border/60 bg-card/95 shadow-xl backdrop-blur-md sm:max-w-2xl">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-2xl border border-border/50 bg-primary/10">
					<UsersRound class="h-5 w-5 text-primary" />
				</div>
				<div>
					<div>创建群组</div>
					<div class="text-sm font-normal text-muted-foreground">邀请伙伴一起交流</div>
				</div>
			</Dialog.Title>
		</Dialog.Header>

		<div class="min-h-0 flex-1 space-y-4 overflow-y-auto px-1 py-4">
			<div class="space-y-2 rounded-2xl border border-border/50 bg-background/65 p-4">
				<Label for="group-name">群组名称</Label>
				<Input id="group-name" class="h-11 rounded-2xl border-border/60 bg-background/75" placeholder="例如：城市漫步俱乐部" bind:value={name} />
			</div>

			<div class="space-y-2 rounded-2xl border border-border/50 bg-background/65 p-4">
				<Label for="group-desc">群组介绍</Label>
				<Textarea id="group-desc" placeholder="简单描述群组用途与氛围" class="min-h-[100px] rounded-2xl border-border/60 bg-background/75" bind:value={description} />
			</div>

			<div class="space-y-2 rounded-2xl border border-border/50 bg-background/65 p-4">
				<Label for="group-tags">群组标签</Label>
				<Input id="group-tags" class="h-11 rounded-2xl border-border/60 bg-background/75" placeholder="兴趣、城市、活动" bind:value={tagsInput} />
				{#if tags.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each tags as tag (tag)}
							<Badge variant="secondary" class="rounded-2xl text-xs">#{tag}</Badge>
						{/each}
					</div>
				{/if}
			</div>

			<div class="grid gap-4 rounded-2xl border border-border/50 bg-background/65 p-4 sm:grid-cols-2">
				<div class="space-y-2">
					<Label for="join-method">加入方式</Label>
					<select
						id="join-method"
						class="h-11 w-full rounded-2xl border border-border/60 bg-background/75 px-3 text-sm"
						bind:value={joinMethod}
					>
						<option value="open">自由加入</option>
						<option value="approval">需要审批</option>
						<option value="invite_only">仅邀请</option>
					</select>
				</div>
				<div class="space-y-2">
					<Label for="group-password">入群口令 (可选)</Label>
					<Input id="group-password" class="h-11 rounded-2xl border-border/60 bg-background/75" type="password" placeholder="设置后需口令加入" bind:value={password} />
				</div>
			</div>

			<div class="grid gap-4 rounded-2xl border border-border/50 bg-background/65 p-4 sm:grid-cols-2">
				<div class="space-y-2">
					<Label for="max-members">最大成员数 (可选)</Label>
					<Input id="max-members" class="h-11 rounded-2xl border-border/60 bg-background/75" type="number" min="2" max="2000" placeholder="例如 200" bind:value={maxMembers} />
				</div>
				<div class="flex items-center justify-between rounded-2xl border border-border/60 bg-background/75 px-3 py-2 text-sm">
					<span>允许被搜索</span>
					<input type="checkbox" class="h-4 w-4" bind:checked={isSearchable} />
				</div>
			</div>

			<AnchorSummaryCard
				label="群组位置"
				description="影响附近推荐与搜索，可单独选点且不修改浏览锚点"
				overrideName={selectedLocation.locationName}
				overrideLatitude={selectedLocation.latitude}
				overrideLongitude={selectedLocation.longitude}
				onAdjust={() => (showLocationPicker = true)}
			/>
		</div>

		<Dialog.Footer class="pt-2">
			<Button variant="outline" class="h-10 rounded-2xl border-border/60 bg-background/70" onclick={() => { open = false; onOpenChange?.(false); }}>
				取消
			</Button>
			<Button class="h-10 rounded-2xl" onclick={handleSubmit} disabled={isSubmitting}>
				{#if isSubmitting}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				创建群组
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
