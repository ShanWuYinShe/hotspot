<script lang="ts">
	import { platform } from '@/stores/platform.svelte';
	import ProfilePage from '@/components/profile/ProfilePage.svelte';
	import ProfileMobile from '@/components/profile/mobile/ProfileMobile.svelte';

	// 个人主页总览（/me）。本页只承载总览；资源分区（moments/groups/friends/
	// following/fans/relationships）是固定路由（见 lib/routes.ts 与 me/ 目录树），
	// 分区页在任意设备上独立可寻址——复用全宽分区视图（ProfilePage 分区模式）。
	// 设备分派：mobile 用 ProfileMobile；tablet 与 desktop 统一走 ProfilePage。
</script>

{#if platform.platform !== 'mobile'}
	 <ProfilePage section={null} />
{:else}
	<ProfileMobile />
{/if}
