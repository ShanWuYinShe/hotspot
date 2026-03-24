<script lang="ts">
	import { browser } from "$app/environment";
	import { page } from "$app/state";
	import { authStore } from "$lib/stores/auth.svelte";
	import { uiStore } from "$lib/stores/ui.svelte";
	import Navbar from "$lib/components/common/navbar.svelte";
	import Sidebar from "$lib/components/common/sidebar.svelte";
	import { Button } from "$lib/components/ui/button";

	let { children } = $props();

	let sidebarOpen = $state(false);

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
			uiStore.saveRedirectUrl(`${page.url.pathname}${page.url.search}`);
		}
	});

	function handleLoginClick() {
		uiStore.saveRedirectUrl(`${page.url.pathname}${page.url.search}`);
		uiStore.openLoginModal();
	}
</script>

<div class="min-h-screen bg-background">
	<Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

	<div
		class="flex h-[calc(100dvh-4rem)] min-h-[calc(100dvh-4rem)] overflow-hidden bg-muted/30"
	>
		<Sidebar class="hidden w-64 shrink-0 md:block" />

		{#if sidebarOpen}
			<Button
				type="button"
				variant="ghost"
				size="icon"
				class="fixed inset-0 z-40 h-full w-full cursor-pointer bg-black/50 p-0 md:hidden border-none shadow-none"
				onclick={closeSidebar}
				onkeydown={(e) => e.key === "Escape" && closeSidebar()}
				aria-label="关闭侧边栏"
			/>
			<div
				class="fixed left-0 top-14 z-50 w-64 md:hidden"
				style="height: calc(100dvh - 3.5rem - env(safe-area-inset-bottom, 0px));"
			>
				<Sidebar class="h-full bg-sidebar" />
			</div>
		{/if}

		<main
			class="mobile-app-main md:mx-4 md:rounded-3xl md:border md:border-border/60 md:bg-card/85 md:p-6 md:shadow-sm md:backdrop-blur-md"
			style="padding-bottom: calc(1rem + env(safe-area-inset-bottom, 0px));"
		>
			<div class="mobile-page-shell md:mx-auto md:max-w-7xl">
				{#if authStore.isAuthenticated}
					{@render children()}
				{:else if authStore.isRestoring}
					<div
						class="flex flex-col items-center justify-center py-20 gap-3"
					>
						<div
							class="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary"
						></div>
						<p class="text-sm text-muted-foreground">
							正在恢复登录状态…
						</p>
					</div>
				{:else}
					<div
						class="flex flex-col items-center justify-center py-20"
					>
						<div class="mb-4 rounded-3xl bg-muted p-6">
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
								<rect
									width="18"
									height="11"
									x="3"
									y="11"
									rx="2"
									ry="2"
								/>
								<path d="M7 11V7a5 5 0 0 1 10 0v4" />
							</svg>
						</div>
						<h2 class="mb-2 text-xl font-semibold">需要登录</h2>
						<p class="mb-6 text-muted-foreground">
							登录后即可访问此页面
						</p>
						<Button class="rounded-2xl px-8" onclick={handleLoginClick}>
							立即登录
						</Button>
					</div>
				{/if}
			</div>
		</main>
	</div>
</div>
