/**
 * 全局禁用 SSR。
 *
 * 本应用不适合 SSR：
 * 1. 所有 API 鉴权依赖 HttpOnly Cookie（spec 规定不走 Authorization Header），
 *    SSR 阶段的服务端 fetch 无法携带浏览器 Cookie，load 函数里的请求必然失败。
 * 2. 数据加载在客户端 $effect / onMount 中完成，SSR 只产出空壳，无首屏价值。
 * 3. 内容在登录后才可见，无 SEO 需求。
 * 4. 混用 SSR (public 组) + CSR (protected 组) 会导致 hydration 时路由 node_ids
 *    解析错误（SvelteKit dev/prod 均复现），刷新公开页面渲染出错误路由内容。
 *
 * 统一 CSR 消除此问题，且不影响任何功能。
 */
export const ssr = false;
