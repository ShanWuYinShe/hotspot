<script lang="ts">
	import { page } from '$app/state';
</script>

<svelte:head>
	<title>{page.status === 404 ? '页面不存在' : '出错了'} · HotSpot</title>
</svelte:head>

<div class="flex min-h-dvh flex-col items-center justify-center gap-4 bg-surface-page p-6 text-center">
	<!-- 字重收敛到 normal/medium/semibold 三档：bold 原全站仅此与地图计数徽章两处，
	     且用途（大号/小号数字强调）与 semibold 完全重合，故归并到 semibold。
	     状态码是整页唯一语义标题（h1），保证错误页也有可读的标题层级。 -->
	<h1 class="text-4xl font-semibold text-text-primary">{page.status}</h1>
	<p class="text-sm text-text-secondary">{page.status === 404 ? `页面不存在：${page.url.pathname}` : (page.error?.message && !/^[\x00-\x7F]+$/.test(page.error.message) ? page.error.message : '加载出错了，请稍后重试')}</p>
	<div class="flex items-center justify-center gap-3">
		<!-- 主次分明：返回首页为主操作（accent 实底），重试/返回上一页为次级描边 -->
		<button
			type="button"
			onclick={() => location.reload()}
			class="hover-btn flex items-center justify-center rounded-full border border-border-default bg-surface-primary px-6 py-2.5 text-sm font-medium text-text-primary shadow-raised-sm hover:border-border-strong"
		>
			重试
		</button>
		<button
			type="button"
			onclick={() => history.back()}
			class="hover-btn flex items-center justify-center rounded-full border border-border-default bg-surface-primary px-6 py-2.5 text-sm font-medium text-text-primary shadow-raised-sm hover:border-border-strong"
		>
			返回上一页
		</button>
		<a
			class="hover-btn focus-ring flex items-center justify-center rounded-full bg-accent-default px-6 py-2.5 text-sm font-medium text-text-on-accent no-underline shadow-raised-sm hover:shadow-raised-md"
			href="/"
		>
			返回首页
		</a>
	</div>
</div>
