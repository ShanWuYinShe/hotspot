<script lang="ts">
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { untrack } from 'svelte';
	import { authStore } from '$lib/stores/auth.svelte';
	import { normalizeRedirectPath, uiStore } from '$lib/stores/ui.svelte';
	import AuthFormContent from '$lib/components/common/auth-form-content.svelte';
	import Shield from 'lucide-svelte/icons/shield';

	$effect(() => {
		if (browser && authStore.isAuthenticated) {
			untrack(() => {
				const redirect = normalizeRedirectPath(page.url.searchParams.get('redirect')) ?? '/';
				goto(redirect, { replaceState: true });
			});
		}
	});

	async function handleSuccess() {
		uiStore.saveRedirectUrl(page.url.searchParams.get('redirect') ?? undefined);
		uiStore.onLoginSuccess();
	}
</script>

<svelte:head>
	<title>登录 - HotSpot</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center px-4 py-10">
	<div class="w-full max-w-md rounded-3xl border border-border/60 bg-card/95 p-8 shadow-xl backdrop-blur-md">
		<div class="mb-6 text-center">
			<div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-2xl border border-border/50 bg-primary/10 text-primary">
				<Shield class="h-6 w-6" />
			</div>
			<h1 class="mb-2 text-2xl font-semibold">登录后继续</h1>
			<p class="text-sm text-muted-foreground">使用你的账号、注册新账号或继续访客体验</p>
		</div>

		<AuthFormContent onSuccess={handleSuccess} />
	</div>
</div>
