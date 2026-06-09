import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  server: {
    port: parseInt(process.env.PORT || '5173'),
    allowedHosts: ['.yinghuoren.com', '.chaoset.com', '.orb.local']
  },
  preview: {
    port: parseInt(process.env.PORT || '5173'),
    allowedHosts: ['.yinghuoren.com', '.chaoset.com', '.orb.local']
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/svelte/')) {
            return 'svelte';
          }
        }
      }
    }
  }
});
