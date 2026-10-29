import type { Supplier } from '../../api/types';
import { optionalSupplierFields, type SupplierInputValues } from './supplierSchema';

type OptionalText = Record<(typeof optionalSupplierFields)[number], string>;

/** The form's values: a new active supplier paid in cash, or the supplier to edit. */
export function supplierDefaults(supplier?: Supplier): SupplierInputValues {
  if (!supplier) return { name: '', nit: '', ...optionalText({}), payment_term_days: '0', active: true };
  return {
    name: supplier.name,
    nit: supplier.nit,
    ...optionalText(supplier),
    payment_term_days: String(supplier.payment_term_days),
    active: supplier.active,
  };
}

// An absent contact field is an empty input.
function optionalText(supplier: Partial<OptionalText>): OptionalText {
  return Object.fromEntries(optionalSupplierFields.map((field) => [field, supplier[field] ?? ''])) as OptionalText;
}
