<script lang="ts">
	import { useReactedMoments } from '../profile-data.svelte';

	// 单测宿主：把 runes 工厂挂到组件实例上对外暴露，仅供测试使用。
	let { uid, onReady }: { uid: () => string | null | undefined; onReady: (r: ReturnType<typeof useReactedMoments>) => void } = $props();
	// svelte-ignore state_referenced_locally：uid 是稳定的 getter（恒等不变），
	// 响应性来自 hook 内部 effect 调用它，而非 prop 本体变更。
	const r = useReactedMoments(uid);
	$effect(() => {
		onReady(r);
	});
</script>
