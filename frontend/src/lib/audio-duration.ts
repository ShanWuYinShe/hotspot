/**
 * WebM（MediaRecorder 录制）不含时长元数据：<audio> 的 duration 为 NaN/Infinity，
 * 原生控件显示 0:00 / 0:00。浏览器在 seek 到极大时间点后会扫描出真实时长，这里封装成
 * Svelte action：加载元数据时探测，拿到有限时长后立即回到 0。
 */

export const DURATION_PROBE_SEEK = 1e101;

export function needsDurationProbe(duration: number): boolean {
	return !Number.isFinite(duration) || duration <= 0;
}

/** 探测 WebM 时长：`preload=metadata` 触发 loadedmetadata → 跳大时间 → durationchange 后回 0。 */
export function probeAudioDuration(node: HTMLAudioElement) {
	const onLoadedMetadata = () => {
		if (!needsDurationProbe(node.duration)) return;
		try {
			node.currentTime = DURATION_PROBE_SEEK;
		} catch {
			// 部分浏览器对未加载完的媒体抛错：保持原状即可
		}
	};
	const onDurationChange = () => {
		if (needsDurationProbe(node.duration)) return;
		try {
			node.currentTime = 0;
		} catch {
			// 同上
		}
	};
	node.addEventListener('loadedmetadata', onLoadedMetadata);
	node.addEventListener('durationchange', onDurationChange);
	return {
		destroy() {
			node.removeEventListener('loadedmetadata', onLoadedMetadata);
			node.removeEventListener('durationchange', onDurationChange);
		}
	};
}
