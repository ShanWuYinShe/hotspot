<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import Navbar from '$lib/components/common/navbar.svelte';
	import Sidebar from '$lib/components/common/sidebar.svelte';

	let { children } = $props();

	let sidebarOpen = $state(false);

	function closeSidebar() {
		sidebarOpen = false;
	}

	$effect(() => {
		if (browser && authStore.initialized && !authStore.isAuthenticated) {
			uiStore.saveRedirectUrl(page.url.pathname);
		}
	});

	function handleLoginClick() {
		uiStore.saveRedirectUrl(page.url.pathname);
		uiStore.openLoginModal();
	}
</script>

<div class="min-h-screen bg-background">
	<Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

	<div class="flex h-[calc(100dvh-4rem)] min-h-[calc(100dvh-4rem)] overflow-hidden bg-muted/30">
		<Sidebar class="hidden w-64 shrink-0 md:block" />

		{#if sidebarOpen}
			<button
				type="button"
				class="fixed inset-0 z-40 bg-black/50 md:hidden"
				onclick={closeSidebar}
				onkeydown={(e) => e.key === 'Escape' && closeSidebar()}
				aria-label="关闭侧边栏"
			></button>
			<div
				class="fixed left-0 top-14 z-50 w-64 md:hidden"
				style="height: calc(100dvh - 3.5rem - env(safe-area-inset-bottom, 0px));"
			>
				<Sidebar class="h-full bg-sidebar" />
			</div>
		{/if}

		<main
			class="flex-1 overflow-y-auto rounded-2xl bg-card/80 p-4 shadow-lg shadow-black/10 ring-1 ring-border/40 mx-3 md:mx-4 md:p-6"
			style="padding-bottom: calc(1rem + env(safe-area-inset-bottom, 0px));"
		>
			<div class="mx-auto w-full max-w-7xl">
				{#if authStore.isAuthenticated}
					{@render children()}
				{:else}
					<div class="flex flex-col items-center justify-center py-20">
						<div class="mb-4 rounded-full bg-muted p-6">
							<svg
								xmlns="http://www.w3.org/2000/svg"
								width="48"
								height="48"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
								class="text-muted-foreground"
							>
								<rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
								<path d="M7 11V7a5 5 0 0 1 10 0v4" />
							</svg>
						</div>
						<h2 class="mb-2 text-xl font-semibold">需要登录</h2>
						<p class="mb-6 text-muted-foreground">登录后即可访问此页面</p>
						<button
							class="inline-flex items-center justify-center rounded-full bg-primary px-8 py-3 text-sm font-medium text-primary-foreground shadow-lg shadow-primary/25 transition-all hover:shadow-xl hover:shadow-primary/30"
							onclick={handleLoginClick}
						>
							立即登录
						</button>
					</div>
				{/if}
			</div>
		</main>
	</div>
</div>
