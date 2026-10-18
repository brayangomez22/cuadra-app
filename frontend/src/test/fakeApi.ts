import type { components } from '@/shared/api/schema';

type Handler = (request: Request) => Response | Promise<Response>;

// Routes are keyed by "METHOD /path", e.g. "POST /api/v1/auth/login".
type Routes = Record<string, Handler | Response>;

export type Me = components['schemas']['Me'];

// fakeApi replaces the global fetch the API client uses. A route without a
// handler answers 404 so a missing stub shows up as a failing assertion.
export function fakeApi(routes: Routes) {
  const requests: Request[] = [];

  vi.stubGlobal('fetch', async (request: Request) => {
    requests.push(request.clone());
    const route = routes[`${request.method} ${new URL(request.url).pathname}`];
    if (!route) return json(404, { error: { code: 'not_found', message: 'No existe' } });
    return typeof route === 'function' ? route(request) : route.clone();
  });

  return {
    // calls lists the requests sent to a route, in order.
    calls: (key: string) => requests.filter((r) => `${r.method} ${new URL(r.url).pathname}` === key),
  };
}

export function json(status: number, body: unknown) {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
}

export function apiError(status: number, code: string, message: string, details?: Record<string, string>) {
  return json(status, { error: { code, message, details } });
}

const permissionsByRole: Record<Me['role'], string[]> = {
  owner: [
    'catalog:read', 'catalog:write', 'inventory:read', 'inventory:adjust',
    'sales:create', 'sales:discount', 'sales:read', 'sales:void', 'users:manage', 'tenant:manage',
  ],
  admin: [
    'catalog:read', 'catalog:write', 'inventory:read', 'inventory:adjust',
    'sales:create', 'sales:discount', 'sales:read', 'sales:void', 'users:manage',
  ],
  cashier: ['catalog:read', 'inventory:read', 'sales:create', 'sales:read'],
  warehouse: ['catalog:read', 'inventory:read', 'inventory:adjust'],
};

export function aUser(role: Me['role'] = 'owner'): Me {
  return {
    id: '01920000-0000-7000-8000-000000000001',
    tenant_id: '01920000-0000-7000-8000-0000000000aa',
    tenant_name: 'Ferretería El Tornillo',
    tenant_nit: '900123456-8',
    business_template: 'ferreteria',
    default_tax_rate: '0.19',
    name: 'Ana Gómez',
    email: 'ana@eltornillo.co',
    role,
    permissions: permissionsByRole[role],
    modules: [],
  };
}

export function anAuthSession(user = aUser()) {
  return { access_token: 'access-token', token_type: 'Bearer', expires_in: 900, user };
}

// signedIn stubs a session restored from the refresh cookie (GET /me answers the user).
export function signedIn(user = aUser()): Routes {
  return { 'GET /api/v1/me': json(200, user) };
}

// signedOut stubs a visitor without a session: /me and the refresh both answer 401.
export function signedOut(): Routes {
  return {
    'GET /api/v1/me': apiError(401, 'unauthorized', 'Inicie sesión'),
    'POST /api/v1/auth/refresh': apiError(401, 'invalid_session', 'La sesión venció'),
  };
}
