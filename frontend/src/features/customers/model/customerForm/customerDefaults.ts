import type { Customer } from '../../api/types';
import { optionalCustomerFields, type CustomerValues } from './customerSchema';
import { NONE } from './taxResponsibilities';

type OptionalText = Record<(typeof optionalCustomerFields)[number], string>;

/** The form's values: a new customer is a natural person with a CC, or the customer to edit. */
export function customerDefaults(customer?: Customer): CustomerValues {
  if (!customer) {
    return {
      person_type: 'natural',
      document_type: 'cc',
      document_number: '',
      name: '',
      ...optionalText({}),
      tax_responsibilities: [NONE],
      active: true,
    };
  }
  return {
    person_type: customer.person_type,
    document_type: customer.document_type,
    document_number: customer.document_number,
    name: customer.name,
    ...optionalText(customer),
    tax_responsibilities: customer.tax_responsibilities,
    active: customer.active,
  };
}

// An absent optional field is an empty input.
function optionalText(customer: Partial<OptionalText>): OptionalText {
  return Object.fromEntries(optionalCustomerFields.map((field) => [field, customer[field] ?? ''])) as OptionalText;
}
