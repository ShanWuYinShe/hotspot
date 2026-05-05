<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from '$lib/i18n';
	import { authStore } from '$lib/stores/auth.svelte';
	import { notificationsService, usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
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

<div class="flex flex-col">
	<!-- Sticky header -->
	<div
		class="sticky top-0 z-10 px-5 pt-3 pb-3"
		style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
	>
		<div>
			<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">{t('dashboard.welcomeBack', { nickname: authStore.currentAccount?.nickname ?? '' })}</h1>
			<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">{t('dashboard.personalDashboard')}</p>
		</div>
	</div>

	<div class="flex flex-col px-4 pt-4 pb-6 gap-5">
		<!-- Stats -->
		{#if isLoading}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each Array(3) as _, i (i)}
					<div class="rounded-[24px]" style="height: 96px; background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"></div>
				{/each}
			</div>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				<div class="rounded-[24px] p-4 flex items-center gap-4" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
					<div class="flex items-center justify-center rounded-full" style="width: 40px; height: 40px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<Bell size={18} color="var(--neu-accent-primary)" strokeWidth={1.5} />
					</div>
					<div>
						<p style="font-size: 22px; font-weight: 700; color: var(--neu-text);">{unreadCount}</p>
						<p style="font-size: 11px; color: var(--neu-text-muted);">{t('dashboard.unreadNotifications')}</p>
					</div>
				</div>
				<div class="rounded-[24px] p-4 flex items-center gap-4" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
					<div class="flex items-center justify-center rounded-full" style="width: 40px; height: 40px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<Heart size={18} color="var(--neu-accent-primary)" strokeWidth={1.5} />
					</div>
					<div>
						<p style="font-size: 22px; font-weight: 700; color: var(--neu-text);">{followersCount}</p>
						<p style="font-size: 11px; color: var(--neu-text-muted);">{t('dashboard.followers')}</p>
					</div>
				</div>
				<div class="rounded-[24px] p-4 flex items-center gap-4" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
					<div class="flex items-center justify-center rounded-full" style="width: 40px; height: 40px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<Users size={18} color="var(--neu-accent-primary)" strokeWidth={1.5} />
					</div>
					<div>
						<p style="font-size: 22px; font-weight: 700; color: var(--neu-text);">{followingCount}</p>
						<p style="font-size: 11px; color: var(--neu-text-muted);">{t('dashboard.following')}</p>
					</div>
				</div>
			</div>
		{/if}

		<!-- Quick Access -->
		<p style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted); text-transform: uppercase; font-weight: 600;">{t('dashboard.quickAccess')}</p>
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<a href="/moments" class="neu-card-hover block rounded-[24px] p-4" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border); text-decoration: none;">
				<div class="flex items-center justify-between mb-3">
					<span style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{t('dashboard.myMoments')}</span>
					<div class="flex items-center justify-center rounded-full" style="width: 28px; height: 28px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<Image size={12} color="var(--neu-text-muted)" strokeWidth={1.5} />
					</div>
				</div>
				<p style="font-size: 12px; color: var(--neu-text-muted);">{t('dashboard.viewAndManageMoments')}</p>
			</a>

			<a href="/groups" class="neu-card-hover block rounded-[24px] p-4" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border); text-decoration: none;">
				<div class="flex items-center justify-between mb-3">
					<span style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{t('dashboard.myGroups')}</span>
					<div class="flex items-center justify-center rounded-full" style="width: 28px; height: 28px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<Users size={12} color="var(--neu-text-muted)" strokeWidth={1.5} />
					</div>
				</div>
				<p style="font-size: 12px; color: var(--neu-text-muted);">{t('dashboard.viewJoinedGroups')}</p>
			</a>

			<a href="/chat" class="neu-card-hover block rounded-[24px] p-4" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border); text-decoration: none;">
				<div class="flex items-center justify-between mb-3">
					<span style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{t('dashboard.messages')}</span>
					<div class="flex items-center justify-center rounded-full" style="width: 28px; height: 28px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<MessageCircle size={12} color="var(--neu-text-muted)" strokeWidth={1.5} />
					</div>
				</div>
				<p style="font-size: 12px; color: var(--neu-text-muted);">{t('dashboard.viewMessages')}</p>
			</a>

			<a href="/users/nearby" class="neu-card-hover block rounded-[24px] p-4" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border); text-decoration: none;">
				<div class="flex items-center justify-between mb-3">
					<span style="font-size: 14px; font-weight: 500; color: var(--neu-text);">{t('dashboard.nearby')}</span>
					<div class="flex items-center justify-center rounded-full" style="width: 28px; height: 28px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
						<MapPin size={12} color="var(--neu-text-muted)" strokeWidth={1.5} />
					</div>
				</div>
				<p style="font-size: 12px; color: var(--neu-text-muted);">{t('dashboard.discoverNearbyPeopleAndMoments')}</p>
			</a>
		</div>
	</div>
</div>
