import type { components } from '@/shared/api/schema';
import { apiError, fakeApi, json } from '@/test/fakeApi';
import { aGarment, ids, variantId, type Product, type ProductVariant } from '@/test/fixtures';
import { axisLabel } from '../model/variantMatrix/variantMatrix';
import { sentBody } from './sentBody';

type ProductInput = components['schemas']['ProductInput'];
type VariantBody = components['schemas']['ProductVariantInput'] & { active?: boolean };

function fromBody(body: VariantBody, id: string, isDefault: boolean): ProductVariant {
  return {
    id,
    sku: body.sku.toUpperCase(),
    price: body.price,
    price_with_tax: body.price,
    cost: body.cost,
    axis_values: body.axis_values,
    label: axisLabel(body.axis_values),
    units: [],
    is_default: isDefault,
    active: body.active !== false,
    created_at: '2026-10-17T15:00:00Z',
  };
}

const defaultId = variantId('S', 'Rojo');

/**
 * Answers like the API, keeping the garment it creates: a new product has
 * only its default variant; adding variants grows it. The variant whose SKU
 * is failSku is rejected as taken.
 */
export function garmentServer({ failSku }: { failSku?: string } = {}) {
  let product = aGarment({ variants: [] });
  const variants = () => product.variants;
  const reply = (next: Product, status = 200) => {
    product = next;
    return json(status, product);
  };
  return {
    'POST /api/v1/products': async (request: Request) => {
      const body = await sentBody<ProductInput>(request);
      const variant = fromBody({ ...body, axis_values: {} }, defaultId, true);
      return reply(aGarment({ name: body.name, variants: [variant] }), 201);
    },
    [`PUT /api/v1/products/${ids.garment}`]: async (request: Request) => {
      const { sku } = await sentBody<ProductInput>(request);
      const [first, ...rest] = variants();
      return reply({ ...product, variants: first ? [{ ...first, sku }, ...rest] : rest });
    },
    [`PUT /api/v1/products/${ids.garment}/variants/${defaultId}`]: async (request: Request) => {
      const variant = fromBody(await sentBody<VariantBody>(request), defaultId, true);
      return reply({ ...product, variants: [variant, ...variants().slice(1)] });
    },
    [`POST /api/v1/products/${ids.garment}/variants`]: async (request: Request) => {
      const body = await sentBody<VariantBody>(request);
      if (body.sku === failSku) {
        return apiError(409, { code: 'sku_taken', message: `Ya existe un producto con el SKU ${body.sku}` });
      }
      const { talla = '', color = '' } = body.axis_values;
      const variant = fromBody(body, variantId(talla, color), false);
      return reply({ ...product, variants: [...variants(), variant] }, 201);
    },
  } satisfies Parameters<typeof fakeApi>[0];
}
