<script lang="ts">
	// DropdownMenu.Separator — 菜单分隔线。
	import { DropdownMenu } from 'bits-ui';
	import { cn } from '@/lib/utils';

	type Props = DropdownMenu.SeparatorProps;

	let { class: className, ...rest }: Props = $props();
</script>

<DropdownMenu.Separator class={cn('my-1 h-px bg-[var(--neu-border)]', className)} {...rest} />
