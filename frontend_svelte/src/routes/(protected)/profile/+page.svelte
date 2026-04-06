<script lang="ts">
	import { onMount } from 'svelte';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Separator } from '$lib/components/ui/separator';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import type { MyProfile } from '$lib/types/api';
	import { formatDate } from '$lib/utils/format';
	import Camera from 'lucide-svelte/icons/camera';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import User from 'lucide-svelte/icons/user';
	import Mail from 'lucide-svelte/icons/mail';
	import Phone from 'lucide-svelte/icons/phone';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Calendar from 'lucide-svelte/icons/calendar';
	import Users from 'lucide-svelte/icons/users';
	import Heart from 'lucide-svelte/icons/heart';
	import MessageSquare from 'lucide-svelte/icons/message-square';
	import Image from 'lucide-svelte/icons/image';
	import Check from 'lucide-svelte/icons/check';
	import X from 'lucide-svelte/icons/x';
	import Shield from 'lucide-svelte/icons/shield';
	import Bell from 'lucide-svelte/icons/bell';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

	import type { PageData } from './$types';

	// 使用 SvelteKit 的 load 函数加载的数据
	let { data }: { data: PageData } = $props();
	
	// 从 load 函数初始化 profile
	let profile = $state<MyProfile | null>(null);
	let isSaving = $state(false);
	let isEditing = $state(false);

	// 表单状态
	let nickname = $state('');
	let bio = $state('');
	let tags = $state('');
	const MAX_TAGS = 10;
	
	// 同步 data.profile 到本地状态
	$effect(() => {
		profile = data.profile;
		if (data.profile) {
			nickname = data.profile.nickname ?? '';
			bio = data.profile.bio ?? '';
			tags = data.profile.tags?.join(', ') ?? '';
		}
	});
	
	// 初始化时更新 authStore（一次性操作）
	onMount(() => {
		if (profile) {
			authStore.setProfile(profile);
		}
	});

	async function saveProfile() {
		const trimmedNickname = nickname.trim();
		const trimmedBio = bio.trim();
		const parsedTags = tags
			.split(',')
			.map((t) => t.trim())
			.filter((t) => t);

		if (!trimmedNickname) {
			toast.error('昵称不能为空');
			return;
		}
		if (trimmedNickname.length > 30) {
			toast.error('昵称长度需在 1-30 字符之间');
			return;
		}
		if (trimmedBio.length > 500) {
			toast.error('个人简介不能超过 500 字符');
			return;
		}
		if (parsedTags.length > MAX_TAGS) {
			toast.error('标签最多 10 个');
			return;
		}
		if (parsedTags.some((tag) => tag.length > 30)) {
			toast.error('每个标签长度不能超过 30 字符');
			return;
		}

		isSaving = true;
		try {
			const response = await usersService.updateMyProfile({
				nickname: trimmedNickname,
				bio: trimmedBio || undefined,
				tags: parsedTags
			});

			if (isSuccess(response)) {
				// updateMyProfile returns void; refresh profile manually
				const profileRes = await usersService.getMyProfile();
				if (isSuccess(profileRes) && profileRes.data) {
					profile = profileRes.data;
					authStore.setProfile(profile);
				}
				toast.success('保存成功');
				isEditing = false;
			} else {
				toast.error(response.message || '保存失败');
			}
		} catch {
			toast.error('保存失败');
		} finally {
			isSaving = false;
		}
	}

	function cancelEdit() {
		if (profile) {
			nickname = profile.nickname;
			bio = profile.bio ?? '';
			tags = profile.tags?.join(', ') ?? '';
		}
		isEditing = false;
	}
</script>

<svelte:head>
	<title>个人资料 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-4xl space-y-5 pb-8">
	<div class="rounded-[1.75rem] border border-border/80 bg-card/92 p-4 shadow-[10px_10px_22px_var(--shadow-dark),-10px_-10px_22px_var(--shadow-light)] backdrop-blur-md">
		<p class="text-sm text-muted-foreground">Profile 页以 Harmony 的身份卡、快捷动作和个人模块分层为主参考，Web mobile 保持同样的信息层次。</p>
	</div>
	<PageHeader
		title="个人资料"
		description="管理你的个人信息和账户设置"
	/>

	{#if profile}
		<div class="grid gap-6 lg:grid-cols-3">
			<!-- 左侧：主要信息 -->
			<div class="space-y-6 lg:col-span-2">
				<!-- 个人资料卡片 -->
				<Card class="overflow-hidden rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
					<!-- 头部背景 -->
					<div class="bg-linear-to-r from-primary/10 via-primary/5 to-transparent p-8">
						<div class="flex flex-col gap-6 sm:flex-row sm:items-start">
							<!-- 头像 -->
							<div class="relative shrink-0">
								<Avatar class="h-24 w-24 border-4 border-background shadow-lg">
									<AvatarImage src={profile.avatar_url} alt={profile.nickname} />
									<AvatarFallback class="text-2xl font-bold">{profile.nickname.slice(0, 2)}</AvatarFallback>
								</Avatar>
								<Button
									size="icon"
									variant="secondary"
									class="absolute -bottom-2 -right-2 h-10 w-10 rounded-2xl shadow-md"
									title="更换头像"
									aria-label="更换头像"
								>
									<Camera class="h-4 w-4" />
								</Button>
								{#if profile.presence_status === 'online'}
									<div class="absolute right-0 top-0 h-6 w-6 rounded-full border-4 border-background bg-green-500"></div>
								{/if}
							</div>

							<!-- 基本信息 -->
							<div class="flex-1 min-w-0">
								<div class="flex items-start justify-between gap-2">
									<div class="flex-1 min-w-0">
										<h2 class="text-2xl font-bold">{profile.nickname}</h2>
									</div>
									{#if !isEditing}
										<Button variant="outline" size="sm" onclick={() => isEditing = true}>
											<User class="mr-2 h-4 w-4" />
											编辑资料
										</Button>
									{/if}
								</div>
								
								{#if profile.bio && !isEditing}
									<p class="mt-3 text-sm leading-relaxed">{profile.bio}</p>
								{/if}

								{#if profile.tags && profile.tags.length > 0 && !isEditing}
									<div class="mt-3 flex flex-wrap gap-2">
										{#each profile.tags as tag (tag)}
											<Badge variant="secondary" class="rounded-2xl">#{tag}</Badge>
										{/each}
									</div>
								{/if}
							</div>
						</div>
					</div>

					<!-- 表单编辑区 -->
					{#if isEditing}
						<CardContent class="space-y-4 border-t p-6">
							<div class="space-y-2">
								<Label for="nickname">昵称 *</Label>
								<Input id="nickname" bind:value={nickname} placeholder="输入你的昵称" />
							</div>

							<div class="space-y-2">
								<Label for="bio">个人简介</Label>
								<Textarea 
									id="bio" 
									bind:value={bio} 
									placeholder="介绍一下自己..." 
									rows={4}
									class="resize-none"
								/>
								<p class="text-xs text-muted-foreground">简短的自我介绍，让大家更了解你</p>
							</div>

							<div class="space-y-2">
								<Label for="tags">兴趣标签</Label>
								<Input 
									id="tags" 
									bind:value={tags} 
									placeholder="摄影, 旅行, 美食, 音乐..." 
								/>
								<p class="text-xs text-muted-foreground">用逗号分隔多个标签，最多 5 个</p>
							</div>

							<div class="flex gap-3 pt-2">
								<Button class="flex-1" onclick={saveProfile} disabled={isSaving}>
									{#if isSaving}
										<Loader2 class="mr-2 h-4 w-4 animate-spin" />
										保存中...
									{:else}
										<Check class="mr-2 h-4 w-4" />
										保存更改
									{/if}
								</Button>
								<Button variant="outline" class="flex-1" onclick={cancelEdit} disabled={isSaving}>
									<X class="mr-2 h-4 w-4" />
									取消
								</Button>
							</div>
						</CardContent>
					{/if}
				</Card>

				<!-- 详细信息 -->
				<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
					<CardHeader>
						<CardTitle class="flex items-center gap-2">
							<User class="h-5 w-5 text-primary" />
							详细信息
						</CardTitle>
					</CardHeader>
					<CardContent class="space-y-4">
						{#if profile.email}
							<div class="flex items-center gap-3">
								<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
									<Mail class="h-5 w-5 text-muted-foreground" />
								</div>
								<div class="flex-1 min-w-0">
									<p class="text-xs text-muted-foreground">邮箱</p>
									<p class="truncate font-medium">{profile.email}</p>
								</div>
							</div>
							<Separator />
						{/if}

						{#if profile.phone}
							<div class="flex items-center gap-3">
								<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
									<Phone class="h-5 w-5 text-muted-foreground" />
								</div>
								<div class="flex-1 min-w-0">
									<p class="text-xs text-muted-foreground">手机号</p>
									<p class="truncate font-medium">{profile.phone}</p>
								</div>
							</div>
							<Separator />
						{/if}

						{#if profile.location?.location_name}
							<div class="flex items-center gap-3">
								<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
									<MapPin class="h-5 w-5 text-muted-foreground" />
								</div>
								<div class="flex-1 min-w-0">
									<p class="text-xs text-muted-foreground">所在地</p>
									<p class="truncate font-medium">{profile.location.location_name}</p>
								</div>
							</div>
							<Separator />
						{/if}

						<div class="flex items-center gap-3">
							<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
								<Calendar class="h-5 w-5 text-muted-foreground" />
							</div>
							<div class="flex-1 min-w-0">
								<p class="text-xs text-muted-foreground">加入时间</p>
								<p class="font-medium">{formatDate(profile.created_at)}</p>
							</div>
						</div>
					</CardContent>
				</Card>
			</div>

			<!-- 右侧：统计数据 -->
			<div class="space-y-6">
				<!-- 数据统计 -->
				<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
					<CardHeader>
						<CardTitle class="text-base">数据概览</CardTitle>
						<CardDescription>你的活动统计</CardDescription>
					</CardHeader>
					<CardContent class="space-y-4">
						<a href="/moments" class="group block rounded-2xl border border-border/50 bg-background/65 p-4 transition-all hover:border-primary/50 hover:bg-accent/55">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-3">
									<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
										<Image class="h-5 w-5 text-primary" />
									</div>
									<div>
										<p class="text-sm text-muted-foreground">发布瞬间</p>
										<p class="text-2xl font-bold">{profile.stats.moments_count}</p>
									</div>
								</div>
							</div>
						</a>

						<a href="/followers" class="group block rounded-2xl border border-border/50 bg-background/65 p-4 transition-all hover:border-primary/50 hover:bg-accent/55">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-3">
									<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-pink-500/10">
										<Heart class="h-5 w-5 text-pink-500" />
									</div>
									<div>
										<p class="text-sm text-muted-foreground">粉丝数</p>
										<p class="text-2xl font-bold">{profile.stats.followers_count}</p>
									</div>
								</div>
							</div>
						</a>

						<a href="/follows" class="group block rounded-2xl border border-border/50 bg-background/65 p-4 transition-all hover:border-primary/50 hover:bg-accent/55">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-3">
									<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10">
										<Users class="h-5 w-5 text-blue-500" />
									</div>
									<div>
										<p class="text-sm text-muted-foreground">关注数</p>
										<p class="text-2xl font-bold">{profile.stats.following_count}</p>
									</div>
								</div>
							</div>
						</a>

						<div class="rounded-2xl border border-border/50 bg-background/65 p-4">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-3">
									<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10">
										<MessageSquare class="h-5 w-5 text-green-500" />
									</div>
									<div>
										<p class="text-sm text-muted-foreground">好友数</p>
										<p class="text-2xl font-bold">{profile.stats.friends_count}</p>
									</div>
								</div>
							</div>
						</div>
					</CardContent>
				</Card>

				<!-- 快捷操作 -->
				<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
					<CardHeader>
						<CardTitle class="text-base">快捷操作</CardTitle>
					</CardHeader>
					<CardContent class="space-y-2">
						<a href="/settings" class="block">
							<Button variant="outline" class="h-11 w-full justify-start rounded-2xl border-border/60 bg-background/70">
								<User class="mr-2 h-4 w-4" />
								账户设置
							</Button>
						</a>
						<a href="/settings/privacy" class="block">
							<Button variant="outline" class="h-11 w-full justify-start rounded-2xl border-border/60 bg-background/70">
								<Shield class="mr-2 h-4 w-4" />
								隐私设置
							</Button>
						</a>
						<a href="/settings/notifications" class="block">
							<Button variant="outline" class="h-11 w-full justify-start rounded-2xl border-border/60 bg-background/70">
								<Bell class="mr-2 h-4 w-4" />
								通知设置
							</Button>
						</a>
					</CardContent>
				</Card>
			</div>
		</div>
	{:else}
		<!-- 错误或空状态 -->
		<Card>
			<CardContent class="flex flex-col items-center justify-center py-16 text-center">
				<AlertTriangle class="mb-4 h-12 w-12 text-muted-foreground/50" />
				<h3 class="mb-2 text-lg font-medium">加载个人资料失败</h3>
				<p class="mb-4 text-sm text-muted-foreground">{data.error ?? '无法获取个人资料，请稍后重试'}</p>
				<Button variant="outline" onclick={() => location.reload()}>重试</Button>
			</CardContent>
		</Card>
	{/if}
</div>
