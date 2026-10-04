<script lang="ts">
	import ReportsPage from './ReportsPage.svelte';

	// 我的举报页（/me/reports，固定路由）：colocate 页面组件（ReportsPage），
	// 任意设备直接渲染。入口：设置页「隐私与安全」→「我的举报」。
</script>
<svelte:head>
	<title>我的举报 · HotSpot</title>
</svelte:head>

<ReportsPage />
