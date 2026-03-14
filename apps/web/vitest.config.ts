import path from 'path';
import { defineConfig, mergeConfig } from 'vitest/config';
import viteConfig from './vite.config';

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/test-utils/setup.ts'],
      include: ['src/**/*.{test,spec}.{ts,tsx}'],
      // Node 25+ enables --experimental-webstorage by default, which provides
      // a built-in localStorage that shadows jsdom's implementation. Disable it
      // so jsdom's full Storage API is available in tests.
      execArgv: ['--no-experimental-webstorage'],
      coverage: {
        provider: 'v8',
        reporter: ['text', 'html', 'lcov'],
        include: ['src/**/*.{ts,tsx}'],
        exclude: ['src/**/*.{test,spec}.{ts,tsx}', 'src/test-utils/**', 'src/vite-env.d.ts'],
      },
    },
    resolve: {
      alias: {
        '@enzyme/api-client': path.resolve(__dirname, '../../packages/api-client/src'),
      },
    },
  }),
);
