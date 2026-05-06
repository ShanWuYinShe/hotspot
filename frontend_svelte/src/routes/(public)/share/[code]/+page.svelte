<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { MapPin, Share2, UsersRound } from 'lucide-svelte';
	import type {
		ShareDetail,
		LocationSharePayload,
		MomentSharePayload,
		GroupSharePayload,
		UserSharePayload
	} from '$lib/types/api';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const share = $derived<ShareDetail | null>(data.share);

	// 用于清理导航延迟定时器
	let navigationTimer: ReturnType<typeof setTimeout> | null = null;

	// 复制分享链接
	async function copyShareLink() {
		const url = window.location.href;
		try {
			await navigator.clipboard.writeText(url);
			toast.success('分享链接已复制到剪贴板');
		} catch (err) {
			toast.error('复制失败，请手动复制');
		}
	}

	// 处理分享（客户端本地导航，不使用 apply 接口）
	function handleApply() {
		if (!share) return;

		const { share_type } = share.share;
		const payload = share.payload;

		if (share_type === 'location') {
			const loc = (payload as LocationSharePayload).shared_location;
			mapAnchorStore.setAnchor(
				loc.point.latitude,
				loc.point.longitude,
				loc.location_name
			);
			toast.success('浏览锚点已更新！');
			navigationTimer = setTimeout(() => goto('/'), 500);
		} else if (share_type === 'user') {
			goto(`/users/${(payload as UserSharePayload).shared_user_id}`);
		} else if (share_type === 'group') {
			goto(`/groups/${(payload as GroupSharePayload).shared_group_id}`);
		} else if (share_type === 'moment') {
			goto(`/moments/${(payload as MomentSharePayload).shared_moment_id}`);
		}
	}

	// 显示错误信息（一次性操作）
	onMount(() => {
		if (data.error) {
			toast.error(data.error);
		}

		return () => {
			if (navigationTimer) clearTimeout(navigationTimer);
		};
	});
</script>

<svelte:head>
	<title>{share?.share.share_type === 'location' && (share.payload as LocationSharePayload).shared_location?.location_name ? (share.payload as LocationSharePayload).shared_location.location_name : '探索新区域'} - HotSpot</title>
</svelte:head>

<div class="container mx-auto max-w-4xl px-4 py-8">
	{#if !share}
		<div
			class="rounded-[24px] p-6"
			style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
		>
			<h2 style="font-size: 18px; font-weight: 600; color: var(--neu-error);">无法加载分享</h2>
			<p style="font-size: 14px; color: var(--neu-text-muted); margin-top: 8px;">{data.error || '此分享不存在或已被删除'}</p>
			<button
				class="mt-4 rounded-full px-5 py-2.5 font-medium outline-none"
				style="box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
				onclick={() => goto('/')}
			>
				返回首页
			</button>
		</div>
	{:else if share.share.share_type === 'location'}
		{@const payload = share.payload as LocationSharePayload}
		<div
			class="overflow-hidden rounded-[24px]"
			style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
		>
			<!-- Header with creator info -->
			<div class="flex items-center justify-between p-6" style="border-bottom: 1px solid var(--neu-border);">
				<div>
					<h2 style="font-size: 20px; font-weight: 700; color: var(--neu-text);">{payload.shared_location.location_name}</h2>
				</div>
				<div class="flex items-center gap-3">
					<div class="overflow-hidden rounded-full" style="width: 32px; height: 32px; border: 1.5px solid var(--neu-border);">
						<img src={share.share.creator.avatar_url} alt={share.share.creator.nickname} class="h-full w-full object-cover" />
					</div>
					<span style="font-size: 13px; color: var(--neu-text-muted);">
						<span style="font-weight: 500; color: var(--neu-text);">{share.share.creator.nickname}</span> 分享了此位置
					</span>
				</div>
			</div>

			<div class="space-y-6 p-6">
				<!-- Primary action button -->
				<button
					class="flex w-full items-center justify-center gap-2 rounded-full py-4 text-base font-semibold outline-none"
					style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);"
					onclick={handleApply}
				>
					<MapPin size={20} />
					设为浏览锚点
				</button>

				<!-- Info card -->
				<div
					class="rounded-[18px] p-4"
					style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
				>
					<p class="flex items-center gap-2" style="font-weight: 500; font-size: 14px; color: var(--neu-text);">
						<UsersRound size={16} />
						设置后你将看到
					</p>
					<ul class="mt-3 space-y-2" style="font-size: 13px; color: var(--neu-text-muted);">
						<li class="flex items-start gap-2">
							<span style="color: var(--neu-accent-primary);">•</span>
							<span>这个区域附近的人</span>
						</li>
						<li class="flex items-start gap-2">
							<span style="color: var(--neu-accent-primary);">•</span>
							<span>这个区域的附近群组和活跃社区</span>
						</li>
						<li class="flex items-start gap-2">
							<span style="color: var(--neu-accent-primary);">•</span>
							<span>这个区域的最新动态和精彩内容</span>
						</li>
					</ul>
				</div>

				<!-- Location note -->
				<div>
					<p class="uppercase" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted); margin-bottom: 8px;">位置说明</p>
					<div
						class="rounded-[14px] p-3"
						style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
					>
						<p style="font-size: 12px; color: var(--neu-text-muted);">以「{payload.shared_location.location_name}」为中心区域进行浏览和推荐。</p>
					</div>
				</div>

				<!-- Secondary actions -->
				<div class="flex items-center gap-2">
					<button
						class="flex flex-1 items-center justify-center gap-2 rounded-full py-2.5 font-medium outline-none"
						style="box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
						onclick={copyShareLink}
					>
						<Share2 size={16} />
						分享给朋友
					</button>
					<button
						class="rounded-full px-5 py-2.5 font-medium outline-none"
						style="color: var(--neu-text-muted); background: transparent;"
						onclick={() => goto('/')}
					>
						返回首页
					</button>
				</div>
			</div>
		</div>
	{:else}
		<!-- Other share types (user/group/moment) -->
		<div
			class="rounded-[24px] p-6"
			style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
		>
			<h2 style="font-size: 18px; font-weight: 600; color: var(--neu-text);">分享内容</h2>
			<p style="font-size: 14px; color: var(--neu-text-muted); margin-top: 8px;">
				分享类型：{share.share.share_type === 'user' ? '用户' : share.share.share_type === 'group' ? '群组' : '瞬间'}
			</p>
			<div class="mt-4 space-y-3">
				<button
					class="w-full rounded-full py-3 font-medium outline-none"
					style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);"
					onclick={handleApply}
				>
					前去查看
				</button>
				<button
					class="w-full rounded-full py-3 font-medium outline-none"
					style="box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
					onclick={() => goto('/')}
				>
					返回首页
				</button>
			</div>
		</div>
	{/if}
</div>
