<!--
  根级错误页面
  当 load 函数抛出 error() 或发生未处理错误时显示
-->
<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import Home from 'lucide-svelte/icons/home';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import TriangleAlert from 'lucide-svelte/icons/triangle-alert';
</script>

<div class="flex min-h-[60vh] flex-col items-center justify-center gap-6 px-4 text-center">
	<div class="flex h-20 w-20 items-center justify-center rounded-full bg-destructive/10">
		<TriangleAlert class="h-10 w-10 text-destructive" />
	</div>

	<div class="space-y-2">
		<h1 class="text-4xl font-bold tracking-tight">
			{page.status}
		</h1>
		<p class="text-lg text-muted-foreground">
			{page.error?.message ?? '页面出了点问题'}
		</p>
	</div>

	<div class="flex gap-3">
		<Button variant="outline" onclick={() => history.back()} aria-label="返回上一页">
			<ArrowLeft class="mr-2 h-4 w-4" />
			返回
		</Button>
		<Button onclick={() => goto('/')} aria-label="回到首页">
			<Home class="mr-2 h-4 w-4" />
			首页
		</Button>
	</div>
</div>
