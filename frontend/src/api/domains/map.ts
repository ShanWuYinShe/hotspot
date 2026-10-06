// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	MapCanvasRequest,
	MapCanvasResponse,
} from '@/api/types';

// Map
export async function getMapCanvas(params: MapCanvasRequest) { return request<MapCanvasResponse>(Endpoints.Map.Canvas, params); }
