<script lang="ts">
	// Popover.Root — 受控开关包装（bits-ui 命名空间 API 透传）。
	// 行为：Escape / 外部点击 / 焦点管理（无 focus-trap，与原 dismissible-layer 一致）。
	import { Popover } from 'bits-ui';

	type Props = Popover.RootProps;

	let { open = $bindable(false), ...rest }: Props = $props();
</script>

<Popover.Root bind:open {...rest} />
