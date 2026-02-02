<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/stores';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { cn } from '$lib/utils';
	import Navbar from '$lib/components/common/navbar.svelte';
	import { Button } from '$lib/components/ui/button';
	import Users from 'lucide-svelte/icons/users';
	import UsersRound from 'lucide-svelte/icons/users-round';
	import Megaphone from 'lucide-svelte/icons/megaphone';
	import FileText from 'lucide-svelte/icons/file-text';
	import Flag from 'lucide-svelte/icons/flag';
	import Shield from 'lucide-svelte/icons/shield';
	import Settings from 'lucide-svelte/icons/settings';
	import LayoutDashboard from 'lucide-svelte/icons/layout-dashboard';

	let { children } = $props();

	let sidebarOpen = $state(false);

	function closeSidebar() {
		sidebarOpen = false;
	}

	const adminNavItems = [
		{ href: '/admin', label: '概览', icon: LayoutDashboard },
		{ href: '/admin/users', label: '用户管理', icon: Users },
		{ href: '/admin/groups', label: '群组管理', icon: UsersRound },
		{ href: '/admin/announcements', label: '公告管理', icon: Megaphone },
		{ href: '/admin/audit', label: '审计日志', icon: FileText },
		{ href: '/admin/reports', label: '举报审核', icon: Flag },
		{ href: '/admin/rbac', label: '权限管理', icon: Shield },
		{ href: '/admin/config', label: '系统配置', icon: Settings }
	];

	function isActive(href: string): boolean {
		const pathname = $page.url.pathname;
		if (href === '/admin') {
			return pathname === '/admin';
		}
		return pathname.startsWith(href);
	}

	// 认证检查
	$effect(() => {
		if (browser && authStore.initialized && !authStore.isAuthenticated) {
			uiStore.saveRedirectUrl(`${$page.url.pathname}${$page.url.search}`);
		}
	});

	function handleLoginClick() {
		uiStore.saveRedirectUrl(`${$page.url.pathname}${$page.url.search}`);
		uiStore.openLoginModal();
	}
</script>

{#if authStore.isAuthenticated}
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

		<div class="flex min-h-[calc(100dvh-4rem)] bg-muted/30">
			<!-- Admin sidebar -->
			<aside class="hidden w-64 shrink-0 bg-sidebar text-sidebar-foreground md:block">
				<nav class="space-y-1 p-4">
					<h2 class="mb-4 px-3 text-lg font-semibold">管理后台</h2>
					{#each adminNavItems as item}
						<a
							href={item.href}
							class={cn(
								'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
								isActive(item.href)
									? 'bg-primary text-primary-foreground'
									: 'hover:bg-accent hover:text-accent-foreground'
							)}
						>
							<item.icon class="h-5 w-5" />
							{item.label}
						</a>
					{/each}
				</nav>
			</aside>

			<!-- Mobile sidebar overlay -->
			{#if sidebarOpen}
				<button
					type="button"
					class="fixed inset-0 z-40 bg-black/50 md:hidden"
					onclick={closeSidebar}
					onkeydown={(e) => e.key === 'Escape' && closeSidebar()}
					aria-label="关闭侧边栏"
				></button>
				<aside 
					class="fixed left-0 top-14 z-50 w-64 bg-sidebar text-sidebar-foreground md:hidden"
					style="height: calc(100dvh - 3.5rem - env(safe-area-inset-bottom, 0px));"
				>
					<nav class="space-y-1 p-4" style="padding-bottom: calc(1rem + env(safe-area-inset-bottom, 0px));">
						<h2 class="mb-4 px-3 text-lg font-semibold">管理后台</h2>
						{#each adminNavItems as item}
							<a
								href={item.href}
								class={cn(
									'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
									isActive(item.href)
										? 'bg-primary text-primary-foreground'
										: 'hover:bg-accent hover:text-accent-foreground'
								)}
								onclick={closeSidebar}
							>
								<item.icon class="h-5 w-5" />
								{item.label}
							</a>
						{/each}
					</nav>
				</aside>
			{/if}

			<!-- Main content -->
			<main class="flex-1 rounded-2xl bg-card/70 p-4 shadow-sm shadow-black/5 mx-3 md:mx-4 md:p-6" style="padding-bottom: calc(1rem + env(safe-area-inset-bottom, 0px));">
				<div class="mx-auto w-full max-w-6xl">
					{@render children()}
				</div>
			</main>
		</div>
	</div>
{:else}
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

		<div class="flex min-h-[calc(100dvh-4rem)] bg-muted/30">
			<main class="flex-1 rounded-2xl bg-card/70 p-4 shadow-sm shadow-black/5 mx-3 md:mx-4 md:p-6" style="padding-bottom: calc(1rem + env(safe-area-inset-bottom, 0px));">
				<div class="mx-auto w-full max-w-6xl">
					<div class="flex flex-col items-center justify-center py-20 text-center">
						<div class="mb-4 rounded-full bg-muted p-6">
							<Shield class="h-10 w-10 text-muted-foreground" />
						</div>
						<h2 class="mb-2 text-xl font-semibold">需要登录</h2>
						<p class="mb-6 text-muted-foreground">登录后可访问管理后台</p>
						<Button class="rounded-full px-8" onclick={handleLoginClick}>
							立即登录
						</Button>
					</div>
				</div>
			</main>
		</div>
	</div>
{/if}
