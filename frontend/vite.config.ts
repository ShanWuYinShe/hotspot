import { defineConfig } from 'vite'
import path from 'path'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'


function figmaAssetResolver() {
  return {
    name: 'figma-asset-resolver',
    resolveId(id: string) {
      if (id.startsWith('figma:asset/')) {
        const filename = id.replace('figma:asset/', '')
        return path.resolve(__dirname, 'src/assets', filename)
      }
    },
  }
}

export default defineConfig({
  plugins: [
    figmaAssetResolver(),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-react': ['react', 'react-dom', 'react-router'],
          'vendor-motion': ['motion/react'],
          'vendor-lucide': ['lucide-react'],
          'vendor-zustand': ['zustand'],
        },
      },
    },
  },
  // 开发期 /api 与 /ws 反代由 nginx 网关统一处理，与生产保持一致。
  // 访问入口为 http://localhost:<NGINX_PORT>（默认 8888，由 .env 配置），
  // dev server 只负责静态资源 + HMR（端口 5173 仍 expose 但不再代理 API）。
  // 详见 docker-compose.dev.yml 与 docker/nginx.dev.conf。
  assetsInclude: ['**/*.svg', '**/*.csv'],
})
