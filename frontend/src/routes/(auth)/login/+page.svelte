<script lang="ts">
	import AuthPage from '../AuthPage.svelte';

	// /login 路由入口：品牌页主体在 (auth)/AuthPage.svelte（与 /register 共用）。
</script>

<svelte:head>
	<title>登录 · HotSpot</title>
</svelte:head>

<AuthPage initialMode="login" />
