import { formatMoney } from '@/shared/format';
import { fromUnits, toUnits } from '@/shared/money/decimal';
import type { Sale } from '../../api/types';
import { methodLabels } from '../payments/paymentMethods';

/** The cash session of a sale: voiding one does not touch a closed drawer (decision of 2026-10-14). */
export type DrawerState = 'open' | 'closed' | 'none';

function paidWith(sale: Sale, method: Sale['payments'][number]['method']): bigint {
  return sale.payments.filter((p) => p.method === method).reduce((sum, p) => sum + toUnits(p.amount), 0n);
}

/** The cash the customer left in the drawer: what they paid in cash less the change. */
export function cashToReturn(sale: Sale): string {
  return fromUnits(paidWith(sale, 'cash') - toUnits(sale.change));
}

/** fromMyDrawer: the cash comes out of the user's open drawer, recorded as an out movement along with the void. */
type Options = { fromMyDrawer?: boolean };

function cashNotice(amount: string, drawer: DrawerState, { fromMyDrawer = false }: Options): string {
  const cash = `Devuelva ${formatMoney(amount)} en efectivo`;
  if (drawer === 'open') return `${cash}. La caja deja de contar esta venta.`;
  const why = drawer === 'closed' ? 'la caja de esta venta ya se cerró' : 'esta venta no quedó en ninguna caja';
  if (fromMyDrawer) return `${cash} de su caja: ${why}, así que sale como egreso de la suya.`;
  return `${cash} y regístrelo como egreso de la caja actual: ${why}.`;
}

/** What voiding the sale does, and what the user must do with the money: the API only reverses stock and debt. */
export function voidNotices(sale: Sale, drawer: DrawerState, options: Options = {}): string[] {
  const notices = ['Lo vendido vuelve al inventario.'];
  const cash = cashToReturn(sale);
  if (toUnits(cash) > 0n) notices.push(cashNotice(cash, drawer, options));
  for (const { method, amount } of sale.payments) {
    if (method === 'credit') {
      notices.push(`Se cancela la deuda de ${formatMoney(amount)} que dejó esta venta en la cuenta del cliente.`);
    } else if (method !== 'cash') {
      const label = methodLabels[method];
      notices.push(`Devuelva ${formatMoney(amount)} de ${label} por ese mismo medio: Cuadra no mueve ese dinero.`);
    }
  }
  return notices;
}
