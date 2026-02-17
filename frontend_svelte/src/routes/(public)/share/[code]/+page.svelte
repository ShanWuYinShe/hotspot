<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { MapPin, Share2, UsersRound } from 'lucide-svelte';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
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
		<Card class="border-destructive/50">
			<CardHeader>
				<CardTitle class="text-destructive">无法加载分享</CardTitle>
				<CardDescription>{data.error || '此分享不存在或已被删除'}</CardDescription>
			</CardHeader>
			<CardContent>
				<Button variant="outline" onclick={() => goto('/')}>返回首页</Button>
			</CardContent>
		</Card>
	{:else if share.share.share_type === 'location'}
		{@const payload = share.payload as LocationSharePayload}
		<!-- 位置分享详情卡片 -->
		<Card class="overflow-hidden">
			<!-- 头部 -->
			<div class="bg-gradient-to-r from-primary/10 via-primary/5 to-transparent p-6">
				<div class="space-y-4">
					<div class="flex-1 min-w-0">
						<h1 class="text-2xl font-bold">{payload.shared_location.location_name}</h1>
					</div>

					<!-- 分享者信息 -->
					<div class="flex items-center gap-3">
						<Avatar class="h-8 w-8">
							<AvatarImage src={share.share.creator.avatar_url} alt={share.share.creator.nickname} />
							<AvatarFallback>{share.share.creator.nickname.slice(0, 2)}</AvatarFallback>
						</Avatar>
						<span class="text-sm text-muted-foreground">
							<span class="font-medium text-foreground">{share.share.creator.nickname}</span> 分享了此位置
						</span>
					</div>
				</div>
			</div>

			<CardContent class="space-y-6 p-6">
				<!-- 主要操作按钮 -->
				<Button
					class="w-full h-14 text-base font-semibold"
					onclick={handleApply}
				>
					<MapPin class="mr-2 h-5 w-5" />
					设为浏览锚点
				</Button>

				<!-- 说明卡片 -->
				<div class="rounded-lg bg-primary/10 p-4 space-y-3">
					<p class="font-medium flex items-center gap-2">
						<UsersRound class="h-4 w-4" />
						设置后你将看到
					</p>
					<ul class="space-y-2 text-sm text-muted-foreground">
						<li class="flex items-start gap-2">
							<span class="text-primary">•</span>
							<span>这个区域附近的人</span>
						</li>
						<li class="flex items-start gap-2">
							<span class="text-primary">•</span>
							<span>这个区域的附近群组和活跃社区</span>
						</li>
						<li class="flex items-start gap-2">
							<span class="text-primary">•</span>
							<span>这个区域的最新动态和精彩内容</span>
						</li>
					</ul>
				</div>

				<!-- 位置信息 -->
				<div class="space-y-2">
					<h3 class="text-sm font-medium text-muted-foreground">位置坐标</h3>
					<div class="rounded-lg bg-muted/50 p-3 text-xs text-muted-foreground font-mono">
						{payload.shared_location.point.latitude.toFixed(6)}, {payload.shared_location.point.longitude.toFixed(6)}
					</div>
				</div>

				<!-- 次要操作 -->
				<div class="flex items-center gap-2">
					<Button variant="outline" size="sm" class="flex-1" onclick={copyShareLink}>
						<Share2 class="mr-2 h-4 w-4" />
						分享给朋友
					</Button>
					<Button variant="ghost" size="sm" onclick={() => goto('/')}>
						返回首页
					</Button>
				</div>
			</CardContent>
		</Card>
	{:else}
		<!-- 其他类型的分享（用户/群组/瞬间） -->
		<Card>
			<CardHeader>
				<CardTitle>分享内容</CardTitle>
				<CardDescription>
					分享类型：{share.share.share_type === 'user' ? '用户' : share.share.share_type === 'group' ? '群组' : '瞬间'}
				</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				<Button class="w-full" onclick={handleApply}>
					前去查看
				</Button>
				<Button variant="outline" onclick={() => goto('/')}>返回首页</Button>
			</CardContent>
		</Card>
	{/if}
</div>
