<script lang="ts">
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import type { UserStats } from '$lib/types/api';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Card, CardContent } from '$lib/components/ui/card';
	import Heart from 'lucide-svelte/icons/heart';
	import Users from 'lucide-svelte/icons/users';
	import TrendingUp from 'lucide-svelte/icons/trending-up';

	interface Props {
		refreshTrigger?: number;
	}

	let { refreshTrigger = 0 }: Props = $props();

	let stats = $state<UserStats | null>(null);
	let loading = $state(false);

	async function loadStats() {
		loading = true;
		try {
			const response = await usersService.getMyStats();
			if (isSuccess(response) && response.data) {
				stats = response.data;
			}
		} catch (error) {
			console.error('Failed to load user stats:', error);
			// 不显示 toast，避免影响首页体验
		} finally {
			loading = false;
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
