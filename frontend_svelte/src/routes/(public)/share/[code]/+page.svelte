<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import { shareService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { MapPin, Share2, UsersRound } from 'lucide-svelte';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import type { ShareDetail, ReferencePointSharePayload } from '$lib/types/api';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const shareCode = $derived(data.shareCode);
	const share = $derived<ShareDetail | null>(data.share);
	let isApplying = $state(false);

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

	// 应用分享
	async function handleApply() {
		if (!share) return;

		isApplying = true;
		try {
			const response = await shareService.apply({ share_code: shareCode });
			if (isSuccess(response) && response.data) {
				const result = response.data;
				if (result.action.type === 'update_reference_point') {
					toast.success('浏览锚点已更新！');
					// 跳转到主页，让用户看到新锚点的效果
					setTimeout(() => {
						goto('/');
					}, 1500);
				} else if (result.action.type === 'navigate_user') {
					goto(`/users/${result.action.user_id}`);
				} else if (result.action.type === 'navigate_group') {
					goto(`/groups/${result.action.group_id}`);
				} else if (result.action.type === 'navigate_moment') {
					goto(`/moments/${result.action.moment_id}`);
				}
			} else {
				toast.error('应用失败', response.message ?? '未知错误');
			}
		} catch (error) {
			toast.error('网络错误', '请检查网络连接');
		} finally {
			isApplying = false;
		}
	}

	// 显示错误信息（一次性操作）
	onMount(() => {
		if (data.error) {
			toast.error(data.error);
		}
	});
</script>

<svelte:head>
	<title>{share?.share.share_type === 'reference_point' && (share.payload as ReferencePointSharePayload).reference_point?.location_name ? (share.payload as ReferencePointSharePayload).reference_point.location_name : '探索新区域'} - HotSpot</title>
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
	{:else if share.share.share_type === 'reference_point'}
		{@const payload = share.payload as ReferencePointSharePayload}
		<!-- 位置分享详情卡片 -->
		<Card class="overflow-hidden">
			<!-- 头部 -->
			<div class="bg-gradient-to-r from-primary/10 via-primary/5 to-transparent p-6">
				<div class="space-y-4">
					<div class="flex-1 min-w-0">
						<h1 class="text-2xl font-bold">{payload.reference_point.location_name}</h1>
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
					disabled={isApplying}
				>
					{#if isApplying}
						应用中...
					{:else}
						<MapPin class="mr-2 h-5 w-5" />
						设为浏览锚点
					{/if}
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
						{payload.reference_point.point.latitude.toFixed(6)}, {payload.reference_point.point.longitude.toFixed(6)}
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
		<!-- 其他类型的分享（暂未实现） -->
		<Card>
			<CardHeader>
				<CardTitle>暂不支持此类型的分享</CardTitle>
				<CardDescription>分享类型：{share.share.share_type}</CardDescription>
			</CardHeader>
			<CardContent>
				<Button variant="outline" onclick={() => goto('/')}>返回首页</Button>
			</CardContent>
		</Card>
	{/if}
</div>
