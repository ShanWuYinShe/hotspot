<script lang="ts">
	import { onMount } from 'svelte';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { MyCard } from '$lib/types/api';
	import Camera from 'lucide-svelte/icons/camera';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	let profile = $state<MyCard | null>(null);
	let isLoading = $state(true);
	let isSaving = $state(false);

	// 表单状态
	let nickname = $state('');
	let bio = $state('');
	let tags = $state('');

	async function loadProfile() {
		try {
			const response = await usersService.getMyProfile();
			if (isSuccess(response) && response.data) {
				profile = response.data;
				nickname = profile.nickname;
				bio = profile.bio ?? '';
				tags = profile.tags?.join(', ') ?? '';
				authStore.setProfile(profile);
			}
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
		}
	}

	async function saveProfile() {
		if (!nickname.trim()) {
			toast.error('昵称不能为空');
			return;
		}

		isSaving = true;
		try {
			const response = await usersService.updateMyProfile({
				nickname: nickname.trim(),
				bio: bio.trim() || undefined,
				tags: tags
					.split(',')
					.map((t) => t.trim())
					.filter((t) => t)
			});

			if (isSuccess(response) && response.data) {
				profile = response.data;
				authStore.setProfile(profile);
				toast.success('保存成功');
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
		loadProfile();
	});
</script>

<svelte:head>
	<title>个人资料 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-2xl space-y-6">
	<h1 class="text-2xl font-bold">个人资料</h1>

	{#if isLoading}
		<Card>
			<CardContent class="space-y-6 p-6">
				<div class="flex justify-center">
					<Skeleton class="h-24 w-24 rounded-full" />
				</div>
				<div class="space-y-4">
					<Skeleton class="h-10 w-full" />
					<Skeleton class="h-24 w-full" />
					<Skeleton class="h-10 w-full" />
				</div>
			</CardContent>
		</Card>
	{:else if profile}
		<Card>
			<CardContent class="space-y-6 p-6">
				<!-- Avatar -->
				<div class="flex flex-col items-center">
					<div class="relative">
						<Avatar class="h-24 w-24">
							<AvatarImage src={profile.avatar_url} alt={profile.nickname} />
							<AvatarFallback class="text-2xl">{profile.nickname.slice(0, 2)}</AvatarFallback>
						</Avatar>
						<Button
							size="icon"
							variant="secondary"
							class="absolute bottom-0 right-0 h-8 w-8 rounded-full"
						>
							<Camera class="h-4 w-4" />
						</Button>
					</div>
					<p class="mt-2 text-sm text-muted-foreground">点击更换头像</p>
				</div>

				<!-- Form -->
				<div class="space-y-4">
					<div class="space-y-2">
						<Label for="nickname">昵称</Label>
						<Input id="nickname" bind:value={nickname} placeholder="你的昵称" />
					</div>

					<div class="space-y-2">
						<Label for="bio">简介</Label>
						<Textarea id="bio" bind:value={bio} placeholder="介绍一下自己" rows={3} />
					</div>

					<div class="space-y-2">
						<Label for="tags">标签</Label>
						<Input id="tags" bind:value={tags} placeholder="用逗号分隔，如：摄影, 旅行, 美食" />
						<p class="text-xs text-muted-foreground">用逗号分隔多个标签</p>
					</div>
				</div>

				<Button class="w-full" onclick={saveProfile} disabled={isSaving}>
					{#if isSaving}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					保存
				</Button>
			</CardContent>
		</Card>

		<!-- Stats -->
		<Card>
			<CardHeader>
				<CardTitle class="text-base">数据统计</CardTitle>
			</CardHeader>
			<CardContent>
				<div class="grid grid-cols-4 gap-4 text-center">
					<div>
						<p class="text-2xl font-bold">{profile.stats.moments_count}</p>
						<p class="text-sm text-muted-foreground">瞬间</p>
					</div>
					<div>
						<p class="text-2xl font-bold">{profile.stats.followers_count}</p>
						<p class="text-sm text-muted-foreground">粉丝</p>
					</div>
					<div>
						<p class="text-2xl font-bold">{profile.stats.following_count}</p>
						<p class="text-sm text-muted-foreground">关注</p>
					</div>
					<div>
						<p class="text-2xl font-bold">{profile.stats.friends_count}</p>
						<p class="text-sm text-muted-foreground">好友</p>
					</div>
				</div>
			</CardContent>
		</Card>
	{/if}
</div>
