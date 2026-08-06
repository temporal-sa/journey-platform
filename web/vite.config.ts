/// <reference types="vitest" />
import { defineConfig, Plugin } from 'vite';
import react from '@vitejs/plugin-react';
import { generateBuildTag } from './scripts/generate-build-tag.js';

function autoBuildTagPlugin(): Plugin {
  return {
    name: 'auto-build-tag-plugin',
    buildStart() {
      generateBuildTag();
    },
    handleHotUpdate({ file }) {
      // Ignore edits to buildTag.ts itself to avoid infinite HMR loops
      if (file.endsWith('buildTag.ts')) return;

      // When any source code file in src/ changes, re-generate build tag
      if (file.includes('/src/')) {
        generateBuildTag();
      }
    },
  };
}

export default defineConfig({
  plugins: [react(), autoBuildTagPlugin()],
  server: {
    port: parseInt(process.env.FRONTEND_PORT || '3000', 10),
    strictPort: true,
    proxy: {
      '/api': {
        target: `http://127.0.0.1:${process.env.CONTROL_API_PORT || '8080'}`,
        changeOrigin: true,
      },
    },
  },
  test: {
    globals: true,
    environment: 'happy-dom',
    setupFiles: './src/test/setup.ts',
    pool: 'forks',
    poolOptions: {
      forks: {
        singleFork: true,
      },
    },
  },
});
