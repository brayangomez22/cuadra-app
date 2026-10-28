import type { Product } from '@/features/catalog';
import { useMutationToast } from '@/shared/hooks/useMutationToast';
import { Button } from '@/shared/ui';
import { useRequestRecount } from '../../../../api/stockCountReview';
import { rowVariant } from '../../../../model/rowVariant';

type Props = { countId: string; variantId: string; product?: Product };

/** Asks to count a doubtful product again: what was counted of it no longer counts. */
export function RecountButton({ countId, variantId, product }: Props) {
  const recount = useRequestRecount(countId);
  const name = rowVariant(product, variantId).name ?? 'el producto';
  const toasts = useMutationToast({
    success: `${name} quedó por contar de nuevo`,
    error: 'No se pudo pedir el recuento',
  });
  return (
    <Button
      variant="secondary"
      aria-label={`Recontar ${name}`}
      loading={recount.isPending}
      onClick={() => recount.mutate([variantId], toasts)}
    >
      Recontar
    </Button>
  );
}
