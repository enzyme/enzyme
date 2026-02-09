import path from 'path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import type { Plugin } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    ...(process.env.VISUALIZE
      ? [(await import('rollup-plugin-visualizer')).visualizer({
          open: true,
          filename: 'dist/stats.html',
        }) as unknown as Plugin]
      : []),
  ],
  resolve: {
    alias: {
      '@feather/api-client': path.resolve(__dirname, '../../packages/api-client/src'),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            if (
              id.includes('@tiptap/') ||
              id.includes('tiptap-markdown') ||
              id.includes('prosemirror-')
            ) {
              return 'vendor-tiptap'
            }
            if (
              id.includes('react-aria-components') ||
              id.includes('@react-aria/') ||
              id.includes('@react-stately/') ||
              id.includes('@react-types/') ||
              id.includes('@internationalized/')
            ) {
              return 'vendor-react-aria'
            }
            if (id.includes('@dnd-kit/')) {
              return 'vendor-dnd'
            }
            if (
              id.includes('react-dom') ||
              id.includes('react-router-dom') ||
              id.includes('@tanstack/react-query') ||
              // Match react/ but not react-dom, react-router, react-aria, etc.
              /\/node_modules\/react\//.test(id)
            ) {
              return 'vendor-react'
            }
          }
        },
      },
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
