<script lang="ts">
	import { browser, dev } from '$app/environment';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { untrack } from 'svelte';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { adminRbacService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
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
	import type { AdminPermissions } from '$lib/types/api';

	let { children } = $props();

	let sidebarOpen = $state(false);
	// 管理员权限状态：null=正在检查, false=非管理员, AdminPermissions=已验证
	let adminState = $state<AdminPermissions | false | null>(null);

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

	// 竞态保护：仅最新一次权限检查的结果会被应用
	let checkVersion = 0;

	// 认证 + 管理员权限检查
	$effect(() => {
		const isAuthenticated = authStore.isAuthenticated;

		if (browser && authStore.initialized && !isAuthenticated && !authStore.isRestoring) {
			uiStore.saveRedirectUrl(`${page.url.pathname}${page.url.search}`);
		}

		// 登录后校验管理员权限
		if (isAuthenticated) {
			const version = ++checkVersion;
			untrack(() => {
				checkAdminPermission(version);
			});
		} else {
			untrack(() => {
				checkVersion++; // 使之前进行中的请求失效
				adminState = null;
			});
		}
	});

	async function checkAdminPermission(version: number) {
		adminState = null; // 重置为"检查中"
		try {
			const res = await adminRbacService.getMyPermissions();
			// 仅应用最新一次请求的结果
			if (version !== checkVersion) return;
			if (isSuccess(res) && res.data) {
				adminState = res.data;
				if (dev) console.log('[Admin] 权限验证通过:', res.data.permissions);
			} else {
				// 10200 权限不足 → 非管理员
				adminState = false;
				if (dev) console.warn('[Admin] 非管理员用户');
			}
		} catch {
			if (version !== checkVersion) return;
			adminState = false;
		}
	}

	function handleLoginClick() {
		uiStore.saveRedirectUrl(`${page.url.pathname}${page.url.search}`);
		uiStore.openLoginModal();
	}
</script>

{#if authStore.isAuthenticated && adminState}
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => (sidebarOpen = !sidebarOpen)} />

		<div class="flex min-h-[calc(100dvh-4rem)] bg-muted/30">
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
			<main class="mx-3 flex-1 rounded-3xl border border-border/60 bg-card/85 p-4 shadow-sm backdrop-blur-md md:mx-4 md:p-6" style="padding-bottom: calc(1rem + env(safe-area-inset-bottom, 0px));">
				<div class="mx-auto w-full max-w-6xl">
					{@render children()}
				</div>
			</main>
		</div>
	</div>
{:else if authStore.isAuthenticated && adminState === null}
	<!-- 管理员权限验证中 -->
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => {}} />
		<div class="flex min-h-[calc(100dvh-4rem)] items-center justify-center bg-muted/30">
			<div class="flex flex-col items-center gap-3 text-muted-foreground">
				<div class="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary"></div>
				<p class="text-sm">验证管理员权限...</p>
			</div>
		</div>
	</div>
{:else if authStore.isAuthenticated && adminState === false}
	<!-- 已登录但非管理员 -->
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => {}} />
		<div class="flex min-h-[calc(100dvh-4rem)] bg-muted/30">
			<main class="mx-3 flex-1 rounded-3xl border border-border/60 bg-card/85 p-4 shadow-sm backdrop-blur-md md:mx-4 md:p-6" style="padding-bottom: calc(1rem + env(safe-area-inset-bottom, 0px));">
				<div class="mx-auto w-full max-w-6xl">
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
{:else if authStore.isRestoring}
	<!-- 登录态恢复中 -->
	<div class="min-h-screen bg-background">
		<Navbar onMenuClick={() => {}} />
		<div class="flex min-h-[calc(100dvh-4rem)] items-center justify-center bg-muted/30">
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
		<div class="flex min-h-[calc(100dvh-4rem)] bg-muted/30">
			<main class="mx-3 flex-1 rounded-3xl border border-border/60 bg-card/85 p-4 shadow-sm backdrop-blur-md md:mx-4 md:p-6" style="padding-bottom: calc(1rem + env(safe-area-inset-bottom, 0px));">
				<div class="mx-auto w-full max-w-6xl">
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
