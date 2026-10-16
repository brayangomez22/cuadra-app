// The units every tenant started with (api/openapi.yaml, UnitOfMeasure). Each
// tenant now has its own units; the product form keeps offering these by
// code until T24j moves it to base_unit_id and the tenant's list.
const units: Record<string, { name: string; short: string }> = {
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

/** The short name shown next to a quantity: "2,5 bulto". A unit the tenant created shows its code. */
export function unitShort(code: string): string {
  return units[code]?.short ?? code;
}
