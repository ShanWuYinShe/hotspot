<script lang="ts">
	import { page } from '$app/state';
</script>

<svelte:head>
	<title>{page.status === 404 ? '页面不存在' : '出错了'} · HotSpot</title>
</svelte:head>

<div class="surface-flat gap-4 p-6 flex min-h-dvh flex-col items-center justify-center text-center">
	<p class="text-4xl font-bold text-text-primary">{page.status}</p>
	<p class="text-text-secondary">{page.error?.message ?? (page.status === 404 ? `页面不存在：${page.url.pathname}` : '加载出错了，请稍后重试')}</p>
	<a class="input-surface rounded-full px-6 py-2.5 no-underline text-text-primary" href="/">返回首页</a>
</div>
