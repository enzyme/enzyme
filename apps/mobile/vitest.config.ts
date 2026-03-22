import path from 'path';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
  },
  resolve: {
    alias: {
      '@enzyme/api-client': path.resolve(__dirname, '../../packages/api-client/src'),
      '@enzyme/shared': path.resolve(__dirname, '../../packages/shared/src'),
      '@/*': path.resolve(__dirname, 'src'),
    },
  },
});
