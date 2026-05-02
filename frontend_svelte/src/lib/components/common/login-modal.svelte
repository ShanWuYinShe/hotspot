<script lang="ts">
	import { t } from '$lib/i18n';
	import { uiStore } from '$lib/stores/ui.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import AuthFormContent from '$lib/components/common/auth-form-content.svelte';

	function handleClose() {
		uiStore.closeLoginModal();
	}

	async function handleSuccess() {
		uiStore.onLoginSuccess();
	}
</script>

<Dialog.Root
	open={uiStore.showLoginModal}
	onOpenChange={(open) => {
		if (!open) {
			handleClose();
		}
	}}
>
	<Dialog.Content
		class="rounded-[24px] border px-5 py-6"
		style="background: var(--neu-bg); border-color: var(--neu-border); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); max-width: 28rem;"
	>
		<div class="mb-5 flex items-center gap-3">
			<div
				class="flex items-center justify-center rounded-full"
				style="width: 40px; height: 40px; box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
			>
				<MapPin size={18} strokeWidth={1.5} color="var(--neu-accent-primary)" />
			</div>
			<div>
				<h2 style="font-size: 18px; font-weight: 600; color: var(--neu-text);">{t('auth.welcome')}</h2>
				<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">{t('auth.loginSubtitle')}</p>
			</div>
		</div>

		<AuthFormContent onSuccess={handleSuccess} />
	</Dialog.Content>
</Dialog.Root>
