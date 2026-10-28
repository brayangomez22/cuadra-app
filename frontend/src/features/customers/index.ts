// The public API of customers: the segments (settings edits them) and the
// picker the POS chooses who buys with. The router takes the routes from
// routes.tsx, which loads pages.ts lazily.
export { customersKeys } from './api/keys';
export { useCustomerSegments } from './api/segments';
export type { ChosenCustomer, CustomerSegment } from './api/types';
export { CustomerPicker } from './components/CustomerPicker/CustomerPicker';
