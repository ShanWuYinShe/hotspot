import { goto } from '$app/navigation';

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

export function navigateToNearbyBackTarget(currentWindow: Window | undefined, fallbackUrl: string) {
	if (currentWindow && hasUsableInAppBackHistory(currentWindow)) {
		currentWindow.history.back();
		return;
	}

	void goto(fallbackUrl);
}
