<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { Bell, Lock, ChevronRight, LogOut, Globe, Info, Volume2, MapPin, User, Palette, Trees, Moon, Sun } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import Toggle from '@/components/forms/Toggle.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import { logout } from '@/stores/auth.svelte';
	import { theme, setTheme, toggleDark, THEME_OPTIONS } from '@/stores/theme.svelte';
	import { getAccount, getSettings, updateSettings } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 设置页：通知/声音/隐私开关（iOS 式）、语言、账号信息、退出登录。
	// 服务端字段对齐 spec UserSettings：位置共享 → location_visible、私密账号 →
	// privacy_level；推送通知/应用声音 spec 无服务端字段，是本地客户端偏好，
	// 用 localStorage 持久化（不冒充服务端设置）。

	// 本地偏好（推送通知/应用声音）——localStorage 持久化
	const NOTIF_KEY = 'hotspot:notif';
	const SOUNDS_KEY = 'hotspot:sounds';
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
	let sounds = $state(readLocal(SOUNDS_KEY, true));
	let privateProfile = $state(false);
	let locationSharing = $state(true);
	let accountName = $state('');
	let accountEmail = $state('');
	let accountAvatar = $state('');
	let loading = $state(true);

	// 服务端设置开关：乐观更新 + 失败回滚（spec: users/update_settings）
	const toggleLocationSharing = () => {
		const next = !locationSharing;
		locationSharing = next;
		void updateSettings({ settings: { location_visible: next } }).then((res) => {
			if (res.code !== 0) locationSharing = !next;
		});
	};
	const togglePrivateProfile = () => {
		const next = !privateProfile;
		privateProfile = next;
		void updateSettings({ settings: { privacy_level: next ? 'private' : 'public' } }).then((res) => {
			if (res.code !== 0) privateProfile = !next;
		});
	};
	const toggleNotif = () => {
		notif = !notif;
		writeLocal(NOTIF_KEY, notif);
	};
	const toggleSounds = () => {
		sounds = !sounds;
		writeLocal(SOUNDS_KEY, sounds);
	};

	$effect(() => {
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				const [accountRes, settingsRes] = await Promise.all([
					getAccount().catch(() => ({ code: -1, data: null })),
					getSettings().catch(() => ({ code: -1, data: null }))
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
			} catch {
				// empty data is normal
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchData();
		return () => {
			cancelled = true;
		};
	});

	const handleLogout = async () => {
		// auth store logout clears client auth state and keeps the user on the
		// current route (anonymous browsing). No more redirect to a dead /login.
		await logout();
		void goto(ROUTES.home);
	};
</script>

<StandardPageLayout contentClassName="py-5 pb-10">
	{#snippet header()}
		<PageHeader title="设置" sticky={false} onBack={() => history.back()} showBackOnDesktop />
	{/snippet}

	<!-- User card — hero with ambient glow -->
	<div class="ambient-glow mb-3">
		<button type="button" class="hover-btn flex items-center gap-4 px-4 py-3.5 rounded-panel w-full outline-none text-left surface-hero">
			<Avatar class="border-2 border-surface-primary shadow-[0_4px_12px_var(--cst-glow-accent)]" src={accountAvatar} name={accountName} size={48}  />
			<div class="flex-1 min-w-0">
				<p class="text-base tracking-tight text-text-primary">{accountName || '用户'}</p>
				<p class="text-xs mt-0.5 text-text-secondary">{accountEmail || '未绑定邮箱'}</p>
			</div>
			<ChevronRight class="size-3.75 shrink-0"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
		</button>
	</div>

	<!-- Appearance（对齐 Figma Make SettingsPage：应用主题二选一 + 深色模式） -->
	<p class="px-1 mt-3 mb-1.5 text-text-secondary text-xxs tracking-[0.14em] uppercase">外观</p>
	<div class="rounded-panel overflow-hidden surface-flat">
		<ListContainer>
			<div class="px-4 py-3.5">
				<div class="flex items-center gap-3">
					<div class="flex items-center justify-center rounded-full shrink-0 icon-badge size-9" >
						<Palette class="size-3.75"  style="color:var(--cst-accent-default)" strokeWidth={1.5} />
					</div>
					<div class="flex-1 min-w-0">
						<p class="text-base text-text-primary">应用主题</p>
						<p class="text-xs text-text-secondary">选择一套全局视觉风格</p>
					</div>
				</div>
				<div class="mt-3 grid grid-cols-2 gap-2">
					{#each THEME_OPTIONS as t}
						<button
							type="button"
							aria-pressed={theme.theme === t.id}
							onclick={() => setTheme(t.id)}
							class="flex items-center gap-2 border px-3 py-2.5 text-left outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default"
							style="border-color:{theme.theme === t.id ? 'var(--cst-accent-default)' : 'var(--cst-border-default)'};background:{theme.theme === t.id ? 'var(--cst-accent-soft)' : 'transparent'};color:var(--cst-text-primary);"
						>
							{#if t.id === 'forest'}
								<Trees class="size-3.75"  style="color:var(--cst-accent-default)" strokeWidth={1.7} />
							{:else}
								<Palette class="size-3.75"  style="color:var(--cst-accent-default)" strokeWidth={1.7} />
							{/if}
							<span class="text-xs">{t.label}</span>
						</button>
					{/each}
				</div>
			</div>
			<div class="flex items-center gap-3 px-4 py-3.5">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge size-9" >
					{#if theme.isDark}
						<Moon class="size-3.75"  style="color:var(--cst-accent-default)" strokeWidth={1.5} />
					{:else}
						<Sun class="size-3.75"  style="color:var(--cst-accent-default)" strokeWidth={1.5} />
					{/if}
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-base text-text-primary">深色模式</p>
					<p class="text-xs text-text-secondary">{theme.isDark ? '当前使用深色外观' : '当前使用浅色外观'}</p>
				</div>
				<Toggle on={theme.isDark} onToggle={toggleDark} label="深色模式" />
			</div>
		</ListContainer>
	</div>

	<!-- Notifications -->
	<p class="px-1 mt-3 mb-1.5 text-text-secondary text-xxs tracking-[0.14em] uppercase">通知</p>
	<div class="rounded-panel overflow-hidden surface-flat">
		<ListContainer>
			<div class="flex items-center gap-3 px-4 py-3.5">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge size-9" >
					<Bell class="size-3.75"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-base text-text-primary">推送通知</p>
					<p class="text-xs text-text-secondary">消息、提及与新动态提醒</p>
				</div>
				<Toggle on={notif} onToggle={toggleNotif} />
			</div>
			<div class="flex items-center gap-3 px-4 py-3.5">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge size-9" >
					<Volume2 class="size-3.75"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-base text-text-primary">应用声音</p>
					<p class="text-xs text-text-secondary">操作反馈音效</p>
				</div>
				<Toggle on={sounds} onToggle={toggleSounds} />
			</div>
		</ListContainer>
	</div>

	<!-- Privacy -->
	<p class="px-1 mt-3 mb-1.5 text-text-secondary text-xxs tracking-[0.14em] uppercase">隐私与安全</p>
	<div class="rounded-panel overflow-hidden surface-flat">
		<ListContainer>
			<div class="flex items-center gap-3 px-4 py-3.5">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge size-9" >
					<Lock class="size-3.75"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-base text-text-primary">私密账号</p>
					<p class="text-xs text-text-secondary">仅关注者可查看你的内容</p>
				</div>
				<Toggle on={privateProfile} onToggle={togglePrivateProfile} />
			</div>
			<div class="flex items-center gap-3 px-4 py-3.5">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge size-9" >
					<MapPin class="size-3.75"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-base text-text-primary">位置共享</p>
					<p class="text-xs text-text-secondary">允许附近用户发现你</p>
				</div>
				<Toggle on={locationSharing} onToggle={toggleLocationSharing} />
			</div>
		</ListContainer>
	</div>

	<!-- General -->
	<p class="px-1 mt-3 mb-1.5 text-text-secondary text-xxs tracking-[0.14em] uppercase">通用</p>
	<div class="rounded-panel overflow-hidden surface-flat">
		<ListContainer>
			<ListItem class="flex items-center gap-3 px-4 py-3.5" ariaLabel="语言设置">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge size-9" >
					<Globe class="size-3.75"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-base text-text-primary">语言</p>
				</div>
				<span class="text-sm shrink-0 text-text-secondary">中文</span>
				<ChevronRight class="size-3.75 shrink-0"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
			</ListItem>
			<ListItem class="flex items-center gap-3 px-4 py-3.5" ariaLabel="账号信息">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge size-9" >
					<User class="size-3.75"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-base text-text-primary">账号信息</p>
				</div>
				<span class="text-sm shrink-0 text-text-secondary"></span>
				<ChevronRight class="size-3.75 shrink-0"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
			</ListItem>
			<ListItem class="flex items-center gap-3 px-4 py-3.5" ariaLabel="关于">
				<div class="flex items-center justify-center rounded-full shrink-0 icon-badge size-9" >
					<Info class="size-3.75"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
				</div>
				<div class="flex-1 min-w-0">
					<p class="text-base text-text-primary">关于</p>
				</div>
				<span class="text-sm shrink-0 text-text-secondary">v 1.0.0</span>
				<ChevronRight class="size-3.75 shrink-0"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
			</ListItem>
		</ListContainer>
	</div>

	<!-- Log Out — flat, less visually heavy -->
	<div class="mt-5">
		<button
			type="button"
			class="hover-btn flex items-center justify-center gap-2 px-4 py-3 rounded-control w-full outline-none surface-flat text-accent-default"
			onclick={() => void handleLogout()}
		>
			<LogOut class="size-3.5"  strokeWidth={1.75} />
			<span class="text-base">退出登录</span>
		</button>
	</div>
</StandardPageLayout>
