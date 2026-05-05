<script lang="ts">
	import { onMount } from 'svelte';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/stores/toast.svelte';
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

{#if profile}
	<div class="mx-auto max-w-4xl space-y-5 pb-8 px-4 pt-4">
		<!-- 个人资料卡片 -->
		<div class="rounded-[24px] p-6" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
			<div class="flex flex-col gap-6 sm:flex-row sm:items-start">
				<!-- 头像 -->
				<div class="relative shrink-0">
					<div class="overflow-hidden rounded-full" style="width: 96px; height: 96px; border: 1.5px solid var(--neu-border); box-shadow: 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light);">
						{#if profile.avatar_url}
							<img src={profile.avatar_url} alt={profile.nickname} class="h-full w-full object-cover" />
						{:else}
							<div class="flex h-full w-full items-center justify-center" style="background: var(--neu-bg); font-size: 28px; font-weight: 700; color: var(--neu-text);">{profile.nickname.slice(0, 2)}</div>
						{/if}
					</div>
					<button
						class="absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none"
						style="width: 32px; height: 32px; background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 2px 8px var(--neu-glow); border: 2px solid var(--neu-bg);"
						title="更换头像"
						aria-label="更换头像"
					>
						<Camera size={14} />
					</button>
					{#if profile.presence_status === 'online'}
						<span class="absolute bottom-1 right-1 rounded-full" style="width: 11px; height: 11px; background: var(--neu-success); border: 2px solid var(--neu-bg);"></span>
					{/if}
				</div>

				<!-- 基本信息 -->
				<div class="flex-1 min-w-0">
					<div class="flex items-start justify-between gap-2">
						<div class="flex-1 min-w-0">
							<h2 style="font-size: 22px; font-weight: 700; color: var(--neu-text);">{profile.nickname}</h2>
						</div>
						{#if !isEditing}
							<button
								class="flex items-center gap-1.5 rounded-[16px] px-4 py-[6px] outline-none"
								style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); font-size: 12px; color: var(--neu-text);"
								onclick={() => isEditing = true}
							>
								<User size={13} />
								编辑资料
							</button>
						{/if}
					</div>

					{#if profile.bio && !isEditing}
						<p style="margin-top: 12px; font-size: 13px; color: var(--neu-text); line-height: 1.7;">{profile.bio}</p>
					{/if}

					{#if profile.tags && profile.tags.length > 0 && !isEditing}
						<div class="mt-3 flex flex-wrap gap-2">
							{#each profile.tags as tag (tag)}
								<span class="rounded-full px-3.5 py-1.5" style="font-size: 12px; color: var(--neu-accent-primary); background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">#{tag}</span>
							{/each}
						</div>
					{/if}
				</div>
			</div>

			<!-- 表单编辑区 -->
			{#if isEditing}
				<div style="height: 1px; background: var(--neu-border); margin: 20px 0;"></div>
				<div class="space-y-4">
					<div>
						<label for="nickname" style="font-size: 12px; font-weight: 500; color: var(--neu-text-muted); display: block; margin-bottom: 6px;">昵称 *</label>
						<input
							id="nickname"
							bind:value={nickname}
							placeholder="输入你的昵称"
							class="w-full rounded-[14px] px-4 py-2.5 outline-none"
							style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text); font-size: 14px;"
						/>
					</div>

					<div>
						<label for="bio" style="font-size: 12px; font-weight: 500; color: var(--neu-text-muted); display: block; margin-bottom: 6px;">个人简介</label>
						<textarea
							id="bio"
							bind:value={bio}
							placeholder="介绍一下自己..."
							rows="4"
							class="w-full resize-none rounded-[14px] px-4 py-2.5 outline-none"
							style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text); font-size: 14px;"
						></textarea>
						<p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 4px;">简短的自我介绍，让大家更了解你</p>
					</div>

					<div>
						<label for="tags" style="font-size: 12px; font-weight: 500; color: var(--neu-text-muted); display: block; margin-bottom: 6px;">兴趣标签</label>
						<input
							id="tags"
							bind:value={tags}
							placeholder="摄影, 旅行, 美食, 音乐..."
							class="w-full rounded-[14px] px-4 py-2.5 outline-none"
							style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text); font-size: 14px;"
						/>
						<p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 4px;">用逗号分隔多个标签，最多 5 个</p>
					</div>

					<div class="flex gap-3 pt-2">
						<button
							class="flex-1 flex items-center justify-center gap-1.5 rounded-[22px] px-4 py-3.5 outline-none"
							style="background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 14px; box-shadow: 0 2px 8px var(--neu-glow);"
							onclick={saveProfile}
							disabled={isSaving}
						>
							{#if isSaving}
								<Loader2 class="h-4 w-4 animate-spin" />
								保存中...
							{:else}
								<Check size={15} />
								保存更改
							{/if}
						</button>
						<button
							class="flex-1 flex items-center justify-center gap-1.5 rounded-[22px] px-4 py-3.5 outline-none"
							style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text); font-size: 14px;"
							onclick={cancelEdit}
							disabled={isSaving}
						>
							<X size={15} />
							取消
						</button>
					</div>
				</div>
			{/if}
		</div>

		<!-- 详细信息 -->
		<div class="rounded-[24px] p-5" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
			<div class="flex items-center gap-2 mb-4">
				<div class="flex items-center justify-center rounded-full" style="width: 32px; height: 32px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
					<User size={14} color="var(--neu-accent-primary)" strokeWidth={1.5} />
				</div>
				<span style="font-size: 16px; font-weight: 600; color: var(--neu-text);">详细信息</span>
			</div>
			<div class="space-y-4">
				{#if profile.email}
					<div class="flex items-center gap-3">
						<div class="flex items-center justify-center rounded-full" style="width: 36px; height: 36px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<Mail size={14} color="var(--neu-text-muted)" strokeWidth={1.5} />
						</div>
						<div class="flex-1 min-w-0">
							<p style="font-size: 11px; color: var(--neu-text-muted);">邮箱</p>
							<p class="truncate" style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{profile.email}</p>
						</div>
					</div>
					<div style="height: 1px; background: var(--neu-border);"></div>
				{/if}

				{#if profile.phone}
					<div class="flex items-center gap-3">
						<div class="flex items-center justify-center rounded-full" style="width: 36px; height: 36px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<Phone size={14} color="var(--neu-text-muted)" strokeWidth={1.5} />
						</div>
						<div class="flex-1 min-w-0">
							<p style="font-size: 11px; color: var(--neu-text-muted);">手机号</p>
							<p class="truncate" style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{profile.phone}</p>
						</div>
					</div>
					<div style="height: 1px; background: var(--neu-border);"></div>
				{/if}

				{#if profile.location?.location_name}
					<div class="flex items-center gap-3">
						<div class="flex items-center justify-center rounded-full" style="width: 36px; height: 36px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<MapPin size={14} color="var(--neu-text-muted)" strokeWidth={1.5} />
						</div>
						<div class="flex-1 min-w-0">
							<p style="font-size: 11px; color: var(--neu-text-muted);">所在地</p>
							<p class="truncate" style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{profile.location.location_name}</p>
						</div>
					</div>
					<div style="height: 1px; background: var(--neu-border);"></div>
				{/if}

				<div class="flex items-center gap-3">
					<div class="flex items-center justify-center rounded-full" style="width: 36px; height: 36px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<Calendar size={14} color="var(--neu-text-muted)" strokeWidth={1.5} />
					</div>
					<div class="flex-1 min-w-0">
						<p style="font-size: 11px; color: var(--neu-text-muted);">加入时间</p>
						<p style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{formatDate(profile.created_at)}</p>
					</div>
				</div>
			</div>
		</div>

		<!-- 数据统计 -->
		<div class="rounded-[24px] p-5" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
			<p style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted); text-transform: uppercase; font-weight: 600; margin-bottom: 12px;">数据概览</p>
			<div class="grid gap-3 grid-cols-2 md:grid-cols-4">
				<a href="/moments" class="block rounded-[18px] p-[14px]" style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); text-decoration: none;">
					<div class="flex items-center gap-3">
						<div class="flex items-center justify-center rounded-full" style="width: 32px; height: 32px; box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<Image size={14} color="var(--neu-accent-primary)" strokeWidth={1.5} />
						</div>
						<div>
							<p style="font-size: 12px; color: var(--neu-text-muted);">发布瞬间</p>
							<p style="font-size: 17px; font-weight: 500; color: var(--neu-text);">{profile.stats.moments_count}</p>
						</div>
					</div>
				</a>

				<a href="/followers" class="block rounded-[18px] p-[14px]" style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); text-decoration: none;">
					<div class="flex items-center gap-3">
						<div class="flex items-center justify-center rounded-full" style="width: 32px; height: 32px; box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<Heart size={14} color="var(--neu-accent-primary)" strokeWidth={1.5} />
						</div>
						<div>
							<p style="font-size: 10px; color: var(--neu-accent-primary); text-transform: uppercase; letter-spacing: 0.10em; opacity: 0.8;">粉丝数</p>
							<p style="font-size: 17px; font-weight: 500; color: var(--neu-text);">{profile.stats.followers_count}</p>
						</div>
					</div>
				</a>

				<a href="/follows" class="block rounded-[18px] p-[14px]" style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); text-decoration: none;">
					<div class="flex items-center gap-3">
						<div class="flex items-center justify-center rounded-full" style="width: 32px; height: 32px; box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<Users size={14} color="var(--neu-accent-primary)" strokeWidth={1.5} />
						</div>
						<div>
							<p style="font-size: 10px; color: var(--neu-accent-primary); text-transform: uppercase; letter-spacing: 0.10em; opacity: 0.8;">关注数</p>
							<p style="font-size: 17px; font-weight: 500; color: var(--neu-text);">{profile.stats.following_count}</p>
						</div>
					</div>
				</a>

				<div class="rounded-[18px] p-[14px]" style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
					<div class="flex items-center gap-3">
						<div class="flex items-center justify-center rounded-full" style="width: 32px; height: 32px; box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<MessageSquare size={14} color="var(--neu-accent-primary)" strokeWidth={1.5} />
						</div>
						<div>
							<p style="font-size: 10px; color: var(--neu-accent-primary); text-transform: uppercase; letter-spacing: 0.10em; opacity: 0.8;">好友数</p>
							<p style="font-size: 17px; font-weight: 500; color: var(--neu-text);">{profile.stats.friends_count}</p>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- 快捷操作 -->
		<div class="rounded-[24px] p-5" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
			<p style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted); text-transform: uppercase; font-weight: 600; margin-bottom: 12px;">快捷操作</p>
			<div class="space-y-2">
				<a href="/settings" class="block" style="text-decoration: none;">
					<div class="flex items-center gap-3 rounded-[14px] px-4 py-3" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<div class="flex items-center justify-center rounded-full" style="width: 28px; height: 28px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<User size={12} color="var(--neu-text-muted)" strokeWidth={1.5} />
						</div>
						<span style="font-size: 14px; color: var(--neu-text);">账户设置</span>
					</div>
				</a>
				<a href="/settings/privacy" class="block" style="text-decoration: none;">
					<div class="flex items-center gap-3 rounded-[14px] px-4 py-3" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<div class="flex items-center justify-center rounded-full" style="width: 28px; height: 28px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<Shield size={12} color="var(--neu-text-muted)" strokeWidth={1.5} />
						</div>
						<span style="font-size: 14px; color: var(--neu-text);">隐私设置</span>
					</div>
				</a>
				<a href="/settings/notifications" class="block" style="text-decoration: none;">
					<div class="flex items-center gap-3 rounded-[14px] px-4 py-3" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<div class="flex items-center justify-center rounded-full" style="width: 28px; height: 28px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<Bell size={12} color="var(--neu-text-muted)" strokeWidth={1.5} />
						</div>
						<span style="font-size: 14px; color: var(--neu-text);">通知设置</span>
					</div>
				</a>
			</div>
		</div>
	</div>
{:else}
	<!-- 错误或空状态 -->
	<div class="flex flex-col items-center justify-center py-16 px-4">
		<div class="mb-4 flex items-center justify-center rounded-full" style="width: 64px; height: 64px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
			<AlertTriangle size={28} color="var(--neu-text-muted)" strokeWidth={1.5} />
		</div>
		<p style="font-size: 16px; font-weight: 500; color: var(--neu-text); margin-bottom: 8px;">加载个人资料失败</p>
		<p style="font-size: 14px; color: var(--neu-text-muted); margin-bottom: 16px;">{data.error ?? '无法获取个人资料，请稍后重试'}</p>
		<button
			class="rounded-[22px] px-5 py-3.5 outline-none"
			style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-accent-primary); font-size: 14px;"
			onclick={() => location.reload()}
		>
			重试
		</button>
	</div>
{/if}
