<script lang="ts">
	import ProfilePage from '@/components/profile/ProfilePage.svelte';

	// 个人主页固定分区页（常驻路由，不走动态段）：PageHeader + Section 全宽视图。
</script>
<svelte:head>
	<title>我的粉丝 · HotSpot</title>
</svelte:head>


 <ProfilePage section="fans" />
