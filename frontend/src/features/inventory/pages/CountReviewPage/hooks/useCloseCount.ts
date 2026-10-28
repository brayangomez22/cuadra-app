import { useState } from 'react';
import { useNavigate } from 'react-router';
import { paths } from '@/app/routes/paths';
import { ApiError, describeError } from '@/shared/api/errors';
import { useToast } from '@/shared/ui';
import { useApproveStockCount, useCancelStockCount } from '../../../api/stockCountReview';

/**
 * Approving or canceling the count: on success, back to the counts with a
 * toast; on error, the API's message stays in the dialog (a product under
 * review, or one sold after counting it). For the latter, short is the
 * variant the adjustment would leave without stock.
 */
export function useCloseCount(countId: string) {
  const navigate = useNavigate();
  const { toast } = useToast();
  const approve = useApproveStockCount(countId);
  const cancel = useCancelStockCount(countId);
  const [error, setError] = useState<string>();
  const [short, setShort] = useState<string>();

  const clearError = () => {
    setError(undefined);
    setShort(undefined);
  };

  const run = (mutation: typeof approve, title: string) => () => {
    clearError();
    mutation.mutate(undefined, {
      onSuccess: () => {
        toast({ tone: 'success', title });
        void navigate(paths.inventory.counts());
      },
      onError: (failure) => {
        setError(describeError(failure).message);
        if (failure instanceof ApiError && failure.code === 'insufficient_stock') setShort(failure.details.variant_id);
      },
    });
  };

  return {
    error,
    short,
    clearError,
    approve: run(approve, 'Conteo aprobado: el inventario quedó ajustado'),
    cancel: run(cancel, 'Conteo cancelado'),
    isApproving: approve.isPending,
    isCanceling: cancel.isPending,
  };
}

export type CloseCount = ReturnType<typeof useCloseCount>;
