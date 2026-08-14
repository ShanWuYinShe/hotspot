import { initMock } from '@/mocks/toggle';

const RELOADED_FLAG = 'hotspot:mock-reloaded';

/**
 * 客户端启动钩子：SvelteKit 在渲染任何组件、发出任何 API 请求之前会
 * `await hooks.init()`，所以在这里完成 MSW 注册，保证 dev 下页面启动时
 * 的 get_status / get_client_config / feed 等请求全部被 mock 拦截。
 *
 * service worker 首次注册后无法控制当前页面（必须刷新一次才接管），
 * 这里检测到未受控时刷新一次；用 sessionStorage 标记防止异常环境下死循环。
 */
export async function init() {
  if (!(import.meta.env.DEV && import.meta.env.PUBLIC_API_MOCK !== 'false')) return;

  try {
    const { worker } = await import('@/mocks/browser');
    const enabled = await initMock(worker, true);
    if (
      enabled &&
      typeof navigator !== 'undefined' &&
      !navigator.serviceWorker.controller &&
      !sessionStorage.getItem(RELOADED_FLAG)
    ) {
      sessionStorage.setItem(RELOADED_FLAG, '1');
      location.reload();
    }
  } catch (err) {
    // 浏览器禁用 service worker 等场景：不阻塞应用，API 请求直连真实后端。
    console.error('[msw] worker 启动失败，API 请求将直连真实后端', err);
  }
}
