import { useState } from 'react';
import { formatMoney, isZero } from '@/shared/format';
import type { Sale } from '../../../api/types';
import { useCashOut } from '../../../hooks/useCashOut';
import { cashToReturn, type DrawerState } from '../../../model/void/voidNotice';

/**
 * Whether the cash to give back comes out of the user's open drawer: offered,
 * and on by default, when the sale's own drawer cannot take it back (closed,
 * or none) and the user has one open.
 */
export function useVoidCashOut(sale: Sale, drawer: DrawerState | undefined) {
  const cashOut = useCashOut();
  const [checked, setChecked] = useState(true);
  const amount = cashToReturn(sale);
  const offered = drawer !== undefined && drawer !== 'open' && !isZero(amount) && cashOut.drawer !== undefined;
  const taking = offered && checked;

  // After the void: the out movement, if it was chosen. The description of the success toast, or null if it failed.
  const afterVoid = async (): Promise<string | undefined | null> => {
    if (!taking) return undefined;
    const reason = `Devolución por anular la venta N.º ${String(sale.number)}`;
    const ok = await cashOut.takeOut({ amount, reason, done: `La venta N.º ${String(sale.number)} quedó anulada` });
    return ok ? `Salieron ${formatMoney(amount)} de su caja.` : null;
  };

  return {
    option: offered ? { amount, location: cashOut.drawer?.location ?? '', checked, onChange: setChecked } : undefined,
    taking,
    afterVoid,
  };
}
