<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { adminPermissionsStore } from '$lib/stores/admin-permissions.svelte';
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
		const pathname = page.url.pathname;
		if (href === '/admin') {
			return pathname === '/admin';
		}
		return pathname.startsWith(href);
	}

	$effect(() => {
		if (browser && authStore.initialized && !authStore.isAuthenticated && !authStore.isRestoring) {
			uiStore.saveRedirectUrl(`${page.url.pathname}${page.url.search}`);
		}
	});
	function handleLoginClick() {
		uiStore.saveRedirectUrl(`${page.url.pathname}${page.url.search}`);
		uiStore.openLoginModal();
	}
</script>

{#if authStore.isAuthenticated && adminPermissionsStore.isAdmin}
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

		<div class="app-shell-frame flex bg-muted/30">
			<!-- Admin sidebar -->
			<aside class="hidden w-64 shrink-0 bg-sidebar text-sidebar-foreground md:block">
				<nav class="space-y-1 p-4">
					<h2 class="mb-4 px-3 text-lg font-semibold">管理后台</h2>
					{#each adminNavItems as item (item.href)}
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
				<Button
					type="button"
					variant="ghost"
					size="icon"
					class="fixed inset-0 z-40 h-full w-full cursor-pointer bg-black/50 p-0 md:hidden border-none shadow-none"
					onclick={closeSidebar}
					onkeydown={(e) => e.key === 'Escape' && closeSidebar()}
					aria-label="关闭侧边栏"
				/>
				<aside 
					class="app-mobile-sidebar fixed left-0 z-50 w-64 bg-sidebar text-sidebar-foreground md:hidden"
				>
					<nav class="space-y-1 p-4" style="padding-bottom: calc(0.75rem + env(safe-area-inset-bottom, 0px));">
						<h2 class="mb-4 px-3 text-lg font-semibold">管理后台</h2>
						{#each adminNavItems as item (item.href)}
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
			<main class="mobile-app-main pb-safe-compact md:mx-4 md:rounded-3xl md:border md:border-border/60 md:bg-card/85 md:p-6 md:shadow-sm md:backdrop-blur-md">
				<div class="mobile-page-shell md:mx-auto md:max-w-6xl">
					{@render children()}
				</div>
			</main>
		</div>
	</div>
{:else if authStore.isAuthenticated && adminPermissionsStore.isLoading}
	<!-- 管理员权限验证中 -->
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => {}} />
		<div class="flex min-h-[calc(100dvh-var(--topbar-height-mobile))] items-center justify-center bg-muted/30 sm:min-h-[calc(100dvh-var(--topbar-height-desktop))]">
			<div class="flex flex-col items-center gap-3 text-muted-foreground">
				<div class="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary"></div>
				<p class="text-sm">验证管理员权限...</p>
			</div>
		</div>
	</div>
{:else if authStore.isAuthenticated && adminPermissionsStore.isForbidden}
	<!-- 已登录但非管理员 -->
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => {}} />
		<div class="app-shell-frame flex bg-muted/30">
			<main class="mobile-app-main pb-safe-compact md:mx-4 md:rounded-3xl md:border md:border-border/60 md:bg-card/85 md:p-6 md:shadow-sm md:backdrop-blur-md">
				<div class="mobile-page-shell md:mx-auto md:max-w-6xl">
					<div class="flex flex-col items-center justify-center py-20 text-center">
						<div class="mb-4 rounded-3xl bg-destructive/10 p-6">
							<Shield class="h-10 w-10 text-destructive" />
						</div>
						<h2 class="mb-2 text-xl font-semibold">无访问权限</h2>
						<p class="mb-6 text-muted-foreground">你的账号不是系统管理员，无法访问管理后台</p>
						<Button class="rounded-2xl px-8" onclick={() => goto('/')}>
							返回首页
						</Button>
					</div>
				</div>
			</main>
		</div>
	</div>
{:else if authStore.isAuthenticated && adminPermissionsStore.isError}
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => {}} />
		<div class="app-shell-frame flex bg-muted/30">
			<main class="mobile-app-main pb-safe-compact md:mx-4 md:rounded-3xl md:border md:border-border/60 md:bg-card/85 md:p-6 md:shadow-sm md:backdrop-blur-md">
				<div class="mobile-page-shell md:mx-auto md:max-w-6xl">
					<div class="flex flex-col items-center justify-center py-20 text-center">
						<div class="mb-4 rounded-3xl bg-muted p-6">
							<Shield class="h-10 w-10 text-muted-foreground" />
						</div>
						<h2 class="mb-2 text-xl font-semibold">权限检查失败</h2>
						<p class="mb-6 text-muted-foreground">暂时无法验证管理员权限，请稍后重试。</p>
						<div class="flex gap-3">
							<Button variant="outline" class="rounded-2xl px-8" onclick={() => adminPermissionsStore.refresh()}>
								重试
							</Button>
							<Button class="rounded-2xl px-8" onclick={() => goto('/')}>
								返回首页
							</Button>
						</div>
					</div>
				</div>
			</main>
		</div>
	</div>
{:else if authStore.isRestoring}
	<!-- 登录态恢复中 -->
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => {}} />
		<div class="flex min-h-[calc(100dvh-var(--topbar-height-mobile))] items-center justify-center bg-muted/30 sm:min-h-[calc(100dvh-var(--topbar-height-desktop))]">
			<div class="flex flex-col items-center gap-3 text-muted-foreground">
				<div class="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary"></div>
				<p class="text-sm">正在恢复登录状态...</p>
			</div>
		</div>
	</div>
{:else}
	<!-- 未登录 -->
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />
		<div class="app-shell-frame flex bg-muted/30">
			<main class="mobile-app-main pb-safe-compact md:mx-4 md:rounded-3xl md:border md:border-border/60 md:bg-card/85 md:p-6 md:shadow-sm md:backdrop-blur-md">
				<div class="mobile-page-shell md:mx-auto md:max-w-6xl">
					<div class="flex flex-col items-center justify-center py-20 text-center">
						<div class="mb-4 rounded-3xl bg-muted p-6">
							<Shield class="h-10 w-10 text-muted-foreground" />
						</div>
						<h2 class="mb-2 text-xl font-semibold">需要登录</h2>
						<p class="mb-6 text-muted-foreground">登录后可访问管理后台</p>
						<Button class="rounded-2xl px-8" onclick={handleLoginClick}>
							立即登录
						</Button>
					</div>
				</div>
			</main>
		</div>
	</div>
{/if}
