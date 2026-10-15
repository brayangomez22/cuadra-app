/// <reference types="vitest/config" />
import { fileURLToPath, URL } from 'node:url';
import { sentryVitePlugin } from '@sentry/vite-plugin';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// Source maps are uploaded to Sentry only from the CI, which has SENTRY_AUTH_TOKEN
// (plus SENTRY_ORG and SENTRY_PROJECT). They are "hidden" (no sourceMappingURL
// comment) and deleted after the upload, so the deployed bundle never serves them.
// Without the token, as in local builds and PRs from forks, no maps are generated.
const uploadSourceMaps = Boolean(process.env.SENTRY_AUTH_TOKEN);

// Conventions in docs/estilos.md.
export default defineConfig(({ mode }) => ({
  plugins: [
    react(),
    uploadSourceMaps &&
      sentryVitePlugin({
        org: process.env.SENTRY_ORG,
        project: process.env.SENTRY_PROJECT,
        authToken: process.env.SENTRY_AUTH_TOKEN,
        // The same name the app reports (VITE_RELEASE), so events find their maps.
        release: { name: process.env.VITE_RELEASE, setCommits: false },
        sourcemaps: { filesToDeleteAfterUpload: ['./dist/**/*.map'] },
        telemetry: false,
      }),
  ],
  build: {
    sourcemap: uploadSourceMaps ? 'hidden' : false,
    rollupOptions: {
      output: {
        // React and the router change only with an upgrade: in their own chunk,
        // browsers keep them cached across deploys.
        manualChunks: (id) =>
          /node_modules\/(react|react-dom|react-router|scheduler)\//.test(id) ? 'vendor' : undefined,
      },
    },
  },
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
