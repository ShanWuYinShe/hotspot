<script lang="ts">
	import type { Snippet } from 'svelte';
	import MainLayout from '@/components/layout/MainLayout.svelte';
	import ConnectionBanner from '@/components/shared/ConnectionBanner.svelte';
	import MockBadge from '@/components/shared/MockBadge.svelte';

	// 主路由组外壳：MainLayout 平台分发（desktop/tablet/mobile）。
	// ConnectionBanner（WS 断线提示）挂在最外层顶部，MockBadge（dev-only 数据源提示）
	// 挂右下角，两者 fixed 定位不占文档流。

	let { children }: { children: Snippet } = $props();
</script>

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
