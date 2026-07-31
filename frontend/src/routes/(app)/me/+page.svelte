<script lang="ts">
	import { page } from '$app/state';
	import { platformStore } from '@/stores/platform.svelte';
	import ProfileDesktop from '@/components/profile/ProfileDesktop.svelte';
	import ProfileMobile from '@/components/profile/ProfileMobile.svelte';
	import { ROUTES } from '@/lib/routes';

	// 个人主页入口（迁移自 React ProfileFeature.tsx 内 ProfilePage）。
	// 资源分区（/me/:section）在任意设备上独立可寻址——复用全宽分区视图。

	const pathname = $derived(page.url.pathname);
	const section = $derived(
		pathname.startsWith(`${ROUTES.me}/`) ? pathname.slice(`${ROUTES.me}/`.length) : null
	);
</script>

{#if section || platformStore.platform === 'desktop'}
	<ProfileDesktop {section} />
{:else}
	<ProfileMobile />
{/if}
