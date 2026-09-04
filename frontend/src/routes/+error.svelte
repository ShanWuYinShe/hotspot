<script lang="ts">
	import { page } from '$app/state';
</script>

<svelte:head>
	<title>{page.status === 404 ? '页面不存在' : '出错了'} · HotSpot</title>
</svelte:head>

<div class="flex min-h-dvh flex-col items-center justify-center gap-4 bg-surface-page p-6 text-center">
	<!-- 字重收敛到 normal/medium/semibold 三档：bold 原全站仅此与地图计数徽章两处，
	     且用途（大号/小号数字强调）与 semibold 完全重合，故归并到 semibold -->
	<p class="text-4xl font-semibold text-text-primary">{page.status}</p>
	<p class="text-sm text-text-secondary">{page.error?.message ?? (page.status === 404 ? `页面不存在：${page.url.pathname}` : '加载出错了，请稍后重试')}</p>
	<!-- base.css 的焦点兜底只覆盖 button，<a> 需显式挂 focus-ring -->
	<!-- 次级描边按钮形态（surface-flat 已被移除，input-surface 是输入框语义且会吞掉焦点环） -->
	<a
		class="hover-btn focus-ring flex items-center justify-center rounded-full border border-border-default bg-surface-primary px-6 py-2.5 text-sm font-medium text-text-primary no-underline shadow-raised-sm hover:border-border-strong"
		href="/"
	>
		返回首页
	</a>
</div>
