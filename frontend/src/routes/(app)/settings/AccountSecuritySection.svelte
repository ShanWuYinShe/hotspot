<script lang="ts">
	import { KeyRound, Smartphone, UserRoundPlus, ShieldCheck, X } from '@lucide/svelte';
	import SettingRow from '@/components/forms/SettingRow.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { auth, refreshUser } from '@/stores/auth.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import {
		changePassword, fetchSessions, revokeSession, revokeOtherSessions, upgradeAccount,
		netError,
	} from '@/api';
	import type { SessionItem } from '@/api/types';
	import { formatDateTime } from '@/lib/utils';

	// 设置页「账号与安全」区块（页面私有组件，SettingsSection 骨架内嵌）：
	//   - 修改密码（auth/change_password；成功后服务端自动吊销其余会话）
	//   - 登录设备管理（auth/fetch_sessions + revoke_session/revoke_other_sessions；
	//     当前会话不可在列内撤销——spec 10400，登出走页面底部「退出登录」）
	//   - 游客升级（users/upgrade；仅 user_type='temp' 时出现，绑定用户名+密码）
	// 表单均为行内展开（extension），不新增弹窗交互模式。

	const isGuest = $derived(auth.user?.user_type === 'temp');

	// ── 游客升级 ────────────────────────────────────────
	let upgradeOpen = $state(false);
	let upgradeLoginId = $state('');
	let upgradePassword = $state('');
	let upgradeConfirm = $state('');
	let upgradeSaving = $state(false);
	let upgradeError = $state<string | null>(null);

	const submitUpgrade = async () => {
		if (upgradeSaving) return;
		const loginId = upgradeLoginId.trim();
		if (!loginId || !upgradePassword) {
			upgradeError = '请填写用户名和密码';
			return;
		}
		if (upgradePassword !== upgradeConfirm) {
			upgradeError = '两次输入的密码不一致';
			return;
		}
		upgradeError = null;
		upgradeSaving = true;
		try {
			// spec: users/upgrade（login_id 仅支持用户名型；10400=已被占用/已是正式用户）
			const res = await upgradeAccount({ login_id: loginId, password: upgradePassword });
			if (res.code !== 0) {
				upgradeError = res.code === 10400 ? '该用户名已被占用，换一个试试' : (res.message || '升级失败，请重试');
				return;
			}
			await refreshUser();
			upgradeOpen = false;
			upgradeLoginId = '';
			upgradePassword = '';
			upgradeConfirm = '';
			showToast('升级成功，账号已永久保留', 'success');
		} catch {
			upgradeError = '升级失败，请重试';
		} finally {
			upgradeSaving = false;
		}
	};

	// ── 修改密码 ────────────────────────────────────────
	let pwdOpen = $state(false);
	let oldPassword = $state('');
	let newPassword = $state('');
	let pwdConfirm = $state('');
	let pwdSaving = $state(false);
	let pwdError = $state<string | null>(null);

	const submitPassword = async () => {
		if (pwdSaving) return;
		if (!oldPassword || !newPassword) {
			pwdError = '请填写旧密码和新密码';
			return;
		}
		if (newPassword !== pwdConfirm) {
			pwdError = '两次输入的新密码不一致';
			return;
		}
		pwdError = null;
		pwdSaving = true;
		try {
			// spec: auth/change_password（10401=旧密码错误；成功后其余会话自动吊销）
			const res = await changePassword({ old_password: oldPassword, new_password: newPassword });
			if (res.code !== 0) {
				pwdError = res.code === 10401 ? '旧密码错误' : (res.message || '修改失败，请重试');
				return;
			}
			pwdOpen = false;
			oldPassword = '';
			newPassword = '';
			pwdConfirm = '';
			showToast('密码已修改，其他设备已自动下线', 'success');
		} catch {
			pwdError = '修改失败，请重试';
		} finally {
			pwdSaving = false;
		}
	};

	// ── 登录设备 ────────────────────────────────────────
	let sessionsOpen = $state(false);
	let sessions = $state<SessionItem[] | null>(null);
	let sessionsLoading = $state(false);
	let sessionsError = $state<string | null>(null);
	let revokeTarget = $state<SessionItem | null>(null);
	let revoking = $state(false);
	let revokeAllOpen = $state(false);

	const otherSessions = $derived(sessions?.filter((s) => !s.is_current) ?? []);

	const loadSessions = async () => {
		sessionsLoading = true;
		sessionsError = null;
		try {
			const res = await fetchSessions().catch(() => netError(null));
			if (res.code === 0 && res.data) {
				sessions = res.data.items;
			} else {
				sessionsError = res.message || '加载失败，请稍后重试';
			}
		} finally {
			sessionsLoading = false;
		}
	};

	const toggleSessions = () => {
		sessionsOpen = !sessionsOpen;
		// 首次展开拉取；之后保留本地状态（撤销后同步移除，避免重复请求）
		if (sessionsOpen && sessions === null) void loadSessions();
	};

	const confirmRevoke = async () => {
		if (!revokeTarget || revoking) return;
		revoking = true;
		try {
			const res = await revokeSession({ session_id: revokeTarget.session_id });
			if (res.code === 0) {
				sessions = sessions?.filter((s) => s.session_id !== revokeTarget?.session_id) ?? [];
				showToast('该设备已下线', 'success');
			} else {
				showToast(res.message || '操作失败，请重试', 'danger');
			}
		} catch {
			showToast('操作失败，请重试', 'danger');
		} finally {
			revoking = false;
			revokeTarget = null;
		}
	};

	const confirmRevokeAll = async () => {
		if (revoking) return;
		revoking = true;
		try {
			const res = await revokeOtherSessions();
			if (res.code === 0) {
				sessions = sessions?.filter((s) => s.is_current) ?? [];
				showToast(`已下线 ${res.data?.revoked_count ?? otherSessions.length} 台其他设备`, 'success');
			} else {
				showToast(res.message || '操作失败，请重试', 'danger');
			}
		} catch {
			showToast('操作失败，请重试', 'danger');
		} finally {
			revoking = false;
			revokeAllOpen = false;
		}
	};
</script>

<SettingRow
	title="登录设备"
	description="查看活跃会话并下线陌生设备"
	onRowClick={toggleSessions}
>
	{#snippet icon()}
		<Smartphone class="size-4 text-accent-default" strokeWidth={1.5} />
	{/snippet}
	{#snippet children()}
		<span class="text-xs text-text-secondary" aria-hidden="true">{sessionsOpen ? '收起' : '展开'}</span>
	{/snippet}
	{#snippet extension()}
		{#if sessionsOpen}
			<div class="flex flex-col gap-2 pb-1">
				{#if sessionsLoading}
					<InlineLoading label="加载设备列表…" />
				{:else if sessionsError}
					<LoadError message={sessionsError} onRetry={() => void loadSessions()} />
				{:else if sessions}
					<ListContainer bordered={false} inset={false} ariaLabel="活跃会话列表">
						{#each sessions as s (s.session_id)}
							<ListItem class="px-3 py-2.5" ariaLabel={s.is_current ? `当前设备 ${s.device_name ?? s.device_type ?? ''}` : `设备 ${s.device_name ?? s.device_type ?? ''}`}>
								<div class="flex min-w-0 flex-1 items-center gap-3">
									<div class="flex size-8 shrink-0 items-center justify-center rounded-full icon-badge">
										<Smartphone class="size-3.5 text-accent-default" strokeWidth={1.5} />
									</div>
									<div class="min-w-0 flex-1">
										<div class="flex items-center gap-2">
											<p class="truncate text-sm text-text-primary">{s.device_name ?? s.device_type ?? '未知设备'}</p>
											{#if s.is_current}
												<span class="inline-flex h-5 items-center justify-center rounded-full bg-accent-soft px-2 text-xxs font-medium leading-none text-accent-default">当前设备</span>
											{/if}
										</div>
										<p class="mt-0.5 truncate text-xs text-text-secondary">
											{[s.ip_location, s.ip_address].filter(Boolean).join(' · ') || '位置未知'}
											· 最近活跃 {formatDateTime(s.last_active_at)}
										</p>
									</div>
									{#if !s.is_current}
										<button
											type="button"
											class="hover-btn hit-slop flex shrink-0 items-center gap-1 rounded-full border border-border-default bg-surface-primary px-2.5 py-1 text-xs text-text-secondary focus-ring hover:bg-accent-soft"
											aria-label={`下线设备 ${s.device_name ?? s.device_type ?? ''}`}
											onclick={() => (revokeTarget = s)}
										>
											<X class="size-3" strokeWidth={2} />
											下线
										</button>
									{/if}
								</div>
							</ListItem>
						{/each}
					</ListContainer>
					{#if otherSessions.length > 0}
						<button
							type="button"
							class="hover-btn flex w-full items-center justify-center gap-1.5 rounded-sm border border-border-default py-2 text-xs text-status-danger focus-ring hover:bg-danger-bg"
							onclick={() => (revokeAllOpen = true)}
						>
							<ShieldCheck class="size-3.5" strokeWidth={1.75} />
							下线其他全部设备（{otherSessions.length}）
						</button>
					{/if}
				{/if}
			</div>
		{/if}
	{/snippet}
</SettingRow>

{#if isGuest}
	<!-- 游客账号（user_type='temp'）：绑定登录凭证升级为正式账号（spec: users/upgrade） -->
	<SettingRow
		title="升级为正式账号"
		description="绑定用户名和密码，游客期的资料将完整保留"
		onRowClick={() => (upgradeOpen = !upgradeOpen)}
	>
		{#snippet icon()}
			<UserRoundPlus class="size-4 text-accent-default" strokeWidth={1.5} />
		{/snippet}
		{#snippet children()}
			<span class="text-xs text-text-secondary" aria-hidden="true">{upgradeOpen ? '收起' : '展开'}</span>
		{/snippet}
		{#snippet extension()}
			{#if upgradeOpen}
				<form class="flex flex-col gap-2.5 pb-1" onsubmit={(e) => { e.preventDefault(); void submitUpgrade(); }}>
					<label class="block">
						<span class="text-xs tracking-field text-text-secondary">用户名</span>
						<input
							type="text"
							bind:value={upgradeLoginId}
							autocomplete="username"
							aria-label="用户名"
							placeholder="用于日后登录"
							class="mt-1 w-full input-surface rounded-md px-3 py-2 text-sm text-text-primary"
						/>
					</label>
					<label class="block">
						<span class="text-xs tracking-field text-text-secondary">密码</span>
						<input
							type="password"
							bind:value={upgradePassword}
							autocomplete="new-password"
							aria-label="设置密码"
							placeholder="设置登录密码"
							class="mt-1 w-full input-surface rounded-md px-3 py-2 text-sm text-text-primary"
						/>
					</label>
					<label class="block">
						<span class="text-xs tracking-field text-text-secondary">确认密码</span>
						<input
							type="password"
							bind:value={upgradeConfirm}
							autocomplete="new-password"
							aria-label="确认密码"
							placeholder="再输入一次密码"
							class="mt-1 w-full input-surface rounded-md px-3 py-2 text-sm text-text-primary"
						/>
					</label>
					{#if upgradeError}
						<p class="text-xs text-status-danger" role="alert">{upgradeError}</p>
					{/if}
					<button
						type="submit"
						disabled={upgradeSaving}
						class="hover-btn mt-1 flex w-full items-center justify-center gap-2 rounded-full bg-accent-default py-2.5 text-sm font-medium text-text-on-accent shadow-raised-sm hover:shadow-raised-md focus-ring disabled:opacity-muted"
					>
						{upgradeSaving ? '升级中…' : '完成升级'}
					</button>
				</form>
			{/if}
		{/snippet}
	</SettingRow>
{:else}
	<SettingRow
		title="修改密码"
		description="定期更换密码，保护账号安全"
		onRowClick={() => (pwdOpen = !pwdOpen)}
	>
		{#snippet icon()}
			<KeyRound class="size-4 text-accent-default" strokeWidth={1.5} />
		{/snippet}
		{#snippet children()}
			<span class="text-xs text-text-secondary" aria-hidden="true">{pwdOpen ? '收起' : '展开'}</span>
		{/snippet}
		{#snippet extension()}
			{#if pwdOpen}
				<form class="flex flex-col gap-2.5 pb-1" onsubmit={(e) => { e.preventDefault(); void submitPassword(); }}>
					<label class="block">
						<span class="text-xs tracking-field text-text-secondary">旧密码</span>
						<input
							type="password"
							bind:value={oldPassword}
							autocomplete="current-password"
							aria-label="旧密码"
							placeholder="输入当前密码"
							class="mt-1 w-full input-surface rounded-md px-3 py-2 text-sm text-text-primary"
						/>
					</label>
					<label class="block">
						<span class="text-xs tracking-field text-text-secondary">新密码</span>
						<input
							type="password"
							bind:value={newPassword}
							autocomplete="new-password"
							aria-label="新密码"
							placeholder="设置新密码"
							class="mt-1 w-full input-surface rounded-md px-3 py-2 text-sm text-text-primary"
						/>
					</label>
					<label class="block">
						<span class="text-xs tracking-field text-text-secondary">确认新密码</span>
						<input
							type="password"
							bind:value={pwdConfirm}
							autocomplete="new-password"
							aria-label="确认新密码"
							placeholder="再输入一次新密码"
							class="mt-1 w-full input-surface rounded-md px-3 py-2 text-sm text-text-primary"
						/>
					</label>
					{#if pwdError}
						<p class="text-xs text-status-danger" role="alert">{pwdError}</p>
					{/if}
					<button
						type="submit"
						disabled={pwdSaving}
						class="hover-btn mt-1 flex w-full items-center justify-center gap-2 rounded-full bg-accent-default py-2.5 text-sm font-medium text-text-on-accent shadow-raised-sm hover:shadow-raised-md focus-ring disabled:opacity-muted"
					>
						{pwdSaving ? '提交中…' : '确认修改'}
					</button>
				</form>
			{/if}
		{/snippet}
	</SettingRow>
{/if}

<ConfirmDialog
	open={revokeTarget !== null}
	title="下线该设备？"
	description="下线后该设备上的登录会立即失效，需要重新登录。"
	icon={Smartphone}
	confirmLabel="确认下线"
	loading={revoking}
	onCancel={() => (revokeTarget = null)}
	onConfirm={() => void confirmRevoke()}
/>

<ConfirmDialog
	open={revokeAllOpen}
	title="下线其他全部设备？"
	description="适用于账号疑似被盗的场景：其他所有设备将立即退出，需重新登录。"
	icon={ShieldCheck}
	confirmLabel="全部下线"
	loading={revoking}
	onCancel={() => (revokeAllOpen = false)}
	onConfirm={() => void confirmRevokeAll()}
/>
