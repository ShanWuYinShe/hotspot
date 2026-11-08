<script lang="ts">
	import { ArrowUp, Bell, ChevronRight, Info, LogOut, Palette, Settings, Shield, User } from '@lucide/svelte';
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

<StandardPageLayout contentMaxWidth={device === 'desktop' ? '1040px' : 'var(--cst-content-form)'} contentClassName="pt-4 sm:pt-6 pb-12">
	{#snippet header()}
		<PageHeader
			variant="integrated"
			title="设置"
			subtitle="管理你的账号偏好、主题外观与系统选项"
			islandMeta="账号 · 外观 · 地图 · 通知"
			onBack={() => smartBack()}
			showBackOnDesktop={false}
			maxWidth={device === 'desktop' ? '1040px' : 'var(--cst-content-form)'}
			titleSize={device === 'desktop' ? 'xl' : '2xl'}
		>
			{#snippet islandIcon()}
				<Settings class="size-3.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
			{#snippet islandExpanded()}
				<div class="flex flex-col gap-2.5">
					<div class="flex items-center justify-between text-xs pb-0.5">
						<span class="font-medium text-text-primary">系统偏好导航</span>
						<span class="font-mono text-xxs text-accent-default">快捷定位</span>
					</div>
					<div class="flex flex-wrap gap-1.5 py-0.5">
						{#if auth.isAuthenticated}
							<button
								type="button"
								class="hover-btn flex items-center gap-1.5 rounded-full border border-border-default/30 bg-surface-subtle/80 px-2.5 py-1 text-xs text-text-secondary transition-all hover:border-accent-border/50 hover:bg-surface-primary hover:text-text-primary active:scale-95"
								onclick={() => {
									document.querySelector('[data-settings-account]')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
								}}
							>
								<User class="size-3 text-accent-default" />
								<span>账号信息</span>
							</button>
						{/if}
						<button
							type="button"
							class="hover-btn flex items-center gap-1.5 rounded-full border border-border-default/30 bg-surface-subtle/80 px-2.5 py-1 text-xs text-text-secondary transition-all hover:border-accent-border/50 hover:bg-surface-primary hover:text-text-primary active:scale-95"
							onclick={() => {
								document.querySelector('[data-settings-appearance]')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
							}}
						>
							<Palette class="size-3 text-accent-default" />
							<span>外观与显示</span>
						</button>
						<button
							type="button"
							class="hover-btn flex items-center gap-1.5 rounded-full border border-border-default/30 bg-surface-subtle/80 px-2.5 py-1 text-xs text-text-secondary transition-all hover:border-accent-border/50 hover:bg-surface-primary hover:text-text-primary active:scale-95"
							onclick={() => {
								document.querySelector('[data-settings-privacy]')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
							}}
						>
							<Bell class="size-3 text-accent-default" />
							<span>通知与隐私</span>
						</button>
						{#if auth.isAuthenticated}
							<button
								type="button"
								class="hover-btn flex items-center gap-1.5 rounded-full border border-border-default/30 bg-surface-subtle/80 px-2.5 py-1 text-xs text-text-secondary transition-all hover:border-accent-border/50 hover:bg-surface-primary hover:text-text-primary active:scale-95"
								onclick={() => {
									document.querySelector('[data-settings-security]')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
								}}
							>
								<Shield class="size-3 text-accent-default" />
								<span>账号安全</span>
							</button>
						{/if}
						<button
							type="button"
							class="hover-btn flex items-center gap-1.5 rounded-full border border-border-default/30 bg-surface-subtle/80 px-2.5 py-1 text-xs text-text-secondary transition-all hover:border-accent-border/50 hover:bg-surface-primary hover:text-text-primary active:scale-95"
							onclick={() => {
								document.querySelector('[data-settings-about]')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
							}}
						>
							<Info class="size-3 text-accent-default" />
							<span>系统关于</span>
						</button>
					</div>
					<div class="flex items-center justify-between border-t border-border-default/15 pt-2 text-xxs text-text-secondary">
						<div class="flex items-center gap-1.5 text-text-secondary/70">
							<kbd class="rounded border border-border-default/30 bg-surface-island px-1 py-0.2 font-mono text-[10px]">Esc</kbd>
							<span>收起</span>
						</div>
						<button
							type="button"
							class="group flex items-center gap-1 text-accent-default hover:underline transition-colors font-medium text-xxs"
							onclick={() => {
								const scroller = document.getElementById('main-content');
								scroller?.scrollTo({ top: 0, behavior: 'smooth' });
							}}
						>
							<span>平滑回顶</span>
							<ArrowUp class="size-3 transition-transform group-hover:-translate-y-0.5" />
						</button>
					</div>
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if page.loading}
		<InlineLoading label="加载设置…" />
	{:else if page.loadError}
		<LoadError message={page.loadError} onRetry={() => page.refresh()} />
	{:else}
		<!-- 桌面端双列 Bento Grid 架构，移动端单列流式呈现 -->
		<div class="w-full grid grid-cols-1 lg:grid-cols-2 gap-5 lg:gap-6 items-start">
			<!-- 左栏：身份档案、外观主题视觉中枢与系统关于 -->
			<div class="flex flex-col w-full">
				<!-- 账号信息 -->
				{#if auth.isAuthenticated}
					<div data-settings-account>
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
					</div>
				{/if}

				<!-- 外观与地图 -->
				<div data-settings-appearance>
					<AppearanceSection />
				</div>

				<!-- 系统与关于 -->
				<div data-settings-about>
					<AboutSection
						isAuthenticated={auth.isAuthenticated}
						{shortCommit}
						onOpenLogout={() => (page.confirmLogoutOpen = true)}
					/>
				</div>
			</div>

			<!-- 右栏：通知、隐私防线与账号安全中心 -->
			<div class="flex flex-col w-full">
				<!-- 通知与隐私 -->
				<div data-settings-privacy>
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
				</div>

				<!-- 账号安全（改密/设备） -->
				{#if auth.isAuthenticated}
					<div data-settings-security>
						<SettingsSection title="账号与安全" description="修改登录密码、管理活跃设备登录态。">
							<ListContainer bordered inset class="overflow-hidden">
								<AccountSecuritySection />
							</ListContainer>
						</SettingsSection>
					</div>
				{/if}
			</div>
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
