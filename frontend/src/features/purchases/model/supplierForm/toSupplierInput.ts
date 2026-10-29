import type { SupplierInput } from '../../api/types';
import { optionalSupplierFields, type SupplierValues } from './supplierSchema';

/** The body of POST and PUT /suppliers: an empty contact field is left out, which the API stores as absent. */
export function toSupplierInput(values: SupplierValues): SupplierInput {
  const input: SupplierInput = {
    name: values.name.trim(),
    nit: values.nit.trim(),
    payment_term_days: values.payment_term_days,
    active: values.active,
  };
  for (const field of optionalSupplierFields) {
    const value = values[field].trim();
    if (value) input[field] = value;
  }
  return input;
}
