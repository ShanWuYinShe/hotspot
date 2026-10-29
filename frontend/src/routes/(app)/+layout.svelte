<script lang="ts">
	import type { Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import MainLayout from '@/components/layout/MainLayout.svelte';
	import ConnectionBanner from '@/components/shared/ConnectionBanner.svelte';
	import MockBadge from '@/components/shared/MockBadge.svelte';
	import { ROUTES } from '@/lib/routes';
	import { resolveSearchShortcut } from '@/lib/search-shortcut';

	// 主路由组外壳：MainLayout 平台分发（desktop/tablet/mobile）。
	// ConnectionBanner（WS 断线提示）挂在最外层顶部，MockBadge（dev-only 数据源提示）
	// 挂右下角，两者 fixed 定位不占文档流。

	let { children }: { children: Snippet } = $props();

	/**
	 * 全局搜索快捷键 ⌘K（mac）/ Ctrl+K（其他）：任意页面唤出搜索。
	 * 决策在 lib/search-shortcut.ts（纯函数 + 单测）；这里只做 DOM 副作用。
	 * 让位规则：弹层打开、非搜索页焦点在表单控件内（避免带走未保存输入）时不劫持。
	 * 搜索页落点：autofocus 与 RoutedContentViewport 的 afterNavigate focus(main)
	 * 存在时序竞争，导航后 rAF 再显式聚焦一次，保证「按下即可打字」。
	 */
	function focusSearchInput() {
		const input = document.querySelector<HTMLInputElement>('input[data-search-input]');
		input?.focus();
		input?.select();
	}

	function handleGlobalKeydown(e: KeyboardEvent) {
		const active = document.activeElement as HTMLElement | null;
		const action = resolveSearchShortcut(e, {
			pathname: window.location.pathname,
			searchPath: ROUTES.search,
			dialogOpen: !!document.querySelector('[role="dialog"]'),
			editing:
				!!active &&
				(active.tagName === 'INPUT' ||
					active.tagName === 'TEXTAREA' ||
					active.isContentEditable)
		});
		if (action === 'none') return;
		e.preventDefault();
		if (action === 'focus-search') {
			focusSearchInput();
			return;
		}
		void goto(ROUTES.search).then(() => {
			requestAnimationFrame(focusSearchInput);
		});
	}
</script>

<svelte:window onkeydown={handleGlobalKeydown} />
<ConnectionBanner />
<MockBadge />
<!-- 键盘用户跳转：直接定位到路由内容滚动区（RoutedContentViewport main#main-content） -->
<a
	href="#main-content"
	class="sr-only focus:not-sr-only focus:fixed focus:left-3 focus:top-3 focus:z-[var(--cst-z-toast)] focus:rounded-md focus:bg-surface-primary focus:px-4 focus:py-2 focus:text-sm focus:text-text-primary focus:ring-2 focus:ring-accent-default"
>
	跳到主要内容
</a>
<!-- SPA 路由播报：导航后由 RoutedContentViewport 更新文本（首屏不播） -->
<div id="route-announcer" class="sr-only" aria-live="polite" aria-atomic="true"></div>
<MainLayout>
	{@render children()}
</MainLayout>
