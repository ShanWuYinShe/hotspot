<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Label } from '$lib/components/ui/label';
	import { Badge } from '$lib/components/ui/badge';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import LocationPickerModal from './location-picker-modal.svelte';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import UsersRound from 'lucide-svelte/icons/users-round';

	type JoinMethod = 'open' | 'approval' | 'invite_only';

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
	let includeLocation = $state(true);
	let isSubmitting = $state(false);
	let showLocationPicker = $state(false);

	let tags = $derived(
		tagsInput
			.split(/[,，\s]+/)
			.map((tag) => tag.replace(/^#/, '').trim())
			.filter(Boolean)
			.slice(0, 10)
	);

	function resetForm() {
		name = '';
		description = '';
		tagsInput = '';
		joinMethod = 'open';
		password = '';
		maxMembers = '';
		isSearchable = true;
		includeLocation = true;
	}

	async function handleSubmit() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			open = false;
			onOpenChange?.(false);
			return;
		}

		if (!name.trim()) {
			toast.error('请输入群组名称');
			return;
		}

		isSubmitting = true;
		try {
			const locationName = mapAnchorStore.name || mapAnchorStore.displayName;
			const payload: {
				name: string;
				description?: string;
				tags?: string[];
				password?: string;
				join_method?: JoinMethod;
				is_searchable?: boolean;
				max_members?: number;
				location?: { latitude: number; longitude: number; location_name: string };
			} = {
				name: name.trim(),
				join_method: joinMethod,
				is_searchable: isSearchable
			};

			if (description.trim()) payload.description = description.trim();
			if (tags.length > 0) payload.tags = tags;
			if (password.trim()) payload.password = password.trim();
			if (maxMembers.trim()) payload.max_members = parseInt(maxMembers, 10);
			if (includeLocation) {
				payload.location = {
					latitude: mapAnchorStore.latitude,
					longitude: mapAnchorStore.longitude,
					location_name: locationName
				};
			}

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
	<Dialog.Content class="sm:max-w-2xl">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<div class="flex h-10 w-10 items-center justify-center rounded-xl bg-linear-to-br from-primary/20 to-primary/10">
					<UsersRound class="h-5 w-5 text-primary" />
				</div>
				<div>
					<div>创建群组</div>
					<div class="text-sm font-normal text-muted-foreground">邀请伙伴一起交流</div>
				</div>
			</Dialog.Title>
		</Dialog.Header>

		<div class="mt-4 space-y-4">
			<div class="space-y-2">
				<Label for="group-name">群组名称</Label>
				<Input id="group-name" placeholder="例如：城市漫步俱乐部" bind:value={name} />
			</div>

			<div class="space-y-2">
				<Label for="group-desc">群组介绍</Label>
				<Textarea id="group-desc" placeholder="简单描述群组用途与氛围" class="min-h-[100px]" bind:value={description} />
			</div>

			<div class="space-y-2">
				<Label for="group-tags">群组标签</Label>
				<Input id="group-tags" placeholder="兴趣、城市、活动" bind:value={tagsInput} />
				{#if tags.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each tags as tag}
							<Badge variant="secondary" class="rounded-full text-xs">#{tag}</Badge>
						{/each}
					</div>
				{/if}
			</div>

			<div class="grid gap-4 sm:grid-cols-2">
				<div class="space-y-2">
					<Label for="join-method">加入方式</Label>
					<select
						id="join-method"
						class="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
						bind:value={joinMethod}
					>
						<option value="open">自由加入</option>
						<option value="approval">需要审批</option>
						<option value="invite_only">仅邀请</option>
					</select>
				</div>
				<div class="space-y-2">
					<Label for="group-password">入群口令 (可选)</Label>
					<Input id="group-password" type="password" placeholder="设置后需口令加入" bind:value={password} />
				</div>
			</div>

			<div class="grid gap-4 sm:grid-cols-2">
				<div class="space-y-2">
					<Label for="max-members">最大成员数 (可选)</Label>
					<Input id="max-members" type="number" min="10" placeholder="例如 200" bind:value={maxMembers} />
				</div>
				<div class="flex items-center justify-between rounded-md border border-input px-3 py-2 text-sm">
					<span>允许被搜索</span>
					<input type="checkbox" class="h-4 w-4" bind:checked={isSearchable} />
				</div>
			</div>

			<Card class="border-0 bg-linear-to-r from-primary/5 to-primary/10">
				<CardContent class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
					<div class="min-w-0">
						<div class="text-xs text-muted-foreground">群组位置</div>
						<div class="truncate font-medium">{mapAnchorStore.displayName}</div>
						<div class="text-xs text-muted-foreground">
							{mapAnchorStore.latitude.toFixed(4)}, {mapAnchorStore.longitude.toFixed(4)}
						</div>
					</div>
					<div class="flex items-center gap-2">
						<Button variant="outline" size="sm" class="gap-2" onclick={() => (showLocationPicker = true)}>
							<MapPin class="h-4 w-4" />
							调整位置
						</Button>
						<label class="flex items-center gap-2 text-xs text-muted-foreground">
							<input type="checkbox" class="h-4 w-4" bind:checked={includeLocation} />
							启用位置
						</label>
					</div>
				</CardContent>
			</Card>
		</div>

		<Dialog.Footer class="mt-6">
			<Button variant="outline" onclick={() => { open = false; onOpenChange?.(false); }}>
				取消
			</Button>
			<Button onclick={handleSubmit} disabled={isSubmitting}>
				{#if isSubmitting}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				创建群组
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<LocationPickerModal bind:open={showLocationPicker} />
