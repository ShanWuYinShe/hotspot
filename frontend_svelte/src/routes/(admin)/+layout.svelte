<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { t } from '$lib/i18n';
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

	const adminNavItems = $derived([
		{ href: '/admin', label: t('admin.overview'), icon: LayoutDashboard },
		{ href: '/admin/users', label: t('admin.users'), icon: Users },
		{ href: '/admin/groups', label: t('admin.groups'), icon: UsersRound },
		{ href: '/admin/announcements', label: t('admin.announcements'), icon: Megaphone },
		{ href: '/admin/audit', label: t('admin.audit'), icon: FileText },
		{ href: '/admin/reports', label: t('admin.reports'), icon: Flag },
		{ href: '/admin/rbac', label: t('admin.rbac'), icon: Shield },
		{ href: '/admin/config', label: t('admin.config'), icon: Settings }
	]);

	function isActive(href: string): boolean {
		const pathname = page.url.pathname;
		if (href === '/admin') {
			return pathname === '/admin';
		}
		return pathname.startsWith(href);
	}

	$effect(() => {
		if (browser && authStore.initialized && !authStore.isAuthenticated && !authStore.isRestoring) {
			uiStore.saveCurrentRedirectUrl();
		}
	});
	function handleLoginClick() {
		uiStore.openLoginModalWithRedirect({});
	}
</script>

{#if authStore.isAuthenticated && adminPermissionsStore.isAdmin}
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

		<div class="app-shell-frame flex bg-muted/30">
			<!-- Admin sidebar -->
			<aside class="hidden w-64 shrink-0 bg-sidebar text-sidebar-foreground md:block">
				<nav class="space-y-1 p-4">
					<h2 class="mb-4 px-3 text-lg font-semibold">{t('nav.adminPanel')}</h2>
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
					aria-label={t('nav.closeSidebar')}
				/>
				<aside 
					class="app-mobile-sidebar fixed left-0 z-50 w-64 bg-sidebar text-sidebar-foreground md:hidden"
				>
					<nav class="space-y-1 p-4" style="padding-bottom: calc(0.75rem + env(safe-area-inset-bottom, 0px));">
						<h2 class="mb-4 px-3 text-lg font-semibold">{t('nav.adminPanel')}</h2>
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
				<p class="text-sm">{t('admin.checkingPermissions')}</p>
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
						<h2 class="mb-2 text-xl font-semibold">{t('admin.accessDenied')}</h2>
						<p class="mb-6 text-muted-foreground">{t('admin.accessDeniedDescription')}</p>
						<Button class="rounded-2xl px-8" onclick={() => goto('/')}>
							{t('common.backHome')}
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
						<h2 class="mb-2 text-xl font-semibold">{t('admin.permissionCheckFailed')}</h2>
						<p class="mb-6 text-muted-foreground">{t('admin.permissionCheckFailedDescription')}</p>
						<div class="flex gap-3">
							<Button variant="outline" class="rounded-2xl px-8" onclick={() => adminPermissionsStore.refresh()}>
								{t('common.retry')}
							</Button>
							<Button class="rounded-2xl px-8" onclick={() => goto('/')}>
								{t('common.backHome')}
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
				<p class="text-sm">{t('auth.restoringSession')}</p>
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
						<h2 class="mb-2 text-xl font-semibold">{t('auth.loginRequired')}</h2>
						<p class="mb-6 text-muted-foreground">{t('admin.loginRequiredDescription')}</p>
						<Button class="rounded-2xl px-8" onclick={handleLoginClick}>
							{t('common.loginNow')}
						</Button>
					</div>
				</div>
			</main>
		</div>
	</div>
{/if}
