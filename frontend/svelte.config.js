import adapter from '@sveltejs/adapter-static';

/**
 * SPA 模式：全局 ssr=false（见 src/routes/+layout.ts），所有页面在浏览器渲染，
 * 无 SSR/预渲染。adapter-static 仅构建静态资源壳 + 一个 index.html 兜底。
 *
 * fallback: 未命中静态文件的所有路径（含动态路由 /users/[id] 等）统一返回 index.html，
 * 由 SvelteKit 客户端路由器解析 URL 并渲染——与 ssr=false 下的运行时行为完全一致，
 * 只是省去了不干活的 Node 进程，由 nginx 直接吐静态文件。
 */
const config = {
  kit: {
    adapter: adapter({
      fallback: 'index.html'
    })
  }
};

export default config;
