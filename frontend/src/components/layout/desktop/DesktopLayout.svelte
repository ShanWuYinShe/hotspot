<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		Home, Users, Rss, MessageCircle, Bell, ImageIcon,
		Heart, Search, Settings, Plus, Navigation2, UserRound, PanelLeftClose, PanelLeftOpen, Shield
	} from '@lucide/svelte';
	import { cn } from '@/lib/utils';
	import { auth, requireAuth } from '@/stores/auth.svelte';
	import { admin, ensureAdminLoaded } from '@/stores/admin.svelte';
	import { composer, openComposer, closeComposer } from '@/stores/composer.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import AppearanceMenu from '@/components/overlays/AppearanceMenu.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Button from '@/components/ui/button.svelte';
	import CreateSheet from '@/components/overlays/CreateSheet.svelte';
	import SidebarNavItem from '@/components/layout/desktop/SidebarNavItem.svelte';
	import LocationCard from '@/components/layout/desktop/LocationCard.svelte';
	import RoutedContentViewport from '@/components/layout/page/RoutedContentViewport.svelte';
	import { ROUTES, ME_SECTION_PATHS, getDesktopNavId, type DesktopNavId } from '@/lib/routes';
	import imgHotSpot from '@/assets/hotspot-logo.png';

	// 桌面全屏布局（≥1024px）：52px 顶栏 + 可折叠侧栏 + 主内容卡片 + 发布模态。

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
		{ id: 'moments', icon: ImageIcon, label: '我的瞬间', path: ME_SECTION_PATHS.moments },
		{ id: 'my-groups', icon: Users, label: '我的群组', path: ME_SECTION_PATHS.groups },
		{ id: 'relationships', icon: Heart, label: '关系', path: ME_SECTION_PATHS.relationships }
	];

	// 登录后拉一次平台管理员权限：决定侧栏「管理后台」入口是否渲染
	$effect(() => {
		if (auth.status === 'authenticated') ensureAdminLoaded();
	});

	const pathname = $derived(page.url.pathname);
	const searchString = $derived(page.url.search.slice(1));
	const showCreate = $derived(composer.composerOpen);
	// 全画布页面（消息分栏 / 地图探索）：主区域不滚动，画布固定为视口高度
	const isFullCanvas = $derived(([ROUTES.messages, ROUTES.map] as string[]).includes(pathname));

	/* rail 高亮只反映「当前路由真实归属的 rail 项」：utility/detail 路由
	   （/search、/settings、/notify、无 from 的详情页等）解析为 null 时
	   清除高亮——宁可不亮，不错亮（此前持久化上一 rail 上下文会误导）。 */
	let activeNavId = $state<DesktopNavId | null>(getDesktopNavId(page.url.pathname, page.url.search.slice(1)));
	let isSidebarCollapsed = $state(false);

	$effect(() => {
		activeNavId = getDesktopNavId(pathname, searchString);
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
					void goto(item.path);
				}
			});
			return;
		}
		activeNavId = item.id;
		void goto(item.path);
	};
</script>

<div
	class="flex flex-col h-screen w-screen overflow-hidden transition-colors duration-gentle font-sans"
	style="background-color:var(--cst-surface-page);color:var(--cst-text-primary);"
>
	<!-- HEADER — 52px 白顶框（与侧栏/主面板同为白浮卡语言，border-b 分隔画布） -->
	<header
		class="relative z-[var(--cst-z-chrome)] shrink-0 flex items-center border-b border-border-default bg-surface-primary px-5 gap-2"
		style="height:var(--cst-desktop-header-height);"
	>
		<!-- Logo / 侧边栏开关：展开态分开（Logo 回首页 + 独立收起钮），折叠态
		     「同一坑位」二合一——默认显示 Logo，hover 原位淡入展开图标，点击展开。
		     Logo hover 不再套整片灰胶囊（hover-btn rounded-full 在图文组合上显笨重），
		     展开态改为品牌式反馈：Logo 微放大 + 字色切 accent，本体变化一眼可辨。 -->
		{#if isSidebarCollapsed}
			<button
				type="button"
				class="group relative mr-3 flex size-9 shrink-0 select-none items-center justify-center rounded-full focus-ring hover:bg-interactive-hover"
				aria-label="展开侧边栏"
				aria-pressed={isSidebarCollapsed}
				title="展开侧边栏"
				onclick={() => (isSidebarCollapsed = false)}
			>
				<img
					src={imgHotSpot}
					alt="HotSpot"
					class="size-7 rounded-full object-cover transition-opacity duration-rapid group-hover:opacity-0"
				/>
				<PanelLeftOpen
					size={18}
					strokeWidth={1.75}
					class="absolute text-text-secondary opacity-0 transition-opacity duration-rapid group-hover:text-accent-default group-hover:opacity-100"
				/>
			</button>
		{:else}
			<button
				type="button"
				class="group flex items-center gap-2 mr-1 select-none rounded-xl px-2 py-1 focus-ring hover:bg-interactive-hover"
				title="返回首页"
				onclick={() => {
					activeNavId = 'home';
					void goto(ROUTES.home);
				}}
			>
				<img src={imgHotSpot} alt="HotSpot Logo" class="rounded-full object-cover shrink-0 size-7 transition-transform duration-rapid group-hover:scale-105 group-active:scale-95" />
				<span class="tracking-tight text-lg text-text-primary transition-colors group-hover:text-accent-default">HotSpot</span>
			</button>

			<!-- 侧边栏折叠开关：展开态独立存在；折叠态与 Logo 同坑位 hover 切换（见上） -->
			<button
				type="button"
				class="header-action hit-slop flex size-9 shrink-0 items-center justify-center rounded-full focus-ring text-text-secondary"
				aria-label="收起侧边栏"
				aria-pressed={isSidebarCollapsed}
				title="收起侧边栏"
				onclick={() => (isSidebarCollapsed = true)}
			>
				<PanelLeftClose size={16} strokeWidth={1.75} />
			</button>
		{/if}

		<!-- Spacer -->
		<div class="flex-1"></div>

		<!-- Right Actions -->
		<div class="flex items-center gap-1.5">
			<ActionIconButton
				icon={Search}
				label="搜索"
				onClick={() => void goto(ROUTES.search)}
				variant="ghost"
				size={36}
				class="header-action"
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
				class="header-action"
			/>

			<ActionIconButton icon={Bell} label="通知" onClick={() => guardNav('查看通知', ROUTES.notify)} variant="ghost" size={36} class="header-action" />

			<AppearanceMenu />

			<ActionIconButton icon={Settings} label="设置" onClick={() => void goto(ROUTES.settings)} variant="ghost" size={36} class="header-action" />

			{#if auth.user}
				<button
					type="button"
					class="hover-btn scale-btn hit-slop relative ml-2 shrink-0 rounded-full focus-ring size-8"
					aria-label="打开个人主页"
					onclick={() => guardNav('打开个人主页', ROUTES.me)}
				>
					<!-- Avatar 组件：空头像/加载失败回退首字母色块（不渲染空 src img） -->
					<Avatar
						src={auth.user.avatar_url || undefined}
						name={auth.user.nickname || '用户'}
						size={32}
						style="border:var(--cst-border-width-regular) solid var(--cst-border-default);"
					/>
				</button>
			{:else}
					<Button
						type="button"
						variant="primary"
						size="sm"
						class="ml-2 rounded-full px-3 text-sm"
						aria-label="登录"
						onclick={() => void goto(ROUTES.login)}
					>
					登录
					</Button>
			{/if}
		</div>
	</header>

	<!-- BODY — 白浮动双卡（侧栏卡 + 内容卡浮于连续画布上，卡片自身边框为边界） -->
	<div class="desktop-workspace relative z-0 flex flex-1 overflow-hidden" >
		<!-- Sidebar (224px expanded / 72px collapsed, fixed width) -->
		<aside
			class="relative flex flex-col shrink-0 overflow-hidden box-border rounded-lg border border-border-default bg-surface-primary shadow-raised-sm transition-[width,box-shadow] duration-standard ease-standard"
			style="width:{isSidebarCollapsed ? 'var(--cst-desktop-sidebar-collapsed)' : 'var(--cst-desktop-sidebar-width)'};padding:0.75rem;"
		>
			<!-- Scrollable nav region — takes all available space -->
			<div class="flex-1 flex flex-col overflow-y-auto no-scrollbar min-h-0">
				<!-- Section 1: 发现导航 -->
				<div class="flex flex-col gap-1">
					{#each TOP_NAV as item (item.id)}
						<SidebarNavItem {item} isActive={activeNavId === item.id} onClick={() => handleNavClick(item)} collapsed={isSidebarCollapsed} />
					{/each}
				</div>

				<!-- Divider — 宽度随侧栏同步收缩 -->
				<div class="my-3 self-center h-px bg-border-default transition-[width] duration-standard ease-standard" style="width:{isSidebarCollapsed ? 24 : '100%'};"></div>

				<!-- Section 2 label — 折叠时高度/透明度同步收起（不用 sr-only 硬切） -->
				<p
					class="overflow-hidden whitespace-nowrap text-text-secondary text-xxs font-semibold tracking-label uppercase transition-[max-height,opacity,margin] duration-standard ease-standard {isSidebarCollapsed ? 'max-h-0 mb-0 opacity-none' : 'max-h-6 px-3 mb-2 opacity-soft'}"
				>
					个人中心
				</p>

				<!-- Section 2: Personal nav -->
				<div class="flex flex-col gap-1">
					{#each PERSONAL_NAV as item (item.id)}
						<SidebarNavItem {item} isActive={activeNavId === item.id} onClick={() => handleNavClick(item)} collapsed={isSidebarCollapsed} />
					{/each}
				</div>

				<!-- Section 3: 管理后台（仅平台管理员；权限随登录态拉取，非管理员不渲染） -->
				{#if auth.status === 'authenticated' && admin.isAdmin}
					<div class="my-3 self-center h-px bg-border-default transition-[width] duration-standard ease-standard" style="width:{isSidebarCollapsed ? 24 : '100%'};"></div>
					<div class="flex flex-col gap-1">
						<SidebarNavItem
							item={{ icon: Shield, label: '管理后台' }}
							isActive={activeNavId === 'admin'}
							onClick={() => void goto(ROUTES.admin)}
							collapsed={isSidebarCollapsed}
						/>
					</div>
				{/if}
			</div>

			<!-- Location Anchor — ALWAYS pinned to bottom -->
			<div class="flex shrink-0 pt-3">
				<LocationCard collapsed={isSidebarCollapsed} onClick={() => void goto(ROUTES.map)} />
			</div>
		</aside>

		<!-- Main Content Card — RoutedContentViewport 是唯一滚动 owner（desktopPane 激活 sticky-header surface-page 强制规则）。
		     全画布页（消息分栏 / 地图探索）禁用滚动，画布固定视口高度。 -->
		<div class="desktop-main-panel flex-1 min-w-0 overflow-hidden rounded-lg border border-border-default transition-colors duration-gentle">
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
		<Dialog.Content side="center" class="overflow-hidden w-120"  style="height:78vh;max-height:700px">
			<CreateSheet onClose={closeComposer} />
		</Dialog.Content>
	</Dialog.Root>
</div>
