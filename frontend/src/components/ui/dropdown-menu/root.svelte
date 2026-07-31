<script lang="ts">
	// DropdownMenu.Root — 受控开关包装（bits-ui 命名空间 API 透传）。
	import { DropdownMenu } from 'bits-ui';

	type Props = DropdownMenu.RootProps;

	let { open = $bindable(false), ...rest }: Props = $props();
</script>

<DropdownMenu.Root bind:open {...rest} />
