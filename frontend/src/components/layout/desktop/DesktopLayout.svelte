<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		Home, Users, Rss, MessageCircle, Bell, ImageIcon,
		Heart, Search, Settings, Plus, Navigation2, UserRound, PanelLeftClose, PanelLeftOpen
	} from '@lucide/svelte';
	import { cn } from '@/lib/utils';
	import { auth, requireAuth } from '@/stores/auth.svelte';
	import { composer, openComposer, closeComposer } from '@/stores/composer.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import AppearanceMenu from '@/components/layout/AppearanceMenu.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import CreateSheet from '@/components/layout/CreateSheet.svelte';
	import SidebarNavItem from '@/components/layout/desktop/SidebarNavItem.svelte';
	import LocationCard from '@/components/layout/desktop/LocationCard.svelte';
	import RoutedContentViewport from '@/components/layout/page/RoutedContentViewport.svelte';
	import { ROUTES, getDesktopNavId, type DesktopNavId } from '@/lib/routes';
	import imgHotSpot from '@/assets/9254a1e0e11b516a6c1aecb80d92b66f2caa243e.png';

	// 桌面全屏布局（≥1024px）：52px 顶栏 + 可折叠侧栏 + 主内容卡片 + 发布模态。
	// 迁移自旧版 DesktopLayout。

	let { children }: { children: Snippet } = $props();

	interface NavItem {
		id: DesktopNavId;
		icon: Component;
		label: string;
		path: string;
	}

	const TOP_NAV: NavItem[] = [
		{ id: 'home', icon: Home, label: '首页', path: ROUTES.home },
		{ id: 'nearby-people', icon: Users, label: '附近的人', path: ROUTES.nearbyPeople },
		{ id: 'nearby-groups', icon: UserRound, label: '附近群组', path: ROUTES.nearbyGroups },
		{ id: 'nearby-moments', icon: Rss, label: '附近瞬间', path: ROUTES.nearbyMoments }
	];

	const PERSONAL_NAV: NavItem[] = [
		{ id: 'messages', icon: MessageCircle, label: '消息', path: ROUTES.messages },
		{ id: 'moments', icon: ImageIcon, label: '我的瞬间', path: ROUTES.meSection('moments') },
		{ id: 'my-groups', icon: Users, label: '我的群组', path: ROUTES.meSection('groups') },
		{ id: 'relationships', icon: Heart, label: '关系', path: ROUTES.meSection('relationships') }
	];

	const LAST_DESKTOP_NAV_KEY = 'hotspot:last-desktop-nav';
	const DESKTOP_NAV_IDS: DesktopNavId[] = [
		'home', 'nearby-people', 'nearby-groups', 'nearby-moments',
		'messages', 'moments', 'my-groups', 'relationships'
	];

	function readLastDesktopNav(): DesktopNavId | null {
		try {
			const value = window.sessionStorage.getItem(LAST_DESKTOP_NAV_KEY);
			return DESKTOP_NAV_IDS.includes(value as DesktopNavId) ? (value as DesktopNavId) : null;
		} catch {
			return null;
		}
	}

	function persistLastDesktopNav(id: DesktopNavId) {
		try {
			window.sessionStorage.setItem(LAST_DESKTOP_NAV_KEY, id);
		} catch {
			// Storage can be unavailable in private or embedded browser contexts.
		}
	}

	const pathname = $derived(page.url.pathname);
	const searchString = $derived(page.url.search.slice(1));
	const showCreate = $derived(composer.composerOpen);
	// 全画布页面（消息分栏 / 地图探索）：主区域不滚动，画布固定为视口高度
	const isFullCanvas = $derived(([ROUTES.messages, ROUTES.map] as string[]).includes(pathname));

	/* Keep the last meaningful rail context while viewing utility/detail routes. */
	let activeNavId = $state<DesktopNavId>(getDesktopNavId(page.url.pathname, page.url.search.slice(1)) ?? readLastDesktopNav() ?? 'home');
	let isSidebarCollapsed = $state(false);

	// 仅显式 rail 路由替换当前 rail 上下文
	$effect(() => {
		const resolvedNav = getDesktopNavId(pathname, searchString);
		if (resolvedNav) {
			activeNavId = resolvedNav;
			persistLastDesktopNav(resolvedNav);
		}
	});

	/** Navigate only when authenticated; otherwise prompt login (anonymous browsing strategy). */
	const guardNav = (label: string, path: string) => {
		if (auth.status === 'authenticated') {
			void goto(path);
			return;
		}
		requireAuth({ action: label, run: () => void goto(path) });
	};

	const handleNavClick = (item: NavItem) => {
		// Personal/private destinations require auth; public discovery does not.
		const requiresAuth =
			item.id === 'messages' || item.id === 'moments' || item.id === 'my-groups' || item.id === 'relationships';
		if (requiresAuth && auth.status !== 'authenticated') {
			requireAuth({
				action: item.label,
				run: () => {
					activeNavId = item.id;
					persistLastDesktopNav(item.id);
					void goto(item.path);
				}
			});
			return;
		}
		activeNavId = item.id;
		persistLastDesktopNav(item.id);
		void goto(item.path);
	};
</script>

<div
	class="flex flex-col h-screen w-screen overflow-hidden transition-colors duration-500 font-sans"
	style="background-color:var(--page-bg);color:var(--neu-text);"
>
	<!-- HEADER — 52px, full width -->
	<header
		class="relative z-30 flex-shrink-0 flex items-center px-5 gap-2"
		style="height:52px;background-color:var(--page-bg);border-bottom:1px solid var(--neu-border);"
	>
		<!-- Logo -->
		<button
			type="button"
			class="neu-hover-btn flex items-center gap-2 mr-3 select-none rounded-full neu-focus-ring focus-visible:ring-offset-[var(--page-bg)]"
			onclick={() => {
				activeNavId = 'home';
				void goto(ROUTES.home);
			}}
		>
			<img src={imgHotSpot} alt="HotSpot Logo" class="rounded-full object-cover flex-shrink-0" style="width:28px;height:28px;" />
			<span class="tracking-tight" style="font-size:17px;color:var(--neu-text);">HotSpot</span>
		</button>

		<!-- Spacer -->
		<div class="flex-1"></div>

		<!-- Right Actions -->
		<div class="flex items-center gap-0.5">
			<ActionIconButton
				icon={Search}
				label="搜索"
				onClick={() => void goto(ROUTES.search)}
				variant="ghost"
				size={36}
				class="neu-header-action"
			/>

			<ActionIconButton
				icon={Plus}
				label="发布"
				onClick={() => {
					if (auth.status === 'authenticated') {
						openComposer();
						return;
					}
					requireAuth({ action: '发布动态', run: openComposer });
				}}
				variant="ghost"
				size={36}
				class="neu-header-action"
			/>

			<ActionIconButton icon={Bell} label="通知" onClick={() => guardNav('查看通知', ROUTES.notifications)} variant="ghost" size={36} class="neu-header-action" />

			<AppearanceMenu />

			<ActionIconButton icon={Settings} label="设置" onClick={() => void goto(ROUTES.settings)} variant="ghost" size={36} class="neu-header-action" />

			{#if auth.user}
				<button
					type="button"
					class="neu-hover-btn neu-scale-btn relative ml-1.5 flex-shrink-0 rounded-full neu-focus-ring focus-visible:ring-offset-[var(--page-bg)]"
					style="width:32px;height:32px;"
					aria-label="打开个人主页"
					onclick={() => guardNav('打开个人主页', ROUTES.me)}
				>
					<img
						src={auth.user.avatar_url || ''}
						alt={auth.user.nickname || '用户'}
						class="w-full h-full rounded-full object-cover"
						style="border:1.5px solid var(--neu-border);"
					/>
				</button>
			{:else}
				<button
					type="button"
					class="neu-hover-btn ml-1.5 flex-shrink-0 rounded-full px-3 neu-focus-ring"
					style="height:32px;background:var(--neu-accent-primary);color:white;font-size:12px;"
					aria-label="登录"
					onclick={() => void goto(ROUTES.login)}
				>
					登录
				</button>
			{/if}
		</div>
	</header>

	<!-- BODY — sidebar + main content -->
	<div class="desktop-workspace relative z-0 flex flex-1 overflow-hidden" style="gap:0px;">
		<!-- Sidebar (224px expanded / 72px collapsed, fixed width) -->
		<aside
			class="relative flex flex-col flex-shrink-0 overflow-visible box-border transition-[width] duration-200"
			style="width:{isSidebarCollapsed ? 72 : 224}px;padding:0.75rem;"
		>
			<!-- Edge-mounted collapse control: leaves the navigation rhythm untouched -->
			<button
				type="button"
				class="neu-hover-btn absolute -right-4 top-1/2 z-20 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full border outline-none transition-colors hover:bg-[var(--neu-accent-soft)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
				style="color:var(--neu-text-muted);background:var(--page-bg);border-color:var(--neu-border);box-shadow:0 2px 8px var(--neu-shadow-dark);"
				aria-label={isSidebarCollapsed ? '展开侧边栏' : '收起侧边栏'}
				aria-pressed={isSidebarCollapsed}
				title={isSidebarCollapsed ? '展开侧边栏' : '收起侧边栏'}
				onclick={() => (isSidebarCollapsed = !isSidebarCollapsed)}
			>
				{#if isSidebarCollapsed}
					<PanelLeftOpen size={16} strokeWidth={1.7} />
				{:else}
					<PanelLeftClose size={16} strokeWidth={1.7} />
				{/if}
			</button>
			<!-- Scrollable nav region — takes all available space -->
			<div class="flex-1 flex flex-col overflow-y-auto no-scrollbar min-h-0">
			<!-- Section 1: 发现导航 -->
			<div class="flex flex-col gap-0.5 {isSidebarCollapsed ? 'items-center' : ''}">
					{#each TOP_NAV as item (item.id)}
						<SidebarNavItem {item} isActive={activeNavId === item.id} onClick={() => handleNavClick(item)} collapsed={isSidebarCollapsed} />
					{/each}
				</div>

				<!-- Divider -->
				<div class="my-3 self-center" style="height:1px;width:{isSidebarCollapsed ? 24 : '100%'};background:var(--neu-border);"></div>

				<!-- Section 2 label -->
				<p
					class={isSidebarCollapsed ? 'sr-only' : 'px-3 mb-2'}
					style="font-size:10px;font-weight:600;letter-spacing:0.12em;text-transform:uppercase;color:var(--neu-text-muted);opacity:0.65;"
				>
					个人中心
				</p>

				<!-- Section 2: Personal nav -->
				<div class="flex flex-col gap-0.5 {isSidebarCollapsed ? 'items-center' : ''}">
					{#each PERSONAL_NAV as item (item.id)}
						<SidebarNavItem {item} isActive={activeNavId === item.id} onClick={() => handleNavClick(item)} collapsed={isSidebarCollapsed} />
					{/each}
				</div>
			</div>

			<!-- Location Anchor — ALWAYS pinned to bottom -->
			<div class="flex flex-shrink-0 pt-3 {isSidebarCollapsed ? 'justify-center' : ''}">
				<LocationCard collapsed={isSidebarCollapsed} onClick={() => void goto(ROUTES.map)} />
			</div>
		</aside>

		<!-- Main Content Card — RoutedContentViewport 是唯一滚动 owner（desktopPane 激活 sticky-header page-bg 强制规则）。
		     全画布页（消息分栏 / 地图探索）禁用滚动，画布固定视口高度。 -->
		<div class="desktop-main-panel flex-1 min-w-0 overflow-hidden border-l transition-colors duration-500" style="border-color:var(--neu-border);">
			<RoutedContentViewport
				desktopPane
				class="desktop-main-scroll"
				contentClassName={cn('desktop-page-canvas', isFullCanvas && 'h-full')}
				style={isFullCanvas ? 'overflow:hidden;' : undefined}
			>
				{@render children()}
			</RoutedContentViewport>
		</div>
	</div>

	<!-- CREATE MODAL — centered dialog on desktop -->
	<Dialog.Root open={showCreate} onOpenChange={(next) => (next ? openComposer() : closeComposer())}>
		<Dialog.Content side="center" class="overflow-hidden" style="width:480px;height:78vh;max-height:700px;">
			<CreateSheet onClose={closeComposer} />
		</Dialog.Content>
	</Dialog.Root>
</div>
