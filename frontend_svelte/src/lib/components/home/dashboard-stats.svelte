<script lang="ts">
	import { dev } from '$app/environment';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import type { MyProfile } from '$lib/types/api';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Card, CardContent } from '$lib/components/ui/card';
	import Heart from 'lucide-svelte/icons/heart';
	import Users from 'lucide-svelte/icons/users';
	import TrendingUp from 'lucide-svelte/icons/trending-up';

	interface Props {
		refreshTrigger?: number;
	}

	let { refreshTrigger = 0 }: Props = $props();

	// 统计数据内嵌在 MyProfile 中（spec 无独立 stats 端点）
	let stats = $state<MyProfile['stats'] | null>(null);
	let loading = $state(false);
	let loadVersion = 0;

	async function loadStats() {
		const version = ++loadVersion;
		loading = true;
		try {
			const response = await usersService.getMyProfile();
			if (version !== loadVersion) return; // 丢弃过期响应
			if (isSuccess(response) && response.data) {
				stats = response.data.stats;
			}
		} catch (error) {
			if (version !== loadVersion) return;
			if (dev) console.error('Failed to load user stats:', error);
		} finally {
			if (version === loadVersion) loading = false;
		}
	}

	$effect(() => {
		refreshTrigger; // Dependency
		loadStats();
	});
</script>

<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
	<Card class="border-0 bg-linear-to-br from-pink-500/10 to-rose-500/10">
		<CardContent class="flex items-center gap-4 p-6">
			<div class="rounded-2xl bg-pink-500/20 p-3">
				<Heart class="h-6 w-6 text-pink-500" />
			</div>
			<div>
				{#if loading}
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
				{#if loading}
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
				{#if loading}
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
				{#if loading}
					<Skeleton class="h-7 w-10" />
				{:else}
					<p class="text-2xl font-bold">{stats?.groups_count ?? 0}</p>
				{/if}
				<p class="text-sm text-muted-foreground">我的群组</p>
			</div>
		</CardContent>
	</Card>
</div>
