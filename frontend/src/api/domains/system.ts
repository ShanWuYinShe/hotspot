// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request, pick } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	BootstrapData,
	ClientConfig,
	BootstrapRequest,
} from '@/api/types';

// System
export async function getClientConfig()                   { return request<ClientConfig>(Endpoints.System.GetClientConfig); }
export async function bootstrap(params?: BootstrapRequest)          { return request<BootstrapData>(pick(Endpoints.System.Bootstrap, Endpoints.Public.System.Bootstrap), params ?? {}); }
