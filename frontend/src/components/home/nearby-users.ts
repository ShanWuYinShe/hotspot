/** 首页「附近的人」列表项。 */
export interface NearbyUser {
	id: string;
	name: string;
	distance: string;
	avatar: string;
	isOnline: boolean;
	tag: string;
	/** 查看者是否已关注（users/search relation.is_following；关注钮初始态基准） */
	isFollowing: boolean;
}
