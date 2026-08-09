<script lang="ts">
	import { useChat, type UseChatOptions } from '../chat.svelte';

	// 测试宿主：在组件初始化上下文中实例化 useChat（$effect 需要组件上下文），
	// 通过 expose 回调把实例交给测试断言。

	let {
		options,
		expose
	}: {
		options: UseChatOptions;
		expose: (chat: ReturnType<typeof useChat>) => void;
	} = $props();

	// svelte-ignore state_referenced_locally：测试宿主每次渲染固定传入 options，无需响应式追踪
	const chat = useChat(options);

	$effect(() => {
		expose(chat);
	});
</script>
