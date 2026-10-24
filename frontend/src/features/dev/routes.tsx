import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';

/**
 * Internal pages that never reach a production build: import.meta.env.DEV is
 * false there, so the bundler drops the route and its lazy import.
 */
export const devRoutes: RouteObject[] = import.meta.env.DEV
  ? [
      {
        path: paths.dev.ui.pattern,
        lazy: async () => ({ Component: (await import('./pages')).UiCatalogPage }),
        // Shown while the lazy page loads on a first visit to /dev/ui.
        HydrateFallback: () => null,
      },
    ]
  : [];
