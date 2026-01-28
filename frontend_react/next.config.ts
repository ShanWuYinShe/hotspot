import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // 实验性功能
  experimental: {
    // React 19 支持
    reactCompiler: false,
  },
  
  // 图片配置
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '**',
      },
    ],
  }
};

export default nextConfig;
