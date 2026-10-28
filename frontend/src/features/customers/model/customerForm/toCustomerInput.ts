import type { CustomerInput } from '../../api/types';
import { optionalCustomerFields, type CustomerValues } from './customerSchema';

/** The body of POST and PUT /customers: an empty optional field is left out, which the API stores as absent. */
export function toCustomerInput(values: CustomerValues): CustomerInput {
  const input: CustomerInput = {
    person_type: values.person_type,
    document_type: values.document_type,
    document_number: values.document_number.trim(),
    name: values.name.trim(),
    tax_responsibilities: values.tax_responsibilities,
    active: values.active,
  };
  for (const field of optionalCustomerFields) {
    const value = values[field].trim();
    if (value) input[field] = value;
  }
  return input;
}
