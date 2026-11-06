<script lang="ts">
	import { ChevronRight, LogOut, Settings } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { buildInfo } from '@/lib/build-info.svelte';
	import { smartBack } from '@/lib/smart-back';

	import SettingsSection from './SettingsSection.svelte';
	import AccountSecuritySection from './AccountSecuritySection.svelte';
	import AppearanceSection from './AppearanceSection.svelte';
	import PrivacySection from './PrivacySection.svelte';
	import AboutSection from './AboutSection.svelte';
	import { useSettingsData } from './useSettingsData.svelte';

	void buildInfo.load();
	const shortCommit = $derived(buildInfo.info?.commit ? buildInfo.info.commit.slice(0, 7) : '');
	const device = $derived(platform.platform);

	const page = useSettingsData();
</script>

<svelte:head>
	<title>设置 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentMaxWidth="var(--cst-content-form)" contentClassName="pt-4 pb-10">
	{#snippet header()}
		<PageHeader
			variant="integrated"
			title="设置"
			subtitle="管理你的账号偏好、主题外观与系统选项"
			islandMeta="账号 · 外观 · 地图 · 通知"
			onBack={() => smartBack()}
			showBackOnDesktop={false}
			maxWidth="var(--cst-content-form)"
			titleSize={device === 'desktop' ? 'xl' : '2xl'}
		>
			{#snippet islandIcon()}
				<Settings class="size-3.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if page.loading}
		<InlineLoading label="加载设置…" />
	{:else if page.loadError}
		<LoadError message={page.loadError} onRetry={() => page.refresh()} />
	{:else}
		<div class="flex w-full flex-col">
			<!-- 账号信息 -->
			{#if auth.isAuthenticated}
				<SettingsSection title="账号信息" description="当前登录的账号信息与个人主页入口">
					<ListContainer bordered inset class="overflow-hidden">
						<ListItem
							onClick={() => void goto(ROUTES.meEdit)}
							ariaLabel="编辑个人资料"
							class="group flex items-center gap-3.5 px-4 py-3.5"
						>
							<Avatar
								class="border-2 border-surface-primary shrink-0 transition-transform duration-rapid group-hover:scale-105"
								src={page.accountAvatar}
								name={page.accountName}
								size="lg"
							/>
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<p class="text-sm font-semibold text-text-primary truncate">{page.accountName || '用户'}</p>
									<Badge tone="solid">已登录</Badge>
								</div>
								<p class="text-xs text-text-secondary mt-0.5 truncate">{page.accountEmail || '未绑定邮箱'}</p>
							</div>
							<div class="flex items-center gap-1 text-xs text-text-secondary group-hover:text-text-primary transition-colors">
								<span>编辑资料</span>
								<ChevronRight class="size-4 text-text-secondary group-hover:translate-x-0.5 group-hover:text-text-primary transition-transform" strokeWidth={1.75} />
							</div>
						</ListItem>
					</ListContainer>
				</SettingsSection>
			{/if}

			<!-- 外观与地图 -->
			<AppearanceSection />

			<!-- 通知与隐私 -->
			<PrivacySection
				notif={page.notif}
				onToggleNotif={page.toggleNotif}
				isAuthenticated={auth.isAuthenticated}
				nearbyMomentNotify={page.nearbyMomentNotify}
				onToggleNearbyMomentNotify={page.toggleNearbyMomentNotify}
				privateProfile={page.privateProfile}
				onTogglePrivateProfile={page.togglePrivateProfile}
				locationSharing={page.locationSharing}
				onToggleLocationSharing={page.toggleLocationSharing}
			/>

			<!-- 账号安全（改密/设备） -->
			{#if auth.isAuthenticated}
				<SettingsSection title="账号与安全" description="修改登录密码、管理活跃设备登录态。">
					<ListContainer bordered inset class="overflow-hidden">
						<AccountSecuritySection />
					</ListContainer>
				</SettingsSection>
			{/if}

			<!-- 系统与关于 -->
			<AboutSection
				isAuthenticated={auth.isAuthenticated}
				{shortCommit}
				onOpenLogout={() => (page.confirmLogoutOpen = true)}
			/>
		</div>

		{#if auth.isAuthenticated}
			<ConfirmDialog
				open={page.confirmLogoutOpen}
				title="退出登录？"
				description="退出后将保持游客浏览，发布、关注等操作需要重新登录。"
				icon={LogOut}
				layout={device === 'mobile' ? 'stacked' : 'row'}
				confirmLabel="退出登录"
				loading={page.loggingOut}
				onCancel={() => (page.confirmLogoutOpen = false)}
				onConfirm={() => void page.handleLogout()}
			/>
		{/if}
	{/if}
</StandardPageLayout>
