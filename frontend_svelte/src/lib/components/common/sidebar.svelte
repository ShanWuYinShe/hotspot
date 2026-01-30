<script lang="ts">
	import { page } from '$app/stores';
	import { cn } from '$lib/utils';
	import { authStore } from '$lib/stores/auth.svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { Button } from '$lib/components/ui/button';
	import LocationPickerModal from './location-picker-modal.svelte';
	import Home from 'lucide-svelte/icons/home';
	import Compass from 'lucide-svelte/icons/compass';
	import Image from 'lucide-svelte/icons/image';
	import UsersRound from 'lucide-svelte/icons/users-round';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import Bell from 'lucide-svelte/icons/bell';
	import UserPlus from 'lucide-svelte/icons/user-plus';
	import Heart from 'lucide-svelte/icons/heart';
	import UserCheck from 'lucide-svelte/icons/user-check';
	import Navigation from 'lucide-svelte/icons/navigation';
	import Settings from 'lucide-svelte/icons/settings';
	import ChevronRight from 'lucide-svelte/icons/chevron-right';

	interface NavItem {
		href: string;
		label: string;
		icon: typeof Home;
		requiresAuth?: boolean;
		badge?: number;
	}

	const mainNavItems: NavItem[] = [
		{ href: '/', label: '首页', icon: Home },
		{ href: '/explore', label: '探索', icon: Compass }
	];

const personalNavItems: NavItem[] = [
	{ href: '/chat', label: '消息', icon: MessageCircle, requiresAuth: true },
	{ href: '/notifications', label: '通知', icon: Bell, requiresAuth: true },
	{ href: '/moments', label: '我的瞬间', icon: Image, requiresAuth: true },
	{ href: '/groups', label: '我的群组', icon: UsersRound, requiresAuth: true },
	{ href: '/friends', label: '好友', icon: Heart, requiresAuth: true },
	{ href: '/follows', label: '关注', icon: UserCheck, requiresAuth: true },
	{ href: '/followers', label: '粉丝', icon: UserPlus, requiresAuth: true }
];

	function isActive(href: string): boolean {
		const pathname = $page.url.pathname;
		const currentParams = $page.url.searchParams;
		const url = new URL(href, $page.url.origin);

		if (url.pathname !== pathname) return false;
		if (url.searchParams.toString().length === 0) {
			return true;
		}

		for (const [key, value] of url.searchParams) {
			if (currentParams.get(key) !== value) return false;
		}
		return true;
	}

	function handleNavClick(item: NavItem, e: MouseEvent) {
		if (item.requiresAuth && !authStore.isAuthenticated) {
			e.preventDefault();
			uiStore.saveRedirectUrl(item.href);
			uiStore.openLoginModal();
		}
	}

	interface Props {
		class?: string;
	}

	let { class: className }: Props = $props();

	// 位置选择弹窗状态
	let showLocationPicker = $state(false);
</script>

<aside
	class={cn(
		'flex h-full min-h-[calc(100vh-4rem)] flex-col border-r border-sidebar-border/60 bg-sidebar/95 text-sidebar-foreground shadow-sm shadow-black/5 backdrop-blur',
		className
	)}
>
	<!-- 主导航区域 -->
	<nav class="flex-1 overflow-y-auto px-3 py-4">
		<!-- 主导航 -->
		<div class="space-y-1">
			{#each mainNavItems as item}
				<a
					href={item.href}
					class={cn(
						'group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-200',
						isActive(item.href)
							? 'bg-primary text-primary-foreground shadow-md shadow-primary/25 ring-1 ring-primary/30'
							: 'text-muted-foreground hover:bg-accent hover:text-foreground'
					)}
				>
					<item.icon class={cn('h-5 w-5 transition-transform group-hover:scale-110', isActive(item.href) ? '' : '')} />
					<span>{item.label}</span>
				</a>
			{/each}
		</div>

		<!-- 个人中心 -->
		<div class="mt-6">
			<h3 class="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground/70">
				个人中心
			</h3>
			<div class="space-y-1">
				{#each personalNavItems as item}
					<a
						href={item.href}
						onclick={(e) => handleNavClick(item, e)}
						class={cn(
							'group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-200',
						isActive(item.href)
							? 'bg-primary text-primary-foreground shadow-md shadow-primary/25 ring-1 ring-primary/30'
							: 'text-muted-foreground hover:bg-accent hover:text-foreground'
						)}
					>
						<item.icon class="h-5 w-5 transition-transform group-hover:scale-110" />
						<span class="flex-1">{item.label}</span>
						{#if !authStore.isAuthenticated}
							<span class="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">登录</span>
						{/if}
					</a>
				{/each}
			</div>
		</div>
	</nav>

	<!-- 底部区域：浏览锚点 -->
	<div class="p-3">
		<button
			type="button"
			class="group flex w-full items-center gap-3 rounded-xl bg-linear-to-r from-primary/5 to-primary/10 p-3 text-left transition-all duration-200 hover:from-primary/10 hover:to-primary/20"
			onclick={() => (showLocationPicker = true)}
		>
			<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary transition-transform group-hover:scale-105">
				<Navigation class="h-5 w-5" />
			</div>
			<div class="min-w-0 flex-1">
				<div class="text-xs font-medium text-muted-foreground">浏览锚点</div>
				<div class="truncate text-sm font-semibold text-foreground">{mapAnchorStore.displayName}</div>
			</div>
			<ChevronRight class="h-4 w-4 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
		</button>
	</div>
</aside>

<LocationPickerModal bind:open={showLocationPicker} />
