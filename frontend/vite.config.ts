import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(() => {
  const apiTarget = process.env.VITE_API_BASE ?? 'http://localhost:8080'
  const wsTarget = apiTarget.replace(/^http/i, 'ws')

  return {
    plugins: [react()],
    server: {
      proxy: {
        '/api/ws': { target: wsTarget, ws: true, changeOrigin: true },
        '/api': { target: apiTarget, changeOrigin: true },
      },
    },
  }
})
