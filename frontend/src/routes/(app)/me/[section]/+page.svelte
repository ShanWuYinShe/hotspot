<script lang="ts">
	import ProfileDesktop from '@/components/profile/desktop/ProfileDesktop.svelte';
	import SectionRelationships from '@/components/profile/SectionRelationships.svelte';

	// 个人主页分段：五个独立分区（moments / groups / friends / following / fans）
	// 走 ProfileDesktop 分区模式（PageHeader + Section 全宽视图）；
	// relationships 聚合页（好友/关注/粉丝 三 tab）是独立样板页（StandardPageLayout），
	// 由本路由在任意设备上直接渲染。
	// SPA 模式：fallback 服务一切路径，无需构建期参数枚举。

	let { params }: { params: { section: string } } = $props();
</script>

{#if params.section === 'relationships'}
	<SectionRelationships />
{:else}
	<ProfileDesktop section={params.section} />
{/if}
