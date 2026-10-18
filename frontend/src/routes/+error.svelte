<script lang="ts">
	import { page } from '$app/state';
	import Button from '@/components/ui/button.svelte';
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
		<!-- 主次分明：返回首页为主操作（primary accent 实底），重试/返回上一页为次级
		     描边（secondary）。页面级动作走 Button 原语 md 档（rounded-md），胶囊仅
	     保留给行内关系 chip——按钮语言两派契约见 skill hotspot-frontend-style。 -->
		<Button variant="secondary" onclick={() => location.reload()}>重试</Button>
		<Button variant="secondary" onclick={() => history.back()}>返回上一页</Button>
		<a
			class="focus-ring inline-flex h-10 items-center justify-center gap-2 rounded-md bg-accent-default px-4 text-sm font-medium text-text-on-accent no-underline shadow-raised-sm hover:shadow-raised-md hover-btn"
			href="/"
		>
			返回首页
		</a>
	</div>
</div>
