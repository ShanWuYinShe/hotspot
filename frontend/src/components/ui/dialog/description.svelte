<script lang="ts">
	// Dialog.Description — 描述包装，自动与 Dialog.Content 的 aria-describedby 关联。
	import { Dialog } from 'bits-ui';

	type Props = Dialog.DescriptionProps;

	let { ...rest }: Props = $props();
</script>

<Dialog.Description {...rest} />
