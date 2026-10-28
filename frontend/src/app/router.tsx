import { createBrowserRouter, type RouteObject } from 'react-router';
import { RequireAuth } from '@/features/auth';
import { authRoutes } from '@/features/auth/routes';
import { catalogRoutes } from '@/features/catalog/routes';
import { devRoutes } from '@/features/dev/routes';
import { importRoutes } from '@/features/imports/routes';
import { inventoryRoutes } from '@/features/inventory/routes';
import { salesRoutes } from '@/features/sales/routes';
import { settingsRoutes } from '@/features/settings/routes';
import { AppLayout } from './layout/AppLayout';
import { HomePage } from './pages/HomePage';
import { paths } from './routes/paths';
import { RouteError } from './RouteError';

// Each feature declares its screens in routes.tsx, loaded with their first
// visit in the chunk of the feature, behind the permission of their API. The
// router imports routes.tsx, not the feature's index.ts, which would bring
// everything the feature exports into the main bundle. It only nests them:
// the layout stays in the main bundle. URLs are in
// Spanish: the user sees them (routes/paths.ts). Everything hangs from one
// root route whose errorElement reports what a screen throws and answers
// unknown URLs.
export const routes: RouteObject[] = [
  {
    errorElement: <RouteError />,
    children: [
      ...authRoutes,
      {
        element: <RequireAuth />,
        children: [
          {
            element: <AppLayout />,
            children: [
              { path: paths.home.pattern, element: <HomePage /> },
              ...catalogRoutes,
              ...importRoutes,
              ...inventoryRoutes,
              ...salesRoutes,
              ...settingsRoutes,
            ],
          },
        ],
      },
      ...devRoutes,
    ],
  },
];

export const router = createBrowserRouter(routes);
