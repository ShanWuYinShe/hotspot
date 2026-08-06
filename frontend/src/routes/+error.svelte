<script lang="ts">
	import { page } from '$app/state';
</script>

<svelte:head>
	<title>页面不存在</title>
</svelte:head>

<div class="surface-flat gap-4"  style="min-height:100vh;display:flex;flex-direction:column;align-items:center;justify-content:center;padding:24px;text-align:center">
	<p class="text-4xl font-bold" style="color: var(--cst-text-primary)">404</p>
	<p style="color: var(--cst-text-secondary)">页面不存在：{page.url.pathname}</p>
	<a class="input-surface rounded-[999px]"  style="padding:10px 24px;text-decoration:none" href="/">返回首页</a>
</div>
