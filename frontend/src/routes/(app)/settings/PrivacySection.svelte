<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, ChevronRight, Flag, Lock, MapPin } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import SettingRow from '@/components/forms/SettingRow.svelte';
	import Toggle from '@/components/forms/Toggle.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import SettingsSection from './SettingsSection.svelte';
	import { ROUTES } from '@/lib/routes';

	let {
		notif,
		onToggleNotif,
		isAuthenticated,
		nearbyMomentNotify,
		onToggleNearbyMomentNotify,
		privateProfile,
		onTogglePrivateProfile,
		locationSharing,
		onToggleLocationSharing
	}: {
		notif: boolean;
		onToggleNotif: () => void;
		isAuthenticated: boolean;
		nearbyMomentNotify: boolean;
		onToggleNearbyMomentNotify: () => void;
		privateProfile: boolean;
		onTogglePrivateProfile: () => void;
		locationSharing: boolean;
		onToggleLocationSharing: () => void;
	} = $props();
</script>

<!-- 通知偏好 -->
<SettingsSection title="通知偏好" description="管理新消息、互动提及与关注流的系统提醒方式">
	<ListContainer bordered inset class="overflow-hidden">
		<SettingRow title="推送通知" description="消息、提及与新动态提醒" onRowClick={onToggleNotif}>
			{#snippet badge()}
				<Badge>仅保存在本设备</Badge>
			{/snippet}
			{#snippet icon()}
				<Bell class="size-4 text-accent-default" strokeWidth={1.5} />
			{/snippet}
			<Toggle on={notif} onToggle={onToggleNotif} label="推送通知" />
		</SettingRow>
		{#if isAuthenticated}
			<SettingRow
				title="附近新瞬间通知"
				description="附近有人发布新瞬间时提醒我"
				onRowClick={onToggleNearbyMomentNotify}
			>
				{#snippet icon()}
					<MapPin class="size-4 text-accent-default" strokeWidth={1.5} />
				{/snippet}
				<Toggle on={nearbyMomentNotify} onToggle={onToggleNearbyMomentNotify} label="附近新瞬间通知" />
			</SettingRow>
		{/if}
	</ListContainer>
</SettingsSection>

<!-- 隐私与安全 -->
{#if isAuthenticated}
	<SettingsSection title="隐私与安全" description="控制个人公开主页以及地理位置在附近雷达中的可见性。">
		<ListContainer bordered inset class="overflow-hidden">
			<SettingRow
				title="私密账号"
				description="开启后仅关注者可查看你的主页与生活瞬间"
				onRowClick={onTogglePrivateProfile}
			>
				{#snippet icon()}
					<Lock class="size-4 text-accent-default" strokeWidth={1.5} />
				{/snippet}
				<Toggle on={privateProfile} onToggle={onTogglePrivateProfile} label="私密账号" />
			</SettingRow>

			<SettingRow
				title="位置共享"
				description="允许附近用户在发现雷达和地图中感知到你"
				onRowClick={onToggleLocationSharing}
			>
				{#snippet icon()}
					<MapPin class="size-4 text-accent-default" strokeWidth={1.5} />
				{/snippet}
				<Toggle on={locationSharing} onToggle={onToggleLocationSharing} label="位置共享" />
			</SettingRow>

			<SettingRow
				title="我的举报"
				description="查看已提交举报的处理进度与结果"
				onRowClick={() => void goto(ROUTES.meReports)}
			>
				{#snippet icon()}
					<Flag class="size-4 text-accent-default" strokeWidth={1.5} />
				{/snippet}
				{#snippet badge()}
					<ChevronRight class="size-4 text-text-secondary" strokeWidth={1.5} />
				{/snippet}
			</SettingRow>
		</ListContainer>
	</SettingsSection>
{/if}
