import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { lazyPage } from '@/shared/routing/lazyPage';
import { GuestOnly } from './guards/GuestOnly/GuestOnly';

/** The login and the sign-up: only for visitors without a session. */
export const authRoutes: RouteObject[] = [
  {
    element: <GuestOnly />,
    children: [
      { path: paths.login.pattern, ...lazyPage(async () => (await import('./pages')).LoginPage) },
      { path: paths.signUp.pattern, ...lazyPage(async () => (await import('./pages')).SignUpPage) },
    ],
  },
];
