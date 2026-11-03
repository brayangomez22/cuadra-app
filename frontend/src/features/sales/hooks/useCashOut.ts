import { useLocations } from '@/features/inventory';
import { formatMoney } from '@/shared/format';
import { useToast } from '@/shared/ui';
import { useCurrentCashSession, useRecordCashMovement } from '../api/cashSessions';

type TakeOut = {
  amount: string;
  reason: string;
  /** What already happened, for the error: «La venta N.º 15 quedó anulada». */
  done: string;
};

/**
 * Cash that leaves the drawer for something that is not a sale (a refund, a
 * supplier paid in cash), taken out of the user's open drawer as an out
 * movement, so the closing still adds up. drawer is undefined when the user
 * has none open: then the screen only says to record it.
 */
export function useCashOut() {
  const session = useCurrentCashSession().data?.session;
  const location = useLocations().data?.find((l) => l.id === session?.location_id)?.name;
  const record = useRecordCashMovement(session?.id ?? '');
  const { toast } = useToast();

  // The operation already went through: a failed out movement must not hide
  // it, only say what is left to do by hand.
  const takeOut = async ({ amount, reason, done }: TakeOut) => {
    try {
      await record.mutateAsync({ kind: 'out', amount, reason });
      return true;
    } catch {
      toast({
        tone: 'danger',
        title: `${done}, pero no pudimos registrar el egreso`,
        description: `Registre la salida de ${formatMoney(amount)} en Caja → Registrar egreso.`,
      });
      return false;
    }
  };

  return { drawer: session && { location: location ?? '' }, takeOut };
}

export type CashOut = ReturnType<typeof useCashOut>;
