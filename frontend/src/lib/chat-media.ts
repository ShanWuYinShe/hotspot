/**
 * 聊天消息媒体字段映射（image / video / audio / file，spec MessageType）。
 * 背景：此前 ChatView/GroupChatView 只识别 type='image'，Android/鸿蒙发来的视频/语音/文件
 * 会渲染成空气泡（text 兜底为空串）。这里集中成纯函数，两视图（HTTP 历史 + WS 新消息）共用。
 */
export interface WireChatMedia {
	url?: string;
	file_name?: string | null;
	file_size?: number | null;
}

export interface ChatMediaFields {
	imageUrl?: string;
	videoUrl?: string;
	audioUrl?: string;
	fileUrl?: string;
	fileName?: string;
}

export function mapChatMediaFields(
	type: string | null | undefined,
	media: WireChatMedia | null | undefined,
	recalled = false
): ChatMediaFields {
	if (recalled || !media?.url) return {};
	const url = media.url;
	switch (type) {
		case 'image':
			return { imageUrl: url };
		case 'video':
			return { videoUrl: url };
		case 'audio':
			return { audioUrl: url };
		case 'file':
			return { fileUrl: url, fileName: media.file_name ?? '附件' };
		default:
			return {};
	}
}

/** 气泡是否为「媒体态」（决定是否去掉文本内边距）。 */
export function hasChatMedia(fields: ChatMediaFields): boolean {
	return Boolean(fields.imageUrl || fields.videoUrl || fields.audioUrl || fields.fileUrl);
}

export type ChatMediaKind = 'image' | 'video';

/** 输入栏选文件后的媒体类型判定（其它类型不进入聊天媒体发送）。 */
export function chatMediaKindOfFile(file: { type: string }): ChatMediaKind | null {
	if (file.type.startsWith('image/')) return 'image';
	if (file.type.startsWith('video/')) return 'video';
	return null;
}


/** 聊天上传计划：按 MIME 决定分类与契约上限（spec api_spec_media.md 分类表）。 */


export interface ChatUploadPlan {
	kind: ChatUploadKind;
	category: 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';
	maxBytes: number;
}

const MB = 1024 * 1024;

export function chatUploadPlanOfFile(file: { type: string }): ChatUploadPlan {
	if (file.type.startsWith('image/')) return { kind: 'image', category: 'chat_image', maxBytes: 10 * MB };
	if (file.type.startsWith('video/')) return { kind: 'video', category: 'chat_video', maxBytes: 100 * MB };
	if (file.type.startsWith('audio/')) return { kind: 'audio', category: 'chat_audio', maxBytes: 20 * MB };
	// 其余一律走 chat_file（服务端有黑名单约束，前端只做契约上限的 100MB 提示）
	return { kind: 'file', category: 'chat_file', maxBytes: 100 * MB };
}

/** 文件大小展示（B/KB/MB，1 位小数；与鸿蒙 formatFileSize 同口径）。 */
export function formatFileSize(bytes: number | null | undefined): string {
	if (!bytes || bytes <= 0) return '';
	if (bytes < 1024) return bytes + ' B';
	if (bytes < MB) return (bytes / 1024).toFixed(1) + ' KB';
	return (bytes / MB).toFixed(1) + ' MB';
}

/** 聊天上传类型（audio 单独一档：契约 chat_audio 白名单只有 mpeg/aac/ogg）。 */
export type ChatUploadKind = 'image' | 'video' | 'audio' | 'file';

/** 契约允许的 chat_audio MIME（spec api_spec_media.md 分类表）。 */
export const CHAT_AUDIO_MIMES = ['audio/mpeg', 'audio/aac', 'audio/ogg'] as const;

/** 扩展名 → 契约 MIME（浏览器给 m4a 常报 audio/x-m4a、audio/mp4，需要归一）。 */
const AUDIO_EXT_MIME: Record<string, string> = {
	mp3: 'audio/mpeg',
	m4a: 'audio/aac',
	aac: 'audio/aac',
	ogg: 'audio/ogg',
	oga: 'audio/ogg'
};

export function normalizeChatAudioMime(file: { name: string; type: string }): string | null {
	if ((CHAT_AUDIO_MIMES as readonly string[]).includes(file.type)) return file.type;
	const ext = file.name.split('.').pop()?.toLowerCase() ?? '';
	return AUDIO_EXT_MIME[ext] ?? null;
}

/** 按契约补正音频 MIME；其它类型原样返回（图片/视频/文件的 MIME 浏览器已给对）。 */
export function normalizeChatUploadFile<T extends File>(file: T): T {
	if (!file.type.startsWith('audio/') && !/\.(mp3|m4a|aac|ogg|oga)$/i.test(file.name)) return file;
	const mime = normalizeChatAudioMime(file);
	if (!mime || mime === file.type) return file;
	return new File([file], file.name, { type: mime }) as T;
}
