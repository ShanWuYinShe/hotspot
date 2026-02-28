/**
 * (protected) 路由组布局配置
 *
 * 禁用 SSR：受保护页面的 API 调用依赖 HttpOnly Cookie 鉴权，
 * SSR 阶段的全局 fetch 无法携带浏览器 Cookie，导致 load 函数中的
 * API 请求无法通过服务端会话校验（业务层返回 `code=10100`）。因此这些页面仅在客户端渲染。
 */
export const ssr = false;
