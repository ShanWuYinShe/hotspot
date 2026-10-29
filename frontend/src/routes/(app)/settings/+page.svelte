<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { Bell, Flag, Lock, ChevronRight, LogOut, Info, MapPin, Monitor, Palette, Moon, Sun, UserRound } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Toggle from '@/components/forms/Toggle.svelte';
	import SettingRow from '@/components/forms/SettingRow.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import Badge from '@/components/data-display/Badge.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import Button from '@/components/ui/button.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import SettingsSection from './SettingsSection.svelte';
	import SettingsControlRow from './SettingsControlRow.svelte';
	import AccountSecuritySection from './AccountSecuritySection.svelte';
	import { auth, logout, requireAuth } from '@/stores/auth.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { theme, setTheme, setColorScheme, THEME_OPTIONS, COLOR_SCHEME_OPTIONS, type ColorSchemeMode, type AppTheme } from '@/stores/theme.svelte';
	import { mapSourceStore, setMapSource, MAP_SOURCE_OPTIONS, initMapSource, type MapSource } from '@/stores/mapSource.svelte';
	import { getAccount, getSettings, updateSettings, netError } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import { buildInfo } from '@/lib/build-info.svelte';
	import { smartBack } from '@/lib/smart-back';

	// 设置页：主题/深色、通知、隐私开关、关于、退出登录。
	// 服务端字段对齐 spec UserSettings：位置共享 → location_visible、私密账号 →
	// privacy_level；推送通知 spec 无服务端字段，是本地客户端偏好，
	// 用 localStorage 持久化（不冒充服务端设置）。

	// 构建版本指纹来自 /version.json（构建期产物），非 package.json——
	// 后者长期不随发布演进，不能作为线上版本依据。
	void buildInfo.load();
	const shortCommit = $derived(buildInfo.info?.commit ? buildInfo.info.commit.slice(0, 7) : '');

	// 本地偏好（推送通知）——localStorage 持久化
	const NOTIF_KEY = 'hotspot:notif';
	const device = $derived(platform.platform);
	const readLocal = (key: string, fallback: boolean) => {
		try {
			const v = localStorage.getItem(key);
			return v === null ? fallback : v === '1';
		} catch {
			return fallback;
		}
	};
	const writeLocal = (key: string, value: boolean) => {
		try {
			localStorage.setItem(key, value ? '1' : '0');
		} catch {}
	};

	let notif = $state(readLocal(NOTIF_KEY, true));
	let privateProfile = $state(false);
	let locationSharing = $state(true);
	// 附近新瞬间通知（服务端设置 nearby_moment_notify，spec api_spec_websocket.md
	// NearbyNewMoment 的用户开关；默认 false）。
	let nearbyMomentNotify = $state(false);
	let accountName = $state('');
	let accountEmail = $state('');
	let accountAvatar = $state('');
	let loading = $state(false);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	// 服务端设置开关：未登录引导登录；已登录乐观更新 + 失败回滚并提示
	// 附近新瞬间通知（服务端设置 nearby_moment_notify）：通知页「附近」分区
	// 的用户开关（spec api_spec_websocket.md NearbyNewMoment 推送对象）。
	const toggleNearbyMomentNotify = () => {
		if (!auth.isAuthenticated) {
			requireAuth({ action: '修改附近新瞬间通知设置', run: toggleNearbyMomentNotify });
			return;
		}
		nearbyMomentNotify = !nearbyMomentNotify;
		void updateSettings({ settings: { nearby_moment_notify: nearbyMomentNotify } }).catch(() => {
			nearbyMomentNotify = !nearbyMomentNotify;
			showToast('设置失败，请重试', 'danger');
		});
	};

	const toggleLocationSharing = () => {
		if (!auth.isAuthenticated) {
			requireAuth({ action: '修改位置共享设置', run: toggleLocationSharing });
			return;
		}
		const next = !locationSharing;
		locationSharing = next;
		void updateSettings({ settings: { location_visible: next } })
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message ?? '设置失败');
			})
			.catch(() => {
				locationSharing = !next;
				showToast('设置失败，请稍后重试', 'danger');
			});
	};
	const togglePrivateProfile = () => {
		if (!auth.isAuthenticated) {
			requireAuth({ action: '修改私密账号设置', run: togglePrivateProfile });
			return;
		}
		const next = !privateProfile;
		privateProfile = next;
		void updateSettings({ settings: { privacy_level: next ? 'private' : 'public' } })
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message ?? '设置失败');
			})
			.catch(() => {
				privateProfile = !next;
				showToast('设置失败，请稍后重试', 'danger');
			});
	};
	const toggleNotif = () => {
		notif = !notif;
		writeLocal(NOTIF_KEY, notif);
	};

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchData() {
			if (!auth.isAuthenticated) {
				// 未登录/访客：本地偏好（主题/外观/通知）可用，不拉敏感设置，不报 401 错误
				accountName = '';
				accountEmail = '';
				accountAvatar = '';
				loading = false;
				loadError = null;
				return;
			}
			try {
				loading = true;
				loadError = null;
				const [accountRes, settingsRes] = await Promise.all([
					getAccount().catch(() => netError(null)),
					getSettings().catch(() => netError(null))
				]);
				if (cancelled) return;

				const accountData = accountRes.code === 0 ? accountRes.data : null;
				const settingsData = settingsRes.code === 0 ? settingsRes.data : null;

				if (accountData) {
					accountName = accountData.nickname ?? '';
					accountEmail = accountData.email ?? '';
					accountAvatar = accountData.avatar_url ?? '';
				}

				if (settingsData) {
					if (settingsData.location_visible !== undefined) locationSharing = settingsData.location_visible;
					if (settingsData.nearby_moment_notify !== undefined) nearbyMomentNotify = settingsData.nearby_moment_notify;
					if (settingsData.privacy_level !== undefined) privateProfile = settingsData.privacy_level === 'private';
				}

				// 仅非 401 网络/服务端严重异常时提示
				if (accountRes.code !== 0 && settingsRes.code !== 0 && accountRes.code !== 401) {
					loadError = accountRes.message ?? '加载失败，请稍后重试';
				}
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchData();
		return () => {
			cancelled = true;
		};
	});

	let confirmLogoutOpen = $state(false);

	// 地图源偏好（localStorage 持久化，本机生效；地图页快捷切换同步此值）
	$effect(() => initMapSource());
	const mapSourceDesc = $derived(
		mapSourceStore.source === 'amap' ? '高德地图 · 国内地点更贴合' : 'OpenStreetMap · 全球开源数据'
	);

	// 当前主题的风格说明（THEME_OPTIONS.note：store 提供但此前页面未消费，s3 P2-5）
	const themeNote = $derived(THEME_OPTIONS.find((t) => t.id === theme.theme)?.note ?? '');

	// 明暗三档描述（跟随系统/浅色/深色，与 TabBar 选择同步）
	const colorSchemeDesc = $derived(
		theme.colorScheme === 'auto' ? '跟随系统外观' : theme.colorScheme === 'dark' ? '当前使用深色外观' : '当前使用浅色外观'
	);

	const handleLogout = async () => {
		await logout();
		void goto(ROUTES.home);
	};
</script>

<svelte:head>
	<title>设置 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentMaxWidth="56rem" contentClassName="animate-fade-in pt-4 pb-10">
	{#snippet header()}
		<PageHeader
			title="设置"
			subtitle="管理你的账号偏好、主题外观与系统选项"
			onBack={() => smartBack()}
			showBackOnDesktop={false}
			maxWidth="56rem"
			titleSize={device === 'desktop' ? 'xl' : '2xl'}
		/>
	{/snippet}

	{#if loading}
		<InlineLoading label="加载设置…" />
	{:else if loadError}
		<LoadError message={loadError} onRetry={() => reloadKey++} />
	{:else}
		<!-- 单列分组（R13）：组标题/描述在上、内容软岛在下。
		     原两栏布局在 1440 宽下会留出 130+px 空档，且右列被压窄导致分段控件文字截断。 -->
		<div class="flex w-full flex-col">
			<!-- Section 1: 账号与身份 -->
			<SettingsSection title="账号信息" description="当前登录的账号信息">
				<ListContainer bordered inset class="overflow-hidden">
					{#if auth.isAuthenticated}
						<ListItem class="flex items-center gap-3.5 px-4 py-3.5">
							<Avatar class="border-2 border-surface-primary shrink-0" src={accountAvatar} name={accountName} size="lg" />
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<p class="text-sm font-semibold text-text-primary truncate">{accountName || '用户'}</p>
									<Badge tone="solid">已登录</Badge>
								</div>
								<p class="text-xs text-text-secondary mt-0.5 truncate">{accountEmail || '未绑定邮箱'}</p>
							</div>
						</ListItem>
					{:else}
						<ListItem class="flex items-center gap-3.5 px-4 py-3.5">
							<div class="flex size-11 items-center justify-center rounded-full bg-accent-soft text-text-accent-on-soft shrink-0">
								<UserRound class="size-5" strokeWidth={1.5} />
							</div>
							<div class="min-w-0 flex-1">
								<p class="text-sm font-semibold text-text-primary">访客浏览中</p>
								<p class="text-xs text-text-secondary mt-0.5">登录后可同步跨端偏好、个人主页与瞬间收藏</p>
							</div>
							<Button
								class="h-auto shrink-0 rounded-md px-4 py-2 text-xs"
								onclick={() => requireAuth({ action: '登录账号', run: () => { reloadKey++; } })}
							>
								登录 / 注册
							</Button>
						</ListItem>
					{/if}
				</ListContainer>
			</SettingsSection>

			<!-- Section 2: 外观与显示（选择器整行宽度，杜绝标签截断） -->
			<SettingsSection title="外观与显示" description="自定义界面的视觉色彩与深浅模式，全局即时生效">
				<ListContainer bordered inset class="overflow-hidden">
					<SettingsControlRow
						title="应用主题"
						description={themeNote ? `选择全局视觉色彩风格 · ${themeNote}` : '选择全局视觉色彩风格'}
					>
						{#snippet icon()}
							<Palette class="size-4 text-accent-default" strokeWidth={1.5} />
						{/snippet}
						{#snippet control()}
							<TabBar
								variant="segmented"
								size="md"
								items={THEME_OPTIONS.map((t) => ({ id: t.id, label: t.label, icon: t.icon, swatch: t.swatch }))}
								value={theme.theme}
								onChange={(id) => setTheme(id as AppTheme)}
							/>
						{/snippet}
					</SettingsControlRow>

					<SettingsControlRow title="外观模式" description={colorSchemeDesc}>
						{#snippet icon()}
							{#if theme.isDark}
								<Moon class="size-4 text-accent-default" strokeWidth={1.5} />
							{:else}
								<Sun class="size-4 text-accent-default" strokeWidth={1.5} />
							{/if}
						{/snippet}
						{#snippet control()}
							<TabBar
								variant="segmented"
								size="md"
								items={COLOR_SCHEME_OPTIONS.map((o) => ({ id: o.id, label: o.label, icon: o.id === 'dark' ? Moon : o.id === 'light' ? Sun : Monitor }))}
								value={theme.colorScheme}
								onChange={(id) => setColorScheme(id as ColorSchemeMode)}
							/>
						{/snippet}
					</SettingsControlRow>
				</ListContainer>
			</SettingsSection>

			<!-- Section 3: 地图与位置 -->
			<SettingsSection
				title="地图与位置"
				description="选择地图页使用的底图，两种底图都能看到附近的人与瞬间，差别在地点名称与街道的贴合程度。"
			>
				<ListContainer bordered inset class="overflow-hidden">
					<SettingsControlRow title="地图源" description={mapSourceDesc}>
						{#snippet icon()}
							<MapPin class="size-4 text-accent-default" strokeWidth={1.5} />
						{/snippet}
						{#snippet control()}
							<TabBar
								variant="segmented"
								size="md"
								items={MAP_SOURCE_OPTIONS.map((o) => ({ id: o.id, label: o.id === 'amap' ? '高德' : 'OSM' }))}
								value={mapSourceStore.source}
								onChange={(id) => setMapSource(id as MapSource)}
							/>
						{/snippet}
					</SettingsControlRow>
				</ListContainer>
			</SettingsSection>

			<!-- Section 4: 通知偏好 -->
			<SettingsSection title="通知偏好" description="管理新消息、互动提及与关注流的系统提醒方式">
				<ListContainer bordered inset class="overflow-hidden">
					<SettingRow title="推送通知" description="消息、提及与新动态提醒" onRowClick={toggleNotif}>
						{#snippet badge()}
							<!-- 纯本机偏好（localStorage），后端 spec 无对应字段；与下方服务端
							     开关（私密账号/位置共享）同列时必须标明生效范围，避免误以为已同步。 -->
							<Badge>仅保存在本设备</Badge>
						{/snippet}
						{#snippet icon()}
							<Bell class="size-4 text-accent-default" strokeWidth={1.5} />
						{/snippet}
						<Toggle on={notif} onToggle={toggleNotif} label="推送通知" />
					</SettingRow>
					<SettingRow
						title="附近新瞬间通知"
						description="附近有人发布新瞬间时提醒我{auth.isAuthenticated ? '' : '（登录后生效）'}"
						onRowClick={toggleNearbyMomentNotify}
					>
						{#snippet icon()}
							<MapPin class="size-4 text-accent-default" strokeWidth={1.5} />
						{/snippet}
						<Toggle on={nearbyMomentNotify} onToggle={toggleNearbyMomentNotify} label="附近新瞬间通知" />
					</SettingRow>
				</ListContainer>
			</SettingsSection>

			<!-- Section 5: 账号与安全（登录用户：修改密码/登录设备管理；游客：升级入口） -->
			{#if auth.isAuthenticated}
				<SettingsSection title="账号与安全" description="修改登录密码、管理活跃设备登录态。">
					<ListContainer bordered inset class="overflow-hidden">
						<AccountSecuritySection />
					</ListContainer>
				</SettingsSection>
			{/if}

			<!-- Section 6: 隐私与安全 -->
			<SettingsSection title="隐私与安全" description="控制个人公开主页以及地理位置在附近雷达中的可见性。">
				<ListContainer bordered inset class="overflow-hidden">
					<SettingRow
						title="私密账号"
						description="开启后仅关注者可查看你的主页与生活瞬间{auth.isAuthenticated ? '' : '（登录后生效）'}"
						onRowClick={togglePrivateProfile}
					>
						{#snippet icon()}
							<Lock class="size-4 text-accent-default" strokeWidth={1.5} />
						{/snippet}
						<Toggle on={privateProfile} onToggle={togglePrivateProfile} label="私密账号" />
					</SettingRow>

					<SettingRow
						title="位置共享"
						description="允许附近用户在发现雷达和地图中感知到你{auth.isAuthenticated ? '' : '（登录后生效）'}"
						onRowClick={toggleLocationSharing}
					>
						{#snippet icon()}
							<MapPin class="size-4 text-accent-default" strokeWidth={1.5} />
						{/snippet}
						<Toggle on={locationSharing} onToggle={toggleLocationSharing} label="位置共享" />
					</SettingRow>

					<!-- 我的举报（spec report.md → reports/fetch）：举报人查看处理进度与结果 -->
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

			<!-- Section 7: 系统与关于 -->
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

					{#if auth.isAuthenticated}
						<!-- 危险动作语气收敛：安静软填底 + danger 文字，不再用整块红底按钮抢视线 -->
						<Button
							variant="soft"
							class="hit-slop-44 h-auto w-full justify-center gap-2 px-4 py-3 text-status-danger hover:bg-danger-bg"
							onclick={() => (confirmLogoutOpen = true)}
						>
							<LogOut class="size-4" strokeWidth={1.75} />
							退出登录当前账号
						</Button>
					{/if}
				</div>
			</SettingsSection>
		</div>

		{#if auth.isAuthenticated}
			<ConfirmDialog
				open={confirmLogoutOpen}
				title="退出登录？"
				description="退出后将保持游客浏览，发布、关注等操作需要重新登录。"
				icon={LogOut}
				layout={device === 'mobile' ? 'stacked' : 'row'}
				confirmLabel="退出登录"
				onCancel={() => (confirmLogoutOpen = false)}
				onConfirm={() => void handleLogout()}
			/>
		{/if}
	{/if}
</StandardPageLayout>