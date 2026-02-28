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
	const hasLetterAndDigit = (value: string) => /[A-Za-z]/.test(value) && /\d/.test(value);

	async function handleSubmit() {
		if (!oldPassword || !newPassword) {
			toast.error('请填写完整信息');
			return;
		}
		if (newPassword.length < 8 || newPassword.length > 128 || !hasLetterAndDigit(newPassword)) {
			toast.error('新密码需为 8-128 位且同时包含字母和数字');
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

<div class="mx-auto max-w-3xl space-y-5">
	<div class="rounded-3xl border border-border/60 bg-linear-to-br from-background via-card/85 to-muted/40 p-5 shadow-sm">
		<h1 class="text-2xl font-bold">修改密码</h1>
		<p class="mt-1 text-sm text-muted-foreground">请确保新密码足够安全，避免与其他平台重复</p>
	</div>

	<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
		<CardHeader>
			<CardTitle class="text-base">账户安全</CardTitle>
		</CardHeader>
		<CardContent class="space-y-4 pt-0">
			<div class="space-y-2">
				<Label for="old-password">当前密码</Label>
				<Input id="old-password" type="password" bind:value={oldPassword} class="h-11 rounded-2xl border-border/60 bg-background/70" />
			</div>
			<div class="space-y-2">
				<Label for="new-password">新密码</Label>
				<Input id="new-password" type="password" bind:value={newPassword} class="h-11 rounded-2xl border-border/60 bg-background/70" />
			</div>
			<div class="space-y-2">
				<Label for="confirm-password">确认新密码</Label>
				<Input id="confirm-password" type="password" bind:value={confirmPassword} class="h-11 rounded-2xl border-border/60 bg-background/70" />
			</div>
			<Button class="h-11 w-full rounded-2xl" onclick={handleSubmit} disabled={isSaving}>
				{#if isSaving}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				保存修改
			</Button>
		</CardContent>
	</Card>
</div>
