<script lang="ts">
	// Popover.Content — Portal + Content 组合封装。
	// 默认右下对齐（side=bottom / align=end / sideOffset=8），floating 定位自动避让边缘；
	// 项目浮层视觉（surface-primary 底 + elevation 阴影 + 动画）在此集中。
	// 手动锚定：传 customAnchor 指向触发元素（如自研按钮），即可不渲染 Trigger。
	import { Popover } from 'bits-ui';
	import { cn } from '@/lib/utils';

	type Props = Popover.ContentProps;

	let {
		class: className,
		children,
		side = 'bottom',
		align = 'end',
		sideOffset = 8,
		...rest
	}: Props = $props();
</script>

<Popover.Portal>
	<Popover.Content
		{side}
		{align}
		{sideOffset}
		class={cn(
			'z-[var(--cst-z-modal)] rounded-sm bg-surface-primary p-1.5 shadow-menu data-[state=open]:animate-in data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95',
			className
		)}
		{...rest}
	>
		{@render children?.()}
	</Popover.Content>
</Popover.Portal>
