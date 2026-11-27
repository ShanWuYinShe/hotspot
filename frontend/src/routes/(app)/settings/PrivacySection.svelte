<script lang="ts">
	import { goto } from '$app/navigation';
	import { Bell, ChevronRight, Flag, Heart, Lock, MapPin, Share2, UserPlus, Users, UsersRound } from '@lucide/svelte';
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
		onToggleLocationSharing,
		socialListsVisible,
		onToggleSocialListsVisible,
		groupsVisible,
		onToggleGroupsVisible,
		reactedMomentsVisible,
		onToggleReactedMomentsVisible,
		allowFollow,
		onToggleAllowFollow
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
		socialListsVisible: boolean;
		onToggleSocialListsVisible: () => void;
		groupsVisible: boolean;
		onToggleGroupsVisible: () => void;
		reactedMomentsVisible: boolean;
		onToggleReactedMomentsVisible: () => void;
		allowFollow: boolean;
		onToggleAllowFollow: () => void;
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
				title="社交列表可见"
				description="关闭后他人无法查看你的好友、关注与粉丝列表"
				onRowClick={onToggleSocialListsVisible}
			>
				{#snippet icon()}
					<Users class="size-4 text-accent-default" strokeWidth={1.5} />
				{/snippet}
				<Toggle on={socialListsVisible} onToggle={onToggleSocialListsVisible} label="社交列表可见" />
			</SettingRow>

			<SettingRow
				title="群组列表可见"
				description="关闭后他人无法查看你加入的群组"
				onRowClick={onToggleGroupsVisible}
			>
				{#snippet icon()}
					<UsersRound class="size-4 text-accent-default" strokeWidth={1.5} />
				{/snippet}
				<Toggle on={groupsVisible} onToggle={onToggleGroupsVisible} label="群组列表可见" />
			</SettingRow>

			<SettingRow
				title="反应瞬间可见"
				description="关闭后他人无法查看你点赞或反应过的瞬间"
				onRowClick={onToggleReactedMomentsVisible}
			>
				{#snippet icon()}
					<Heart class="size-4 text-accent-default" strokeWidth={1.5} />
				{/snippet}
				<Toggle on={reactedMomentsVisible} onToggle={onToggleReactedMomentsVisible} label="反应瞬间可见" />
			</SettingRow>

			<SettingRow
				title="允许被关注"
				description="关闭后他人无法关注你，已有关注关系保留"
				onRowClick={onToggleAllowFollow}
			>
				{#snippet icon()}
					<UserPlus class="size-4 text-accent-default" strokeWidth={1.5} />
				{/snippet}
				<Toggle on={allowFollow} onToggle={onToggleAllowFollow} label="允许被关注" />
			</SettingRow>

			<SettingRow
				title="我的举报"
				description="查看已提交举报的处理进度与结果"
				onRowClick={() => void goto(ROUTES.meReports)}
			>
				{#snippet icon()}
					<Flag class="size-4 text-accent-default" strokeWidth={1.5} />
				{/snippet}
				<ChevronRight class="size-4 text-text-secondary transition-transform duration-rapid group-hover:translate-x-0.5 group-hover:text-text-primary" strokeWidth={1.75} />
			</SettingRow>

			<SettingRow
				title="我的分享"
				description="管理已创建分享链接的复制与撤销"
				onRowClick={() => void goto(ROUTES.meShares)}
			>
				{#snippet icon()}
					<Share2 class="size-4 text-accent-default" strokeWidth={1.5} />
				{/snippet}
				<ChevronRight class="size-4 text-text-secondary transition-transform duration-rapid group-hover:translate-x-0.5 group-hover:text-text-primary" strokeWidth={1.75} />
			</SettingRow>
		</ListContainer>
	</SettingsSection>
{/if}
