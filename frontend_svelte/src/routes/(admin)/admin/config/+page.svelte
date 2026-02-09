<script lang="ts">
	import { onMount } from 'svelte';
	import { adminServerConfigService, adminClientConfigService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { ServerConfig, ClientConfig } from '$lib/types/api';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Save from 'lucide-svelte/icons/save';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Lock from 'lucide-svelte/icons/lock';
	import FileText from 'lucide-svelte/icons/file-text';
	import Bell from 'lucide-svelte/icons/bell';

	let serverConfig = $state<ServerConfig | null>(null);
	let clientConfig = $state<ClientConfig | null>(null);
	let isLoading = $state(true);
	let isSaving = $state(false);

	// Server config form fields
	let imageMaxSize = $state('');
	let videoMaxSize = $state('');
	let audioMaxSize = $state('');
	let fileMaxSize = $state('');

	// Client config form fields
	let enablePublicRegistration = $state(true);
	let enableGuestLogin = $state(true);
	let enableMoments = $state(true);
	let enableNearbyPeople = $state(true);
	let maxMomentTags = $state('');
	let maxMomentImages = $state('');
	let momentDescriptionMaxLength = $state('');
	let commentMaxLength = $state('');
	let maxGroupMembers = $state('');
	let searchRadiusMax = $state('');

	async function loadConfig() {
		isLoading = true;
		try {
			// 并行加载服务端配置和客户端配置
			const [serverRes, clientRes] = await Promise.all([
				adminServerConfigService.get(),
				adminClientConfigService.get()
			]);

			if (isSuccess(serverRes) && serverRes.data) {
				serverConfig = serverRes.data;
				imageMaxSize = String(serverConfig.upload?.image_max_size || 5242880);
				videoMaxSize = String(serverConfig.upload?.video_max_size || 52428800);
				audioMaxSize = String(serverConfig.upload?.audio_max_size || 10485760);
				fileMaxSize = String(serverConfig.upload?.file_max_size || 10485760);
			}

			if (isSuccess(clientRes) && clientRes.data) {
				clientConfig = clientRes.data;
				enablePublicRegistration = clientConfig.features?.enable_public_registration ?? true;
				enableGuestLogin = clientConfig.features?.enable_guest_login ?? true;
				enableMoments = clientConfig.features?.enable_moments ?? true;
				enableNearbyPeople = clientConfig.features?.enable_nearby_people ?? true;
				maxMomentTags = String(clientConfig.limits?.max_moment_tags || 10);
				maxMomentImages = String(clientConfig.limits?.max_moment_images || 9);
				momentDescriptionMaxLength = String(clientConfig.limits?.moment_description_max_length || 300);
				commentMaxLength = String(clientConfig.limits?.comment_max_length || 200);
				maxGroupMembers = String(clientConfig.limits?.max_group_members || 500);
				searchRadiusMax = String(clientConfig.limits?.search_radius_max || 50);
			}
		} catch {
			toast.error('加载配置失败');
		} finally {
			isLoading = false;
		}
	}

	async function handleSaveUpload() {
		isSaving = true;
		try {
			const response = await adminServerConfigService.updateUpload({
				upload: {
					image_max_size: parseInt(imageMaxSize) || undefined,
					video_max_size: parseInt(videoMaxSize) || undefined,
					audio_max_size: parseInt(audioMaxSize) || undefined,
					file_max_size: parseInt(fileMaxSize) || undefined
				}
			});

			if (isSuccess(response)) {
				toast.success('上传配置已保存');
			} else {
				toast.error(response.message || '保存失败');
			}
		} catch {
			toast.error('保存失败');
		} finally {
			isSaving = false;
		}
	}

	async function handleSaveFeatures() {
		isSaving = true;
		try {
			const response = await adminClientConfigService.updateFeatures({
				enable_public_registration: enablePublicRegistration,
				enable_guest_login: enableGuestLogin,
				enable_moments: enableMoments,
				enable_nearby_people: enableNearbyPeople
			});

			if (isSuccess(response)) {
				toast.success('功能配置已保存');
			} else {
				toast.error(response.message || '保存失败');
			}
		} catch {
			toast.error('保存失败');
		} finally {
			isSaving = false;
		}
	}

	async function handleSaveLimits() {
		isSaving = true;
		try {
			const response = await adminClientConfigService.updateLimits({
				limits: {
					max_moment_tags: parseInt(maxMomentTags) || undefined,
					max_moment_images: parseInt(maxMomentImages) || undefined,
					moment_description_max_length: parseInt(momentDescriptionMaxLength) || undefined,
					comment_max_length: parseInt(commentMaxLength) || undefined,
					max_group_members: parseInt(maxGroupMembers) || undefined,
					search_radius_max: parseInt(searchRadiusMax) || undefined
				}
			});

			if (isSuccess(response)) {
				toast.success('限制配置已保存');
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
		loadConfig();
	});
</script>

<svelte:head>
	<title>系统配置 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">系统配置</h1>
		<Button variant="outline" size="sm" onclick={loadConfig} disabled={isLoading}>
			<RefreshCw class="mr-2 h-4 w-4 {isLoading ? 'animate-spin' : ''}" />
			刷新
		</Button>
	</div>

	{#if isLoading}
		<div class="grid gap-6 lg:grid-cols-2">
			{#each Array(3) as _}
				<Card>
					<CardHeader>
						<Skeleton class="h-5 w-32" />
						<Skeleton class="h-4 w-48" />
					</CardHeader>
					<CardContent class="space-y-4">
						{#each Array(3) as _}
							<div class="space-y-2">
								<Skeleton class="h-4 w-20" />
								<Skeleton class="h-9 w-full" />
							</div>
						{/each}
					</CardContent>
				</Card>
			{/each}
		</div>
	{:else}
		<div class="grid gap-6 lg:grid-cols-2">
			<!-- 上传设置（服务端配置） -->
			<Card>
				<CardHeader>
					<CardTitle class="flex items-center gap-2">
						<FileText class="h-5 w-5" />
						上传设置
					</CardTitle>
					<CardDescription>配置文件上传相关限制（服务端配置）</CardDescription>
				</CardHeader>
				<CardContent class="space-y-4">
					<div class="space-y-2">
						<Label for="image-max-size">图片最大大小 (字节)</Label>
						<Input
							id="image-max-size"
							type="number"
							bind:value={imageMaxSize}
							placeholder="5242880"
						/>
						<p class="text-xs text-muted-foreground">
							当前: {(Number(imageMaxSize || 0) / 1024 / 1024).toFixed(1)} MB
						</p>
					</div>
					<div class="space-y-2">
						<Label for="video-max-size">视频最大大小 (字节)</Label>
						<Input
							id="video-max-size"
							type="number"
							bind:value={videoMaxSize}
							placeholder="52428800"
						/>
						<p class="text-xs text-muted-foreground">
							当前: {(Number(videoMaxSize || 0) / 1024 / 1024).toFixed(1)} MB
						</p>
					</div>
					<div class="space-y-2">
						<Label for="audio-max-size">音频最大大小 (字节)</Label>
						<Input
							id="audio-max-size"
							type="number"
							bind:value={audioMaxSize}
							placeholder="10485760"
						/>
						<p class="text-xs text-muted-foreground">
							当前: {(Number(audioMaxSize || 0) / 1024 / 1024).toFixed(1)} MB
						</p>
					</div>
					<div class="space-y-2">
						<Label for="file-max-size">文件最大大小 (字节)</Label>
						<Input id="file-max-size" type="number" bind:value={fileMaxSize} placeholder="10485760" />
						<p class="text-xs text-muted-foreground">
							当前: {(Number(fileMaxSize || 0) / 1024 / 1024).toFixed(1)} MB
						</p>
					</div>
					<Button onclick={handleSaveUpload} disabled={isSaving}>
						{#if isSaving}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{:else}
							<Save class="mr-2 h-4 w-4" />
						{/if}
						保存上传设置
					</Button>
				</CardContent>
			</Card>

			<!-- 功能开关（客户端配置） -->
			<Card>
				<CardHeader>
					<CardTitle class="flex items-center gap-2">
						<Lock class="h-5 w-5" />
						功能开关
					</CardTitle>
					<CardDescription>启用或禁用系统功能（客户端配置）</CardDescription>
				</CardHeader>
				<CardContent class="space-y-4">
					<div class="flex items-center justify-between">
						<div>
							<Label>允许新用户注册</Label>
							<p class="text-sm text-muted-foreground">关闭后新用户无法注册</p>
						</div>
						<label class="relative inline-flex cursor-pointer items-center">
							<input type="checkbox" bind:checked={enablePublicRegistration} class="peer sr-only" />
							<div class="peer h-6 w-11 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-full peer-checked:after:border-white dark:border-gray-600 dark:bg-gray-700"></div>
						</label>
					</div>
					<div class="flex items-center justify-between">
						<div>
							<Label>允许访客模式</Label>
							<p class="text-sm text-muted-foreground">允许用户以访客身份使用</p>
						</div>
						<label class="relative inline-flex cursor-pointer items-center">
							<input type="checkbox" bind:checked={enableGuestLogin} class="peer sr-only" />
							<div class="peer h-6 w-11 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-full peer-checked:after:border-white dark:border-gray-600 dark:bg-gray-700"></div>
						</label>
					</div>
					<div class="flex items-center justify-between">
						<div>
							<Label>启用动态功能</Label>
							<p class="text-sm text-muted-foreground">控制是否开放生活瞬间</p>
						</div>
						<label class="relative inline-flex cursor-pointer items-center">
							<input type="checkbox" bind:checked={enableMoments} class="peer sr-only" />
							<div class="peer h-6 w-11 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-full peer-checked:after:border-white dark:border-gray-600 dark:bg-gray-700"></div>
						</label>
					</div>
					<div class="flex items-center justify-between">
						<div>
							<Label>启用附近的人</Label>
							<p class="text-sm text-muted-foreground">是否展示附近用户</p>
						</div>
						<label class="relative inline-flex cursor-pointer items-center">
							<input type="checkbox" bind:checked={enableNearbyPeople} class="peer sr-only" />
							<div class="peer h-6 w-11 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-full peer-checked:after:border-white dark:border-gray-600 dark:bg-gray-700"></div>
						</label>
					</div>
					<Button onclick={handleSaveFeatures} disabled={isSaving}>
						{#if isSaving}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{:else}
							<Save class="mr-2 h-4 w-4" />
						{/if}
						保存功能设置
					</Button>
				</CardContent>
			</Card>

			<!-- 限制设置（客户端配置） -->
			<Card>
				<CardHeader>
					<CardTitle class="flex items-center gap-2">
						<Bell class="h-5 w-5" />
						限制设置
					</CardTitle>
					<CardDescription>配置各种数量限制（客户端配置）</CardDescription>
				</CardHeader>
				<CardContent class="space-y-4">
					<div class="space-y-2">
						<Label for="max-tags">动态最大标签数</Label>
						<Input id="max-tags" type="number" bind:value={maxMomentTags} placeholder="10" min="1" max="50" />
					</div>
					<div class="space-y-2">
						<Label for="max-moment-images">动态最大图片数</Label>
						<Input
							id="max-moment-images"
							type="number"
							bind:value={maxMomentImages}
							placeholder="9"
							min="1"
							max="20"
						/>
					</div>
					<div class="space-y-2">
						<Label for="moment-description-max">动态描述最大长度</Label>
						<Input
							id="moment-description-max"
							type="number"
							bind:value={momentDescriptionMaxLength}
							placeholder="300"
							min="1"
							max="1000"
						/>
					</div>
					<div class="space-y-2">
						<Label for="comment-max">评论最大长度</Label>
						<Input
							id="comment-max"
							type="number"
							bind:value={commentMaxLength}
							placeholder="200"
							min="1"
							max="1000"
						/>
					</div>
					<div class="space-y-2">
						<Label for="max-group-members">群组最大成员数</Label>
						<Input
							id="max-group-members"
							type="number"
							bind:value={maxGroupMembers}
							placeholder="500"
							min="10"
							max="10000"
						/>
					</div>
					<div class="space-y-2">
						<Label for="search-radius-max">搜索最大半径 (km)</Label>
						<Input
							id="search-radius-max"
							type="number"
							bind:value={searchRadiusMax}
							placeholder="50"
							min="1"
							max="500"
						/>
					</div>
					<Button onclick={handleSaveLimits} disabled={isSaving}>
						{#if isSaving}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{:else}
							<Save class="mr-2 h-4 w-4" />
						{/if}
						保存限制设置
					</Button>
				</CardContent>
			</Card>
		</div>
	{/if}
</div>
