<script lang="ts">
	// Dialog.Title — 标题包装，自动与 Dialog.Content 的 aria-labelledby 关联。
	import { Dialog } from 'bits-ui';

	type Props = Dialog.TitleProps;

	let { ...rest }: Props = $props();
</script>

<Dialog.Title {...rest} />
