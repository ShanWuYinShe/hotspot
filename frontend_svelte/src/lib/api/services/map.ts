/**
 * 地图服务
 */

import { post } from '../client';
import { MAP } from '../endpoints';
import type { MapCanvas, MapDataType, MapViewport, TimeRangeParams } from '$lib/types/api';

export const mapService = {
	/**
	 * 获取地图渲染数据
	 */
	render(data: {
		viewport: MapViewport;
		filter?: {
			data_type?: MapDataType;
			time_range?: TimeRangeParams;
			only_friends?: boolean;
		};
	}) {
		return post<MapCanvas>(MAP.RENDER, data);
	}
};
