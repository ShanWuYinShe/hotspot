---
name: hotspot-frontend-prod-console
description: "Hotspot Web 线上（chaoset.com，Cloudflare 橙云 + 源站 nginx）控制台报错分诊：CF Web Analytics beacon 的 CORS/integrity 报错、Firefox 字体 preload「未使用」告警、Firefox 对 -webkit-text-size-adjust 的解析告警、弱网/部署窗口下 chunk 动态 import 失败。触发：线上控制台出现上述报错；判断某条报错是否为站点 bug；复核 CF beacon 注入与完整性哈希；排查 Firefox 特有的字体/CSS 告警。"
---

# Hotspot Web 线上控制台报错分诊

chaoset.com 控制台常见的四类报错**都不是站点代码 bug**。先按本文件分诊，再决定是否动手；不要为消除这些告警改业务代码或构建配置。

## 1. CF Web Analytics beacon：CORS + integrity 两条报错

**报错原文（Firefox）**

- `已拦截跨源请求：同源策略禁止读取位于 https://static.cloudflareinsights.com/beacon.min.js/... 的远程资源。（原因：CORS 请求未能成功）。状态码：(null)。`
- `integrity 属性中的“sha512”散列值没有一个符合位于“...”的子资源的内容。计算出的散列值为“z4PhN...”`

**根因**：橙云站点由 Cloudflare 边缘自动向 HTML 注入 Web Analytics beacon：

```html
<script type="module" src="https://static.cloudflareinsights.com/beacon.min.js/<版本路径>"
  integrity="sha512-..." crossorigin="anonymous" data-cf-beacon='{"token":"..."}'>
```

该跨源请求失败（弱网、广告/跟踪拦截扩展、网络策略）时响应为空：Firefox 先报 CORS 失败，再报空内容的哈希不符。

**判定要点**

- 空响应的 SHA-512（base64）恒为 `z4PhNX7vuL3xVChQ1m2AB9Yg5AULVxXcg/SpIdNs6c5H0NE8XYXysP+DGNKHfuwvY7kxvUdBeoGlODJ6+SfaPg==`。控制台出现这个值 = 响应为空，不是资源被篡改。
- 源站 HTML 不含这段脚本（不带浏览器头的 `curl` 抓不到），注入发生在 CF 边缘，站点侧无法用代码移除。
- CSP 已放行（`docker/nginx.csp.inc` 的 `script-src` 含 `https://static.cloudflareinsights.com`、`connect-src` 含 `https://cloudflareinsights.com`），不是 CSP 拦截。

**处理**：站点侧不改。用户侧检查 uBlock/AdGuard/ETP 等拦截扩展；不需要该统计时在 CF 控制台关闭 Web Analytics。

## 2. Firefox：字体 preload「未使用」

**报错原文**：`通过 link preload 预加载的资源“https://chaoset.com/fonts/inter-latin-wght.woff2”并未在加载后几秒内使用。请确保 preload 标签的所有属性均设置无误。`

**根因**：前端 `ssr=false`（SPA），首屏 HTML 是空壳；Inter 是 latin 子集，只有应用渲染出拉丁字形后才被使用。弱网下首屏渲染超过几秒，Firefox 的启发式检查就报「未使用」；健康网络下实测字体确实被加载并使用（`document.fonts.check('16px Inter') === true`，且页面文本含拉丁字符）。

**处理**：忽略。不要删 `frontend/src/app.html` 的字体 preload（`crossorigin` 已正确设置，删除会让拉丁文本 FOUT）；也不要改 `frontend/src/styles/tailwind.css` 的 `--font-sans`（已包含 Inter）。

## 3. Firefox：`-webkit-text-size-adjust:100%` 解析告警

**报错原文**：`解析 '-webkit-text-size-adjust:100%;' 的值时出错。声明被丢弃。`，位置指向 `_app/immutable/assets/0.<hash>.css`。

**根因**：Firefox 桌面版不支持 `text-size-adjust`（MDN BCD `css/properties/text-size-adjust` 的 `firefox.version_added` 为 `false`；只有 Firefox Android 支持 `-webkit-` 前缀），而 Tailwind v4 preflight 会输出该声明。Firefox 解析时丢弃并打警告，控制台的行号列号与线上 CSS 中该声明位置一致。

**处理**：无视觉影响（Firefox 桌面本来就不做文字自动缩放），不改。删掉该声明会失去 iOS Safari 的文字自动缩放保护。

## 4. 弱网/部署窗口：chunk 动态 import 失败

**报错原文**：`Uncaught (in promise) TypeError: error loading dynamically imported module: https://chaoset.com/_app/immutable/chunks/<hash>.js`（Chrome 文案为 `Failed to fetch dynamically imported module`）。

**根因**：某次 `_app/immutable` 静态资源请求在弱网或部署滚动窗口失败。国内直连 chaoset.com 到 CF 边缘的 TLS 干扰是已确认的 P0（判定方式见下）；失败落在根布局 node 的依赖链时首屏会白屏，刷新可恢复。

**判定要点**：先排除构建缺文件——线上 `/version.json` 的 `commit` 应等于 `git rev-parse HEAD`；再请求报错的文件，应返回 200。文件存在且版本一致时，问题在链路，不在产物。

本机到 CF 边缘可能对同一 URL 时通时断：连续请求同一 chunk，成功与 `code=000`（TLS/超时）会夹杂出现。遇到这种"有时白屏、刷新恢复"且本机取资源不稳时，换**源站侧视角**复核（服务器出网 → CF 边缘 → 源站）：同一文件返回 200 即证明产物健康，属链路问题；只有 404 或版本不符时才按产物问题处理（回滚/重跑构建）。

**处理**：根因是入口链路（换香港中转/迁移等基础设施决策），不要为此重跑构建找「缺文件」。

## 复核命令

```bash
# 0) 源站侧视角：本机取 chunk 偶发超时/白屏时先跑这条，避免把链路问题误判成产物缺失
ssh workwork-server "curl -s -o /dev/null -w '%{http_code}\n' 'https://chaoset.com/_app/immutable/chunks/<报错文件>'"

# 1) 线上版本与报错文件
curl -sS https://chaoset.com/version.json
curl -sS -o /dev/null -w '%{http_code}\n' 'https://chaoset.com/_app/immutable/chunks/<报错文件>'

# 2) CF beacon 注入与完整性（用浏览器 UA 才会注入）
curl -sS -A '<Chrome UA>' -H 'Accept: text/html' https://chaoset.com/ | grep -oE '<script[^>]*cloudflareinsights[^>]*>'
curl -sS --compressed -o /tmp/beacon.js 'https://static.cloudflareinsights.com/beacon.min.js/<版本路径>'
echo "sha512-$(openssl dgst -sha512 -binary /tmp/beacon.js | openssl base64 -A)"   # 应与标签内 integrity 一致

# 3) Firefox CSS 告警定位（行号列号应与控制台一致）
curl -sS --compressed -o /tmp/app.css 'https://chaoset.com/_app/immutable/assets/<报错css>'
grep -o 'text-size-adjust:[^;}]*' /tmp/app.css | head

# 4) Firefox 兼容数据
curl -sS https://raw.githubusercontent.com/mdn/browser-compat-data/main/css/properties/text-size-adjust.json | grep -A3 '"firefox"'
```

## 不要做的事

- 不要为消 beacon 报错改 CSP。
- 不要为消 preload 告警删字体 preload。
- 不要为消 CSS 告警改 Tailwind preflight 或删 `-webkit-text-size-adjust`。
- chunk 报错先验证线上文件存在且版本一致，再查链路；不要重跑构建。

## 相关

- 字体自托管与 preload 口径：skill `hotspot-frontend-theme`
