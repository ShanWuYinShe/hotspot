<script lang="ts">
	import { authService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	let oldPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let isSaving = $state(false);

	async function handleSubmit() {
		if (!oldPassword || !newPassword) {
			toast.error('请填写完整信息');
			return;
		}
		if (newPassword.length < 6) {
			toast.error('新密码至少 6 位');
			return;
		}
		if (newPassword !== confirmPassword) {
			toast.error('两次密码不一致');
			return;
		}

		isSaving = true;
		try {
			const response = await authService.changePassword({
				old_password: oldPassword,
				new_password: newPassword
			});
			if (isSuccess(response)) {
				toast.success('密码已更新');
				oldPassword = '';
				newPassword = '';
				confirmPassword = '';
			} else {
				toast.error(response.message || '修改失败');
			}
		} catch {
			toast.error('修改失败');
		} finally {
			isSaving = false;
		}
	}
</script>

<svelte:head>
	<title>修改密码 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-2xl space-y-6">
	<div>
		<h1 class="text-2xl font-bold">修改密码</h1>
		<p class="text-sm text-muted-foreground">请确保新密码足够安全</p>
	</div>

	<Card>
		<CardHeader>
			<CardTitle class="text-base">账户安全</CardTitle>
		</CardHeader>
		<CardContent class="space-y-4">
			<div class="space-y-2">
				<Label for="old-password">当前密码</Label>
				<Input id="old-password" type="password" bind:value={oldPassword} />
			</div>
			<div class="space-y-2">
				<Label for="new-password">新密码</Label>
				<Input id="new-password" type="password" bind:value={newPassword} />
			</div>
			<div class="space-y-2">
				<Label for="confirm-password">确认新密码</Label>
				<Input id="confirm-password" type="password" bind:value={confirmPassword} />
			</div>
			<Button class="w-full" onclick={handleSubmit} disabled={isSaving}>
				{#if isSaving}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				保存修改
			</Button>
		</CardContent>
	</Card>
</div>
