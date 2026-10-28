import type { Customer } from '../api/types';

const shortNames: Record<Customer['document_type'], string> = { cc: 'CC', nit: 'NIT', ce: 'CE', passport: 'Pasaporte' };

/** A customer's document as a person reads it: "NIT 900123456-8", "CC 1020304050". */
export function documentLabel(customer: Pick<Customer, 'document_type' | 'document_number'>) {
  return `${shortNames[customer.document_type]} ${customer.document_number}`;
}
