import { fakeApi, json, signedOut } from '@/test/fakeApi';

// The templates GET /business-templates answers, in the order they are shown.
export const businessTemplates = {
  items: [
    { id: 'generica', name: 'Negocio general', description: 'Cualquier comercio.' },
    { id: 'ferreteria', name: 'Ferretería', description: 'Herramientas, construcción y acabados.' },
    { id: 'ropa_calzado', name: 'Ropa y calzado', description: 'Prendas y zapatos por talla y color.' },
  ],
};

/** A visitor without a session on the sign-up, with the business types loaded. */
export function signUpApi(routes: Parameters<typeof fakeApi>[0] = {}) {
  return fakeApi({ ...signedOut(), 'GET /api/v1/business-templates': json(200, businessTemplates), ...routes });
}
