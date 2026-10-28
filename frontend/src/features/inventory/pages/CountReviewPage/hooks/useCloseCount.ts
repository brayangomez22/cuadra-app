import { useState } from 'react';
import { useNavigate } from 'react-router';
import { paths } from '@/app/routes/paths';
import { describeError } from '@/shared/api/errors';
import { useToast } from '@/shared/ui';
import { useApproveStockCount, useCancelStockCount } from '../../../api/stockCountReview';

/**
 * Approving or canceling the count: on success, back to the counts with a
 * toast; on error, the API's message stays in the dialog (a product under
 * review, or one sold after counting it).
 */
export function useCloseCount(countId: string) {
  const navigate = useNavigate();
  const { toast } = useToast();
  const approve = useApproveStockCount(countId);
  const cancel = useCancelStockCount(countId);
  const [error, setError] = useState<string>();

  const run = (mutation: typeof approve, title: string) => () => {
    setError(undefined);
    mutation.mutate(undefined, {
      onSuccess: () => {
        toast({ tone: 'success', title });
        void navigate(paths.inventory.counts());
      },
      onError: (failure) => setError(describeError(failure).message),
    });
  };

  return {
    error,
    clearError: () => setError(undefined),
    approve: run(approve, 'Conteo aprobado: el inventario quedó ajustado'),
    cancel: run(cancel, 'Conteo cancelado'),
    isApproving: approve.isPending,
    isCanceling: cancel.isPending,
  };
}
