<script lang="ts">
	import type { Snippet } from 'svelte';

	// 平板端右侧「陪伴栏」容器（迁移自旧版 TabletRail）。

	let { children }: { children: Snippet } = $props();
</script>

<div class="flex flex-col gap-3">
	{@render children()}
</div>
