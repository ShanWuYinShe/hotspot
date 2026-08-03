<script lang="ts">
	import { platform } from '@/stores/platform.svelte';
	import ProfileDesktop from '@/components/profile/ProfileDesktop.svelte';
	import ProfileMobile from '@/components/profile/ProfileMobile.svelte';

	// 个人主页总览（/me）。本页只承载总览：/me/xxx 由 me/[section] 路由匹配，
	// 因此这里 section 恒为 null。资源分区（/me/:section）在任意设备上独立可寻址——复用全宽分区视图。
</script>

{#if platform.platform === 'desktop'}
	<ProfileDesktop section={null} />
{:else}
	<ProfileMobile />
{/if}
