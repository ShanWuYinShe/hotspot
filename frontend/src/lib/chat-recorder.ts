/**
 * Web 端语音录制（MediaRecorder）：契约 chat_audio 白名单含 audio/webm（#432）。
 * 纯函数（MIME 选择 / 计时格式）单独导出便于单测；录制成败依赖浏览器能力，不做全局兜底。
 */

/** MediaRecorder 可录制的优先 MIME（Chrome/Firefox 产 webm；Safari 仅 audio/mp4，契约未收录则不展示入口）。 */
export const RECORD_MIME_CANDIDATES = ['audio/webm;codecs=opus', 'audio/webm'] as const;

export function pickRecordMimeType(isSupported: (mime: string) => boolean): string | null {
	for (const mime of RECORD_MIME_CANDIDATES) {
		if (isSupported(mime)) return mime;
	}
	return null;
}

/** 录音入口是否可用：需要 MediaRecorder + getUserMedia + 可录制 MIME。 */
export function canRecordAudio(): boolean {
	if (typeof window === 'undefined') return false;
	if (typeof MediaRecorder === 'undefined') return false;
	if (!navigator.mediaDevices?.getUserMedia) return false;
	return pickRecordMimeType((mime) => MediaRecorder.isTypeSupported(mime)) !== null;
}

/** 录音计时展示：mm:ss（00:00 起）。 */
export function formatRecordDuration(ms: number): string {
	const total = Math.max(0, Math.floor(ms / 1000));
	const m = Math.floor(total / 60);
	const s = total % 60;
	return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
}

export interface AudioRecording {
	stop: () => Promise<File>;
	cancel: () => void;
}

/** 开始录音：申请麦克风 → MediaRecorder 收集分片；stop 返回可直接上传的 File。 */
export async function startAudioRecording(): Promise<AudioRecording> {
	const mime = pickRecordMimeType((m) => MediaRecorder.isTypeSupported(m));
	if (!mime) throw new Error('当前浏览器不支持语音录制');
	const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
	const recorder = new MediaRecorder(stream, { mimeType: mime });
	const chunks: BlobPart[] = [];
	recorder.addEventListener('dataavailable', (event) => {
		if (event.data.size > 0) chunks.push(event.data);
	});
	const stopTracks = () => stream.getTracks().forEach((track) => track.stop());
	recorder.start();

	return {
		stop: () =>
			new Promise<File>((resolve, reject) => {
				recorder.addEventListener('stop', () => {
					stopTracks();
					const type = mime.split(';')[0];
					const blob = new Blob(chunks, { type });
					if (blob.size === 0) {
						reject(new Error('没有录到声音，请重试'));
						return;
					}
					const ext = type === 'audio/webm' ? 'webm' : 'bin';
					resolve(new File([blob], `voice-${Date.now()}.${ext}`, { type }));
				});
				recorder.addEventListener('error', () => {
					stopTracks();
					reject(new Error('录音失败，请重试'));
				});
				recorder.stop();
			}),
		cancel: () => {
			try {
				if (recorder.state !== 'inactive') recorder.stop();
			} catch {
				// 取消路径忽略 recorder 状态异常
			}
			stopTracks();
		}
	};
}
