<script lang="ts">
	// 测试宿主：CreateGroupSheet 生产环境始终位于 Dialog.Root 内（SheetShell 使用
	// Dialog.Title / Dialog.Close，需要 bits-ui 的 Dialog context）。单测直接渲染
	// 组件会丢 context，这里补一层与真实调用方一致的外壳。
	import * as Dialog from '@/components/ui/dialog';
	import CreateGroupSheet from '../CreateGroupSheet.svelte';

	let {
		onClose = () => {},
		onCreated
	}: {
		onClose?: () => void;
		onCreated?: (groupId: string) => void;
	} = $props();
</script>

<Dialog.Root open={true} onOpenChange={() => {}}>
	<Dialog.Content side="center" aria-label="创建群组">
		<CreateGroupSheet {onClose} {onCreated} />
	</Dialog.Content>
</Dialog.Root>
