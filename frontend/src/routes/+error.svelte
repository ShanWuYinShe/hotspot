<script lang="ts">
	import { page } from '$app/state';
</script>

<svelte:head>
	<title>页面不存在</title>
</svelte:head>

<div class="neu-surface-flat" style="min-height: 100vh; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 16px; padding: 24px; text-align: center;">
	<p class="text-4xl font-bold" style="color: var(--text-primary)">404</p>
	<p style="color: var(--text-muted)">页面不存在：{page.url.pathname}</p>
	<a class="neu-input-surface" style="padding: 10px 24px; border-radius: 999px; text-decoration: none;" href="/">返回首页</a>
</div>
