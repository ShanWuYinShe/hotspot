<script lang="ts">
	// Dialog.Root — 受控开关包装（bits-ui 命名空间 API 透传）。
	// 使用：let open = $state(false); <Root bind:open>...</Root>
	import { Dialog } from 'bits-ui';

	type Props = Dialog.RootProps;

	let { open = $bindable(false), ...rest }: Props = $props();
</script>

<Dialog.Root bind:open {...rest} />
