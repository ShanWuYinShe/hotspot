// 附近三页（附近的人 / 附近群组 / 附近动态）共享的内容宽度契约。
// 对齐 Figma Make NearbyPeoplePage/NearbyGroupsPage 的 ContentColumn maxWidth：
//   桌面 1152px / 平板 964px / 移动 680px
// 三页必须统一调用本函数，禁止各页手写不同 maxWidth —— 否则桌面右侧主内容区
// 的视觉留白会漂移（历史上 nearby/moments 全宽、人/群组 1152px 就曾不一致）。
export function nearbyContentMaxWidth(platform: 'mobile' | 'tablet' | 'desktop'): string {
	return platform === 'tablet' ? '964px' : platform === 'desktop' ? '1152px' : '680px';
}
