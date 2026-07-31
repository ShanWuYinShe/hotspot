<script lang="ts">
	// Dialog.Close — 关闭按钮包装，点击后触发 onOpenChange(false)。
	import { Dialog } from 'bits-ui';

	type Props = Dialog.CloseProps;

	let { ...rest }: Props = $props();
</script>

<Dialog.Close {...rest} />
