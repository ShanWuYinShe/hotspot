<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from '$lib/i18n';
	import { authStore } from '$lib/stores/auth.svelte';
	import { notificationsService, usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import Image from 'lucide-svelte/icons/image';
	import Users from 'lucide-svelte/icons/users';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Bell from 'lucide-svelte/icons/bell';
	import Heart from 'lucide-svelte/icons/heart';

	let isLoading = $state(true);
	let unreadCount = $state(0);
	let followersCount = $state(0);
	let followingCount = $state(0);

	async function loadDashboardData() {
		isLoading = true;
		try {
			const [notifRes, profileRes] = await Promise.allSettled([
				notificationsService.unreadCount(),
				usersService.getMyProfile()
			]);

			if (notifRes.status === 'fulfilled' && isSuccess(notifRes.value) && notifRes.value.data) {
				unreadCount = notifRes.value.data.total;
			}
			if (
				profileRes.status === 'fulfilled' &&
				isSuccess(profileRes.value) &&
				profileRes.value.data
			) {
				followersCount = profileRes.value.data.stats?.followers_count ?? 0;
				followingCount = profileRes.value.data.stats?.following_count ?? 0;
			}
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		loadDashboardData();
	});
</script>

<svelte:head>
	<title>{t('dashboard.title')}</title>
</svelte:head>

<div class="space-y-6">
	<PageHeader
		title={t('dashboard.welcomeBack', { nickname: authStore.currentAccount?.nickname ?? '' })}
		description={t('dashboard.personalDashboard')}
	/>

	{#if isLoading}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each Array(3) as _, i (i)}
				<Skeleton class="h-24 w-full rounded-xl" />
			{/each}
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			<Card class="border-primary/20 bg-primary/5">
				<CardContent class="flex items-center gap-4 p-4">
					<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10">
						<Bell class="h-5 w-5 text-primary" />
					</div>
					<div>
						<p class="text-2xl font-bold">{unreadCount}</p>
						<p class="text-xs text-muted-foreground">{t('dashboard.unreadNotifications')}</p>
					</div>
				</CardContent>
			</Card>
			<Card>
				<CardContent class="flex items-center gap-4 p-4">
					<div
						class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-secondary/10"
					>
						<Heart class="h-5 w-5 text-secondary" />
					</div>
					<div>
						<p class="text-2xl font-bold">{followersCount}</p>
						<p class="text-xs text-muted-foreground">{t('dashboard.followers')}</p>
					</div>
				</CardContent>
			</Card>
			<Card>
				<CardContent class="flex items-center gap-4 p-4">
					<div
						class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-accent/10"
					>
						<Users class="h-5 w-5 text-accent" />
					</div>
					<div>
						<p class="text-2xl font-bold">{followingCount}</p>
						<p class="text-xs text-muted-foreground">{t('dashboard.following')}</p>
					</div>
				</CardContent>
			</Card>
		</div>
	{/if}

	<h2 class="text-lg font-semibold">{t('dashboard.quickAccess')}</h2>
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		<Card class="cursor-pointer transition-shadow hover:shadow-lg">
			<a href="/moments">
				<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
					<CardTitle class="text-sm font-medium">{t('dashboard.myMoments')}</CardTitle>
					<Image class="h-4 w-4 text-muted-foreground" />
				</CardHeader>
				<CardContent>
					<p class="text-xs text-muted-foreground">{t('dashboard.viewAndManageMoments')}</p>
				</CardContent>
			</a>
		</Card>

		<Card class="cursor-pointer transition-shadow hover:shadow-lg">
			<a href="/groups">
				<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
					<CardTitle class="text-sm font-medium">{t('dashboard.myGroups')}</CardTitle>
					<Users class="h-4 w-4 text-muted-foreground" />
				</CardHeader>
				<CardContent>
					<p class="text-xs text-muted-foreground">{t('dashboard.viewJoinedGroups')}</p>
				</CardContent>
			</a>
		</Card>

		<Card class="cursor-pointer transition-shadow hover:shadow-lg">
			<a href="/chat">
				<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
					<CardTitle class="text-sm font-medium">{t('dashboard.messages')}</CardTitle>
					<MessageCircle class="h-4 w-4 text-muted-foreground" />
				</CardHeader>
				<CardContent>
					<p class="text-xs text-muted-foreground">{t('dashboard.viewMessages')}</p>
				</CardContent>
			</a>
		</Card>

		<Card class="cursor-pointer transition-shadow hover:shadow-lg">
			<a href="/users/nearby">
				<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
					<CardTitle class="text-sm font-medium">{t('dashboard.nearby')}</CardTitle>
					<MapPin class="h-4 w-4 text-muted-foreground" />
				</CardHeader>
				<CardContent>
					<p class="text-xs text-muted-foreground">{t('dashboard.discoverNearbyPeopleAndMoments')}</p>
				</CardContent>
			</a>
		</Card>
	</div>
</div>
