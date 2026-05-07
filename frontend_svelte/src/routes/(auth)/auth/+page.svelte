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
				const redirect = page.url.searchParams.get('redirect');
				const target = normalizeRedirectPath(redirect) ?? '/';
				goto(target, { replaceState: true });
			});
		}
	});

	async function handleSuccess() {
		uiStore.onLoginSuccess();
		const redirect = page.url.searchParams.get('redirect');
		const target = normalizeRedirectPath(redirect) ?? '/';
		goto(target, { replaceState: true });
	}
</script>

<svelte:head>
	<title>登录 - HotSpot</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center px-4 py-10">
	<div class="w-full max-w-md rounded-[24px] p-6" style="background: var(--neu-bg); box-shadow: var(--neu-shadow-lg); border: 1px solid var(--neu-border);">
		<div class="mb-6 text-center">
			<div class="mx-auto mb-4 flex items-center justify-center rounded-full" style="width: 48px; height: 48px; box-shadow: var(--neu-shadow-inset); border: 1px solid var(--neu-border);">
				<Shield size={19} color="var(--neu-accent-primary)" />
			</div>
			<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">登录后继续</h1>
			<p style="font-size: 13px; color: var(--neu-text-muted); margin-top: 6px;">使用你的账号、注册新账号或继续访客体验</p>
		</div>

		<AuthFormContent onSuccess={handleSuccess} />
	</div>
</div>
