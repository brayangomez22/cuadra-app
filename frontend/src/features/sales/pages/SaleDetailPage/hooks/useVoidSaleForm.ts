import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { isZero } from '@/shared/format';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { useToast } from '@/shared/ui';
import { useCashSession } from '../../../api/cashSessions';
import { useVoidSale } from '../../../api/sales';
import type { Sale } from '../../../api/types';
import { cashToReturn, voidNotices, type DrawerState } from '../../../model/void/voidNotice';
import { voidSaleSchema, type VoidSaleValues } from '../../../model/void/voidSaleSchema';

const fields = { invalid_void_reason: 'reason' } as const;

// What the drawer's state is only matters when cash has to go back. Mounted per
// opening, so a drawer closed since the last time is read again.
function useDrawerState(sale: Sale) {
  const sessionId = sale.cash_session_id;
  const needed = Boolean(sessionId) && !isZero(cashToReturn(sale));
  const session = useCashSession(sessionId ?? '', { enabled: needed });
  if (!needed) return { drawer: 'none' as DrawerState, failed: false, ready: true };
  const drawer: DrawerState = session.data?.status === 'open' ? 'open' : 'closed';
  return { drawer, failed: session.isError, ready: session.isSuccess };
}

/** The reason of the void, what it will do with the money, and sending it. */
export function useVoidSaleForm(sale: Sale, onDone: () => void) {
  const voidSale = useVoidSale(sale.id);
  const { toast } = useToast();
  const { drawer, failed, ready } = useDrawerState(sale);
  const form = useForm<VoidSaleValues>({ resolver: zodResolver(voidSaleSchema), defaultValues: { reason: '' } });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, fields);

  const submit = form.handleSubmit(({ reason }) => {
    clearFormError();
    voidSale.mutate(reason, {
      onSuccess: () => {
        toast({ tone: 'success', title: `Venta N.º ${sale.number} anulada` });
        onDone();
      },
      onError: handleError,
    });
  });

  return {
    form,
    submit,
    formError,
    notices: ready ? voidNotices(sale, drawer) : [],
    drawerFailed: failed,
    canSubmit: ready && !voidSale.isPending,
    pending: voidSale.isPending,
  };
}
