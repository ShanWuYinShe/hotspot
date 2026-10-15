<script lang="ts">
	import { adminListPermissions, adminListRoles, adminCreateRole, adminUpdateRole, adminDeleteRole, adminSetRolePermissions, adminListAdmins, adminAwardAdmin, adminRevokeAdmin } from '@/api';
	import type { AdminPermissionItem, AdminRoleItem, AdminOperatorItem } from '@/api/types';
	import { hasAdminPermission, isSuperAdmin } from '@/stores/admin.svelte';
	import { showToast } from '@/stores/toast.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import Button from '@/components/ui/button.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { TABLE_WRAP, TABLE, TH, TR, TD, TD_ACTIONS, ROW_BTN, ROW_BTN_DANGER, ROW_BTN_PRIMARY } from '../shared/table';
	import { formatDateTime } from '@/lib/utils';

	// 角色权限（RBAC）：角色 CRUD + 权限设置（权限字典多选）+ 平台管理员授予/撤销。
	// spec 约束：仅 superadmin 可设置 '*'；award_all_permissions 与 role_ids 互斥。

	const canManage = $derived(hasAdminPermission('admin.rbac.manage'));
	const isSuper = $derived(isSuperAdmin());

	// ── 角色 ──
	let permissions = $state<AdminPermissionItem[]>([]);
	let roles = $state<AdminRoleItem[]>([]);
	let rolesLoading = $state(true);
	let rolesError = $state<string | null>(null);
	let opPending = $state(false);

	async function loadRoles() {
		rolesLoading = true;
		rolesError = null;
		const [permRes, roleRes] = await Promise.all([
			adminListPermissions().catch(() => ({ code: -1, message: '网络异常', data: null }) as const),
			adminListRoles().catch(() => ({ code: -1, message: '网络异常', data: null }) as const)
		]);
		if (permRes.code === 0 && permRes.data) permissions = permRes.data.items ?? [];
		if (roleRes.code === 0 && roleRes.data) roles = roleRes.data.items ?? [];
		if (roleRes.code !== 0) rolesError = roleRes.message ?? '加载失败';
		rolesLoading = false;
	}

	$effect(() => {
		if (!canManage) {
			rolesLoading = false;
			return;
		}
		void loadRoles();
	});

	// 角色创建/编辑弹层
	let editOpen = $state(false);
	let editTarget = $state<AdminRoleItem | null>(null);
	let roleName = $state('');
	let roleDesc = $state('');

	const openCreate = () => {
		editTarget = null;
		roleName = '';
		roleDesc = '';
		editOpen = true;
	};
	const openEdit = (r: AdminRoleItem) => {
		editTarget = r;
		roleName = r.name;
		roleDesc = r.description ?? '';
		editOpen = true;
	};
	const submitEdit = async () => {
		if (!roleName.trim() || opPending) return;
		opPending = true;
		try {
			const res = editTarget
				? await adminUpdateRole({ role_id: editTarget.role_id, name: roleName.trim(), description: roleDesc.trim() || null, clear_description: !roleDesc.trim() })
				: await adminCreateRole({ name: roleName.trim(), description: roleDesc.trim() || null });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast(editTarget ? '角色已更新' : '角色已创建', 'success');
			editOpen = false;
			void loadRoles();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			opPending = false;
		}
	};

	// 权限设置弹层（'*' 仅 superadmin 可设置）
	let permTarget = $state<AdminRoleItem | null>(null);
	let permDraft = $state<Set<string>>(new Set());
	const openPerms = (r: AdminRoleItem) => {
		permTarget = r;
		permDraft = new Set(r.permissions);
	};
	const togglePerm = (key: string) => {
		const n = new Set(permDraft);
		n.has(key) ? n.delete(key) : n.add(key);
		permDraft = n;
	};
	const submitPerms = async () => {
		if (!permTarget || opPending) return;
		opPending = true;
		try {
			const res = await adminSetRolePermissions({ role_id: permTarget.role_id, permission_keys: Array.from(permDraft) });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast('权限已更新', 'success');
			permTarget = null;
			void loadRoles();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			opPending = false;
		}
	};

	// 角色删除确认
	let deleteTarget = $state<AdminRoleItem | null>(null);
	const submitDelete = async () => {
		if (!deleteTarget || opPending) return;
		opPending = true;
		try {
			const res = await adminDeleteRole({ role_id: deleteTarget.role_id });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast('角色已删除', 'success');
			deleteTarget = null;
			void loadRoles();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			opPending = false;
		}
	};

	// ── 平台管理员 ──
	let admins = $state<AdminOperatorItem[]>([]);
	let adminsLoading = $state(true);
	let adminsError = $state<string | null>(null);
	async function loadAdmins() {
		adminsLoading = true;
		adminsError = null;
		const res = await adminListAdmins({ pagination: { limit: 50 } }).catch(() => ({ code: -1, message: '网络异常', data: null }) as const);
		if (res.code === 0 && res.data) admins = res.data.items ?? [];
		else adminsError = res.message ?? '加载失败';
		adminsLoading = false;
	}
	$effect(() => {
		if (canManage) void loadAdmins();
	});

	// 授予弹层
	let awardOpen = $state(false);
	let awardUserId = $state('');
	let awardAll = $state(false);
	let awardRoleIds = $state<Set<string>>(new Set());
	const openAward = () => {
		awardUserId = '';
		awardAll = false;
		awardRoleIds = new Set();
		awardOpen = true;
	};
	const toggleAwardRole = (id: string) => {
		const n = new Set(awardRoleIds);
		n.has(id) ? n.delete(id) : n.add(id);
		awardRoleIds = n;
	};
	const submitAward = async () => {
		if (!awardUserId.trim() || opPending) return;
		if (!awardAll && awardRoleIds.size === 0) {
			showToast('请选择至少一个角色，或勾选全部权限', 'danger');
			return;
		}
		opPending = true;
		try {
			const res = await adminAwardAdmin({
				user_id: awardUserId.trim(),
				award_all_permissions: awardAll ? true : null,
				role_ids: awardAll ? [] : Array.from(awardRoleIds)
			});
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast('已授予平台管理员', 'success');
			awardOpen = false;
			void loadAdmins();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			opPending = false;
		}
	};

	// 撤销确认
	let revokeTarget = $state<AdminOperatorItem | null>(null);
	const submitRevoke = async () => {
		if (!revokeTarget || opPending) return;
		opPending = true;
		try {
			const res = await adminRevokeAdmin({ user_id: revokeTarget.user.user_id });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
			showToast('已撤销平台管理员', 'success');
			revokeTarget = null;
			void loadAdmins();
		} catch (err) {
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			opPending = false;
		}
	};
</script>

<svelte:head>
	<title>角色权限 · HotSpot</title>
</svelte:head>

{#if !canManage}
	<EmptyState title="无访问权限" description="需要 admin.rbac.manage 权限" />
{:else}
	<div class="flex flex-col gap-8 max-w-5xl">
		<!-- 角色管理 -->
		<section class="flex flex-col gap-4">
			<div class="flex items-center justify-between">
				<div>
					<h2 class="text-sm font-semibold text-text-primary">角色</h2>
					<p class="mt-1 text-xs text-text-secondary">角色绑定权限键，授予平台管理员时选择角色。</p>
				</div>
				<button type="button" class={ROW_BTN_PRIMARY} onclick={openCreate}>新建角色</button>
			</div>

			{#if rolesLoading}
				<InlineLoading label="加载角色…" />
			{:else if rolesError}
				<LoadError message={rolesError} onRetry={() => void loadRoles()} />
			{:else}
				<div class={TABLE_WRAP}>
					<table class={TABLE}>
						<thead>
							<tr>
								<th scope="col" class={TH}>角色</th>
								<th scope="col" class={TH}>描述</th>
								<th scope="col" class={TH}>权限</th>
								<th scope="col" class={TH}></th>
							</tr>
						</thead>
						<tbody>
							{#if roles.length === 0}
								<tr class={TR}><td class={TD} colspan={4}>暂无角色，点击「新建角色」创建。</td></tr>
							{/if}
							{#each roles as r (r.role_id)}
								<tr class={TR}>
									<td class={TD}>{r.name}</td>
									<td class={`${TD} max-w-56 truncate`} title={r.description ?? ''}>{r.description ?? '—'}</td>
									<td class={TD}>
										{#if r.permissions.includes('*')}
											<span class="badge bg-accent-soft text-accent-default">全部权限</span>
										{:else}
											<span class="badge bg-surface-subtle text-text-secondary">{r.permissions.length} 项</span>
										{/if}
									</td>
									<td class={TD_ACTIONS}>
										<span class="flex justify-end gap-1.5">
											<button type="button" class={ROW_BTN} onclick={() => openPerms(r)}>权限</button>
											<button type="button" class={ROW_BTN} onclick={() => openEdit(r)}>编辑</button>
											<button type="button" class={ROW_BTN_DANGER} onclick={() => (deleteTarget = r)}>删除</button>
										</span>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>

		<!-- 平台管理员 -->
		<section class="flex flex-col gap-4">
			<div class="flex items-center justify-between">
				<div>
					<h2 class="text-sm font-semibold text-text-primary">平台管理员</h2>
					<p class="mt-1 text-xs text-text-secondary">按最近授权时间排序；撤销 superadmin 仅 superadmin 可执行。</p>
				</div>
				<button type="button" class={ROW_BTN_PRIMARY} onclick={openAward}>授予管理员</button>
			</div>

			{#if adminsLoading}
				<InlineLoading label="加载管理员列表…" />
			{:else if adminsError}
				<LoadError message={adminsError} onRetry={() => void loadAdmins()} />
			{:else}
				<div class={TABLE_WRAP}>
					<table class={TABLE}>
						<thead>
							<tr>
								<th scope="col" class={TH}>用户</th>
								<th scope="col" class={TH}>权限</th>
								<th scope="col" class={TH}>角色</th>
								<th scope="col" class={TH}>授权时间</th>
								<th scope="col" class={TH}></th>
							</tr>
						</thead>
						<tbody>
							{#each admins as a (a.user.user_id)}
								<tr class={TR}>
									<td class={TD}>
										<span class="text-text-primary">{a.user.nickname}</span>
										<span class="ml-1.5 text-xs text-text-secondary font-mono">{a.user.user_id}</span>
									</td>
									<td class={TD}>
										{#if a.permissions.includes('*')}
											<span class="badge bg-accent-soft text-accent-default">superadmin</span>
										{:else}
											<span class="badge bg-surface-subtle text-text-secondary">{a.permissions.length} 项</span>
										{/if}
									</td>
									<td class={TD}>{a.roles.map((r) => r.name).join('、') || '—'}</td>
									<td class={TD}>{formatDateTime(a.granted_at)}</td>
									<td class={TD_ACTIONS}>
										<button type="button" class={ROW_BTN_DANGER} onclick={() => (revokeTarget = a)}>撤销</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>
	</div>

	<!-- 角色创建/编辑 -->
	<Dialog.Root open={editOpen} onOpenChange={(next) => (next ? undefined : (editOpen = false))}>
		<Dialog.Content class="max-w-90 p-0">
			<div class="px-5 pt-6 pb-4">
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">{editTarget ? '编辑角色' : '新建角色'}</Dialog.Title>
				<div class="mt-4 flex flex-col gap-3">
					<label class="flex flex-col gap-1.5">
						<span class="text-xs text-text-secondary">名称</span>
						<input bind:value={roleName} maxlength={50} class="input-surface rounded-md px-3 py-2 text-sm text-text-primary" />
					</label>
					<label class="flex flex-col gap-1.5">
						<span class="text-xs text-text-secondary">描述（可留空）</span>
						<textarea bind:value={roleDesc} rows={2} maxlength={200} class="input-surface resize-none rounded-md px-3 py-2 text-sm text-text-primary"></textarea>
					</label>
				</div>
			</div>
			<div class="flex justify-end gap-2 px-5 pb-5">
				<Button type="button" variant="secondary" size="md" onclick={() => (editOpen = false)}>取消</Button>
				<Button type="button" variant="primary" size="md" loading={opPending} onclick={() => void submitEdit()}>保存</Button>
			</div>
		</Dialog.Content>
	</Dialog.Root>

	<!-- 权限设置 -->
	<Dialog.Root open={permTarget !== null} onOpenChange={(next) => (next ? undefined : (permTarget = null))}>
		<Dialog.Content class="max-w-105 p-0">
			<div class="px-5 pt-6 pb-4">
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">设置角色权限「{permTarget?.name}」</Dialog.Title>
				<Dialog.Description class="mt-1 text-sm text-text-secondary">权限键来自权限字典；{'*'}（全量权限）仅超级管理员可设置。</Dialog.Description>
				<div class="mt-4 grid max-h-96 grid-cols-1 gap-1.5 overflow-y-auto sm:grid-cols-2">
					{#if isSuper}
						<label class="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-text-primary hover:bg-interactive-hover">
							<input type="checkbox" checked={permDraft.has('*')} onchange={() => togglePerm('*')} />
							<span class="font-mono text-xs">*</span>（全部权限）
						</label>
					{/if}
					{#each permissions as p (p.key)}
						<label class="flex items-start gap-2 rounded-md px-2 py-1.5 hover:bg-interactive-hover" title={p.description}>
							<input type="checkbox" class="mt-0.5" checked={permDraft.has(p.key)} onchange={() => togglePerm(p.key)} />
							<span class="min-w-0">
								<span class="block font-mono text-xs text-text-primary">{p.key}</span>
								<span class="block truncate text-xxs text-text-secondary">{p.description}</span>
							</span>
						</label>
					{/each}
				</div>
			</div>
			<div class="flex justify-end gap-2 px-5 pb-5">
				<Button type="button" variant="secondary" size="md" onclick={() => (permTarget = null)}>取消</Button>
				<Button type="button" variant="primary" size="md" loading={opPending} onclick={() => void submitPerms()}>保存权限</Button>
			</div>
		</Dialog.Content>
	</Dialog.Root>

	<!-- 角色删除确认 -->
	<ConfirmDialog
		open={deleteTarget !== null}
		title="删除角色「{deleteTarget?.name}」？"
		description="持有该角色的平台管理员将失去其权限。"
		confirmLabel="确认删除"
		loading={opPending}
		onCancel={() => (deleteTarget = null)}
		onConfirm={() => void submitDelete()}
	/>

	<!-- 授予管理员 -->
	<Dialog.Root open={awardOpen} onOpenChange={(next) => (next ? undefined : (awardOpen = false))}>
		<Dialog.Content class="max-w-90 p-0">
			<div class="px-5 pt-6 pb-4">
				<Dialog.Title class="text-lg font-semibold tracking-title text-text-primary">授予平台管理员</Dialog.Title>
				<div class="mt-4 flex flex-col gap-3">
					<label class="flex flex-col gap-1.5">
						<span class="text-xs text-text-secondary">用户 ID</span>
						<input bind:value={awardUserId} placeholder="user_xxx" class="input-surface rounded-md px-3 py-2 text-sm font-mono text-text-primary" />
					</label>
					{#if isSuper}
						<label class="flex items-center gap-2 text-sm text-text-primary">
							<input type="checkbox" bind:checked={awardAll} />
							授予全部权限（superadmin；与角色互斥）
						</label>
					{/if}
					{#if !awardAll}
						<div class="flex flex-col gap-1.5">
							<span class="text-xs text-text-secondary">选择角色（必选）</span>
							<div class="flex flex-col gap-1">
								{#each roles as r (r.role_id)}
									<label class="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-text-primary hover:bg-interactive-hover">
										<input type="checkbox" checked={awardRoleIds.has(r.role_id)} onchange={() => toggleAwardRole(r.role_id)} />
										{r.name}
									</label>
								{/each}
							</div>
						</div>
					{/if}
				</div>
			</div>
			<div class="flex justify-end gap-2 px-5 pb-5">
				<Button type="button" variant="secondary" size="md" onclick={() => (awardOpen = false)}>取消</Button>
				<Button type="button" variant="primary" size="md" loading={opPending} onclick={() => void submitAward()}>确认授予</Button>
			</div>
		</Dialog.Content>
	</Dialog.Root>

	<!-- 撤销确认 -->
	<ConfirmDialog
		open={revokeTarget !== null}
		title="撤销 {revokeTarget?.user.nickname} 的平台管理员？"
		description="撤销后该用户立即失去全部平台管理权限。"
		confirmLabel="确认撤销"
		loading={opPending}
		onCancel={() => (revokeTarget = null)}
		onConfirm={() => void submitRevoke()}
	/>
{/if}
