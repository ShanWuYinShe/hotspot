import { goto } from '$app/navigation';
import { dismissActiveKeyboardFocus } from '$lib/attachments';

const CHAT_FALLBACK_URL = '/chat';
const KEYBOARD_DISMISS_WAIT_MS = 250;

function hasUsableInAppBackHistory(currentWindow: Window): boolean {
	if (currentWindow.history.length <= 1) {
		return false;
	}

	const referrer = currentWindow.document.referrer;
	if (!referrer) {
		return false;
	}

	try {
		const referrerUrl = new URL(referrer);
		return referrerUrl.origin === currentWindow.location.origin;
	} catch {
		return false;
	}
}

async function waitForKeyboardDismiss(currentWindow: Window): Promise<void> {
	if (!dismissActiveKeyboardFocus(currentWindow)) {
		return;
	}

	await new Promise<void>((resolve) => {
		currentWindow.setTimeout(resolve, KEYBOARD_DISMISS_WAIT_MS);
	});
}

export async function navigateToChatBackTarget(currentWindow?: Window): Promise<void> {
	if (!currentWindow) {
		await goto(CHAT_FALLBACK_URL);
		return;
	}

	await waitForKeyboardDismiss(currentWindow);

	if (hasUsableInAppBackHistory(currentWindow)) {
		currentWindow.history.back();
		return;
	}

	await goto(CHAT_FALLBACK_URL, { replaceState: true });
}
