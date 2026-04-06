<script lang="ts">
	import { browser } from "$app/environment";
	import { onMount } from "svelte";
	import { t } from '$lib/i18n';
	import { authStore } from "$lib/stores/auth.svelte";
	import { uiStore } from "$lib/stores/ui.svelte";
	import Navbar from "$lib/components/common/navbar.svelte";
	import Sidebar from "$lib/components/common/sidebar.svelte";
	import MobileBottomNav from "$lib/components/common/mobile-bottom-nav.svelte";
	import { Button } from "$lib/components/ui/button";

	let { children } = $props();

	let sidebarOpen = $state(false);
	let isMobile = $state(false);

	function closeSidebar() {
		sidebarOpen = false;
	}

	$effect(() => {
		if (
			browser &&
			authStore.initialized &&
			!authStore.isAuthenticated &&
			!authStore.isRestoring
		) {
			uiStore.saveCurrentRedirectUrl();
		}
	});

	function handleLoginClick() {
		uiStore.openLoginModalWithRedirect({});
	}

	onMount(() => {
		const mediaQuery = window.matchMedia('(max-width: 767px)');
		const update = () => {
			isMobile = mediaQuery.matches;
		};
		update();
		mediaQuery.addEventListener('change', update);
		return () => mediaQuery.removeEventListener('change', update);
	});
</script>

<div class="min-h-screen bg-background">
	{#if isMobile}
		<div class="mobile-shell-frame">
			<Navbar showMenu={false} />
			<main class="mobile-app-main mobile-shell-content">
				<div class="mobile-page-shell">
					{#if authStore.isAuthenticated}
						{@render children()}
					{:else if authStore.isRestoring}
						<div class="flex flex-col items-center justify-center gap-3 py-20">
							<div class="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary"></div>
							<p class="text-sm text-muted-foreground">{t('auth.restoringSession')}</p>
						</div>
					{:else}
						<div class="flex flex-col items-center justify-center py-20">
							<div class="mb-4 rounded-[1.75rem] bg-card p-6 shadow-[10px_10px_22px_var(--shadow-dark),-10px_-10px_22px_var(--shadow-light)]">
								<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-muted-foreground">
									<rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
									<path d="M7 11V7a5 5 0 0 1 10 0v4" />
								</svg>
							</div>
							<h2 class="mb-2 text-xl font-semibold">{t('auth.loginRequired')}</h2>
							<p class="mb-6 text-muted-foreground">{t('auth.loginRequiredDescription')}</p>
							<Button class="rounded-[1.35rem] px-8" onclick={handleLoginClick}>{t('common.loginNow')}</Button>
						</div>
					{/if}
				</div>
			</main>
			<MobileBottomNav />
		</div>
	{:else}
		<div>
			<Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

			<div class="app-shell-frame flex overflow-hidden bg-muted/20">
				<Sidebar class="hidden w-64 shrink-0 md:block" />

				{#if sidebarOpen}
					<Button
						type="button"
						variant="ghost"
						size="icon"
						class="fixed inset-0 z-40 h-full w-full cursor-pointer bg-black/50 p-0 md:hidden border-none shadow-none"
						onclick={closeSidebar}
						onkeydown={(e) => e.key === "Escape" && closeSidebar()}
						aria-label={t('nav.closeSidebar')}
					/>
					<div class="app-mobile-sidebar fixed left-0 z-50 w-64 md:hidden">
						<Sidebar class="h-full bg-sidebar" />
					</div>
				{/if}

				<main class="mobile-app-main pb-safe-compact md:mx-4 md:rounded-3xl md:border md:border-border/60 md:bg-card/85 md:p-6 md:shadow-sm md:backdrop-blur-md">
					<div class="mobile-page-shell md:mx-auto md:max-w-7xl">
						{#if authStore.isAuthenticated}
							{@render children()}
						{:else if authStore.isRestoring}
							<div class="flex flex-col items-center justify-center gap-3 py-20">
								<div class="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary"></div>
								<p class="text-sm text-muted-foreground">{t('auth.restoringSession')}</p>
							</div>
						{:else}
							<div class="flex flex-col items-center justify-center py-20">
								<div class="mb-4 rounded-3xl bg-muted p-6">
									<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-muted-foreground">
										<rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
										<path d="M7 11V7a5 5 0 0 1 10 0v4" />
									</svg>
								</div>
								<h2 class="mb-2 text-xl font-semibold">{t('auth.loginRequired')}</h2>
								<p class="mb-6 text-muted-foreground">{t('auth.loginRequiredDescription')}</p>
								<Button class="rounded-2xl px-8" onclick={handleLoginClick}>{t('common.loginNow')}</Button>
							</div>
						{/if}
					</div>
				</main>
			</div>
		</div>
	{/if}
</div>

