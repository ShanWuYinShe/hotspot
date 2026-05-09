import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  server: {
    allowedHosts: ['.yinghuoren.com', '.chaoset.com']
  },
  preview: {
    allowedHosts: ['.yinghuoren.com', '.chaoset.com']
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
