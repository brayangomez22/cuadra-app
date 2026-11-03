import { useState } from 'react';
import { useWatch, type Control } from 'react-hook-form';
import { useCashOut } from '@/features/sales';
import { parseMoneyInput } from '@/shared/format';
import type { Purchase } from '../../../api/types';
import type { PaymentFormInput } from '../../../model/payment/paymentSchema';

type Args = { purchase: Purchase; supplierName: string; control: Control<PaymentFormInput> };

/**
 * A payment in cash can come out of the user's open drawer, recorded as an out
 * movement after the payment, so the closing still adds up. Offered, on by
 * default, only for cash and with a drawer open; without one the dialog only
 * says to record it.
 */
export function usePaymentCashOut({ purchase, supplierName, control }: Args) {
  const cashOut = useCashOut();
  const [checked, setChecked] = useState(true);
  const [method, typed] = useWatch({ control, name: ['method', 'amount'] });
  const parsed = parseMoneyInput(typed);
  const inCash = method === 'cash';
  const offered = inCash && cashOut.drawer !== undefined && parsed.ok;

  const afterPayment = (amount: string) =>
    offered && checked
      ? cashOut.takeOut({
          amount,
          reason: `Pago a ${supplierName}, factura ${purchase.invoice_number}`,
          done: 'El pago quedó registrado',
        })
      : Promise.resolve(true);

  return {
    option: offered
      ? { amount: parsed.value, location: cashOut.drawer?.location ?? '', checked, onChange: setChecked }
      : undefined,
    /** Cash without a drawer open: the money still leaves the drawer by hand. */
    recordByHand: inCash && cashOut.drawer === undefined,
    afterPayment,
  };
}
