// The public API of customers: the segments (settings edits them), the
// picker the POS chooses who buys with, and a customer with its document
// (receivables titles a customer's account with them). The router takes the routes from
// routes.tsx, which loads pages.ts lazily.
export { useCustomer } from './api/customers';
export { customersKeys } from './api/keys';
export { useCustomerSegments } from './api/segments';
export type { ChosenCustomer, CustomerSegment } from './api/types';
export { CustomerPicker } from './components/CustomerPicker/CustomerPicker';
export { documentLabel } from './model/documentLabel';
