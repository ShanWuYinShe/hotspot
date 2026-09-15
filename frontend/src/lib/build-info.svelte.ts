// 构建版本指纹（runes 模块单例）——读取构建期生成的 /version.json
// （frontend/scripts/write-version-info.mjs 产物，随 SPA 静态发布）。
// 设置页「关于 HotSpot」徽标据此展示；读取失败（如本地 dev 未构建过、
// 文件缺失）兜底 version="dev"，不阻塞页面。

export interface BuildInfo {
	project?: string;
	version: string;
	commit: string;
	build_time: string;
}

const FALLBACK: BuildInfo = { version: 'dev', commit: '', build_time: '' };

async function defaultFetch(): Promise<BuildInfo> {
	const res = await fetch('/version.json');
	if (!res.ok) throw new Error(`version.json ${res.status}`);
	return (await res.json()) as BuildInfo;
}

/** 工厂形式便于测试注入 fetch；页面使用下方 buildInfo 单例。 */
export function createBuildInfo(fetchJson: () => Promise<BuildInfo> = defaultFetch) {
	let info = $state<BuildInfo | null>(null);
	let pending: Promise<void> | null = null;

	function load(): Promise<void> {
		// 幂等：并发调用共享同一次请求；失败兜底 dev，不抛出
		pending ??= fetchJson()
			.then((value) => {
				info = value?.version ? value : FALLBACK;
			})
			.catch(() => {
				info = FALLBACK;
			});
		return pending;
	}

	return {
		load,
		get info() {
			return info;
		}
	};
}

export const buildInfo = createBuildInfo();
