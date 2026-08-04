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

<div class="hover-bg flex items-center gap-2.5 rounded-[10px] {onClick ? 'cursor-pointer' : ''}">
	<button
		type="button"
		class="tap-985 flex items-center gap-2.5 flex-1 min-w-0 text-left rounded-[10px] outline-none"
		onclick={onClick}
	>
		<div class="rounded-full overflow-hidden flex-shrink-0" style="width:34px;height:34px;border:1px solid var(--border-default);">
			<img src={avatar} alt={name} class="w-full h-full object-cover" />
		</div>
		<div class="flex-1 min-w-0">
			<p style="font-size:13px;color:var(--text-primary);" class="truncate">{name}</p>
			{#if sub}
				<p style="font-size:11px;color:var(--text-secondary);margin-top:1px;" class="truncate">{sub}</p>
			{/if}
		</div>
	</button>
	{#if trailing}
		{@render trailing()}
	{/if}
</div>
