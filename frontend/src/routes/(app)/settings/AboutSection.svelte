<script lang="ts">
	import { Info, LogOut } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import Button from '@/components/ui/button.svelte';
	import SettingsSection from './SettingsSection.svelte';
	import { buildInfo } from '@/lib/build-info.svelte';

	let {
		isAuthenticated,
		shortCommit,
		onOpenLogout
	}: {
		isAuthenticated: boolean;
		shortCommit: string;
		onOpenLogout: () => void;
	} = $props();
</script>

<SettingsSection title="系统与关于" description="客户端版本及当前设备会话状态。">
	<div class="flex flex-col gap-4">
		<ListContainer bordered inset class="overflow-hidden">
			<ListItem class="flex items-center justify-between gap-3 px-4 py-3.5">
				<div class="flex items-center gap-3">
					<div class="flex size-9 items-center justify-center rounded-full icon-badge shrink-0">
						<Info class="size-4 text-accent-default" strokeWidth={1.5} />
					</div>
					<div>
						<p class="text-sm font-medium text-text-primary">关于 HotSpot</p>
						<p class="text-xs text-text-secondary">基于地理位置的实时社交与社区发现应用</p>
					</div>
				</div>
				<div class="flex flex-col items-end gap-0.5">
					<Badge tone="outline" size="sm" class="font-mono">{buildInfo.info?.version ?? '…'}</Badge>
					{#if shortCommit}
						<span class="text-xxs font-mono text-text-secondary">构建 {shortCommit}</span>
					{/if}
				</div>
			</ListItem>
		</ListContainer>

		{#if isAuthenticated}
			<Button
				variant="soft"
				class="hit-slop-44 h-auto w-full justify-center gap-2 px-4 py-3 text-status-danger hover:bg-danger-bg/80 border border-status-danger/20 hover:border-status-danger/40 transition-all duration-rapid shadow-2xs"
				onclick={onOpenLogout}
			>
				<LogOut class="size-4" strokeWidth={1.75} />
				退出登录当前账号
			</Button>
		{/if}
	</div>
</SettingsSection>
