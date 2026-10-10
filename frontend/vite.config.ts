/// <reference types="vitest/config" />
import { fileURLToPath, URL } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// Conventions in docs/estilos.md.
export default defineConfig(({ mode }) => ({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  css: {
    modules: {
      localsConvention: 'camelCaseOnly',
      // Readable class names in DevTools during development, short hashes in production.
      generateScopedName: mode === 'production' ? '[hash:base64:6]' : '[name]__[local]__[hash:base64:4]',
    },
  },
  server: {
    // Same origin as the API in development: the refresh cookie (SameSite=Strict,
    // path /api/v1/auth) reaches it and the API needs no CORS.
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: { modules: { classNameStrategy: 'non-scoped' } },
  },
}));
