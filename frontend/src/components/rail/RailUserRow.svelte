<script lang="ts">
	import type { Snippet } from 'svelte';
	import Avatar from '@/components/shared/Avatar.svelte';

	// 详情栏复用的「小头像 + 名称 + 副标 + 操作」行。

	let {
		avatar,
		name,
		sub,
		trailing,
		onClick
	}: {
		avatar: string;
		name: string;
		sub?: string;
		trailing?: Snippet;
		onClick?: () => void;
	} = $props();
</script>

<div class="hover-bg flex items-center gap-2.5 rounded-sm {onClick ? 'cursor-pointer' : ''}">
	<button
		type="button"
		class="tap-97 flex items-center gap-2.5 flex-1 min-w-0 text-left rounded-sm focus-ring"
		onclick={onClick}
	>
		<Avatar src={avatar} name={name} size={34} class="border border-border-default" />
		<div class="flex-1 min-w-0">
			<p   class="truncate text-sm text-text-primary">{name}</p>
			{#if sub}
				<p   class="truncate text-xs mt-px text-text-secondary">{sub}</p>
			{/if}
		</div>
	</button>
	{#if trailing}
		{@render trailing()}
	{/if}
</div>
