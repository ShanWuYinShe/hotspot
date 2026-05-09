<script lang="ts">
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { t } from '$lib/i18n';
	import { feedService, momentsService, usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import DashboardStats from '$lib/components/home/dashboard-stats.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { MomentItem, UserItem } from '$lib/types/api';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Users from 'lucide-svelte/icons/users';
	import Compass from 'lucide-svelte/icons/compass';
	import ChevronRight from 'lucide-svelte/icons/chevron-right';
	import Sparkles from 'lucide-svelte/icons/sparkles';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Star from 'lucide-svelte/icons/star';
	import TrendingUp from 'lucide-svelte/icons/trending-up';
	import Coffee from 'lucide-svelte/icons/coffee';
	import Music from 'lucide-svelte/icons/music';
	import Navigation2 from 'lucide-svelte/icons/navigation-2';
	import { goto } from '$app/navigation';

	type HomeMode = 'guest' | 'explore' | 'authenticated';

	import { platformStore } from '$lib/stores/platform.svelte';

	let refreshTrigger = $state(0);
	let showExplore = $state(false);
	let lastMode = $state<HomeMode>('guest');
	let lastAnchorUpdatedAt = $state<number | null>(null);
	let locationDebounceTimer: ReturnType<typeof setTimeout> | null = null;

	// Nearby users (for Figma Make layout)
	let nearbyUsers = $state.raw<UserItem[]>([]);
	let loadingNearby = $state(false);

	const feed = usePagination<MomentItem>({
		uniqueKey: 'moment_id',
		initialLoading: true,
	});

	function getAnchor() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude,
		};
	}

	function getHomeMode(): HomeMode {
		if (authStore.isAuthenticated) return 'authenticated';
		if (showExplore) return 'explore';
		return 'guest';
	}

	function isContentActive() {
		return getHomeMode() !== 'guest';
	}

	function formatDistance(meters?: number): string {
		if (meters == null) return '';
		if (meters < 1000) return `${Math.round(meters)}m`;
		return `${(meters / 1000).toFixed(1)}km`;
	}

	async function loadNearbyUsers() {
		loadingNearby = true;
		try {
			const response = await usersService.search({
				radius: mapAnchorStore.radius * 1000,
				reference_point: getAnchor(),
				pagination: { limit: 10 },
			});
			if (isSuccess(response) && response.data) {
				const currentUserId = authStore.currentUserId;
				const items = response.data.items ?? [];
				nearbyUsers = currentUserId
					? items.filter((item) => item.user.user_id !== currentUserId)
					: items;
			}
		} catch {
			// ignore
		} finally {
			loadingNearby = false;
		}
	}

	async function loadTrendingTags() {
		
		try {
			const response = await feedService.getTrending();
			if (isSuccess(response) && response.data) {
				// trending tags loaded by TrendingTags component
			}
		} catch {
			// ignore
		} finally {

		}
	}

	async function loadFeed(refresh = false) {
		if (!isContentActive()) return;

		await feed.load(
			async (cursor) => {
				if (authStore.isAuthenticated) {
					return await feedService.getMoments({
						reference_point: getAnchor(),
						source: 'all',
						radius: mapAnchorStore.radius * 1000,
						pagination: { limit: 20, cursor },
					});
				}

				return await momentsService.search({
					radius: mapAnchorStore.radius * 1000,
					reference_point: getAnchor(),
					pagination: { limit: 20, cursor },
				});
			},
			refresh,
		);
	}

	async function refreshHome(options: { refreshFeed?: boolean; refreshWidgets?: boolean; refreshGuestTrending?: boolean } = {}) {
		const {
			refreshFeed = true,
			refreshWidgets = authStore.isAuthenticated,
			refreshGuestTrending = showExplore,
		} = options;

		if (refreshWidgets && authStore.isAuthenticated) {
			refreshTrigger++;
		}

		const tasks: Promise<void>[] = [];

		if (refreshGuestTrending && showExplore) {
			tasks.push(loadTrendingTags());
		}

		if (refreshFeed && isContentActive()) {
			tasks.push(loadFeed(true));
		}

		if (refreshWidgets && authStore.isAuthenticated) {
			tasks.push(loadNearbyUsers());
		}

		await Promise.all(tasks);
	}

	function refreshAll() {
		void refreshHome();
	}

	function handleGetStarted() {
		uiStore.openLoginModal(() => {
			showExplore = false;
			if (getHomeMode() === 'authenticated') {
				void refreshHome({ refreshWidgets: true, refreshGuestTrending: false });
			}
		});
	}

	function handleExplore() {
		if (showExplore) {
			void refreshHome({ refreshWidgets: false, refreshGuestTrending: true });
			return;
		}
		showExplore = true;
	}

	$effect(() => {
		const mode = getHomeMode();
		if (mode === lastMode) return;
		lastMode = mode;
		lastAnchorUpdatedAt = null;
		if (mode === 'guest') {
			feed.reset();
			return;
		}
		void refreshHome({
			refreshWidgets: mode === 'authenticated',
			refreshGuestTrending: mode === 'explore',
		});
	});

	$effect(() => {
		const mode = getHomeMode();
		if (mode === 'guest') {
			if (locationDebounceTimer) {
				clearTimeout(locationDebounceTimer);
				locationDebounceTimer = null;
			}
			lastAnchorUpdatedAt = null;
			return;
		}
		const anchorUpdatedAt = mapAnchorStore.anchor.updatedAt;
		if (lastAnchorUpdatedAt === null) {
			lastAnchorUpdatedAt = anchorUpdatedAt;
			return;
		}
		if (anchorUpdatedAt === lastAnchorUpdatedAt) return;
		lastAnchorUpdatedAt = anchorUpdatedAt;
		if (locationDebounceTimer) clearTimeout(locationDebounceTimer);
		locationDebounceTimer = setTimeout(() => {
			void refreshHome({
				refreshWidgets: mode === 'authenticated',
				refreshGuestTrending: mode === 'explore',
			});
		}, 500);
		return () => {
			if (locationDebounceTimer) {
				clearTimeout(locationDebounceTimer);
				locationDebounceTimer = null;
			}
		};
	});

	function handleFollowToggle(user: UserItem) {
		if (!authStore.isAuthenticated) {
			uiStore.openLoginModalWithRedirect({});
			return;
		}
		void goto(`/users/${user.user.user_id}`);
	}
</script>

<svelte:head>
	<title>{t('home.title')}</title>
</svelte:head>

{#if !authStore.isAuthenticated && !showExplore}
	<!-- Guest landing (Figma Make style) -->
	<div class="flex flex-col px-5 pt-4 pb-6">
		<!-- Sticky header -->
		<div
			class="sticky top-0 z-10 flex items-center justify-between px-1 py-3"
			style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
		>
			<div>
				<h1 style="font-size: 22px; color: var(--neu-text);">{t('home.title')}</h1>
				<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">{t('home.discoverNearbyPeopleAndThings')}</p>
			</div>
			<div
				class="flex cursor-pointer items-center gap-1.5 rounded-full py-1.5 px-3.5"
				style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
			>
				<MapPin size={12} strokeWidth={1.5} color="var(--neu-accent-primary)" />
				<span style="font-size: 11px; color: var(--neu-text-muted);">{mapAnchorStore.displayName}</span>
			</div>
		</div>

		<!-- Hero card -->
		<div class="mt-4">
			<div
				class="flex flex-col items-center rounded-[24px] px-6 py-10 text-center"
				style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
			>
				<div
					class="mb-4 flex items-center justify-center rounded-full"
					style="width: 56px; height: 56px; background: var(--neu-accent-primary); box-shadow: 0 6px 20px var(--neu-glow);"
				>
					<Sparkles size={24} color="var(--neu-active-text)" />
				</div>
				<h2 style="font-size: 22px; font-weight: 700; color: var(--neu-text);">{t('home.discoverNearby')}</h2>
				<p style="font-size: 14px; color: var(--neu-text-muted); margin-top: 8px; max-width: 320px;">{t('home.heroDescription')}</p>
				<div class="mt-6 flex gap-3">
					<button
						class="rounded-[22px] px-6 py-3.5 font-medium transition-transform duration-150 active:scale-[0.95]"
						style="background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);"
						onclick={handleGetStarted}
					>
						{t('home.getStarted')}
					</button>
					<button
						class="rounded-[22px] px-6 py-3.5 font-medium transition-transform duration-150 active:scale-[0.97]"
						style="box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
						onclick={handleExplore}
					>
						{t('home.browseFirst')}
					</button>
				</div>
			</div>
		</div>

		<!-- Features grid -->
		<div class="mt-5">
			<p class="mb-3 uppercase" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);">{t('home.coreFeatures')}</p>
			<div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px;">
				{#each [
					{ icon: MapPin, label: t('nav.nearby'), color: 'var(--neu-accent-primary)' },
					{ icon: Compass, label: t('nav.discover'), color: 'var(--neu-fallback-text)' },
					{ icon: Users, label: t('nav.messagesTab'), color: 'var(--neu-accent-primary)' },
					{ icon: Star, label: t('nav.me'), color: 'var(--neu-warning)' },
				] as feat}
					<div
						class="flex flex-col items-center justify-center rounded-[22px] neu-card-hover transition-transform duration-150 active:scale-[0.97]"
						style="aspect-ratio: 1.2; background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
					>
						<div class="flex items-center justify-center rounded-[14px]" style="width: 36px; height: 36px; background: var(--neu-fallback-bg);">
							<feat.icon size={17} strokeWidth={1.75} color={feat.color} />
						</div>
						<span style="font-size: 12px; color: var(--neu-text); margin-top: 8px;">{feat.label}</span>
					</div>
				{/each}
			</div>
		</div>
	</div>
{:else if !platformStore.isDesktop}
	<!-- Mobile Home (Figma Make HomeMobile) -->
	<div class="flex flex-col">
		<!-- Sticky header -->
		<div
			class="sticky top-0 z-10 flex items-center justify-between px-5 py-3"
			style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
		>
			<div>
				<h1 style="font-size: 22px; color: var(--neu-text);">{t('nav.nearby')}</h1>
				<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">{t('home.discoverNearbyPeopleAndThings')}</p>
			</div>
			<div class="flex items-center gap-2">
				<div
					class="flex cursor-pointer items-center gap-1.5 rounded-full py-1.5 px-3.5"
					style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
				>
					<MapPin size={12} strokeWidth={1.5} color="var(--neu-accent-primary)" />
					<span style="font-size: 11px; color: var(--neu-text-muted);">{mapAnchorStore.displayName}</span>
				</div>
				<button
					class="flex items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.9]"
					style="width: 28px; height: 28px; background: var(--neu-accent-primary); box-shadow: 0 2px 8px var(--neu-glow);"
					onclick={refreshAll}
					aria-label={t('common.refresh')}
				>
					<RefreshCw size={12} color="var(--neu-active-text)" />
				</button>
			</div>
		</div>

		<div class="flex flex-col px-5 pt-4 pb-6">
			<!-- Map Card (Figma Make HomeMobile) -->
			<div class="relative mb-5 w-full overflow-hidden rounded-[24px]" style="aspect-ratio: 2.4/1; box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
				<div class="absolute inset-0" style="background: linear-gradient(135deg, var(--neu-fallback-bg), var(--neu-bg));"></div>
				<div class="absolute inset-0" style="background: linear-gradient(to bottom, transparent 30%, rgba(0,0,0,0.3) 100%);"></div>
				<div class="absolute inset-0 flex items-center justify-center">
					<div class="flex items-center justify-center rounded-full" style="width: 44px; height: 44px; background: var(--neu-bg); opacity: 0.88; backdrop-filter: blur(8px); border: 1.5px solid rgba(255,255,255,0.35); box-shadow: 0 4px 16px rgba(0,0,0,0.18);">
						<span class="rounded-full animate-pulse" style="width: 14px; height: 14px; background: var(--neu-accent-primary); box-shadow: 0 0 0 4px var(--neu-glow);"></span>
					</div>
				</div>
				<div class="absolute bottom-3 right-3 flex flex-col gap-2">
					{#each [Navigation2, Compass] as Icon}
						<button
							class="flex items-center justify-center rounded-xl outline-none"
							style="width: 34px; height: 34px; background: var(--neu-bg); opacity: 0.88; backdrop-filter: blur(8px); border: 1px solid rgba(255,255,255,0.25); box-shadow: 2px 2px 6px rgba(0,0,0,0.15);"
						>
							<Icon size={14} strokeWidth={1.5} color="var(--neu-text)" />
						</button>
					{/each}
				</div>
				<div class="absolute bottom-3 left-3 flex items-center gap-1 rounded-full px-2.5 py-1" style="background: var(--neu-bg); opacity: 0.88; backdrop-filter: blur(8px); border: 1px solid rgba(255,255,255,0.2);">
					<span style="font-size: 11px; color: var(--neu-accent-primary); font-weight: 600;">{nearbyUsers.length}</span>
					<span style="font-size: 10px; color: var(--neu-text-muted);">人附近</span>
				</div>
			</div>

			<!-- Nearby People -->
			{#if authStore.isAuthenticated}
				<div class="mb-3 flex items-center justify-between">
					<span class="uppercase" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);">附近的人</span>
					<button class="flex items-center gap-0.5 outline-none" style="color: var(--neu-accent-primary);" onclick={() => void goto('/users/nearby')}>
						<span style="font-size: 12px;">{t('common.viewAll')}</span>
						<ChevronRight size={13} strokeWidth={1.5} />
					</button>
				</div>
				<div class="mb-5 flex flex-col gap-2.5">
					{#if loadingNearby}
						{#each Array(3) as _}
							<div class="flex items-center gap-3.5 rounded-[22px] px-4 py-3.5" style="height: 68px; background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
								<div class="rounded-full" style="width: 44px; height: 44px; background: var(--neu-border);"></div>
								<div class="flex-1">
									<div class="mb-1.5 rounded" style="width: 60px; height: 12px; background: var(--neu-border);"></div>
									<div class="rounded" style="width: 80px; height: 10px; background: var(--neu-border);"></div>
								</div>
							</div>
						{/each}
					{:else}
						{#each nearbyUsers.slice(0, 3) as user (user.user.user_id)}
							<div role="button" tabindex="0"
								class="flex cursor-pointer items-center gap-3.5 rounded-[22px] px-4 py-3.5 text-left neu-row-hover transition-transform duration-150 active:scale-[0.97]"
								style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
								aria-label={user.user.nickname}
								onclick={() => void goto(`/users/${user.user.user_id}`)}
							>
								<div class="relative flex-shrink-0">
									<div class="overflow-hidden rounded-full" style="width: 44px; height: 44px; border: 1.5px solid var(--neu-border);">
										<img src={user.user.avatar_url || ''} alt={user.user.nickname} class="h-full w-full object-cover" />
									</div>
									{#if user.presence_status === 'online'}
										<span class="absolute bottom-0 right-0 rounded-full" style="width: 11px; height: 11px; background: var(--neu-success); border: 2px solid var(--neu-bg);"></span>
									{/if}
								</div>
								<div class="min-w-0 flex-1">
									<p style="font-size: 14px; color: var(--neu-text);">{user.user.nickname}</p>
									<div class="mt-0.5 flex items-center gap-1.5">
										{#if user.presence_status}
											<span style="font-size: 10px; color: {user.presence_status === 'online' ? 'var(--neu-success)' : 'var(--neu-text-muted)'};">
												{user.presence_status === 'online' ? t('common.online') : t('common.offline')}
											</span>
											<span style="font-size: 10px; color: var(--neu-text-muted); opacity: 0.5;">·</span>
										{/if}
										{#if user.tags && user.tags.length > 0}
											<span style="font-size: 10px; color: var(--neu-accent-primary); opacity: 0.8;">{user.tags[0]}</span>
										{/if}
									</div>
								</div>
								<div class="flex flex-col items-end gap-2">
									{#if user.location?.distance != null}
										<span style="font-size: 12px; color: var(--neu-accent-primary);">{formatDistance(user.location.distance)}</span>
									{/if}
									<button
										class="rounded-full px-3.5 py-1 outline-none transition-all active:scale-[0.93]"
										style={user.relation?.is_following
											? 'box-shadow: inset 3px 3px 6px var(--neu-shadow-dark), inset -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-accent-primary); background: var(--neu-bg); font-size: 11px;'
											: 'box-shadow: 3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg); font-size: 11px;'}
										onclick={(e) => { e.stopPropagation(); handleFollowToggle(user); }}
									>
										{user.relation?.is_following ? t('common.following') : t('common.follow')}
									</button>
								</div>
							</div>
						{/each}
					{/if}
				</div>
			{/if}

			<!-- Activities (Figma Make) -->
			<div class="mb-3 flex items-center justify-between">
				<span class="uppercase" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);">附近活动</span>
				<button class="flex items-center gap-0.5 outline-none" style="color: var(--neu-accent-primary);">
					<span style="font-size: 12px;">更多</span>
					<ChevronRight size={13} strokeWidth={1.5} />
				</button>
			</div>
			<div class="mb-5" style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; width: 100%;">
				{#each [
					{ icon: Coffee, title: '晨间咖啡会', sub: '12人参加 · 0.3km', color: '#F59E0B', bg: 'rgba(245,158,11,0.1)' },
					{ icon: Music, title: '街头音乐节', sub: '38人参加 · 0.7km', color: '#8B5CF6', bg: 'rgba(139,92,246,0.1)' },
				] as a}
					{@const Icon = a.icon}
					<button
						class="flex flex-col justify-between rounded-[22px] cursor-pointer neu-card-hover transition-transform duration-150 active:scale-[0.97]"
						style="aspect-ratio: 1/1; padding: 16px; background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
					>
						<div class="flex items-center justify-center rounded-[12px]" style="width: 36px; height: 36px; background: {a.bg};">
							<Icon size={17} strokeWidth={1.75} color={a.color} />
						</div>
						<div>
							<p style="font-size: 13px; color: var(--neu-text);">{a.title}</p>
							<p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 2px;">{a.sub}</p>
						</div>
					</button>
				{/each}
			</div>

			<!-- Moments Feed -->
			<div class="mb-2 flex items-center justify-between">
				<span class="uppercase" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);">{t('home.recommendedMoments')}</span>
				<button class="flex items-center gap-0.5 outline-none" style="color: var(--neu-accent-primary);" onclick={refreshAll}>
					<RefreshCw size={12} class={feed.loading ? 'animate-spin' : ''} color="var(--neu-accent-primary)" />
				</button>
			</div>

			{#if feed.loading}
				<div class="flex flex-col gap-3">
					{#each Array(3) as _}
						<div class="rounded-[22px] p-4" style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
							<div class="flex items-center gap-3">
								<div class="rounded-full" style="width: 36px; height: 36px; background: var(--neu-border);"></div>
								<div class="flex-1">
									<div class="mb-1 rounded" style="width: 60px; height: 12px; background: var(--neu-border);"></div>
									<div class="rounded" style="width: 100px; height: 10px; background: var(--neu-border);"></div>
								</div>
							</div>
						</div>
					{/each}
				</div>
			{:else if feed.error}
				<EmptyState
					icon={Compass}
					title={t('home.loadMomentsFailed')}
					description={t('home.pleaseCheckNetwork')}
					actionText={t('common.retry')}
					onAction={() => loadFeed(true)}
				/>
			{:else if feed.items.length === 0}
				<EmptyState
					icon={Compass}
					title={t('home.noRecommendations')}
					description={t('home.noRecommendationsDesc')}
				/>
			{:else}
				<div class="flex flex-col gap-3">
					{#each feed.items as moment (moment.moment_id)}
						<MomentCard {moment} />
					{/each}
				</div>
				<LoadMoreButton
					loading={feed.loadingMore}
					hasMore={feed.hasMore}
					onLoadMore={() => loadFeed(false)}
				/>
			{/if}
		</div>
	</div>
{:else}
	<!-- Desktop Home (Figma Make HomeDesktop) -->
	<div class="flex h-full flex-col">
		<!-- Sticky header -->
		<div
			class="sticky top-0 z-10 flex items-center justify-between px-6 py-4"
			style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
		>
			<div>
				<h1 style="font-size: 24px; font-weight: 600; color: var(--neu-text);">{t('nav.nearby')}</h1>
				<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">{t('home.discoverNearbyPeopleAndThings')}</p>
			</div>
			<div class="flex items-center gap-2">
				<div
					class="flex items-center gap-1.5 rounded-full py-1.5 px-3.5"
					style="box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);"
				>
					<MapPin size={12} strokeWidth={1.5} color="var(--neu-accent-primary)" />
					<span style="font-size: 11px; color: var(--neu-text-muted);">{mapAnchorStore.displayName}</span>
				</div>
				<button
					class="flex items-center gap-1 rounded-full px-3 py-1.5"
					style="background: var(--neu-accent-primary); box-shadow: 0 4px 12px var(--neu-glow);"
					onclick={refreshAll}
				>
					<TrendingUp size={11} strokeWidth={2} color="var(--neu-active-text)" />
					<span style="font-size: 11px; color: var(--neu-active-text);">{nearbyUsers.length}{t('sidebar.nearbyPeople')}</span>
				</button>
			</div>
		</div>

		<div class="flex min-h-0 flex-1 gap-0">
			<!-- Left column -->
			<div
				class="flex flex-col gap-4 overflow-y-auto no-scrollbar p-5"
				style="width: 340px; flex-shrink: 0; border-right: 1px solid var(--neu-border);"
			>
				<!-- Dashboard Stats -->
				<DashboardStats {refreshTrigger} />


			</div>

			<!-- Right column — people + feed -->
			<div class="flex-1 overflow-y-auto no-scrollbar p-5">
				<!-- Nearby People -->
				{#if authStore.isAuthenticated}
					<div class="mb-4 flex items-center justify-between">
						<p class="uppercase" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);">{t('sidebar.nearbyPeople')}</p>
						<a href="/users/nearby" class="flex items-center gap-0.5 outline-none" style="color: var(--neu-accent-primary);">
							<span style="font-size: 12px;">{t('common.viewAll')}</span>
							<ChevronRight size={13} strokeWidth={1.5} />
						</a>
					</div>
					<div class="mb-5 flex flex-col gap-3">
						{#if loadingNearby}
							{#each Array(4) as _}
								<div class="flex items-center gap-3.5 rounded-[22px] px-4 py-3.5" style="height: 68px; background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
									<div class="rounded-full" style="width: 44px; height: 44px; background: var(--neu-border);"></div>
									<div class="flex-1">
										<div class="mb-1.5 rounded" style="width: 60px; height: 12px; background: var(--neu-border);"></div>
										<div class="rounded" style="width: 80px; height: 10px; background: var(--neu-border);"></div>
									</div>
								</div>
							{/each}
						{:else}
							{#each nearbyUsers as user (user.user.user_id)}
								<div role="button" tabindex="0"
									class="flex cursor-pointer items-center gap-3.5 rounded-[22px] px-4 py-3.5 text-left neu-row-hover transition-transform duration-150 active:scale-[0.97]"
									style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);"
									aria-label={user.user.nickname}
									onclick={() => void goto(`/users/${user.user.user_id}`)}
								>
									<div class="relative flex-shrink-0">
										<div class="overflow-hidden rounded-full" style="width: 44px; height: 44px; border: 1.5px solid var(--neu-border);">
											<img src={user.user.avatar_url || ''} alt={user.user.nickname} class="h-full w-full object-cover" />
										</div>
										{#if user.presence_status === 'online'}
											<span class="absolute bottom-0 right-0 rounded-full" style="width: 11px; height: 11px; background: var(--neu-success); border: 2px solid var(--neu-bg);"></span>
										{/if}
									</div>
									<div class="min-w-0 flex-1">
										<p style="font-size: 14px; color: var(--neu-text);">{user.user.nickname}</p>
										<div class="mt-0.5 flex items-center gap-1.5">
											{#if user.presence_status}
												<span style="font-size: 10px; color: {user.presence_status === 'online' ? 'var(--neu-success)' : 'var(--neu-text-muted)'};">
													{user.presence_status === 'online' ? t('common.online') : t('common.offline')}
												</span>
												<span style="font-size: 10px; color: var(--neu-text-muted); opacity: 0.5;">·</span>
											{/if}
											{#if user.tags && user.tags.length > 0}
												<span style="font-size: 10px; color: var(--neu-accent-primary); opacity: 0.8;">{user.tags[0]}</span>
											{/if}
										</div>
									</div>
									<div class="flex flex-col items-end gap-2">
										{#if user.location?.distance != null}
											<span style="font-size: 12px; color: var(--neu-accent-primary);">{formatDistance(user.location.distance)}</span>
										{/if}
										<button
											class="rounded-full px-3.5 py-1.5 outline-none transition-all active:scale-[0.93]"
											style={user.relation?.is_following
												? 'box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border); color: var(--neu-accent-primary); background: var(--neu-bg); font-size: 11px;'
												: 'box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg); font-size: 11px;'}
											onclick={(e) => { e.stopPropagation(); handleFollowToggle(user); }}
										>
											{user.relation?.is_following ? t('common.following') : t('common.follow')}
										</button>
									</div>
								</div>
							{/each}
						{/if}
					</div>
				{/if}

				<!-- Moments Feed -->
				<div class="mb-4 flex items-center justify-between">
					<p class="uppercase" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted);">{t('home.recommendedMoments')}</p>
					<button class="flex items-center gap-0.5 outline-none" style="color: var(--neu-accent-primary);" onclick={refreshAll}>
						<RefreshCw size={12} class={feed.loading ? 'animate-spin' : ''} color="var(--neu-accent-primary)" />
					</button>
				</div>

				{#if feed.loading}
					<div class="flex flex-col gap-3">
						{#each Array(4) as _}
							<div class="rounded-[22px] p-4" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
								<div class="flex items-center gap-3">
									<div class="rounded-full" style="width: 36px; height: 36px; background: var(--neu-border);"></div>
									<div class="flex-1">
										<div class="mb-1 rounded" style="width: 60px; height: 12px; background: var(--neu-border);"></div>
										<div class="rounded" style="width: 100px; height: 10px; background: var(--neu-border);"></div>
									</div>
								</div>
							</div>
						{/each}
					</div>
				{:else if feed.error}
					<EmptyState
						icon={Compass}
						title={t('home.loadMomentsFailed')}
						description={t('home.pleaseCheckNetwork')}
						actionText={t('common.retry')}
						onAction={() => loadFeed(true)}
					/>
				{:else if feed.items.length === 0}
					<EmptyState
						icon={Compass}
						title={t('home.noRecommendations')}
						description={t('home.noRecommendationsDesc')}
					/>
				{:else}
					<div class="columns-1 gap-3 sm:columns-2">
						{#each feed.items as moment (moment.moment_id)}
							<div class="mb-3 break-inside-avoid">
								<MomentCard {moment} />
							</div>
						{/each}
					</div>
					<LoadMoreButton
						loading={feed.loadingMore}
						hasMore={feed.hasMore}
						onLoadMore={() => loadFeed(false)}
					/>
				{/if}
			</div>
		</div>
	</div>
{/if}
