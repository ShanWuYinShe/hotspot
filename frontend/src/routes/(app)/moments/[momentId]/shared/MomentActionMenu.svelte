<script lang="ts">
	import { MoreHorizontal, Link2, RotateCw, PencilLine, Trash2, Flag } from '@lucide/svelte';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';

	let {
		isAuthor = false,
		onCopyLink,
		onReload,
		onEdit,
		onDelete,
		onReport,
		size = 36,
		class: className = ''
	}: {
		isAuthor?: boolean;
		onCopyLink: () => void;
		onReload?: () => void;
		onEdit?: () => void;
		onDelete?: () => void;
		onReport?: () => void;
		size?: 32 | 34 | 36 | 42;
		class?: string;
	} = $props();

	let open = $state(false);
	let menuAnchor = $state<HTMLElement | null>(null);
</script>

<DropdownMenu.Root bind:open>
	<div bind:this={menuAnchor} class="relative shrink-0">
		<ActionIconButton
			icon={MoreHorizontal}
			label="更多操作"
			variant="ghost"
			tone="current"
			{size}
			class={className}
			ariaExpanded={open}
			ariaHasPopup="menu"
			onClick={() => (open = !open)}
		/>
	</div>
	<DropdownMenu.Content customAnchor={menuAnchor}>
		<DropdownMenu.Item onSelect={() => { open = false; onCopyLink(); }}>
			<Link2 class="size-3.5 text-text-secondary" strokeWidth={1.75} />
			复制链接
		</DropdownMenu.Item>

		{#if onReload}
			<DropdownMenu.Item onSelect={() => { open = false; onReload(); }}>
				<RotateCw class="size-3.5 text-text-secondary" strokeWidth={1.75} />
				刷新动态
			</DropdownMenu.Item>
		{/if}

		{#if isAuthor}
			{#if onEdit}
				<DropdownMenu.Item onSelect={() => { open = false; onEdit(); }}>
					<PencilLine class="size-3.5 text-accent-default" strokeWidth={1.75} />
					编辑动态
				</DropdownMenu.Item>
			{/if}
			{#if onDelete}
				<DropdownMenu.Item
					class="text-status-danger hover:text-text-status-danger-on-soft data-[highlighted]:text-text-status-danger-on-soft hover:bg-danger-bg data-[highlighted]:bg-danger-bg"
					onSelect={() => { open = false; onDelete(); }}
				>
					<Trash2 class="size-3.5" strokeWidth={1.75} />
					删除动态
				</DropdownMenu.Item>
			{/if}
		{:else if onReport}
			<DropdownMenu.Item
				class="text-status-danger hover:text-text-status-danger-on-soft data-[highlighted]:text-text-status-danger-on-soft hover:bg-danger-bg data-[highlighted]:bg-danger-bg"
				onSelect={() => { open = false; onReport(); }}
			>
				<Flag class="size-3.5" strokeWidth={1.75} />
				举报动态
			</DropdownMenu.Item>
		{/if}
	</DropdownMenu.Content>
</DropdownMenu.Root>
