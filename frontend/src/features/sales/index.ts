// The public API of sales. The router takes the routes from routes.tsx.
// Export here what another feature needs, when it needs it.

// Cash that leaves the user's open drawer for something that is not a sale (a supplier paid in cash).
export { CashOutOption } from './components/CashOutOption/CashOutOption';
export { useCashOut } from './hooks/useCashOut';

/** How the cashier, the receipt and the reports name each payment method. */
export { methodLabels } from './model/payments/paymentMethods';

/** The drawers open right now, with the button to close each one, for the owner's home screen. */
export { OpenCashSessions } from './components/OpenCashSessions/OpenCashSessions';
