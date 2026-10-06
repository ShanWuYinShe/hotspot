// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request, newIdempotencyKey } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	GroupMessageItem,
	MessageSentResult,
	PaginatedData,
	PrivateChatItem,
	PrivateChatOpenResult,
	PrivateMessageItem,
	SendPrivateMessageRequest,
	ListPrivateMessagesRequest,
	SendGroupMessageRequest,
	ListGroupMessagesRequest,
	MarkPrivateReadRequest,
	MarkGroupReadRequest,
	UpdatePrivateChatPreferencesRequest,
	PaginationParams,
} from '@/api/types';

// Chat
export async function getPrivateChats(params?: { pagination?: PaginationParams })    { return request<PaginatedData<PrivateChatItem>>(Endpoints.PrivateChats.List, params ?? {}); }
// spec: api_spec_chat.md → private_chats/delete { target_user_id }（仅删除自己视角的会话）
export async function deletePrivateChat(params: { target_user_id: string }) { return request<void>(Endpoints.PrivateChats.Delete, params); }
export async function getPrivateChatDetail(params: { target_user_id: string }) { return request<PrivateChatOpenResult>(Endpoints.PrivateChats.GetDetail, params); }
export async function getPrivateMessages(params: ListPrivateMessagesRequest)  { return request<PaginatedData<PrivateMessageItem>>(Endpoints.PrivateMessages.List, params); }
export async function sendPrivateMessage(params: SendPrivateMessageRequest)  { return request<MessageSentResult>(Endpoints.PrivateMessages.Send, params, { 'Idempotency-Key': newIdempotencyKey() }); }
export async function getGroupMessages(params: ListGroupMessagesRequest)    { return request<PaginatedData<GroupMessageItem>>(Endpoints.GroupMessages.List, params); }
export async function sendGroupMessage(params: SendGroupMessageRequest)    { return request<MessageSentResult>(Endpoints.GroupMessages.Send, params, { 'Idempotency-Key': newIdempotencyKey() }); }
// spec: api_spec_chat.md → 消息撤回/编辑（recall 受窗口限制/群管理员豁免，update 仅 type=text）
export async function recallPrivateMessage(params: { message_id: string }) { return request<void>(Endpoints.PrivateMessages.Recall, params); }
export async function updatePrivateMessage(params: { message_id: string; content: string }) { return request<PrivateMessageItem>(Endpoints.PrivateMessages.Update, params); }
export async function recallGroupMessage(params: { message_id: string }) { return request<void>(Endpoints.GroupMessages.Recall, params); }
export async function updateGroupMessage(params: { message_id: string; content: string }) { return request<GroupMessageItem>(Endpoints.GroupMessages.Update, params); }
export async function markPrivateRead(params: MarkPrivateReadRequest)     { return request<void>(Endpoints.PrivateMessages.MarkRead, params); }
export async function markGroupRead(params: MarkGroupReadRequest)       { return request<void>(Endpoints.GroupMessages.MarkRead, params); }
// spec: api_spec_chat.md → private_chats/update_preferences（静音/置顶）
export async function updatePrivateChatPreferences(params: UpdatePrivateChatPreferencesRequest) { return request<void>(Endpoints.PrivateChats.UpdatePreferences, params); }
// spec: api_spec_group.md → groups/update_preferences（静音/置顶）
