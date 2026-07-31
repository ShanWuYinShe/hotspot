<script lang="ts">
	import ProfileDesktop from '@/components/profile/ProfileDesktop.svelte';

	// 个人主页分段（moments / groups / relationships）。
	// SPA 模式无需 generateStaticParams（fallback 服务一切路径）。

	let { params }: { params: { section: string } } = $props();
</script>

<ProfileDesktop section={params.section} />
