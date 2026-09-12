<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { Bell, Lock, ChevronRight, LogOut, Info, MapPin, Palette, Trees, Moon, Sun, UserRound } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Toggle from '@/components/forms/Toggle.svelte';
	import SettingRow from '@/components/forms/SettingRow.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import { auth, logout, requireAuth } from '@/stores/auth.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import { theme, setTheme, setColorScheme, THEME_OPTIONS, COLOR_SCHEME_OPTIONS, type ColorSchemeMode, type AppTheme } from '@/stores/theme.svelte';
	import { mapSourceStore, setMapSource, MAP_SOURCE_OPTIONS, initMapSource, type MapSource } from '@/stores/mapSource.svelte';
	import { getAccount, getSettings, updateSettings, netError } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import pkg from '../../../../package.json';

	// 设置页：主题/深色、通知、隐私开关、关于、退出登录。
	// 服务端字段对齐 spec UserSettings：位置共享 → location_visible、私密账号 →
	// privacy_level；推送通知 spec 无服务端字段，是本地客户端偏好，
	// 用 localStorage 持久化（不冒充服务端设置）。

	const APP_VERSION = `v${pkg.version}`;

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
	let accountName = $state('');
	let accountEmail = $state('');
	let accountAvatar = $state('');
	let loading = $state(false);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	// 服务端设置开关：未登录引导登录；已登录乐观更新 + 失败回滚并提示
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
		mapSourceStore.source === 'amap' ? '高德瓦片，国内街巷细节丰富' : 'OpenStreetMap，全球开源数据'
	);

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

<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-16">
	{#snippet header()}
		<PageHeader
			title="设置"
			subtitle="管理你的账号偏好、主题外观与系统选项"
			onBack={() => history.back()}
			showBackOnDesktop={false}
			titleSize={device === 'desktop' ? 'xl' : '2xl'}
		/>
	{/snippet}

	{#if loadError}
		<LoadError message={loadError} onRetry={() => reloadKey++} />
	{:else}
		<div class="flex flex-col divide-y divide-border-default max-w-5xl">
			<!-- Section 1: 账号与身份 -->
			<div class="grid grid-cols-1 lg:grid-cols-[220px_minmax(0,1fr)] gap-4 lg:gap-10 py-6 first:pt-4">
				<div>
					<h2 class="text-sm font-semibold text-text-primary">账号信息</h2>
					<p class="text-xs text-text-secondary mt-1 leading-relaxed">
						当前登录的账号信息。
					</p>
				</div>
				<div>
					{#if auth.isAuthenticated}
						<div class="flex items-center justify-between rounded-lg border border-border-default bg-surface-primary p-4 shadow-raised-sm">
							<div class="flex items-center gap-3.5 min-w-0">
								<Avatar class="border-2 border-surface-primary shrink-0" src={accountAvatar} name={accountName} size={48} />
								<div class="min-w-0">
									<div class="flex items-center gap-2">
										<p class="text-sm font-semibold text-text-primary truncate">{accountName || '用户'}</p>
										<span class="text-xxs px-2 py-0.5 rounded-full bg-accent-soft text-accent-default font-medium">已登录</span>
									</div>
									<p class="text-xs text-text-secondary mt-0.5 truncate">{accountEmail || '未绑定邮箱'}</p>
								</div>
							</div>
						</div>
					{:else}
						<div class="flex items-center justify-between rounded-lg border border-border-default bg-surface-primary p-4 shadow-raised-sm">
							<div class="flex items-center gap-3.5 min-w-0">
								<div class="flex size-11 items-center justify-center rounded-full bg-accent-soft text-accent-default shrink-0">
									<UserRound class="size-5" strokeWidth={1.5} />
								</div>
								<div class="min-w-0">
									<p class="text-sm font-semibold text-text-primary">访客浏览中</p>
									<p class="text-xs text-text-secondary mt-0.5">登录后可同步跨端偏好、个人主页与动态收藏</p>
								</div>
							</div>
							<button
								type="button"
								onclick={() => requireAuth({ action: '登录账号', run: () => { reloadKey++; } })}
								class="hover-btn px-4 py-2 text-xs font-medium rounded-md bg-accent-default text-text-on-accent shadow-raised-sm hover:shadow-raised-md transition-colors focus-ring shrink-0"
							>
								登录 / 注册
							</button>
						</div>
					{/if}
				</div>
			</div>

			<!-- Section 2: 外观与显示 -->
			<div class="grid grid-cols-1 lg:grid-cols-[220px_minmax(0,1fr)] gap-4 lg:gap-10 py-6">
				<div>
					<h2 class="text-sm font-semibold text-text-primary">外观与显示</h2>
					<p class="text-xs text-text-secondary mt-1 leading-relaxed">
						自定义界面的视觉色彩与深浅模式，全局即时生效。
					</p>
				</div>
				<div class="rounded-lg border border-border-default bg-surface-primary divide-y divide-border-default shadow-raised-sm overflow-hidden">
					<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4">
						<div class="flex items-center gap-3">
							<div class="flex size-9 items-center justify-center rounded-full icon-badge shrink-0">
								<Palette class="size-4 text-accent-default" strokeWidth={1.5} />
							</div>
							<div>
								<p class="text-sm font-medium text-text-primary">应用主题</p>
								<p class="text-xs text-text-secondary">选择全局视觉色彩风格</p>
							</div>
						</div>
						<TabBar
							variant="segmented"
							items={THEME_OPTIONS.map((t) => ({ id: t.id, label: t.label, icon: t.id === 'forest' ? Trees : Palette }))}
							value={theme.theme}
							onChange={(id) => setTheme(id as AppTheme)}
						/>
					</div>

					<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4">
						<div class="flex items-center gap-3">
							<div class="flex size-9 items-center justify-center rounded-full icon-badge shrink-0">
								{#if theme.isDark}
									<Moon class="size-4 text-accent-default" strokeWidth={1.5} />
								{:else}
									<Sun class="size-4 text-accent-default" strokeWidth={1.5} />
								{/if}
							</div>
							<div>
								<p class="text-sm font-medium text-text-primary">外观模式</p>
								<p class="text-xs text-text-secondary">{colorSchemeDesc}</p>
							</div>
						</div>
						<TabBar
							variant="segmented"
							items={COLOR_SCHEME_OPTIONS.map((o) => ({ id: o.id, label: o.label, icon: o.id === 'dark' ? Moon : o.id === 'light' ? Sun : undefined }))}
							value={theme.colorScheme}
							onChange={(id) => setColorScheme(id as ColorSchemeMode)}
						/>
					</div>
				</div>
			</div>

			<!-- Section 3: 地图与位置 -->
			<div class="grid grid-cols-1 lg:grid-cols-[220px_minmax(0,1fr)] gap-4 lg:gap-10 py-6">
				<div>
					<h2 class="text-sm font-semibold text-text-primary">地图与位置</h2>
					<p class="text-xs text-text-secondary mt-1 leading-relaxed">
						选择地图页使用的底图源。两种底图均经站点同源代理加载，密钥不下发到浏览器。
					</p>
				</div>
				<div class="rounded-lg border border-border-default bg-surface-primary divide-y divide-border-default shadow-raised-sm overflow-hidden">
					<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4">
						<div class="flex items-center gap-3">
							<div class="flex size-9 items-center justify-center rounded-full icon-badge shrink-0">
								<MapPin class="size-4 text-accent-default" strokeWidth={1.5} />
							</div>
							<div>
								<p class="text-sm font-medium text-text-primary">地图源</p>
								<p class="text-xs text-text-secondary">{mapSourceDesc}</p>
							</div>
						</div>
						<TabBar
							variant="segmented"
							items={MAP_SOURCE_OPTIONS.map((o) => ({ id: o.id, label: o.label }))}
							value={mapSourceStore.source}
							onChange={(id) => setMapSource(id as MapSource)}
						/>
					</div>
				</div>
			</div>

			<!-- Section 4: 通知偏好 -->
			<div class="grid grid-cols-1 lg:grid-cols-[220px_minmax(0,1fr)] gap-4 lg:gap-10 py-6">
				<div>
					<h2 class="text-sm font-semibold text-text-primary">通知偏好</h2>
					<p class="text-xs text-text-secondary mt-1 leading-relaxed">
						管理新消息、互动提及与关注动态的系统提醒方式。
					</p>
				</div>
				<ListContainer bordered inset>
					<SettingRow
						title="推送通知"
						description="消息、提及与新动态提醒"
						onRowClick={toggleNotif}
					>
						{#snippet badge()}
							<!-- 纯本机偏好（localStorage），后端 spec 无对应字段；与下方服务端
							     开关（私密账号/位置共享）同列时必须标明生效范围，避免误以为已同步。 -->
							<span class="badge bg-surface-subtle text-text-secondary">仅本机</span>
						{/snippet}
						{#snippet icon()}
							<Bell class="size-4 text-accent-default" strokeWidth={1.5} />
						{/snippet}
						<Toggle on={notif} onToggle={toggleNotif} label="推送通知" />
					</SettingRow>
				</ListContainer>
			</div>

			<!-- Section 5: 隐私与安全 -->
			<div class="grid grid-cols-1 lg:grid-cols-[220px_minmax(0,1fr)] gap-4 lg:gap-10 py-6">
				<div>
					<h2 class="text-sm font-semibold text-text-primary">隐私与安全</h2>
					<p class="text-xs text-text-secondary mt-1 leading-relaxed">
						控制个人公开主页以及地理位置在附近雷达中的可见性。
					</p>
				</div>
				<ListContainer bordered inset>
					<SettingRow
						title="私密账号"
						description="开启后仅关注者可查看你的主页与生活瞬间"
						onRowClick={togglePrivateProfile}
					>
						{#snippet icon()}
							<Lock class="size-4 text-accent-default" strokeWidth={1.5} />
						{/snippet}
						<Toggle on={privateProfile} onToggle={togglePrivateProfile} label="私密账号" />
					</SettingRow>

					<SettingRow
						title="位置共享"
						description="允许附近用户在发现雷达和地图中感知到你"
						onRowClick={toggleLocationSharing}
					>
						{#snippet icon()}
							<MapPin class="size-4 text-accent-default" strokeWidth={1.5} />
						{/snippet}
						<Toggle on={locationSharing} onToggle={toggleLocationSharing} label="位置共享" />
					</SettingRow>
				</ListContainer>
			</div>

			<!-- Section 5: 系统与关于 -->
			<div class="grid grid-cols-1 lg:grid-cols-[220px_minmax(0,1fr)] gap-4 lg:gap-10 py-6">
				<div>
					<h2 class="text-sm font-semibold text-text-primary">系统与关于</h2>
					<p class="text-xs text-text-secondary mt-1 leading-relaxed">
						客户端版本及当前设备会话状态。
					</p>
				</div>
				<div class="flex flex-col gap-3">
					<div class="flex items-center justify-between rounded-lg border border-border-default bg-surface-primary p-4 shadow-raised-sm">
						<div class="flex items-center gap-3">
							<div class="flex size-9 items-center justify-center rounded-full icon-badge shrink-0">
								<Info class="size-4 text-accent-default" strokeWidth={1.5} />
							</div>
							<div>
								<p class="text-sm font-medium text-text-primary">关于 HotSpot</p>
								<p class="text-xs text-text-secondary">基于地理位置的实时社交与社区发现应用</p>
							</div>
						</div>
						<span class="badge text-xs font-mono font-medium border border-border-default bg-transparent text-text-secondary">{APP_VERSION}</span>
					</div>

					{#if auth.isAuthenticated}
						<div class="flex justify-end pt-2">
							<button
								type="button"
								class="hover-btn inline-flex items-center gap-2 px-4 py-2 rounded-md focus-ring text-xs font-medium text-status-danger hover:bg-danger-bg border border-border-default bg-surface-primary transition-colors"
								onclick={() => (confirmLogoutOpen = true)}
							>
								<LogOut class="size-3.5" strokeWidth={1.75} />
								退出登录当前账号
							</button>
						</div>
					{/if}
				</div>
			</div>
		</div>

		{#if auth.isAuthenticated}
			<Dialog.Root bind:open={confirmLogoutOpen}>
				<Dialog.Content class="max-w-90 p-0">
					<div class="px-5 pt-6 pb-4">
						<div class="mb-3.5 flex size-10 items-center justify-center rounded-full bg-status-danger/10 text-status-danger ring-1 ring-status-danger/20">
							<LogOut class="size-5" />
						</div>
						<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">退出登录？</Dialog.Title>
						<Dialog.Description class="mt-1 text-sm text-text-secondary leading-relaxed">退出后将保持游客浏览，发布、关注等操作需要重新登录。</Dialog.Description>
					</div>
					<div class="flex flex-col gap-2 px-5 pb-5">
						<Button type="button" variant="danger" size="lg" class="w-full" onclick={() => void handleLogout()}>退出登录</Button>
						<Button type="button" variant="secondary" size="lg" class="w-full" onclick={() => (confirmLogoutOpen = false)}>取消</Button>
					</div>
				</Dialog.Content>
			</Dialog.Root>
		{/if}
	{/if}
</StandardPageLayout>
