<script lang="ts">
	import BspCardCanvas from '../BspCardCanvas.svelte';

	let { items }: { items: { name: string }[] } = $props();
</script>

<BspCardCanvas
	{items}
	gap={24}
	minRatio={0.68}
	maxRatio={1.48}
	minCardWidth={160}
	minCardHeight={250}
	minCanvasHeight={560}
	ariaLabel="测试布局"
>
	{#snippet renderItem(item: { name: string }, index: number)}
		<div class="test-card" data-index={index}>{item.name}</div>
	{/snippet}
</BspCardCanvas>
