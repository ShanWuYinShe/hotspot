<script lang="ts">
	import type { Snippet } from 'svelte';
	import MainLayout from '@/components/layout/MainLayout.svelte';

	// 主路由组外壳：MainLayout 平台分发（desktop/tablet/mobile）。

	let { children }: { children: Snippet } = $props();
</script>

<MainLayout>
	{@render children()}
</MainLayout>
