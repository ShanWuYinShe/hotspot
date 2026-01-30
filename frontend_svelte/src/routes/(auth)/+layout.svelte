<script lang="ts">
	let { children } = $props();
</script>

<!-- 认证页面布局：居中显示，无导航栏 -->
<div class="min-h-screen bg-gradient-to-br from-background via-background to-muted">
	{@render children()}
</div>
