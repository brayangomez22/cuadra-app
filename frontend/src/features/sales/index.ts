// The public API of sales. The router takes the routes from routes.tsx.
// Export here what another feature needs, when it needs it.

// Cash that leaves the user's open drawer for something that is not a sale (a supplier paid in cash).
export { CashOutOption } from './components/CashOutOption/CashOutOption';
export { useCashOut } from './hooks/useCashOut';
