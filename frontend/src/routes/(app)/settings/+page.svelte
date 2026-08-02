<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import { Bell, Lock, ChevronRight, LogOut, Globe, Info, Volume2, MapPin, User } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import NeuToggle from '@/components/forms/NeuToggle.svelte';
	import { logout } from '@/stores/auth.svelte';
	import { getAccount, getSettings } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 设置页：通知/声音/隐私开关（iOS 式）、语言、账号信息、退出登录。
	// 迁移自旧版 SettingsPage。

	let notif = $state(true);
	let sounds = $state(true);
	let privateProfile = $state(false);
	let locationSharing = $state(true);
	let accountName = $state('');
	let accountHandle = $state('');
	let accountEmail = $state('');
	let accountAvatar = $state('');
	let loading = $state(true);

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

				const accountData = (accountRes.code === 0 ? accountRes.data : null) as any;
				const settingsData = (settingsRes.code === 0 ? settingsRes.data : null) as any;

				if (accountData) {
					accountName = accountData.name ?? accountData.nickname ?? '';
					accountHandle = accountData.handle ?? accountData.username ?? '';
					accountEmail = accountData.email ?? '';
					accountAvatar = accountData.avatar_url ?? accountData.avatar ?? '';
				}

				if (settingsData) {
					if (settingsData.notifications !== undefined) notif = settingsData.notifications;
					if (settingsData.sounds !== undefined) sounds = settingsData.sounds;
					if (settingsData.private_profile !== undefined) privateProfile = settingsData.private_profile;
					if (settingsData.location_sharing !== undefined) locationSharing = settingsData.location_sharing;
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

<PageHeader title="设置" subtitle="管理你的偏好与隐私" sticky onBack={() => history.back()} />

<!-- Page content -->
<div class="content-mode-reading flex flex-col px-5 pt-4 pb-10" style="max-width:var(--content-reading-max);margin-left:auto;margin-right:auto;width:100%;">
	<!-- User card — hero with ambient glow -->
	<div class="neu-ambient-glow mb-3">
		<button type="button" class="neu-hover-btn flex items-center gap-4 px-4 py-3.5 rounded-[20px] w-full outline-none text-left neu-surface-hero">
			<Avatar src={accountAvatar} name={accountName} size={48} style="border:2px solid var(--neu-bg);box-shadow:0 4px 12px var(--neu-glow);" />
			<div class="flex-1 min-w-0">
				<p class="text-[15px] tracking-tight text-[var(--neu-text)]">{accountName || '用户'}</p>
				<p class="text-[11px] mt-0.5 text-[var(--neu-text-muted)]">{accountHandle}{accountEmail ? ` · ${accountEmail}` : ''}</p>
			</div>
			<ChevronRight style="width:15px;height:15px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.5} />
		</button>
	</div>

	<!-- Notifications -->
	<p class="px-1 mt-3 mb-1.5 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em] uppercase">通知</p>
	<div class="rounded-[22px] overflow-hidden neu-surface-flat">
		<div class="flex items-center gap-3 px-4 py-3.5">
			<div class="flex items-center justify-center rounded-full flex-shrink-0 neu-icon-badge" style="width:36px;height:36px;">
				<Bell style="width:15px;height:15px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-[14px] text-[var(--neu-text)]">推送通知</p>
				<p class="text-[11px] text-[var(--neu-text-muted)]">消息、提及与新动态提醒</p>
			</div>
			<NeuToggle on={notif} onToggle={() => (notif = !notif)} />
		</div>
		<div class="mx-4 bg-[var(--neu-border)]" style="height:1px;"></div>
		<div class="flex items-center gap-3 px-4 py-3.5">
			<div class="flex items-center justify-center rounded-full flex-shrink-0 neu-icon-badge" style="width:36px;height:36px;">
				<Volume2 style="width:15px;height:15px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-[14px] text-[var(--neu-text)]">应用声音</p>
				<p class="text-[11px] text-[var(--neu-text-muted)]">操作反馈音效</p>
			</div>
			<NeuToggle on={sounds} onToggle={() => (sounds = !sounds)} />
		</div>
	</div>

	<!-- Privacy -->
	<p class="px-1 mt-3 mb-1.5 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em] uppercase">隐私与安全</p>
	<div class="rounded-[22px] overflow-hidden neu-surface-flat">
		<div class="flex items-center gap-3 px-4 py-3.5">
			<div class="flex items-center justify-center rounded-full flex-shrink-0 neu-icon-badge" style="width:36px;height:36px;">
				<Lock style="width:15px;height:15px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-[14px] text-[var(--neu-text)]">私密账号</p>
				<p class="text-[11px] text-[var(--neu-text-muted)]">仅关注者可查看你的内容</p>
			</div>
			<NeuToggle on={privateProfile} onToggle={() => (privateProfile = !privateProfile)} />
		</div>
		<div class="mx-4 bg-[var(--neu-border)]" style="height:1px;"></div>
		<div class="flex items-center gap-3 px-4 py-3.5">
			<div class="flex items-center justify-center rounded-full flex-shrink-0 neu-icon-badge" style="width:36px;height:36px;">
				<MapPin style="width:15px;height:15px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-[14px] text-[var(--neu-text)]">位置共享</p>
				<p class="text-[11px] text-[var(--neu-text-muted)]">允许附近用户发现你</p>
			</div>
			<NeuToggle on={locationSharing} onToggle={() => (locationSharing = !locationSharing)} />
		</div>
	</div>

	<!-- General -->
	<p class="px-1 mt-3 mb-1.5 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em] uppercase">通用</p>
	<div class="rounded-[22px] overflow-hidden neu-surface-flat">
		<button type="button" class="neu-hover-bg flex items-center gap-3 px-4 py-3.5 w-full outline-none text-left" style="transition:background 0.16s ease;">
			<div class="flex items-center justify-center rounded-full flex-shrink-0 neu-icon-badge" style="width:36px;height:36px;">
				<Globe style="width:15px;height:15px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-[14px] text-[var(--neu-text)]">语言</p>
			</div>
			<span class="text-[12px] flex-shrink-0 text-[var(--neu-text-muted)]">中文</span>
			<ChevronRight style="width:15px;height:15px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.5} />
		</button>
		<div class="mx-4 bg-[var(--neu-border)]" style="height:1px;"></div>
		<button type="button" class="neu-hover-bg flex items-center gap-3 px-4 py-3.5 w-full outline-none text-left" style="transition:background 0.16s ease;">
			<div class="flex items-center justify-center rounded-full flex-shrink-0 neu-icon-badge" style="width:36px;height:36px;">
				<User style="width:15px;height:15px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-[14px] text-[var(--neu-text)]">账号信息</p>
			</div>
			<span class="text-[12px] flex-shrink-0 text-[var(--neu-text-muted)]"></span>
			<ChevronRight style="width:15px;height:15px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.5} />
		</button>
		<div class="mx-4 bg-[var(--neu-border)]" style="height:1px;"></div>
		<button type="button" class="neu-hover-bg flex items-center gap-3 px-4 py-3.5 w-full outline-none text-left" style="transition:background 0.16s ease;">
			<div class="flex items-center justify-center rounded-full flex-shrink-0 neu-icon-badge" style="width:36px;height:36px;">
				<Info style="width:15px;height:15px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-[14px] text-[var(--neu-text)]">关于</p>
			</div>
			<span class="text-[12px] flex-shrink-0 text-[var(--neu-text-muted)]">v 1.0.0</span>
			<ChevronRight style="width:15px;height:15px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.5} />
		</button>
	</div>

	<!-- Log Out — flat, less visually heavy -->
	<div class="mt-5">
		<button
			type="button"
			class="neu-hover-btn flex items-center justify-center gap-2 px-4 py-3 rounded-[14px] w-full outline-none neu-surface-flat text-[var(--neu-accent-primary)]"
			onclick={() => void handleLogout()}
		>
			<LogOut style="width:14px;height:14px;" strokeWidth={1.75} />
			<span class="text-[14px]">退出登录</span>
		</button>
	</div>
</div>
