	// 详情页与 rail 子组件共享的轻量类型（数据源为 +page.svelte 的 $state）。
	export type MomentDetail = import('../MomentPostBody.svelte').MomentData;
	export interface RailPerson {
		id: string;
		name: string;
		sub: string;
		avatar: string;
	}
	export interface RailMoment {
		id: string;
		title: string;
		meta: string;
		image: string;
	}
