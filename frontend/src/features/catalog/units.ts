import type { UnitOfMeasure } from './api';

// The fixed units of the catalog (api/openapi.yaml, UnitOfMeasure).
const units: Record<UnitOfMeasure, { name: string; short: string }> = {
  und: { name: 'Unidad (und)', short: 'und' },
  m: { name: 'Metro (m)', short: 'm' },
  kg: { name: 'Kilogramo (kg)', short: 'kg' },
  l: { name: 'Litro (l)', short: 'l' },
  caja: { name: 'Caja', short: 'caja' },
  rollo: { name: 'Rollo', short: 'rollo' },
  bulto: { name: 'Bulto', short: 'bulto' },
  galon: { name: 'Galón', short: 'galón' },
};

export const unitOptions = Object.entries(units).map(([value, unit]) => ({ value, label: unit.name }));

/** The short name shown next to a quantity: "2,5 bulto". */
export function unitShort(unit: UnitOfMeasure): string {
  return units[unit].short;
}
