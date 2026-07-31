<script lang="ts">
	// DropdownMenu.Item — 菜单项包装（默认项视觉：文本 + 悬停 accent-soft）。
	import { DropdownMenu } from 'bits-ui';
	import { cn } from '@/lib/utils';

	type Props = DropdownMenu.ItemProps;

	let { class: className, children, ...rest }: Props = $props();
</script>

<DropdownMenu.Item
	class={cn(
		'flex w-full cursor-pointer items-center gap-2 rounded-[10px] px-2.5 py-2 text-left text-xs text-[var(--neu-text)] outline-none transition-colors hover:bg-[var(--neu-accent-soft)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] data-[highlighted]:bg-[var(--neu-accent-soft)]',
		className
	)}
	{...rest}
>
	{@render children?.()}
</DropdownMenu.Item>
