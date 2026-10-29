import { fakeApi, json, signedOut } from '@/test/fakeApi';

// The demos GET /auth/demo answers: a hardware store and a clothing store.
export const demoOptions = {
  items: [
    { business_template: 'ferreteria', name: 'Ferretería' },
    { business_template: 'ropa_calzado', name: 'Ropa y calzado' },
  ],
};

/** A visitor without a session, with the demo on. */
export function demoApi(routes: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({ ...signedOut(), 'GET /api/v1/auth/demo': json(200, demoOptions), ...routes });
}
