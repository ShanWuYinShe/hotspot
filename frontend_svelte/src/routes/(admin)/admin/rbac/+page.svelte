<script lang="ts">
	import { onMount } from 'svelte';
	import { adminRbacService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Label } from '$lib/components/ui/label';
	import type { AdminRole, AdminPermission } from '$lib/types/api';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Plus from 'lucide-svelte/icons/plus';
	import Shield from 'lucide-svelte/icons/shield';
	import Key from 'lucide-svelte/icons/key';
	import Edit from 'lucide-svelte/icons/edit';
	import Trash2 from 'lucide-svelte/icons/trash-2';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Users from 'lucide-svelte/icons/users';

	let roles = $state<AdminRole[]>([]);
	let permissions = $state<AdminPermission[]>([]);
	let isLoading = $state(true);

	// Create/Edit role dialog
	let showRoleDialog = $state(false);
	let isSubmitting = $state(false);
	let editingRoleId = $state<string | null>(null);
	let roleName = $state('');
	let roleDescription = $state('');
	let rolePermissions = $state<string[]>([]);

	async function loadData() {
		isLoading = true;
		try {
			const [rolesRes, permissionsRes] = await Promise.all([
				adminRbacService.getRoles(),
				adminRbacService.getPermissions()
			]);

			if (isSuccess(rolesRes) && rolesRes.data) {
				roles = rolesRes.data.items;
			}
			if (isSuccess(permissionsRes) && permissionsRes.data) {
				permissions = permissionsRes.data;
			}
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
		}
	}

	function openCreateRoleDialog() {
		editingRoleId = null;
		roleName = '';
		roleDescription = '';
		rolePermissions = [];
		showRoleDialog = true;
	}

	function openEditRoleDialog(role: AdminRole) {
		editingRoleId = role.role_id;
		roleName = role.name;
		roleDescription = role.description || '';
		rolePermissions = role.permissions || [];
		showRoleDialog = true;
	}

	async function handleSubmitRole() {
		if (!roleName.trim()) {
			toast.error('请填写角色名称');
			return;
		}

		isSubmitting = true;
		try {
			if (editingRoleId) {
				const response = await adminRbacService.updateRole({
					role_id: editingRoleId,
					name: roleName.trim(),
					description: roleDescription.trim() || undefined
				});

				if (isSuccess(response)) {
					// 更新权限
					await adminRbacService.setRolePermissions({
						role_id: editingRoleId,
						permissions: rolePermissions
					});
					loadData();
					toast.success('更新成功');
					showRoleDialog = false;
				} else {
					toast.error(response.message || '更新失败');
				}
			} else {
				const response = await adminRbacService.createRole({
					name: roleName.trim(),
					description: roleDescription.trim() || undefined,
					permissions: rolePermissions
				});

				if (isSuccess(response)) {
					loadData();
					toast.success('创建成功');
					showRoleDialog = false;
				} else {
					toast.error(response.message || '创建失败');
				}
			}
		} catch {
			toast.error('操作失败');
		} finally {
			isSubmitting = false;
		}
	}

	async function handleDeleteRole(roleId: string) {
		if (!confirm('确定要删除该角色吗？')) return;

		try {
			const response = await adminRbacService.deleteRole({ role_id: roleId });
			if (isSuccess(response)) {
				roles = roles.filter(r => r.role_id !== roleId);
				toast.success('已删除');
			} else {
				toast.error(response.message || '删除失败');
			}
		} catch {
			toast.error('删除失败');
		}
	}

	function togglePermission(permissionId: string) {
		if (rolePermissions.includes(permissionId)) {
			rolePermissions = rolePermissions.filter(p => p !== permissionId);
		} else {
			rolePermissions = [...rolePermissions, permissionId];
		}
	}

	// Group permissions by category
	function groupPermissions(perms: AdminPermission[]): Map<string, AdminPermission[]> {
		const groups = new Map<string, AdminPermission[]>();
		for (const perm of perms) {
			const category = perm.category || '其他';
			if (!groups.has(category)) {
				groups.set(category, []);
			}
			groups.get(category)!.push(perm);
		}
		return groups;
	}

	onMount(() => {
		loadData();
	});
</script>

<svelte:head>
	<title>权限管理 - HotSpot 管理后台</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">权限管理</h1>
		<div class="flex gap-2">
			<Button variant="outline" size="sm" onclick={loadData} disabled={isLoading}>
				<RefreshCw class="mr-2 h-4 w-4 {isLoading ? 'animate-spin' : ''}" />
				刷新
			</Button>
			<Button size="sm" onclick={openCreateRoleDialog}>
				<Plus class="mr-2 h-4 w-4" />
				创建角色
			</Button>
		</div>
	</div>

	<div class="grid gap-6 lg:grid-cols-2">
		<!-- Roles -->
		<Card>
			<CardHeader>
				<CardTitle class="flex items-center gap-2">
					<Shield class="h-5 w-5" />
					角色列表
				</CardTitle>
				<CardDescription>管理系统中的角色及其权限</CardDescription>
			</CardHeader>
			<CardContent>
				{#if isLoading}
					<div class="space-y-3">
						{#each Array(4) as _}
							<div class="flex items-center gap-4 rounded-lg border p-3">
								<Skeleton class="h-10 w-10 rounded" />
								<div class="flex-1 space-y-2">
									<Skeleton class="h-4 w-24" />
									<Skeleton class="h-3 w-40" />
								</div>
							</div>
						{/each}
					</div>
				{:else if roles.length === 0}
					<div class="py-8 text-center text-muted-foreground">暂无角色</div>
				{:else}
					<div class="space-y-3">
						{#each roles as role (role.role_id)}
							<div class="flex items-center gap-4 rounded-lg border p-3">
								<div class="flex h-10 w-10 items-center justify-center rounded bg-primary/10">
									<Shield class="h-5 w-5 text-primary" />
								</div>
								<div class="min-w-0 flex-1">
									<p class="font-medium">{role.name}</p>
									{#if role.description}
										<p class="text-sm text-muted-foreground">{role.description}</p>
									{/if}
									<div class="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
										<Key class="h-3 w-3" />
										{role.permissions?.length || 0} 权限
									</div>
								</div>
								<div class="flex gap-1">
									<Button variant="ghost" size="icon" onclick={() => openEditRoleDialog(role)}>
										<Edit class="h-4 w-4" />
									</Button>
									<Button
										variant="ghost"
										size="icon"
										class="text-destructive"
										onclick={() => handleDeleteRole(role.role_id)}
									>
										<Trash2 class="h-4 w-4" />
									</Button>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</CardContent>
		</Card>

		<!-- Permissions -->
		<Card>
			<CardHeader>
				<CardTitle class="flex items-center gap-2">
					<Key class="h-5 w-5" />
					权限列表
				</CardTitle>
				<CardDescription>系统中可分配的权限</CardDescription>
			</CardHeader>
			<CardContent>
				{#if isLoading}
					<div class="space-y-3">
						{#each Array(6) as _}
							<div class="flex items-center gap-4">
								<Skeleton class="h-4 w-4" />
								<Skeleton class="h-4 w-32" />
							</div>
						{/each}
					</div>
				{:else if permissions.length === 0}
					<div class="py-8 text-center text-muted-foreground">暂无权限数据</div>
				{:else}
					<div class="space-y-4">
						{#each [...groupPermissions(permissions)] as [category, perms]}
							<div>
								<h4 class="mb-2 text-sm font-medium text-muted-foreground">{category}</h4>
								<div class="space-y-1">
									{#each perms as perm (perm.permission_id)}
										<div class="flex items-center gap-3 rounded px-2 py-1 hover:bg-muted/50">
											<Key class="h-4 w-4 text-muted-foreground" />
											<div class="flex-1">
												<p class="text-sm font-medium">{perm.name}</p>
												{#if perm.description}
													<p class="text-xs text-muted-foreground">{perm.description}</p>
												{/if}
											</div>
										</div>
									{/each}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</CardContent>
		</Card>
	</div>
</div>

<!-- Create/Edit role dialog -->
<Dialog.Root bind:open={showRoleDialog}>
	<Dialog.Content class="sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>{editingRoleId ? '编辑角色' : '创建角色'}</Dialog.Title>
		</Dialog.Header>
		<div class="space-y-4 py-4">
			<div class="space-y-2">
				<Label for="role-name">角色名称</Label>
				<Input id="role-name" placeholder="例如：内容审核员" bind:value={roleName} autocomplete="off" />
			</div>
			<div class="space-y-2">
				<Label for="role-desc">角色描述</Label>
				<Textarea id="role-desc" placeholder="描述该角色的职责" bind:value={roleDescription} />
			</div>
			<div class="space-y-2">
				<Label>分配权限</Label>
				<div class="max-h-60 space-y-4 overflow-y-auto rounded-lg border p-3">
					{#each [...groupPermissions(permissions)] as [category, perms]}
						<div>
							<h5 class="mb-2 text-xs font-medium text-muted-foreground">{category}</h5>
							<div class="space-y-1">
								{#each perms as perm (perm.permission_id)}
									<label class="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-muted/50">
										<input
											type="checkbox"
											checked={rolePermissions.includes(perm.permission_id)}
											onchange={() => togglePermission(perm.permission_id)}
											class="h-4 w-4 rounded border-gray-300"
										/>
										<span class="text-sm">{perm.name}</span>
									</label>
								{/each}
							</div>
						</div>
					{/each}
				</div>
			</div>
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (showRoleDialog = false)}>取消</Button>
			<Button onclick={handleSubmitRole} disabled={isSubmitting}>
				{#if isSubmitting}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				{editingRoleId ? '保存' : '创建'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
