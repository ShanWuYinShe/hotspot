<script lang="ts">
	import { authService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import { validateField } from '$lib/utils/form-validation';
	import type { FieldRule } from '$lib/utils/form-validation';

	let oldPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let isSaving = $state(false);
	let touched = $state<Record<string, boolean>>({});

	const hasLetterAndDigit = (value: string) => /[A-Za-z]/.test(value) && /\d/.test(value);

	const oldPasswordRules: FieldRule[] = [{ required: true, requiredMessage: '请输入当前密码' }];
	const newPasswordRules: FieldRule[] = [
		{ required: true, requiredMessage: '请输入新密码' },
		{ minLength: 8, minLengthMessage: '新密码至少 8 个字符' },
		{ maxLength: 128, maxLengthMessage: '新密码最多 128 个字符' },
		{ custom: (v) => (v && !hasLetterAndDigit(v) ? '新密码需同时包含字母和数字' : undefined) }
	];
	const confirmPasswordRules: FieldRule[] = [
		{ required: true, requiredMessage: '请确认新密码' },
		{ custom: (v) => (v && v !== newPassword ? '两次密码不一致' : undefined) }
	];

	let oldPasswordError = $derived(
		touched['oldPassword'] ? validateField(oldPassword, oldPasswordRules) : undefined
	);
	let newPasswordError = $derived(
		touched['newPassword'] ? validateField(newPassword, newPasswordRules) : undefined
	);
	let confirmPasswordError = $derived(
		touched['confirmPassword'] ? validateField(confirmPassword, confirmPasswordRules) : undefined
	);

	function markTouched(field: string) {
		touched = { ...touched, [field]: true };
	}

	function markAllTouched() {
		touched = { oldPassword: true, newPassword: true, confirmPassword: true };
	}

	function hasErrors(): boolean {
		const nextOldPasswordError = validateField(oldPassword, oldPasswordRules);
		const nextNewPasswordError = validateField(newPassword, newPasswordRules);
		const nextConfirmPasswordError = validateField(confirmPassword, confirmPasswordRules);
		return !!nextOldPasswordError || !!nextNewPasswordError || !!nextConfirmPasswordError;
	}

	async function handleSubmit() {
		markAllTouched();
		if (hasErrors()) return;

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
				touched = {};
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
	<PageHeader
		title="修改密码"
		description="请确保新密码足够安全，避免与其他平台重复"
	/>

	<Card class="rounded-3xl border border-border/60 bg-card/80 shadow-sm backdrop-blur-md">
		<CardHeader>
			<CardTitle class="text-base">账户安全</CardTitle>
		</CardHeader>
		<CardContent class="space-y-4 pt-0">
			<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-4">
			<div class="space-y-2">
				<Label for="old-password">当前密码</Label>
				<Input
					id="old-password"
					type="password"
					bind:value={oldPassword}
					onblur={() => markTouched('oldPassword')}
					class="h-11 rounded-2xl border-border/60 bg-background/70 {oldPasswordError ? 'border-destructive focus-visible:ring-destructive' : ''}"
				/>
				{#if oldPasswordError}
					<p class="text-xs text-destructive">{oldPasswordError}</p>
				{/if}
			</div>
			<div class="space-y-2">
				<Label for="new-password">新密码</Label>
				<Input
					id="new-password"
					type="password"
					bind:value={newPassword}
					onblur={() => markTouched('newPassword')}
					class="h-11 rounded-2xl border-border/60 bg-background/70 {newPasswordError ? 'border-destructive focus-visible:ring-destructive' : ''}"
				/>
				{#if newPasswordError}
					<p class="text-xs text-destructive">{newPasswordError}</p>
				{/if}
			</div>
			<div class="space-y-2">
				<Label for="confirm-password">确认新密码</Label>
				<Input
					id="confirm-password"
					type="password"
					bind:value={confirmPassword}
					onblur={() => markTouched('confirmPassword')}
					class="h-11 rounded-2xl border-border/60 bg-background/70 {confirmPasswordError ? 'border-destructive focus-visible:ring-destructive' : ''}"
				/>
				{#if confirmPasswordError}
					<p class="text-xs text-destructive">{confirmPasswordError}</p>
				{/if}
			</div>
			<Button type="submit" class="h-11 w-full rounded-2xl" disabled={isSaving}>
				{#if isSaving}
					<Loader2 class="mr-2 h-4 w-4 animate-spin" />
				{/if}
				保存修改
			</Button>
			</form>
		</CardContent>
	</Card>
</div>
