<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { Button } from '$lib/components/ui/button';
	import Shield from 'lucide-svelte/icons/shield';

	// 如果已登录，直接跳转
	$effect(() => {
		if (browser && authStore.isAuthenticated) {
			const redirect = page.url.searchParams.get('redirect') || '/';
			goto(redirect);
		}
	});

	function handleLoginClick() {
		const redirect = page.url.searchParams.get('redirect') || '/';
		uiStore.openLoginModal(() => goto(redirect));
	}
</script>

<svelte:head>
	<title>登录 - HotSpot</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center bg-background">
	<div class="rounded-2xl border border-border/60 bg-card/80 px-8 py-10 text-center shadow-sm shadow-black/5">
		<div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-muted">
			<Shield class="h-6 w-6 text-muted-foreground" />
		</div>
		<h1 class="mb-2 text-xl font-semibold">登录后继续</h1>
		<p class="mb-6 text-sm text-muted-foreground">使用登录/注册弹窗完成认证</p>
		<Button class="rounded-full px-6" onclick={handleLoginClick}>打开登录</Button>
	</div>
</div>
