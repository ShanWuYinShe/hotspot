<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { UserStats } from '$lib/types/api';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Users from 'lucide-svelte/icons/users';
	import Image from 'lucide-svelte/icons/image';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import ArrowRight from 'lucide-svelte/icons/arrow-right';
	import Sparkles from 'lucide-svelte/icons/sparkles';
	import Compass from 'lucide-svelte/icons/compass';
	import TrendingUp from 'lucide-svelte/icons/trending-up';
	import Heart from 'lucide-svelte/icons/heart';

	const features = [
		{
			icon: MapPin,
			title: '基于位置',
			description: '发现你附近的精彩内容和有趣的人',
			color: 'from-blue-500/20 to-cyan-500/20',
			iconColor: 'text-blue-500'
		},
		{
			icon: Image,
			title: '生活瞬间',
			description: '分享你的生活点滴，记录美好时刻',
			color: 'from-pink-500/20 to-rose-500/20',
			iconColor: 'text-pink-500'
		},
		{
			icon: Users,
			title: '群组互动',
			description: '加入兴趣群组，结识志同道合的朋友',
			color: 'from-purple-500/20 to-violet-500/20',
			iconColor: 'text-purple-500'
		},
		{
			icon: MessageCircle,
			title: '即时聊天',
			description: '与好友实时沟通，保持紧密联系',
			color: 'from-green-500/20 to-emerald-500/20',
			iconColor: 'text-green-500'
		}
	];

	const quickActions = [
		{ icon: Compass, title: '探索', description: '发现精彩内容', href: '/explore', color: 'bg-linear-to-br from-blue-500 to-cyan-500' },
		{ icon: MapPin, title: '附近', description: '浏览附近的人和动态', href: '/explore?tab=nearby', color: 'bg-linear-to-br from-green-500 to-emerald-500' },
		{ icon: Image, title: '我的瞬间', description: '管理我的发布', href: '/moments', color: 'bg-linear-to-br from-pink-500 to-rose-500' },
		{ icon: Users, title: '我的群组', description: '查看群组动态', href: '/groups', color: 'bg-linear-to-br from-purple-500 to-violet-500' }
	];

	let stats = $state<UserStats | null>(null);
	let statsLoading = $state(false);

	async function loadStats() {
		if (!authStore.isAuthenticated) return;
		statsLoading = true;
		try {
			const response = await usersService.getMyStats();
			if (isSuccess(response) && response.data) {
				stats = response.data;
			}
		} catch {
			// ignore
		} finally {
			statsLoading = false;
		}
	}

	function handleGetStarted() {
		uiStore.openLoginModal(() => goto('/explore'));
	}

	onMount(() => {
		if (authStore.isAuthenticated) {
			loadStats();
		}
	});

	$effect(() => {
		if (authStore.isAuthenticated) {
			loadStats();
		} else {
			stats = null;
		}
	});
</script>

<svelte:head>
	<title>HotSpot - 发现附近的精彩</title>
</svelte:head>

<div class="space-y-12">
	{#if !authStore.isAuthenticated}
		<!-- Hero section for guests -->
		<section class="relative overflow-hidden rounded-3xl bg-linear-to-br from-primary/10 via-primary/15 to-secondary/10 p-8 shadow-xl shadow-primary/10 ring-1 ring-primary/10 md:p-16">
			<!-- Decorative elements -->
			<div class="absolute -right-20 -top-20 h-64 w-64 rounded-full bg-primary/10 blur-3xl"></div>
			<div class="absolute -bottom-20 -left-20 h-64 w-64 rounded-full bg-secondary/10 blur-3xl"></div>
			
			<div class="relative mx-auto max-w-3xl text-center">
				<div class="mb-6 inline-flex items-center gap-2 rounded-full bg-primary/10 px-4 py-2 text-sm font-medium text-primary backdrop-blur-sm">
					<Sparkles class="h-4 w-4" />
					基于位置的社交平台
				</div>
				<h1 class="mb-6 text-4xl font-bold tracking-tight md:text-5xl lg:text-6xl">
					<span class="bg-linear-to-r from-primary via-primary/80 to-primary/60 bg-clip-text text-transparent">
						发现附近的精彩
					</span>
				</h1>
				<p class="mx-auto mb-10 max-w-xl text-lg text-muted-foreground md:text-xl">
					HotSpot 帮助你发现身边有趣的人、精彩的瞬间和活跃的群组。
					开始探索你的世界吧！
				</p>
				<div class="flex flex-col gap-4 sm:flex-row sm:justify-center">
					<Button size="lg" class="gap-2 rounded-full px-8 shadow-lg shadow-primary/25 transition-all hover:shadow-xl hover:shadow-primary/30" onclick={handleGetStarted}>
						立即开始
						<ArrowRight class="h-4 w-4" />
					</Button>
					<Button size="lg" variant="outline" class="rounded-full px-8" onclick={() => goto('/explore')}>
						先逛逛看
					</Button>
				</div>
			</div>
		</section>

		<!-- Features -->
		<section>
			<div class="mb-8 text-center">
				<h2 class="mb-2 text-2xl font-bold md:text-3xl">核心功能</h2>
				<p class="text-muted-foreground">探索 HotSpot 为你提供的精彩体验</p>
			</div>
			<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
				{#each features as feature}
					<Card class="group cursor-pointer border-0 bg-linear-to-br {feature.color} backdrop-blur-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-primary/10">
						<CardHeader>
							<div class="mb-3 inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-background/80 shadow-sm transition-transform group-hover:scale-110">
								<feature.icon class="h-6 w-6 {feature.iconColor}" />
							</div>
							<CardTitle class="text-lg">{feature.title}</CardTitle>
						</CardHeader>
						<CardContent>
							<CardDescription class="text-foreground/70">{feature.description}</CardDescription>
						</CardContent>
					</Card>
				{/each}
			</div>
		</section>

		<!-- Stats section -->
		<section class="rounded-2xl bg-linear-to-r from-primary/5 to-secondary/5 p-8 md:p-12">
			<div class="grid gap-8 text-center sm:grid-cols-3">
				<div>
					<div class="mb-2 text-3xl font-bold text-primary md:text-4xl">10K+</div>
					<div class="text-sm text-muted-foreground">活跃用户</div>
				</div>
				<div>
					<div class="mb-2 text-3xl font-bold text-primary md:text-4xl">50K+</div>
					<div class="text-sm text-muted-foreground">精彩瞬间</div>
				</div>
				<div>
					<div class="mb-2 text-3xl font-bold text-primary md:text-4xl">1K+</div>
					<div class="text-sm text-muted-foreground">活跃群组</div>
				</div>
			</div>
		</section>
	{:else}
		<!-- Dashboard for authenticated users -->
		<section>
			<div class="mb-8">
				<h1 class="mb-2 text-2xl font-bold md:text-3xl">
					欢迎回来，<span class="text-primary">{authStore.currentAccount?.nickname}</span>！
				</h1>
				<p class="text-muted-foreground">看看今天有什么新鲜事</p>
			</div>

			<!-- Quick Actions -->
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				{#each quickActions as action}
					<a href={action.href} class="group">
						<Card class="h-full border-0 bg-card/50 backdrop-blur-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-primary/10">
							<CardContent class="flex items-center gap-4 p-6">
								<div class="{action.color} rounded-2xl p-3 text-white shadow-lg transition-transform group-hover:scale-110">
									<action.icon class="h-6 w-6" />
								</div>
								<div>
									<h3 class="font-semibold">{action.title}</h3>
									<p class="text-sm text-muted-foreground">{action.description}</p>
								</div>
								<ArrowRight class="ml-auto h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1" />
							</CardContent>
						</Card>
					</a>
				{/each}
			</div>
		</section>

		<!-- Activity summary -->
		<section class="space-y-4">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-semibold">数据概览</h2>
				<p class="text-sm text-muted-foreground">你的动态与社交数据</p>
			</div>
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<Card class="border-0 bg-linear-to-br from-pink-500/10 to-rose-500/10">
					<CardContent class="flex items-center gap-4 p-6">
						<div class="rounded-2xl bg-pink-500/20 p-3">
							<Heart class="h-6 w-6 text-pink-500" />
						</div>
						<div>
							{#if statsLoading}
								<Skeleton class="h-7 w-10" />
							{:else}
								<p class="text-2xl font-bold">{stats?.moments_count ?? 0}</p>
							{/if}
							<p class="text-sm text-muted-foreground">我的动态</p>
						</div>
					</CardContent>
				</Card>
				<Card class="border-0 bg-linear-to-br from-blue-500/10 to-cyan-500/10">
					<CardContent class="flex items-center gap-4 p-6">
						<div class="rounded-2xl bg-blue-500/20 p-3">
							<Users class="h-6 w-6 text-blue-500" />
						</div>
						<div>
							{#if statsLoading}
								<Skeleton class="h-7 w-10" />
							{:else}
								<p class="text-2xl font-bold">{stats?.followers_count ?? 0}</p>
							{/if}
							<p class="text-sm text-muted-foreground">关注者</p>
						</div>
					</CardContent>
				</Card>
				<Card class="border-0 bg-linear-to-br from-purple-500/10 to-violet-500/10">
					<CardContent class="flex items-center gap-4 p-6">
						<div class="rounded-2xl bg-purple-500/20 p-3">
							<TrendingUp class="h-6 w-6 text-purple-500" />
						</div>
						<div>
							{#if statsLoading}
								<Skeleton class="h-7 w-10" />
							{:else}
								<p class="text-2xl font-bold">{stats?.friends_count ?? 0}</p>
							{/if}
							<p class="text-sm text-muted-foreground">好友数量</p>
						</div>
					</CardContent>
				</Card>
				<Card class="border-0 bg-linear-to-br from-emerald-500/10 to-teal-500/10">
					<CardContent class="flex items-center gap-4 p-6">
						<div class="rounded-2xl bg-emerald-500/20 p-3">
							<Users class="h-6 w-6 text-emerald-500" />
						</div>
						<div>
							{#if statsLoading}
								<Skeleton class="h-7 w-10" />
							{:else}
								<p class="text-2xl font-bold">{stats?.groups_count ?? 0}</p>
							{/if}
							<p class="text-sm text-muted-foreground">我的群组</p>
						</div>
					</CardContent>
				</Card>
			</div>
		</section>
	{/if}
</div>
