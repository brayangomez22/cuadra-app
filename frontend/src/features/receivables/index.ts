// The public API of receivables: the customer's credit, which the POS reads to
// offer selling on credit, and its keys, which a confirmed sale refreshes. The
// router takes the routes from routes.tsx, which loads pages.ts lazily.
export { useCustomerCredit } from './api/credit';
export { receivablesKeys } from './api/keys';
