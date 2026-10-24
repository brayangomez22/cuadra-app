import type { UnitOfMeasure } from './types';

// unitOf is the tenant's unit with a code, as the API embeds it in a product.
const unitNames: Record<string, string> = {
  und: 'Unidad',
  m: 'Metro',
  bulto: 'Bulto',
  caja: 'Caja',
  rollo: 'Rollo',
  paq: 'Paquete',
};

export function unitOf(code: string): UnitOfMeasure {
  const index = Object.keys(unitNames).indexOf(code) + 1;
  return {
    id: `01920000-0000-7000-8000-00000000050${index}`,
    code,
    name: unitNames[code] ?? code,
  };
}

/** The tenant's units, as GET /units answers them. */
export const units = Object.keys(unitNames).map((code) => unitOf(code));
