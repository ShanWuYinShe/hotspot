<script lang="ts">
	import ProfileDesktop from '@/components/profile/ProfileDesktop.svelte';

	// 个人主页分段（moments / groups / friends / following / fans）。
	// SPA 模式：fallback 服务一切路径，无需构建期参数枚举。

	let { params }: { params: { section: string } } = $props();
</script>

<ProfileDesktop section={params.section} />
