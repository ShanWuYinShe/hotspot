/** @type {import('next').NextConfig} */
const isProd = process.env.NODE_ENV === 'production'

const nextConfig = {
  // 静态导出约束（generateStaticParams 枚举校验）仅在构建期生效。
  // dev 模式关闭 output: 'export'，否则 dev server 只允许访问
  // generateStaticParams 枚举出的路径，所有动态详情页（/moments/:id 等）都会 500。
  output: isProd ? 'export' : undefined,
  // dev 与 build 产物目录分离：build 会清空 distDir，若共用同一目录会
  // 反复破坏正在运行的 dev server 缓存（宿主机 bun run build 时尤其容易踩）。
  distDir: isProd ? 'out' : 'out-dev',
  trailingSlash: false,
  images: {
    unoptimized: true,
  },
}

export default nextConfig
