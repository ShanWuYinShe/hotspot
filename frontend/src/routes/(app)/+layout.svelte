<script lang="ts">
	import type { Snippet } from 'svelte';
	import MainLayout from '@/components/layout/MainLayout.svelte';
	import ConnectionBanner from '@/components/shared/ConnectionBanner.svelte';

	// 主路由组外壳：MainLayout 平台分发（desktop/tablet/mobile）。
	// ConnectionBanner（WS 断线提示）挂在最外层顶部，fixed 定位不占文档流。

	let { children }: { children: Snippet } = $props();
</script>

<ConnectionBanner />
<MainLayout>
	{@render children()}
</MainLayout>
