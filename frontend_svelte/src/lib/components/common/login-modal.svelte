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
	<Dialog.Content class="sm:max-w-md rounded-3xl border border-border/60 bg-card/95 shadow-xl backdrop-blur-md">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-2xl border border-border/50 bg-primary/10 text-primary">
					<MapPin class="h-5 w-5" />
				</div>
				<div>{t('auth.welcome')}</div>
			</Dialog.Title>
			<Dialog.Description>
				{t('auth.loginSubtitle')}
			</Dialog.Description>
		</Dialog.Header>

		<AuthFormContent onSuccess={handleSuccess} />
	</Dialog.Content>
</Dialog.Root>
