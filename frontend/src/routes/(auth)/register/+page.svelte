<script lang="ts">
	import AuthPage from '../AuthPage.svelte';

	// /register 路由入口：与 /login 共用品牌页，仅初始模式不同。
	// 独立路由是为了外部链接（文档/活动页）直接进注册态不再落 404（设计评审 P1）。
</script>

<AuthPage initialMode="register" />
