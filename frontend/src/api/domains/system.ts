// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request, pick } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	BootstrapData,
	ClientConfig,
	BootstrapRequest,
	SystemAnnouncementItem,
} from '@/api/types';

// System
export async function getClientConfig()                   { return request<ClientConfig>(Endpoints.System.GetClientConfig); }
export async function bootstrap(params?: BootstrapRequest)          { return request<BootstrapData>(pick(Endpoints.System.Bootstrap, Endpoints.Public.System.Bootstrap), params ?? {}); }

/** 公开系统公告（spec api_spec_public.md system/announcements/fetch；条数由服务端 AppConfig 控制）。 */
export async function listSystemAnnouncements() {
	return request<{ items: SystemAnnouncementItem[] }>(Endpoints.System.ListAnnouncements);
}
