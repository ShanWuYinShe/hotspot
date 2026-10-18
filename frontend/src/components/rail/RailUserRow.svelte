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

<!-- hover 底色挂在**可点元素本体**上，不挂到含 trailing 动作的行容器：挂容器会让高亮吞掉
     trailing（如动态详情「作者」行的「私信」pill），与之形成「12px 圆角矩形 + full 圆角 pill」
     的同心嵌套，且两者同为 accent 系近似透明度会叠成脏色（2026-10-18 实测反馈）。 -->
<div class="flex items-center gap-2.5">
	<button
		type="button"
		class="tap-97 flex items-center gap-2.5 flex-1 min-w-0 text-left rounded-sm focus-ring {onClick ? 'hover-bg cursor-pointer' : 'cursor-default focus-visible:outline-none'}"
		disabled={!onClick}
		onclick={onClick}
	>
		<Avatar src={avatar} name={name} size="md" class="border border-border-default" />
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
