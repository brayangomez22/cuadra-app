import type { components } from '@/shared/api/schema';
import { apiError, fakeApi, json } from '@/test/fakeApi';
import { aGarment, ids, variantId, type Product, type ProductVariant } from '@/test/fixtures';
import { axisLabel } from '../model/variantMatrix/variantMatrix';
import { sentBody } from './sentBody';

type ProductInput = components['schemas']['ProductInput'];
type WithVariantsInput = components['schemas']['ProductWithVariantsInput'];
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

const idOf = ({ axis_values: { talla = '', color = '' } }: VariantBody) => variantId(talla, color);

const taken = (sku: string) =>
  apiError(409, { code: 'sku_taken', message: `Ya existe un producto con el SKU ${sku}`, details: { sku } });

type Options = {
  /** The variant whose SKU is rejected as taken. */
  failSku?: string;
  /** The garment as it is stored, to edit it; without it, the server creates it. */
  initial?: Product;
};

/**
 * Answers like the API, keeping the garment it creates or edits: a new
 * product comes with its variants (products-with-variants) or only its
 * default one; adding variants grows it.
 */
export function garmentServer({ failSku, initial }: Options = {}) {
  let product = initial ?? aGarment({ variants: [] });
  const variants = () => product.variants;
  const reply = (next: Product, status = 200) => {
    product = next;
    return json(status, product);
  };
  return {
    [`GET /api/v1/products/${ids.garment}`]: () => json(200, product),
    'POST /api/v1/products-with-variants': async (request: Request) => {
      const body = await sentBody<WithVariantsInput>(request);
      const failed = body.variants.find((variant) => variant.sku === failSku);
      if (failed) return taken(failed.sku);
      const created = body.variants.map((variant, index) => fromBody(variant, idOf(variant), index === 0));
      return reply(aGarment({ name: body.name, variants: created }), 201);
    },
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
      if (body.sku === failSku) return taken(body.sku);
      const variant = fromBody(body, idOf(body), false);
      return reply({ ...product, variants: [...variants(), variant] }, 201);
    },
  } satisfies Parameters<typeof fakeApi>[0];
}
