<script lang="ts">
	import { browser } from '$app/environment';
	import { t } from '$lib/i18n';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { mode, toggleMode } from 'mode-watcher';
	import { platformStore } from '$lib/stores/platform.svelte';
	import MobileBottomNav from '$lib/components/common/mobile-bottom-nav.svelte';
	import TabletBottomNav from '$lib/components/common/tablet-bottom-nav.svelte';
	import Search from 'lucide-svelte/icons/search';
	import Sun from 'lucide-svelte/icons/sun';
	import Moon from 'lucide-svelte/icons/moon';
	import Bell from 'lucide-svelte/icons/bell';
	import Settings from 'lucide-svelte/icons/settings';
	import {
		notificationStore,
	} from '$lib/stores/notification.svelte';
	import Sidebar from '$lib/components/common/sidebar.svelte';

	let { children } = $props();

	$effect(() => {
		if (
			browser &&
			authStore.initialized &&
			!authStore.isAuthenticated &&
			!authStore.isRestoring
		) {
			uiStore.saveRedirectUrl();
		}
	});

	const currentMode = $derived(mode.current);
</script>

{#if platformStore.isMobile}
	<!-- Mobile: full-screen content + edge-to-edge bottom nav -->
	<div
		class="flex h-[100dvh] w-full flex-col overflow-hidden transition-colors duration-500"
		style="background-color: var(--neu-bg); color: var(--neu-text);"
	>
		<main class="mobile-app-main no-scrollbar scroll-smooth page-enter">
			<div class="mobile-shell-content">
				{@render children()}
			</div>
		</main>
		<MobileBottomNav />
	</div>
{:else if platformStore.isTablet}
	<!-- Tablet: full-width content + floating glass pill bottom nav -->
	<div
		class="flex h-[100dvh] w-full flex-col overflow-hidden transition-colors duration-500"
		style="background-color: var(--neu-bg); color: var(--neu-text);"
	>
		<main class="flex-1 overflow-y-auto overflow-x-hidden no-scrollbar" style="background-color: var(--neu-bg);">
			{@render children()}
		</main>
		<TabletBottomNav />
	</div>
{:else}
	<!-- Desktop: header + sidebar + main card (≥1280px) -->
	<div
		class="flex h-screen w-screen flex-col overflow-hidden transition-colors duration-500"
		style="background-color: var(--page-bg); color: var(--neu-text);"
	>
		<!-- Header — 52px -->
		<header class="desktop-header">
			<!-- Logo -->
			<a href="/" class="mr-3 flex cursor-pointer items-center gap-2 select-none">
				<img src="/logo_32.png" alt="HotSpot" class="h-7 w-7 rounded-full object-cover" />
				<span class="tracking-tight" style="font-size: 17px; color: var(--neu-text);">HotSpot</span>
			</a>

			<div class="flex-1"></div>

			<!-- Right actions -->
			<div class="flex items-center gap-0.5">
				<button class="desktop-header__btn" aria-label={t('nav.openSearch')} onclick={() => uiStore.openSearchModal()}>
					<Search size={17} color="var(--neu-text-muted)" />
				</button>
				<button class="desktop-header__btn" aria-label={t('sidebar.toggleTheme')} onclick={toggleMode}>
					{#if currentMode === 'dark'}
						<Sun size={17} color="var(--neu-text-muted)" />
					{:else}
						<Moon size={17} color="var(--neu-text-muted)" />
					{/if}
				</button>
				<button
					class="mx-1.5 flex items-center justify-center rounded-full outline-none transition-shadow duration-200"
					style="width: 32px; height: 32px; background: var(--neu-accent-primary); box-shadow: 0 4px 12px var(--neu-glow);"
					onclick={() => uiStore.openCreateSheet()}
					aria-label={t('nav.publish')}
				>
					<svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="var(--neu-active-text)" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>
				</button>
				<button class="desktop-header__btn" aria-label={t('sidebar.notifications')} onclick={() => void goto('/notifications')}>
					<Bell size={17} color="var(--neu-text-muted)" />
				</button>
				<button class="desktop-header__btn" aria-label={t('sidebar.settingsLabel')} onclick={() => void goto('/settings')}>
					<Settings size={17} color="var(--neu-text-muted)" />
				</button>
				{#if authStore.isAuthenticated}
					<button type="button" class="relative ml-1.5 cursor-pointer" style="width: 32px; height: 32px; padding: 0; border: none; background: none;" aria-label="个人主页" onclick={() => void goto('/profile')}>
						<img
							src={authStore.currentAccount?.avatar_url || ''}
							alt={authStore.currentAccount?.nickname || "Profile"}
							class="h-full w-full rounded-full object-cover"
							style="border: 1.5px solid var(--neu-border);"
						/>
						{#if notificationStore.unreadCount > 0}
							<div class="desktop-header__badge" style="top: -3px; right: -3px;">
								{notificationStore.unreadCount}
							</div>
						{/if}
					</button>
				{/if}
			</div>
		</header>

		<!-- Body: sidebar + main content -->
		<div class="flex flex-1 gap-4 overflow-hidden p-4">
			<!-- Sidebar -->
			<Sidebar />

			<!-- Main content card -->
			<main class="desktop-main-card page-enter">
				<div class="h-full overflow-y-auto no-scrollbar">
					{@render children()}
				</div>
			</main>
		</div>
	</div>
{/if}
