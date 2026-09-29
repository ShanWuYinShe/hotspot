<script lang="ts">
	import { useGroupDetail } from '../group-data.svelte';

	// 单测宿主：把 runes 工厂挂到组件实例上对外暴露，仅供测试使用。
	let { onReady }: { onReady: (gd: ReturnType<typeof useGroupDetail>) => void } = $props();
	const gd = useGroupDetail({ groupId: () => 'g1' });
	$effect(() => {
		onReady(gd);
	});
</script>
