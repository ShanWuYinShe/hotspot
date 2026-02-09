<script lang="ts">
	import { onMount } from 'svelte';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Label } from '$lib/components/ui/label';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { PrivacyLevel, PrivateChatPermission, UserSettings } from '$lib/types/api';
	import Loader2 from 'lucide-svelte/icons/loader-2';

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

	async function loadSettings() {
		isLoading = true;
		try {
			const response = await usersService.getMySettings();
			if (isSuccess(response) && response.data) {
				settings = response.data;
				privacyLevel = settings.privacy_level;
				allowPrivateChat = settings.allow_private_chat;
				allowFollow = settings.allow_follow;
				searchable = settings.searchable;
				locationSearchable = settings.location_searchable;
				locationVisible = settings.location_visible;
				presenceVisible = settings.presence_visible;
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
				presence_visible: presenceVisible
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

<div class="mx-auto max-w-3xl space-y-6">
	<div>
		<h1 class="text-2xl font-bold">隐私设置</h1>
		<p class="text-sm text-muted-foreground">控制你在 HotSpot 的可见范围</p>
	</div>

	{#if isLoading}
		<Card>
			<CardContent class="space-y-4 p-6">
				{#each Array(5) as _}
					<Skeleton class="h-10 w-full" />
				{/each}
			</CardContent>
		</Card>
	{:else}
		<Card>
			<CardHeader>
				<CardTitle class="text-base">基础可见性</CardTitle>
				<CardDescription>控制大部分信息默认可见范围</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<div class="space-y-2">
					<Label for="privacy-level">隐私级别</Label>
					<select
						id="privacy-level"
						class="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
						bind:value={privacyLevel}
					>
						<option value="public">公开</option>
						<option value="friends">仅好友</option>
						<option value="private">仅自己</option>
					</select>
				</div>
				<div class="flex items-center justify-between">
					<div>
						<p class="font-medium">位置可见</p>
						<p class="text-sm text-muted-foreground">允许展示你的地理位置</p>
					</div>
					<input type="checkbox" class="h-4 w-4" bind:checked={locationVisible} />
				</div>
				<div class="flex items-center justify-between">
					<div>
						<p class="font-medium">在线状态可见</p>
						<p class="text-sm text-muted-foreground">允许显示在线/离线状态</p>
					</div>
					<input type="checkbox" class="h-4 w-4" bind:checked={presenceVisible} />
				</div>
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle class="text-base">社交与私聊</CardTitle>
				<CardDescription>管理关注与私聊权限</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<div class="flex items-center justify-between">
					<div>
						<p class="font-medium">允许关注</p>
						<p class="text-sm text-muted-foreground">关闭后他人关注需审批</p>
					</div>
					<input type="checkbox" class="h-4 w-4" bind:checked={allowFollow} />
				</div>
				<div class="space-y-2">
					<Label for="private-chat">私聊权限</Label>
					<select
						id="private-chat"
						class="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
						bind:value={allowPrivateChat}
					>
						<option value="everyone">所有人可私聊</option>
						<option value="friends">仅好友可私聊</option>
						<option value="nobody">不允许私聊</option>
					</select>
				</div>
			</CardContent>
		</Card>

		<Card>
			<CardHeader>
				<CardTitle class="text-base">搜索与发现</CardTitle>
				<CardDescription>控制你在搜索/附近中的展示</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<div class="flex items-center justify-between">
					<div>
						<p class="font-medium">允许被搜索</p>
						<p class="text-sm text-muted-foreground">关闭后不会出现在搜索结果</p>
					</div>
					<input type="checkbox" class="h-4 w-4" bind:checked={searchable} />
				</div>
				<div class="flex items-center justify-between">
					<div>
						<p class="font-medium">附近可见</p>
						<p class="text-sm text-muted-foreground">允许出现在附近的人</p>
					</div>
					<input type="checkbox" class="h-4 w-4" bind:checked={locationSearchable} />
				</div>
			</CardContent>
		</Card>

		<Button class="w-full" onclick={saveSettings} disabled={isSaving}>
			{#if isSaving}
				<Loader2 class="mr-2 h-4 w-4 animate-spin" />
			{/if}
			保存设置
		</Button>
	{/if}
</div>
