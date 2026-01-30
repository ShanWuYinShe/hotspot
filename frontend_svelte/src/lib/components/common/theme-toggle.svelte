<script lang="ts">
	import { toggleMode, mode } from 'mode-watcher';
	import { Button } from '$lib/components/ui/button';
	import Sun from 'lucide-svelte/icons/sun';
	import Moon from 'lucide-svelte/icons/moon';

	// 使用 $derived 来响应式获取当前模式
	let currentMode = $derived(mode.current);
</script>

<Button variant="ghost" size="icon" onclick={toggleMode}>
	{#if currentMode === 'dark'}
		<Sun class="h-5 w-5" />
	{:else}
		<Moon class="h-5 w-5" />
	{/if}
	<span class="sr-only">切换主题</span>
</Button>
