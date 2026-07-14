/**
 * (public) 路由组布局配置
 *
 * 禁用 SSR：与 (protected) 组保持一致。此前 (public) 启用 SSR 而
 * (protected) 禁用 SSR，导致刷新 (public) 页面时 SSR 返回正确内容、
 * 但 hydration 阶段客户端路由重新解析时加载了错误的 page node
 * (如 /discover 刷新后渲染 /relations 内容)。统一 CSR 消除此不对称。
 *
 * 公开页面的数据加载本就在客户端 $effect 中完成（无 +page.ts load 函数），
 * SSR 只产出空壳 HTML，禁用 SSR 不影响功能。
 */
export const ssr = false;
