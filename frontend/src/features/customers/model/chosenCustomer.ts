import type { ChosenCustomer, Customer } from '../api/types';
import { customerDefaults } from './customerForm/customerDefaults';
import type { CustomerValues } from './customerForm/customerSchema';
import { documentLabel } from './documentLabel';

/** What the POS keeps of the customer of a sale: enough to show it and print it on the receipt. */
export function chosenCustomer(customer: Customer): ChosenCustomer {
  return { id: customer.id, name: customer.name, document: documentLabel(customer) };
}

/**
 * The quick creation starts from what the cashier searched: digits are the
 * document's number, anything else the name.
 */
export function quickCustomerDefaults(searched: string): CustomerValues {
  const term = searched.trim();
  const isDocument = /^[\d.\-\s]+$/.test(term);
  return { ...customerDefaults(), document_number: isDocument ? term : '', name: isDocument ? '' : term };
}
