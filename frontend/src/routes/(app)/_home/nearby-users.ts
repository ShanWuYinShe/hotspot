/** 首页「附近的人」列表项（原 PeopleRadarStrip 组件导出，组件移除后独立成类型）。 */
export interface NearbyUser {
	id: string;
	name: string;
	distance: string;
	avatar: string;
	status: string;
	tag: string;
}
