import { ids } from './ids';
import type { Product, ProductVariant } from './types';
import { unitOf } from './units';

const sizes = ['S', 'M', 'L'];
const colors = ['Rojo', 'Azul'];

/** The id of the garment's variant in a size and color: variantId('M', 'Azul'). */
export function variantId(size: string, color: string) {
  const index = sizes.indexOf(size) * 2 + colors.indexOf(color) + 1;
  return `01920000-0000-7000-8000-00000000060${index}`;
}

/** The barcode of the garment's variant in a size and color. */
export function variantBarcode(size: string, color: string) {
  return `770999${variantId(size, color).slice(-6)}`;
}

function garmentVariant(size: string, color: string, isDefault: boolean): ProductVariant {
  return {
    id: variantId(size, color),
    sku: `CAM-${size}-${color.toUpperCase()}`,
    barcode: variantBarcode(size, color),
    cost: '15000.00',
    price: size === 'L' ? '27000.00' : '25000.00',
    price_with_tax: size === 'L' ? '32130.00' : '29750.00',
    margin_percent: '40',
    axis_values: { talla: size, color },
    // Sorted by axis name, as ProductVariant.Label(): color before talla.
    label: `${color} · ${size}`,
    units: [],
    is_default: isDefault,
    active: true,
    created_at: '2026-10-16T15:00:00Z',
  };
}

/**
 * aGarment is a product with 6 variants (3 sizes × 2 colors) created after
 * variants existed: its default variant (S, Rojo) has its own id.
 */
export function aGarment(overrides: Partial<Product> = {}): Product {
  const variants = sizes.flatMap((size, i) => colors.map((color, j) => garmentVariant(size, color, i + j === 0)));
  const unit = unitOf('und');
  return {
    id: ids.garment,
    name: 'Camiseta básica',
    base_unit_id: unit.id,
    unit,
    tax_rate: '0.19',
    attributes: { marca: 'Acme', material: 'Algodón' },
    active: true,
    created_at: '2026-10-16T15:00:00Z',
    variants,
    ...overrides,
  };
}
