import type { components } from '@/shared/api/schema';
import type { ResolveKey, ResponseBody } from './routeKeys';

declare const body: unique symbol;

/** A Response that carries the type of its JSON body, so tsc checks it against the contract. */
export type Reply<B> = Response & { readonly [body]?: B };

type Stub<B> = Reply<B> | ((request: Request) => Reply<B> | Promise<Reply<B>>);

// Checked gives each key the stub its operation allows: never for a key the
// contract does not have. A computed key (`GET /api/v1/products/${id}`) is a
// plain string to tsc and takes any reply: use the template key with a
// handler when the body matters more than the ID.
type Checked<R> = {
  [K in keyof R]: K extends string
    ? string extends K
      ? Stub<unknown>
      : [ResolveKey<K>] extends [never]
        ? never
        : Stub<ResponseBody<ResolveKey<K>>>
    : never;
};

export type Me = components['schemas']['Me'];

const keyOf = (request: Request) => `${request.method} ${new URL(request.url).pathname}`;

// A template key ("GET /api/v1/products/{productId}") answers any concrete ID.
function templateMatcher(key: string) {
  const source = key.replace(/[.*+?^$()|[\]\\]/g, '\\$&').replace(/\{[^}]+\}/g, '[^/]+');
  return new RegExp(`^${source}$`);
}

// fakeApi replaces the global fetch the API client uses. Keys are "METHOD /path",
// with a concrete path or the contract's template; the exact key wins. A route
// without a stub answers 404 so a missing stub shows up as a failing assertion.
export function fakeApi<const R extends Record<string, Stub<unknown>>>(routes: R & Checked<R>) {
  const requests: Request[] = [];
  const stubs = routes as Record<string, Stub<unknown> | undefined>;
  const templates = Object.keys(stubs)
    .filter((key) => key.includes('{'))
    .map((key) => ({ key, pattern: templateMatcher(key) }));

  vi.stubGlobal('fetch', async (request: Request) => {
    requests.push(request.clone());
    const key = keyOf(request);
    const route = stubs[key] ?? stubs[templates.find((t) => t.pattern.test(key))?.key ?? ''];
    if (!route) return json(404, { error: { code: 'not_found', message: 'No existe' } });
    return typeof route === 'function' ? route(request) : route.clone();
  });

  return {
    // calls lists the requests sent to a route, in order.
    calls: (key: string) => requests.filter((r) => keyOf(r) === key),
  };
}

export function json<B>(status: number, body: B): Reply<B> {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
}

/** An API error reply: apiError(409, { code: 'sku_taken', message: '…', details }). */
export function apiError(
  status: number,
  error: components['schemas']['Error']['error'],
): Reply<components['schemas']['Error']> {
  return json(status, { error });
}

const permissionsByRole: Record<Me['role'], string[]> = {
  owner: [
    'catalog:read',
    'catalog:write',
    'imports:run',
    'inventory:read',
    'inventory:adjust',
    'inventory:count',
    'inventory:count_manage',
    'sales:create',
    'sales:discount',
    'sales:read',
    'sales:void',
    'users:manage',
    'tenant:manage',
  ],
  admin: [
    'catalog:read',
    'catalog:write',
    'imports:run',
    'inventory:read',
    'inventory:adjust',
    'inventory:count',
    'inventory:count_manage',
    'sales:create',
    'sales:discount',
    'sales:read',
    'sales:void',
    'users:manage',
  ],
  cashier: ['catalog:read', 'inventory:read', 'inventory:count', 'sales:create', 'sales:read'],
  warehouse: ['catalog:read', 'inventory:read', 'inventory:adjust', 'inventory:count'],
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

export function anAuthSession(user = aUser()): components['schemas']['AuthSession'] {
  return { access_token: 'access-token', token_type: 'Bearer', expires_in: 900, user };
}

// signedIn stubs a session restored from the refresh cookie (GET /me answers the user).
export function signedIn(user = aUser()) {
  return { 'GET /api/v1/me': json(200, user) };
}

// signedOut stubs a visitor without a session: /me and the refresh both answer 401.
export function signedOut() {
  return {
    'GET /api/v1/me': apiError(401, { code: 'unauthorized', message: 'Inicie sesión' }),
    'POST /api/v1/auth/refresh': apiError(401, { code: 'invalid_session', message: 'La sesión venció' }),
  };
}
