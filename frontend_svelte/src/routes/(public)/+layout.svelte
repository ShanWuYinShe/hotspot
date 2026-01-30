<script lang="ts">
	import Navbar from '$lib/components/common/navbar.svelte';
	import Sidebar from '$lib/components/common/sidebar.svelte';

	let { children } = $props();

	let sidebarOpen = $state(false);

	function closeSidebar() {
		sidebarOpen = false;
	}
</script>

<div class="min-h-screen bg-background">
	<Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

	<div class="flex h-[calc(100vh-4rem)] min-h-[calc(100vh-4rem)] overflow-hidden bg-muted/30">
		<!-- Desktop sidebar -->
		<Sidebar class="hidden w-64 shrink-0 md:block" />

		<!-- Mobile sidebar overlay -->
		{#if sidebarOpen}
			<button
				type="button"
				class="fixed inset-0 z-40 bg-black/50 md:hidden"
				onclick={closeSidebar}
				onkeydown={(e) => e.key === 'Escape' && closeSidebar()}
				aria-label="关闭侧边栏"
			></button>
			<Sidebar class="fixed left-0 top-14 z-50 h-[calc(100vh-3.5rem)] w-64 bg-sidebar md:hidden" />
		{/if}

		<!-- Main content -->
		<main class="flex-1 overflow-y-auto rounded-2xl bg-card/80 p-4 shadow-lg shadow-black/10 ring-1 ring-border/40 mx-3 md:mx-4 md:p-6">
			<div class="mx-auto w-full max-w-7xl">
				{@render children()}
			</div>
		</main>
	</div>
</div>
