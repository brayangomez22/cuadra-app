import type { RouteObject } from 'react-router';
import { paths } from '@/app/routes/paths';
import { lazyPage } from '@/shared/routing/lazyPage';
import { GuestOnly } from './guards/GuestOnly/GuestOnly';

/** The login, the sign-up, the demo and the invitation and password links: only for visitors without a session. */
export const authRoutes: RouteObject[] = [
  {
    element: <GuestOnly />,
    children: [
      { path: paths.login.pattern, ...lazyPage(async () => (await import('./pages')).LoginPage) },
      { path: paths.signUp.pattern, ...lazyPage(async () => (await import('./pages')).SignUpPage) },
      { path: paths.demo.pattern, ...lazyPage(async () => (await import('./pages')).DemoPage) },
      { path: paths.acceptInvitation.pattern, ...lazyPage(async () => (await import('./pages')).AcceptInvitationPage) },
      { path: paths.resetPassword.pattern, ...lazyPage(async () => (await import('./pages')).ResetPasswordPage) },
    ],
  },
];
