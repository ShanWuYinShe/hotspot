/**
 * (admin) 路由组布局配置
 *
 * 禁用 SSR：管理后台页面的 API 调用依赖 HttpOnly Cookie 鉴权，
 * SSR 阶段的全局 fetch 无法携带浏览器 Cookie，因此仅在客户端渲染。
 */
export const ssr = false;
