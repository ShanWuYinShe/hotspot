<script lang="ts">
	import { onMount } from 'svelte';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { adminRbacService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import ConfirmDialog from '$lib/components/common/confirm-dialog.svelte';
	import type { AdminRoleItem, AdminPermissionItem } from '$lib/types/api';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Plus from 'lucide-svelte/icons/plus';
	import Shield from 'lucide-svelte/icons/shield';
	import Key from 'lucide-svelte/icons/key';
	import Edit from 'lucide-svelte/icons/edit';
	import Trash2 from 'lucide-svelte/icons/trash-2';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	let roles = $state.raw<AdminRoleItem[]>([]);
	let permissions = $state.raw<AdminPermissionItem[]>([]);
	let isLoading = $state(true);
	let deleteRoleConfirmOpen = $state(false);
	let pendingDeleteRoleId = $state<string | null>(null);
	let showRoleDialog = $state(false);
	let isSubmitting = $state(false);
	let editingRoleId = $state<string | null>(null);
	let roleName = $state('');
	let roleDescription = $state('');
	let rolePermissions = $state.raw<string[]>([]);

	function closeOverlayDialogs() { showRoleDialog = false; deleteRoleConfirmOpen = false; pendingDeleteRoleId = null; }

	async function loadData() {
		isLoading = true;
		try {
			const [rolesRes, permissionsRes] = await withMinimumLoadingTime(() =>
				Promise.all([adminRbacService.getRoles(), adminRbacService.getPermissions()])
			);
			if (isSuccess(rolesRes) && rolesRes.data) roles = rolesRes.data.items;
			if (isSuccess(permissionsRes) && permissionsRes.data) permissions = permissionsRes.data.items;
		} catch { toast.error('加载失败'); }
		finally { isLoading = false; }
	}

	function openCreateRoleDialog() { closeOverlayDialogs(); editingRoleId = null; roleName = ''; roleDescription = ''; rolePermissions = []; showRoleDialog = true; }
	function openEditRoleDialog(role: AdminRoleItem) { closeOverlayDialogs(); editingRoleId = role.role_id; roleName = role.name; roleDescription = role.description || ''; rolePermissions = role.permissions || []; showRoleDialog = true; }

	async function handleSubmitRole() {
		if (!roleName.trim()) { toast.error('请填写角色名称'); return; }
		isSubmitting = true;
		try {
			if (editingRoleId) {
				const response = await adminRbacService.updateRole({ role_id: editingRoleId, name: roleName.trim(), description: roleDescription.trim() || undefined });
				if (isSuccess(response)) { await adminRbacService.setRolePermissions({ role_id: editingRoleId, permission_keys: rolePermissions }); loadData(); toast.success('更新成功'); showRoleDialog = false; } else { toast.error(response.message || '更新失败'); }
			} else {
				const response = await adminRbacService.createRole({ name: roleName.trim(), description: roleDescription.trim() || undefined });
				if (isSuccess(response) && response.data) { await adminRbacService.setRolePermissions({ role_id: response.data.role_id, permission_keys: rolePermissions }); loadData(); toast.success('创建成功'); showRoleDialog = false; } else { toast.error(response.message || '创建失败'); }
			}
		} catch { toast.error('操作失败'); }
		finally { isSubmitting = false; }
	}

	function requestDeleteRole(roleId: string) { closeOverlayDialogs(); pendingDeleteRoleId = roleId; deleteRoleConfirmOpen = true; }
	async function handleDeleteRole() {
		if (!pendingDeleteRoleId) return;
		try {
			const response = await adminRbacService.deleteRole({ role_id: pendingDeleteRoleId });
			if (isSuccess(response)) { roles = roles.filter(r => r.role_id !== pendingDeleteRoleId); toast.success('已删除'); } else { toast.error(response.message || '删除失败'); }
		} catch { toast.error('删除失败'); }
		finally { pendingDeleteRoleId = null; }
	}

	function togglePermission(permissionId: string) {
		if (rolePermissions.includes(permissionId)) { rolePermissions = rolePermissions.filter(p => p !== permissionId); }
		else { rolePermissions = [...rolePermissions, permissionId]; }
	}

	function groupPermissions(perms: AdminPermissionItem[]): Map<string, AdminPermissionItem[]> {
		const groups = new Map<string, AdminPermissionItem[]>();
		for (const perm of perms) {
			const keyParts = perm.key.split('.');
			const category = keyParts.length > 1 ? keyParts.slice(0, -1).join('.') : '其他';
			if (!groups.has(category)) groups.set(category, []);
			groups.get(category)!.push(perm);
		}
		return groups;
	}

	onMount(() => { loadData(); });
</script>

<svelte:head>
	<title>权限管理 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 style="font-size: 22px; font-weight: 600; color: var(--neu-text);">权限管理</h1>
		<div class="flex gap-2">
			<button class="flex items-center gap-1.5 rounded-full px-4 py-2 outline-none" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); font-size: 13px; color: var(--neu-text);" onclick={loadData} disabled={isLoading}>
				<RefreshCw size={14} strokeWidth={1.5} class={isLoading ? 'animate-spin' : ''} />
				刷新
			</button>
			<button class="flex items-center gap-1.5 rounded-full px-4 py-2 outline-none" style="background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 13px; box-shadow: 0 3px 10px var(--neu-glow);" onclick={openCreateRoleDialog}>
				<Plus size={14} strokeWidth={1.5} />
				创建角色
			</button>
		</div>
	</div>

	<div class="grid gap-6 lg:grid-cols-2">
		<!-- Roles -->
		<div class="rounded-[24px] overflow-hidden" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
			<div class="flex items-center gap-2 px-4 py-3" style="border-bottom: 1px solid var(--neu-border);">
				<Shield size={16} strokeWidth={1.5} color="var(--neu-accent-primary)" />
				<span style="font-size: 14px; font-weight: 500; color: var(--neu-text);">角色列表</span>
			</div>
			<div class="p-4">
				{#if isLoading}
					<div class="space-y-3">
						{#each Array(4) as _, i (i)}
							<div class="rounded-[18px] p-3" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
								<div class="flex items-center gap-3">
									<div class="rounded" style="width: 36px; height: 36px; background: var(--neu-border);"></div>
									<div class="flex-1 space-y-1.5">
										<div class="rounded" style="height: 12px; width: 80px; background: var(--neu-border);"></div>
										<div class="rounded" style="height: 10px; width: 140px; background: var(--neu-border);"></div>
									</div>
								</div>
							</div>
						{/each}
					</div>
				{:else if roles.length === 0}
					<p class="py-8 text-center" style="font-size: 13px; color: var(--neu-text-muted);">暂无角色</p>
				{:else}
					<div class="space-y-2">
						{#each roles as role (role.role_id)}
							<div class="flex items-center gap-3 rounded-[18px] p-3" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
								<div class="flex flex-shrink-0 items-center justify-center rounded-full" style="width: 36px; height: 36px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
									<Shield size={15} strokeWidth={1.5} color="var(--neu-accent-primary)" />
								</div>
								<div class="min-w-0 flex-1">
									<p style="font-size: 13px; font-weight: 500; color: var(--neu-text);">{role.name}</p>
									{#if role.description}
										<p style="font-size: 11px; color: var(--neu-text-muted); margin-top: 1px;">{role.description}</p>
									{/if}
									<div class="flex items-center gap-1 mt-1">
										<Key size={11} strokeWidth={1.5} color="var(--neu-text-muted)" />
										<span style="font-size: 11px; color: var(--neu-text-muted);">{role.permissions?.length || 0} 权限</span>
									</div>
								</div>
								<div class="flex gap-1">
									<button class="flex items-center justify-center rounded-full outline-none" style="width: 30px; height: 30px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);" onclick={() => openEditRoleDialog(role)} aria-label="编辑角色">
										<Edit size={13} strokeWidth={1.5} color="var(--neu-text-muted)" />
									</button>
									<button class="flex items-center justify-center rounded-full outline-none" style="width: 30px; height: 30px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);" onclick={() => requestDeleteRole(role.role_id)} aria-label="删除角色">
										<Trash2 size={13} strokeWidth={1.5} color="var(--neu-error)" />
									</button>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>

		<!-- Permissions -->
		<div class="rounded-[24px] overflow-hidden" style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
			<div class="flex items-center gap-2 px-4 py-3" style="border-bottom: 1px solid var(--neu-border);">
				<Key size={16} strokeWidth={1.5} color="var(--neu-accent-primary)" />
				<span style="font-size: 14px; font-weight: 500; color: var(--neu-text);">权限列表</span>
			</div>
			<div class="p-4">
				{#if isLoading}
					<div class="space-y-2">
						{#each Array(6) as _, i (i)}
							<div class="flex items-center gap-3">
								<div class="rounded" style="width: 14px; height: 14px; background: var(--neu-border);"></div>
								<div class="rounded" style="height: 12px; width: 120px; background: var(--neu-border);"></div>
							</div>
						{/each}
					</div>
				{:else if permissions.length === 0}
					<p class="py-8 text-center" style="font-size: 13px; color: var(--neu-text-muted);">暂无权限数据</p>
				{:else}
					<div class="space-y-4">
						{#each [...groupPermissions(permissions)] as [category, perms] (category)}
							<div>
								<p class="uppercase" style="font-size: 10px; letter-spacing: 0.14em; color: var(--neu-text-muted); margin-bottom: 6px;">{category}</p>
								<div class="space-y-1">
									{#each perms as perm (perm.key)}
										<div class="flex items-center gap-2 rounded-[14px] px-2 py-1.5" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
											<Key size={13} strokeWidth={1.5} color="var(--neu-text-muted)" />
											<div class="flex-1">
												<p style="font-size: 12px; font-weight: 500; color: var(--neu-text);">{perm.key}</p>
												{#if perm.description}
													<p style="font-size: 11px; color: var(--neu-text-muted);">{perm.description}</p>
												{/if}
											</div>
										</div>
									{/each}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	</div>
</div>

<!-- Create/Edit role dialog -->
<Dialog.Root bind:open={showRoleDialog}>
	<Dialog.Content class="sm:max-w-lg rounded-[24px]" style="background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light);">
		<Dialog.Header>
			<Dialog.Title>{editingRoleId ? '编辑角色' : '创建角色'}</Dialog.Title>
		</Dialog.Header>
		<div class="space-y-4 py-4 px-1">
			<div>
				<label for="role-name" style="font-size: 12px; color: var(--neu-text);">角色名称</label>
				<input id="role-name" placeholder="例如：内容审核员" bind:value={roleName} autocomplete="off" class="mt-1.5 w-full rounded-[14px] px-4 py-2.5 outline-none" style="background: var(--neu-bg); color: var(--neu-text); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); font-size: 13px;" />
			</div>
			<div>
				<label for="role-desc" style="font-size: 12px; color: var(--neu-text);">角色描述</label>
				<textarea id="role-desc" placeholder="描述该角色的职责" bind:value={roleDescription} class="mt-1.5 w-full rounded-[14px] px-4 py-2.5 outline-none resize-none" rows="3" style="background: var(--neu-bg); color: var(--neu-text); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); font-size: 13px;"></textarea>
			</div>
			<div>
				<label style="font-size: 12px; color: var(--neu-text);">分配权限</label>
				<div class="mt-1.5 max-h-60 space-y-3 overflow-y-auto rounded-[14px] p-3" style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
					{#each [...groupPermissions(permissions)] as [category, perms] (category)}
						<div>
							<p class="uppercase" style="font-size: 10px; letter-spacing: 0.12em; color: var(--neu-text-muted); margin-bottom: 4px;">{category}</p>
							<div class="space-y-1">
								{#each perms as perm (perm.key)}
									<label class="flex cursor-pointer items-center gap-2 rounded-[12px] px-2 py-1">
										<input type="checkbox" checked={rolePermissions.includes(perm.key)} onchange={() => togglePermission(perm.key)} class="h-4 w-4 rounded" style="accent-color: var(--neu-accent-primary);" />
										<span style="font-size: 12px; color: var(--neu-text);">{perm.description || perm.key}</span>
									</label>
								{/each}
							</div>
						</div>
					{/each}
				</div>
			</div>
		</div>
		<Dialog.Footer>
			<button class="rounded-full px-5 py-2 outline-none" style="background: var(--neu-bg); color: var(--neu-text); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); font-size: 13px;" onclick={closeOverlayDialogs}>取消</button>
			<button class="rounded-full px-5 py-2 outline-none" style="background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 13px; box-shadow: 0 3px 10px var(--neu-glow);" onclick={handleSubmitRole} disabled={isSubmitting}>
				{#if isSubmitting}<Loader2 size={14} strokeWidth={1.5} class="animate-spin" style="display: inline; vertical-align: middle;" />{/if}
				{editingRoleId ? '保存' : '创建'}
			</button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<ConfirmDialog bind:open={deleteRoleConfirmOpen} title="删除角色" description="确定要删除该角色吗？已分配此角色的管理员将失去对应权限。" confirmText="删除" variant="destructive" onconfirm={handleDeleteRole} />
