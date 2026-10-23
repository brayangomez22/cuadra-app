import { apiError, fakeApi, json } from './fakeApi';
import { aProduct, anOpenCashSession } from './fixtures';

const productId = '01920000-0000-7000-8000-000000000101';
const product = aProduct({ id: productId });
const get = (path: string) => fetch(new Request(`http://localhost${path}`));

describe('fakeApi', () => {
  it('resuelve /api/v1/products/{productId} para un ID concreto', async () => {
    fakeApi({ 'GET /api/v1/products/{productId}': json(200, product) });

    const response = await get(`/api/v1/products/${productId}`);

    expect(response.status).toBe(200);
    expect(await response.json()).toEqual(product);
  });

  it('prefiere la llave exacta sobre la plantilla', async () => {
    fakeApi({
      'GET /api/v1/cash-sessions/{cashSessionId}': json(200, anOpenCashSession({ id: 'por plantilla' })),
      'GET /api/v1/cash-sessions/current': json(200, { required: true, session: anOpenCashSession({ id: 'exacta' }) }),
    });

    expect(await (await get('/api/v1/cash-sessions/current')).json()).toMatchObject({ session: { id: 'exacta' } });
  });

  it('no confunde un segmento de más (products/{id} frente a products/{id}/kardex)', async () => {
    fakeApi({ 'GET /api/v1/products/{productId}': json(200, product) });

    expect((await get(`/api/v1/products/${productId}/kardex`)).status).toBe(404);
  });

  it('responde 404 a una ruta sin mock', async () => {
    fakeApi({});

    const response = await get('/api/v1/products');

    expect(response.status).toBe(404);
    expect(await response.json()).toEqual({ error: { code: 'not_found', message: 'No existe' } });
  });

  it('calls lista las peticiones de una ruta', async () => {
    const api = fakeApi({ [`GET /api/v1/products/${productId}`]: apiError(404, 'not_found', 'No existe') });

    await get(`/api/v1/products/${productId}`);
    await get('/api/v1/products');

    expect(api.calls(`GET /api/v1/products/${productId}`)).toHaveLength(1);
  });

  it('rechaza en tsc una ruta que no existe en el contrato', () => {
    // @ts-expect-error: /api/v1/productos is not a path of api/openapi.yaml.
    fakeApi({ 'GET /api/v1/productos': json(200, []) });
    // @ts-expect-error: the products/{productId} path has no DELETE.
    fakeApi({ [`DELETE /api/v1/products/${productId}`]: json(200, {}) });
  });

  it('rechaza en tsc un cuerpo que no calza con el esquema', () => {
    // @ts-expect-error: a product's name is a string.
    fakeApi({ 'GET /api/v1/products/{productId}': json(200, { ...product, name: 42 }) });
  });
});
