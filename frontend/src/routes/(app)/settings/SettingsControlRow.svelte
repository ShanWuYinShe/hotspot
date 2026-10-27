<script lang="ts">
	import type { Snippet } from 'svelte';
	import ListItem from '@/components/list/ListItem.svelte';

	// 设置「选择器行」（R13）：图标 + 标题/描述 + 全宽控件。
	// 主题/明暗/地图源这类分段控件必须给整行宽度——放在窄右列时 TabBar 内部标签
	// 会被压成一字（改前 D-settings-vibrant-light 实测「星…」），用户看不出选项含义。
	// 开关类行仍用 forms/SettingRow（控件贴右、拇指可达），两类行的图标/标题/描述栅格一致。

	let {
		icon,
		title,
		description,
		control
	}: {
		icon: Snippet;
		title: string;
		description: string;
		control: Snippet;
	} = $props();
</script>

<ListItem class="px-4 py-3.5">
	<div class="flex flex-col gap-3">
		<div class="flex items-center gap-3">
			<span class="icon-badge flex size-9 shrink-0 items-center justify-center rounded-full">
				{@render icon()}
			</span>
			<div class="min-w-0 flex-1">
				<p class="text-sm font-medium text-text-primary">{title}</p>
				<p class="mt-0.5 text-xs leading-relaxed text-text-secondary">{description}</p>
			</div>
		</div>
		<div class="min-w-0">
			{@render control()}
		</div>
	</div>
</ListItem>
