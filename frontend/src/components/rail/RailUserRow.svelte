<script lang="ts">
	import type { Snippet } from 'svelte';

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

<div class="hover-bg flex items-center gap-2.5 rounded-tight {onClick ? 'cursor-pointer' : ''}">
	<button
		type="button"
		class="tap-985 flex items-center gap-2.5 flex-1 min-w-0 text-left rounded-tight outline-none"
		onclick={onClick}
	>
		<div class="rounded-full overflow-hidden shrink-0 size-8.5 border border-border-default"  >
			<img src={avatar} alt={name} class="w-full h-full object-cover" />
		</div>
		<div class="flex-1 min-w-0">
			<p  style="color:var(--text-primary)" class="truncate text-base">{name}</p>
			{#if sub}
				<p  style="color:var(--text-secondary)" class="truncate text-xs mt-px">{sub}</p>
			{/if}
		</div>
	</button>
	{#if trailing}
		{@render trailing()}
	{/if}
</div>
