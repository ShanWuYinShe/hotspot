// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	NotifyItem,
	PaginatedData,
	UnreadSummaryResult,
	MarkNotifyReadRequest,
	PaginationParams,
} from '@/api/types';

// Notify
export async function getUnreadCount()                    { return request<UnreadSummaryResult>(Endpoints.Notify.GetUnreadCount); }
export async function listNotifies(params?: { pagination?: PaginationParams })  { return request<PaginatedData<NotifyItem>>(Endpoints.Notify.List, params ?? {}); }
// spec: notify.md → mark_read（参数必显式）：scope='ids' 指定 1-50 条；scope='all' 全部已读。
export async function markRead(params: MarkNotifyReadRequest)            { return request<void>(Endpoints.Notify.MarkRead, params); }
