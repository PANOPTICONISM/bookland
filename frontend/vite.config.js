import { defineConfig, loadEnv } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '../', '')

  return {
    plugins: [svelte()],
    server: {
      proxy: {
        '/api': {
          target: `http://localhost:${env.PORT}`,
          changeOrigin: true
        }
      }
    },
    optimizeDeps: {
      include: ['pdfjs-dist']
    }
  }
})
