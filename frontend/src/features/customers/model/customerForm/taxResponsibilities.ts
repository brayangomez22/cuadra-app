import type { TaxResponsibility } from '../../api/types';

/**
 * The RUT's responsibilities the electronic invoice asks for, as the RUT
 * names them. The codes use a non-breaking hyphen: "(O-47)" never splits.
 */
export const taxResponsibilityOptions: { value: TaxResponsibility; label: string }[] = [
  { value: 'O-13', label: 'Gran contribuyente (O\u201113)' },
  { value: 'O-15', label: 'Autorretenedor (O\u201115)' },
  { value: 'O-23', label: 'Agente de retención de IVA (O\u201123)' },
  { value: 'O-47', label: 'Régimen simple de tributación (O\u201147)' },
  { value: 'R-99-PN', label: 'No responsable (R\u201199\u2011PN)' },
];

/** R-99-PN means "none of the others": it stands alone. */
export const NONE: TaxResponsibility = 'R-99-PN';

/**
 * Checks or unchecks a responsibility, in the RUT's order. Checking
 * R-99-PN leaves it alone; checking another one takes R-99-PN out.
 */
export function toggleResponsibility(current: TaxResponsibility[], value: TaxResponsibility, checked: boolean) {
  if (!checked) return current.filter((v) => v !== value);
  if (value === NONE) return [NONE];
  const next = new Set([...current.filter((v) => v !== NONE), value]);
  return taxResponsibilityOptions.map((o) => o.value).filter((v) => next.has(v));
}
