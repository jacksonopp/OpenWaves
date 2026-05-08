import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/admin/ui/',
  build: {
    outDir: '../internal/adminui/dist',
    emptyOutDir: true,
  },
  server: {
    allowedHosts: true,
    proxy: {
      // Match /admin/* but NOT /admin/ui (which is the SPA itself)
      '^/admin(?!/ui)': 'http://localhost:8080',
      '/stations': 'http://localhost:8080',
    },
  },
})
