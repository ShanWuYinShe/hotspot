<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { ArrowLeft, Home, RotateCw } from '@lucide/svelte';
	import Button from '@/components/ui/button.svelte';
	import AmbientBackdrop from '@/components/layout/ambient/AmbientBackdrop.svelte';
	import BrandLockup from '@/components/layout/brand/BrandLockup.svelte';
	import { ROUTES } from '@/lib/routes';

	// 全站错误页（含 404）。
	// R13 品牌化：顶部品牌字标 + 中央大号状态码 + 一句人话 + 主/次 CTA，
	// 继续复用认证页同款氛围画布与品牌字标，让「走错路」也保持同一套城市语言。
	// 状态码是整页唯一数字锚点（h1 之前的大字），正文一句话不堆技术细节；
	// 404 的次操作是「返回上一页」，其它错误是「重试」，各自贴合用户此刻意图。
	const is404 = $derived(page.status === 404);
	// 面向用户的文案不打印路径（路径只在 sr-only 里给排障/读屏语境，避免"像报错"）；
	// 设计评审 P2-16。
	const detail = $derived(
		is404
			? '这个页面可能已被移动或删除，回首页继续逛逛吧。'
			: (page.error?.message && !/^[\x00-\x7F]+$/.test(page.error.message)
				? page.error.message
				: '加载出错了，请稍后重试。')
	);
</script>

<svelte:head>
	<title>{is404 ? '页面不存在' : '出错了'} · HotSpot</title>
</svelte:head>

<main class="relative isolate flex min-h-dvh flex-col bg-surface-page px-6 py-8">
	<!-- 氛围画布：错误页整屏留白大，主题世界层兜底视觉密度 -->
	<AmbientBackdrop />

	<header class="relative flex items-center">
		<a
			href={ROUTES.home}
			class="focus-ring -m-2 inline-flex rounded-full p-2"
			aria-label="回到 HotSpot 首页"
		>
			<BrandLockup size="sm" />
		</a>
	</header>

	<div class="relative flex flex-1 flex-col items-center justify-center py-12 text-center">
		<p class="text-4xl font-semibold tabular-nums tracking-title text-text-primary">{page.status}</p>
		<h1 class="mt-3 text-2xl font-semibold tracking-title text-text-primary">
			{is404 ? '这条街还没开通。' : '这里暂时出了点状况。'}
		</h1>
		<p class="mt-2 max-w-md text-sm leading-6 text-text-secondary">{detail}</p>
		{#if is404}
			<p class="sr-only">当前地址：{page.url.pathname}</p>
		{/if}

		<div class="mt-8 flex flex-wrap items-center justify-center gap-3">
			<Button variant="primary" size="md" onclick={() => void goto(ROUTES.home)}>
				<Home class="size-4" strokeWidth={1.75} aria-hidden="true" />
				返回首页
			</Button>
			<Button
				variant="soft"
				size="md"
				onclick={() => (is404 ? history.back() : location.reload())}
			>
				{#if is404}
					<ArrowLeft class="size-4" strokeWidth={1.75} aria-hidden="true" />
					返回上一页
				{:else}
					<RotateCw class="size-4" strokeWidth={1.75} aria-hidden="true" />
					重试
				{/if}
			</Button>
		</div>
	</div>
</main>
