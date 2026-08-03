import { redirect } from '@sveltejs/kit';

// 旧路径 /notify 已更名为 /notifications（页面语义为「通知」），保留旧路径重定向。
export const load = () => {
	redirect(307, '/notifications');
};
