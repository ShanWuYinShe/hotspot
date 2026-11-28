/**
 * 消息引用回复（spec private_messages.send → reply_to_message_id）。
 * 服务端在消息上回填 reply_to: MessageBrief；这里负责把 brief 归一成气泡展示用的预览。
 */

export interface ReplyPreview {
	id: string;
	senderName: string;
	preview: string;
}

export interface WireReplyBrief {
	message_id?: string;
	content?: string | null;
	type?: string;
	sender?: { nickname?: string } | null;
}

/** 引用预览：媒体消息用类型占位，文本按 40 字截断（含省略号）。 */
export function buildReplyPreview(brief: WireReplyBrief | null | undefined): ReplyPreview | null {
	if (!brief?.message_id) return null;
	const type = brief.type ?? 'text';
	const text = (brief.content ?? '').trim();
	let preview: string;
	if (text) {
		preview = text.length > 40 ? text.slice(0, 40) + '…' : text;
	} else {
		preview = mediaPlaceholder(type);
	}
	return {
		id: brief.message_id,
		senderName: brief.sender?.nickname ?? '',
		preview
	};
}

export function mediaPlaceholder(type: string): string {
	switch (type) {
		case 'image':
			return '[图片]';
		case 'video':
			return '[视频]';
		case 'audio':
			return '[语音]';
		case 'file':
			return '[文件]';
		case 'share':
			return '[分享]';
		default:
			return '[消息]';
	}
}
