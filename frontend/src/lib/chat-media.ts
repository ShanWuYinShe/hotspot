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
