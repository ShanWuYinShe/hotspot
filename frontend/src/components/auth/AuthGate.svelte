<script lang="ts">
	import LoginPromptSheet from '@/components/auth/LoginPromptSheet.svelte';
	import BannedScreen from '@/components/auth/BannedScreen.svelte';
	import { auth, closeLoginPrompt, dismissBanned } from '@/stores/auth.svelte';

	// 全局认证覆盖层：动作拦截登录提示 + 账号封禁屏。
	// 挂载在根布局一次，覆盖所有路由，与布局形态无关。

	const loginPromptOpen = $derived(auth.loginPromptOpen);
	const pendingAction = $derived(auth.pendingAction);
	const bannedMessage = $derived(auth.bannedMessage);
</script>

<LoginPromptSheet
	open={loginPromptOpen}
	actionLabel={pendingAction?.action ?? null}
	onClose={closeLoginPrompt}
	onSuccess={closeLoginPrompt}
/>
{#if bannedMessage}
	<BannedScreen message={bannedMessage} onAcknowledge={dismissBanned} />
{/if}
